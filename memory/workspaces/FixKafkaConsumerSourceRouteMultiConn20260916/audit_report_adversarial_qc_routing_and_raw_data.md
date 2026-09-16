# BÁO CÁO QC PHẢN BIỆN CHUYÊN SÂU (ADVERSARIAL AUDIT REPORT)
## RÀ SOÁT TOÀN DIỆN MÃ NGUỒN ĐỊNH TUYẾN ĐA KẾT NỐI SOURCE & MASTER VÀ CHUẨN HÓA `_raw_data`

---
**Thời gian kiểm định:** 2026-09-16T15:15:00+07:00  
**Người thực hiện:** Role: BRAIN (CHAIRMAN & ARCHITECT)  
**Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`  
**Mục tiêu:** Thực hiện kiểm định đối kháng (Adversarial Audit), soi từng dòng mã nguồn đã update, đối chiếu 1-1 với logic tài liệu Plan, rà soát lỗ hổng tiềm ẩn, kiểm tra tính chân thực (chống báo cáo láo/suy diễn), và áp dụng Self-Improvement Loop.

---

## I. BẢNG ĐỐI CHIẾU 1-1: CÁC TỆP TIN ĐÃ SỬA ĐỔI VS LOGIC TÀI LIỆU PLAN

| STT | Tệp tin mã nguồn thực tế trên đĩa | Trạng thái trên đĩa | Đánh giá khớp Plan | Điểm rà soát chi tiết |
|:---:|:---|:---:|:---:|:---|
| 1 | `internal/model/shadow/cdc_event.go` | Đã sửa (L9-11) | Khớp 100% | Bổ sung `SourceConn`, `SourceDB`, `SourceTable` vào struct `CDCEvent`. Giữ nguyên toàn bộ các trường OCC, Kafka metadata. |
| 2 | `internal/service/metadata/metadata_registry.go` | Đã sửa (L14, L18-22, L41) | Khớp 100% | Mở rộng variadic `sourceConn ...string` cho `ResolveSourceRoute`, `ResolveSourceRoutes`, `GetTableConfigBySource`. Thêm `SourceConnectionKey` vào `ResolvedSourceRoute`. |
| 3 | `internal/service/source/metadata_registry_utils.go` | Đã sửa (L160-192) | Khớp 95% | `buildRouteLookupKeys` nhận variadic `sourceConn`. Xử lý candidates (gốc, cắt prefix `cdc.goopay.`, cắt prefix `cdc.`). Sinh các key ưu tiên connection. |
| 4 | `internal/service/source/metadata_registry_service.go` | Đã sửa (L218, L386-401, L553-592) | **CẢNH BÁO: Phát hiện Lỗ hổng Logic dòng 587-591** | `SourceConnectionKey` được gán chuẩn. `GetTableConfigBySource` ưu tiên `conn:table`. Tuy nhiên, trong `ResolveSourceRoutes` có kẽ hở fallback khi truyền conn lạ (Xem chi tiết Mục II). |
| 5 | `internal/service/source/registry_service.go` | Đã sửa (L29-41, L177) | Khớp 100% | Đồng bộ signature variadic `sourceConn ...string` cho `RegistryService` legacy để pass compiler. |
| 6 | `internal/handler/shadow/kafka_consumer.go` | Đã sửa (L652-692) | Khớp 95% | Trích xuất `sourceConnCode` từ `sourceRaw["name"]`, fallback topic. (Xem chi tiết Mục II về edge case 4-parts). |
| 7 | `internal/handler/shadow/event_handler.go` | Đã sửa (L167-250) | Khớp 100% | Bóc tách tương đối `table = parts[len-1]`, `db = parts[len-2]`, `sourceConn = parts[len-3]`. Truyền `sourceConn` vào `ResolveSourceRoutes`. |
| 8 | `internal/handler/source/bridge_handler.go` | Đã sửa (L201, L262-274) | Khớp 100% | `resolveCollection` nhận `connectorName`, truyền vào `ResolveSourceRoutes`. |
| 9 | `internal/handler/orchestration/snapshot_runner_utils.go` | Đã sửa (L144-155) | Khớp 100% | `buildSnapshotEnvelope` nhận variadic `sourceConn`, đóng gói `source_conn` vào JSON CDCEvent. |
| 10 | `internal/handler/orchestration/snapshot_runner_handler.go` | Đã sửa (L511, L795) | Khớp 100% | Truyền `conn.ConnectionCode` vào `ResolveSourceRoutes` và `buildSnapshotEnvelope`. |
| 11 | `internal/service/shadow/dynamic_mapper.go` | Đã sửa (L71-75, L394-463) | Khớp 100% | Viết `normalizeMongoExtJSONWithDepth` với guard `depth > 32`. Gọi ngay đầu vào `MapData` cho toàn bộ mapping rules và `_raw_data`. |
| 12 | `cdc-cms-web/src/pages/SourceConnectors.tsx` | Đã sửa (L488) | Khớp 100% | Đổi gán topic prefix MongoDB sang `${TOPIC_PREFIX_MONGODB}.${name}` khi tạo connector. |
| 13 | `metadata_registry_service_test.go` | Đã sửa (L280-420) | Khớp 90% | Bổ sung unit tests phân lập routing và unwrap Extended JSON. (Thiếu negative test case cho connection lạ). |

---

## II. PHÁT HIỆN LỖ HỔNG & ĐIỂM YẾU BẰNG TƯ DUY PHẢN BIỆN (ADVERSARIAL FINDINGS)

### 🔴 PHÁT HIỆN 1 (CRITICAL LOGIC LEAK): LỖ HỔNG FALLBACK SAI CONNECTION TRONG `ResolveSourceRoutes`
- **Vị trí:** `centralized-data-service/internal/service/source/metadata_registry_service.go`, dòng 572–592.
- **Hiện trạng mã nguồn:**
  ```go
  var conn string
  if len(sourceConn) > 0 {
      conn = strings.TrimSpace(sourceConn[0])
  }
  if conn != "" {
      cleanConn := strings.TrimPrefix(strings.TrimPrefix(conn, "cdc.goopay."), "cdc.")
      filtered := make([]*metadata.ResolvedSourceRoute, 0, len(routes))
      for _, r := range routes {
          if r != nil {
              rConn := r.SourceConnectionKey
              if rConn == conn || rConn == cleanConn {
                  filtered = append(filtered, r)
              }
          }
      }
      if len(filtered) > 0 {
          return filtered
      }
  }
  return routes // <--- LỖ HỔNG CHÍ MẠNG Ở ĐÂY!
  ```
- **Kịch bản phản biện (Adversarial Exploit):**
  1. Giả sử hệ thống đang cấu hình 1 connection active: `traitestctphs` với DB: `core-trans-proxy-history-service`, table: `trans_his`. Bảng shadow là `shadow_traitestctphs.trans_his`.
  2. Một connection thứ hai `traitestmongodevct` được khởi tạo trên Kafka Connect nhưng CHƯA ĐƯỢC TẠO HOẶC CHƯA ACTIVE trong registry metadata.
  3. Khi Kafka Consumer nhận event từ `traitestmongodevct`, nó gọi `ResolveSourceRoutes("core-trans-proxy-history-service", "trans_his", "traitestmongodevct")`.
  4. `buildRouteLookupKeys` tìm key `traitestmongodevct:core-trans-proxy-history-service|trans_his` $\rightarrow$ Không có.
  5. Nó fallback xuống key chung `core-trans-proxy-history-service|trans_his` $\rightarrow$ Trúng cache của `traitestctphs`! Biến `routes` chứa route của `traitestctphs`.
  6. Vòng lặp lọc: `r.SourceConnectionKey` là `"traitestctphs"`, khác với `conn` (`"traitestmongodevct"`). Kết quả `filtered` có độ dài = 0!
  7. Vì `len(filtered) == 0`, nó BỎ QUA `if len(filtered) > 0` và trôi xuống dòng 591: `return routes`!
  8. **HẬU QUẢ NGHIÊM TRỌNG:** Hàm trả về route của `traitestctphs`, dẫn tới dữ liệu của kết nối `traitestmongodevct` BỊ GHI TRỘM / CHÈN CHÉO vào bảng shadow của kết nối `traitestctphs`! Hoàn toàn vi phạm nguyên tắc phân lập dữ liệu!
- **Đúng:** Khi caller ĐÃ chỉ định đích danh `conn != ""`, nếu lọc không ra route nào thuộc connection đó (`len(filtered) == 0`), hàm **BẮT BUỘC PHẢI TRẢ VỀ `nil`**! Tuyệt đối không được fallback sang trả về route của connection khác!

---

### 🟡 PHÁT HIỆN 2 (EDGE CASE): ĐIỀU KIỆN PARSE TOPIC 4-PARTS TRONG `kafka_consumer.go`
- **Vị trí:** `centralized-data-service/internal/handler/shadow/kafka_consumer.go`, dòng 670–675.
- **Hiện trạng mã nguồn:**
  ```go
  if sourceConnCode == "" || sourceConnCode == "mongodb" {
      parts := strings.Split(msg.Topic, ".")
      if len(parts) >= 5 && parts[0] == "cdc" {
          sourceConnCode = parts[2]
      }
  }
  ```
- **Kịch bản phản biện:**
  Nếu topic name có định dạng 4 segments dạng `cdc.<conn>.<db>.<table>` (không có segment `goopay` ở giữa), và `sourceRaw` không chứa trường `name`:
  `len(parts)` là 4 $\rightarrow$ Điều kiện `len(parts) >= 5` không thỏa mãn. `sourceConnCode` sẽ bị bỏ trống và để lại cho fallback của `event_handler.go`.
- **Đúng:** Nên đồng nhất công thức bóc tách tương đối:
  Nếu `len(parts) >= 5 && parts[0] == "cdc"`: `sourceConnCode = parts[2]` (cho `cdc.goopay.<conn>.<db>.<table>`).
  Nếu `len(parts) == 4 && parts[0] == "cdc" && parts[1] != "goopay"`: `sourceConnCode = parts[1]` (cho `cdc.<conn>.<db>.<table>`).

---

### 🟡 PHÁT HIỆN 3 (TEST COVERAGE GAP): THIẾU NEGATIVE TEST TRONG UNIT TEST SUITE
- **Vị trí:** `centralized-data-service/internal/service/source/metadata_registry_service_test.go`, dòng 310–328.
- **Hiện trạng:**
  Test suite hiện tại chỉ kiểm tra 3 case:
  1. Khớp `traitestmongodevct` $\rightarrow$ trả về route 1 (Pass).
  2. Khớp `traitestctphs` $\rightarrow$ trả về route 2 (Pass).
  3. Không truyền connection $\rightarrow$ trả về cả 2 routes (Pass).
- **Điểm mù:** Chưa test trường hợp truyền connection lạ `unknown_conn`. Nếu chạy test case `ResolveSourceRoutes("core_db", "trans_his", "unknown_conn")` với code hiện tại, nó sẽ trả về cả 2 routes thay vì `nil` do dính lỗi ở Phát hiện 1!

---

## III. KIỂM TRA TÍNH TOÀN VẸN VÀ BÁO CÁO TRUNG THỰC (ANTI-SPECULATION CHECK)

1. **Có báo cáo láo về việc sửa file không?**
   - **KHÔNG.** Toàn bộ 13 files mã nguồn đều được kiểm tra trực tiếp bằng `view_file` và `grep_search` trên filesystem thực tế. Mọi dòng code, hàm số, struct đều tồn tại và được định nghĩa chính xác.
2. **Có vi phạm Architecture / Core Systems không?**
   - **KHÔNG.** Hệ thống không dùng bất kỳ cheat DB hay hardcode config tạm bợ nào. Tầng Source được định danh bằng `source_conn`, tầng Master định danh bằng `master_binding_id`. Realtime fan-out dùng `ListMasterTargetsByShadowIdentity`.
3. **Có suy diễn vô căn cứ không?**
   - **KHÔNG.** Toàn bộ các phát hiện lỗi ở Mục II đều được chứng minh bằng code logic và luồng thực thi thực tế (Trace Execution Path).

---

## IV. KẾ HOẠCH HÀNH ĐỘNG KHẮC PHỤC (ACTION PLAN - SELF-IMPROVEMENT LOOP)

Để đạt chất lượng xuất sắc tuyệt đối (Staff Engineer Level) trước khi đóng task:

1. **Sửa dứt điểm Lỗ hổng 1 trong `metadata_registry_service.go`:**
   Thay thế đoạn code dòng 572–592 thành:
   ```go
   var conn string
   if len(sourceConn) > 0 {
       conn = strings.TrimSpace(sourceConn[0])
   }
   if conn != "" {
       cleanConn := strings.TrimPrefix(strings.TrimPrefix(conn, "cdc.goopay."), "cdc.")
       filtered := make([]*metadata.ResolvedSourceRoute, 0, len(routes))
       for _, r := range routes {
           if r != nil {
               rConn := r.SourceConnectionKey
               if rConn == conn || rConn == cleanConn {
                   filtered = append(filtered, r)
               }
           }
       }
       // BẮT BUỘC: Khi caller đã truyền conn cụ thể, chỉ trả về filtered
       // (nếu không có route nào khớp thì trả về nil, CẤM fallback sang connection khác)
       if len(filtered) > 0 {
           return filtered
       }
       return nil
   }
   return routes
   ```

2. **Cập nhật chuẩn hóa bóc tách Topic trong `kafka_consumer.go`:**
   Hỗ trợ trích xuất cả format 4-parts và 5-parts một cách an toàn.

3. **Bổ sung Negative Unit Test Case trong `metadata_registry_service_test.go`:**
   Viết test case xác nhận khi query với connection không tồn tại `unknown_conn`, hàm trả về `nil` chứ không rò rỉ routes của connection khác.
