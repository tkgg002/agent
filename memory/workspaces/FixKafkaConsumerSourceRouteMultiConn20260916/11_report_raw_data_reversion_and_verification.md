# BÁO CÁO NGHIỆM THU: REVERT 100% CAN THIỆP _RAW_DATA & BẢO TOÀN NGUYÊN BẢN GROUND TRUTH

**Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`  
**Thời gian:** 2026-09-16T16:50:00+07:00  
**Thực hiện bởi:** Brain (Chairman & Architect)  
**Tiêu chuẩn nghiệm thu:** Feature Output Quality Gate (DoD G1–G8), Simplicity First & Minimal Impact (Rule #12).

---

## I. TỔNG QUAN HẠNG MỤC (EXECUTIVE SUMMARY)

Thực hiện theo lệnh phê duyệt của User, toàn bộ can thiệp làm phẳng `_raw_data` đã được **REVERT 100%**:
1. **Khôi phục Ground Truth nguyên bản:** Cột `_raw_data` trong PostgreSQL Shadow table được trả về đúng vai trò nguyên thủy là kho lưu trữ thô trung thực 100% với dữ liệu nguồn gửi sang từ Kafka/Debezium/Oplog.
2. **Loại bỏ triệt để rủi ro phá vỡ tương thích ngược:**
   - Dữ liệu MongoDB gửi Extended JSON BSON (`{"$oid": "...", "$date": ...}`) $\rightarrow$ Lưu nguyên Extended JSON.
   - Dữ liệu quan hệ (PostgreSQL, MariaDB, MySQL) gửi phẳng $\rightarrow$ Lưu nguyên phẳng.
   - Toàn bộ các mapping rule cũ trỏ `_id.$oid`, `created_at.$date` tiếp tục chạy hoàn hảo trên `rawData` gốc mà không bị lỗi `nil`.
   - Master Transmuter (`flatten.go`) tiếp tục đọc được `_id.$oid` bình thường, không bị drop mất bản ghi.
   - SQL BatchTransform (`BuildCastExpr`) tiếp tục chạy trên cả 2 định dạng JSONB.
3. **Bảo toàn các cải tiến định tuyến đa kết nối cần thiết:**
   - Hoàn thiện `toTimestamp` hỗ trợ đầy đủ `case int64:`, `case int:`, `case json.Number:`.
   - Bổ sung `IsGenericConnectionCode` fallback cho Kafka Consumer, Event Handler và Metadata Registry để không drop traffic của các topic 4-parts cổ điển.

---

## II. CHI TIẾT CÁC TỆP TIN ĐÃ THAY ĐỔI & DIFF MÃ NGUỒN

### 1. [`centralized-data-service/internal/service/shadow/dynamic_mapper.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/dynamic_mapper.go)
- **Mục đích:** Revert việc làm phẳng `_raw_data`, lưu `rawData` gốc 100%, giữ `unwrapMongoTypes` chỉ cho typed columns, và bổ sung hỗ trợ kiểu số nguyên cho `toTimestamp`.
- **Số dòng thay đổi:** ~50 dòng (xóa bỏ 60 dòng dead code của `normalizeMongoExtJSON`).
- **Diff chi tiết:**
  ```go
  // TRƯỚC (Code gây rủi ro phá vỡ):
  normalizedRaw := rawData
  if nMap, ok := normalizeMongoExtJSON(rawData).(map[string]interface{}); ok {
      normalizedRaw = nMap
  }
  ...
  rawJSON, _ := json.Marshal(dm.maskRawData(bindingID, normalizedRaw))

  // SAU (Đã Revert - Khôi phục Ground Truth 100%):
  rules := dm.registry.GetMappingRules(bindingID)
  if len(rules) == 0 {
      rawJSON, _ := json.Marshal(dm.maskRawData(bindingID, rawData))
      return &MappedData{... RawJSON: rawJSON}, nil
  }
  ...
  val, exists := rawData[rule.SourceField]
  if !exists {
      val = getNestedField(rawData, rule.SourceField)
  }
  // Chỉ unwrap MongoDB types cho các cột định kiểu (typed columns):
  val = unwrapMongoTypes(val)
  ...
  // Lưu nguyên vẹn rawData của nguồn vào _raw_data:
  rawJSON, _ := json.Marshal(dm.maskRawData(bindingID, rawData))
  ```

---

### 2. [`centralized-data-service/internal/service/source/metadata_registry_utils.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_utils.go)
- **Mục đích:** Xuất bản hàm `IsGenericConnectionCode` nhận diện các tiền tố chung (`goopay`, `gpay`, `mongodb`, `postgres`, `postgresql`, `mysql`, `mariadb`, `sftp`).
- **Nội dung thêm mới:**
  ```go
  func IsGenericConnectionCode(code string) bool {
      switch strings.ToLower(strings.TrimSpace(code)) {
      case "goopay", "gpay", "mongodb", "postgres", "postgresql", "mysql", "mariadb", "sftp":
          return true
      default:
          return false
      }
  }
  ```

---

### 3. [`centralized-data-service/internal/service/source/metadata_registry_service.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go)
- **Mục đích:** Xử lý fallback an toàn cho topic 4-parts cổ điển trong `ResolveSourceRoutes`.
- **Nội dung thay đổi:**
  ```go
  func (rs *MetadataRegistryService) ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*metadata.ResolvedSourceRoute {
      if len(sourceConn) > 0 && IsGenericConnectionCode(sourceConn[0]) {
          sourceConn = nil
      }
      ...
  ```

---

### 4. [`centralized-data-service/internal/handler/shadow/kafka_consumer.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go)
- **Mục đích:** Reset `sourceConnCode = ""` khi gặp generic engine prefix để không chặn nhầm traffic cổ điển downstream.
- **Nội dung thay đổi:**
  ```go
  if source.IsGenericConnectionCode(sourceConnCode) {
      sourceConnCode = ""
  }
  ```

---

### 5. [`centralized-data-service/internal/handler/shadow/event_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
- **Mục đích:** Reset `sourceConn = ""` khi gặp generic prefix trước khi phân giải route.
- **Nội dung thay đổi:**
  ```go
  if servicesource.IsGenericConnectionCode(sourceConn) {
      sourceConn = ""
  }
  ```

---

### 6. [`centralized-data-service/test/internal/service/dynamic_mapper_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/test/internal/service/dynamic_mapper_test.go)
- **Mục đích:** Kiểm thử nghiêm ngặt tính toàn vẹn của Ground Truth.
- **Nội dung thêm mới:** `TestDynamicMapper_PreservesRawDataGroundTruth_AndUnwrapsTypedColumns`
  - Assert 1: `mapped.Columns["id"]` và `mapped.Columns["created_at"]` được unwrap đúng kiểu dữ liệu.
  - Assert 2: `_raw_data` chứa nguyên vẹn `{"$oid": "...", "$date": float64(1770102689256)}` 100% không bị mutate.

---

### 7. [`centralized-data-service/internal/service/source/metadata_registry_service_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service_test.go)
- **Mục đích:** Kiểm thử nhận diện generic connection code và cơ chế fallback route.
- **Nội dung thêm mới:**
  - `TestIsGenericConnectionCode`: Xác thực toàn bộ các mã generic và specific.
  - `TestResolveSourceRoutes_GenericConnectionFallback`: Xác thực caller truyền generic prefix trả về đầy đủ routes tương thích, truyền specific code trả về chính xác 1 route, và truyền unknown code trả về `nil`.

---

## III. MA TRẬN ĐỐI SOÁT CHẤT LƯỢNG ĐẦU RA (QUALITY GATES DOD G1–G8)

| Cổng Kiểm Định | Nội dung Đối soát | Kết quả Thực tế | Đánh giá |
| :--- | :--- | :--- | :---: |
| **(G1) Truy vết Yêu cầu** | Revert triệt để biến dạng `_raw_data`, khôi phục Ground Truth và kiểm soát topic cổ điển | 7/7 tệp tin mã nguồn và tests đã phản ánh chính xác | ✅ **PASS** |
| **(G2) Reproduce trước khi Fix** | Nguy cơ gãy mapping rules cũ và drop topic cổ điển | Đã viết test case tái hiện và kiểm chứng fallback thành công | ✅ **PASS** |
| **(G3) Test thật, không đoán mò** | Unit tests cho `dynamic_mapper` và `metadata_registry_service` | Toàn bộ tests chạy PASS, assertion Ground Truth đạt 100% | ✅ **PASS** |
| **(G4) Edge-case & Negative-path** | Test unknown connection code trả về `nil`, test generic prefix trả về all | Assert chặt chẽ trong `TestResolveSourceRoutes_GenericConnectionFallback` | ✅ **PASS** |
| **(G5) Chống Regression** | Kiểm tra tương thích với dữ liệu lịch sử và mapping rules cũ | `_raw_data` nguyên bản, `unwrapMongoTypes` chạy trên typed columns | ✅ **PASS** |
| **(G6) Output Correctness** | `_raw_data` trong DB giữ nguyên 100% payload nguồn | Xác nhận qua test assertion không có mutilation | ✅ **PASS** |
| **(G7) Adversarial Self-Review** | Rà soát unused imports, type safety, scope biến | Dọn sạch unused imports, code clean, zero dead code | ✅ **PASS** |
| **(G8) Bằng chứng Vật lý** | Ghi nhận tài liệu và audit log vào workspace | Đầy đủ tại `05_progress.md` và `11_report_*.md` | ✅ **PASS** |
