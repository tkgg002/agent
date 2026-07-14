# Kế hoạch Triển khai Sửa lỗi Báo cáo & Chữa lành Segment B (Shadow ↔ Master)

## 1. Phân tích lỗi thực tế từ dữ liệu đối soát & Quét tĩnh ảnh hưởng
Từ kết quả đối soát thực tế và kết quả quét tĩnh codebase bằng script `check_segment_b_impact.py`, chúng ta xác định được các lỗi hệ thống:
1. **Sai kiểu CheckType:** `CheckType` của Segment B ghi nhận là `segment_b_window` khiến hàm `getCheckTypesForTier` loại bỏ chúng, làm ẩn báo cáo trên UI/CMS. Cần đổi sang `"hash_window"` (Tier 2) và `"bucket_hash"` (Tier 3).
2. **Thừa & sai lệch trường SourceDB:** Các trường `source_db`, `source_type`, `source_host`, `source_table` chỉ dành riêng cho Segment A. Ở Segment B chặng Transmute, chúng phải để trống.
3. **Lỗi map SourceDB khi Chữa lành (Heal):** Hàm `executeHealSegB` đọc Shadow relation từ `rpt.SourceDB` để map `_gpay_id` sang `_source_id`. Khi gán `SourceDB = ""` ở Segment B, hàm heal sẽ ném lỗi `invalid shadow relation ""`.
   - **Giải pháp:** Nếu segment là `shadow_master`, ta phải lấy Shadow relation FQN bằng cách ghép `rpt.ShadowSchema + "." + rpt.ShadowTable` thay vì đọc `rpt.SourceDB`.
4. **Sai định dạng StaleIDs:** Cột `StaleIDs` trong DB cho Segment B ghi map `{stale_ids, orphan_in_master}`. Phải dùng format chuẩn: `{"mismatched": [...], "missing_from_src": [...], "missing_from_dest": [...]}`.
   - **Giải pháp:** Cập nhật struct `staleSegmentB` và các hàm parse/gọi trong `recon_execute_heal_handler.go` và `recon_check_heal_handler.go`.
5. **Thiếu StaleCount:** GORM struct của Segment B chỉ gán `StaleCount = len(staleIDs)`, bỏ quên `orphanInMaster`. Cần gán `StaleCount = len(staleIDs) + len(orphanInMaster)`.

---

## 2. Giải pháp kỹ thuật sửa đổi chi tiết

### 2.1. internal/service/recon/recon_tier_b.go
- Cập nhật `RunHashWindowCheckB`:
  - Gán `SourceDB: ""` khi khởi tạo report.
  - Sử dụng format JSON chuẩn cho `staleJSON` (mapping `missingFromMaster` -> `missing_from_dest`, `orphanInMaster` -> `missing_from_src`, và `staleIDs` -> `mismatched`).
  - Gán `StaleCount: len(staleIDs) + len(orphanInMaster)`.
  - Thay đổi `CheckType` thành `"hash_window"`.
- Cập nhật `RunDeepCheckB`:
  - Gán `SourceDB: ""` khi khởi tạo report.
  - Gán `StaleCount: len(staleIDs) + len(orphanInMaster)`.
  - Thay đổi `CheckType` thành `"bucket_hash"`.

### 2.2. internal/service/recon/recon_engine_segment_b.go
- Cập nhật `stampB`: Loại bỏ việc gán các trường `SourceType`, `SourceHost`, `SourceTable` (để mặc định trống).

### 2.3. internal/handler/recon/recon_base_handler.go
- Cập nhật struct `staleSegmentB`:
  ```go
  type staleSegmentB struct {
      Mismatched      []string `json:"mismatched"`
      MissingFromSrc  []string `json:"missing_from_src"`
      MissingFromDest []string `json:"missing_from_dest"`
  }
  ```
- Cập nhật hàm `parseStaleSegmentB` để parse đúng theo struct mới và hỗ trợ fallback.

### 2.4. internal/handler/recon/recon_execute_heal_handler.go
- Cập nhật `executeHealSegB`:
  - Lấy `shadowRel` từ `rpt.ShadowSchema + "." + rpt.ShadowTable` nếu `rpt.Segment == SegmentShadowMaster` thay vì đọc `rpt.SourceDB`.
  - Thay đổi gọi trường cũ sang trường mới:
    - `staleB.StaleIDs` $\rightarrow$ `staleB.Mismatched`
    - `staleB.OrphanInMaster` $\rightarrow$ `staleB.MissingFromSrc`

### 2.5. internal/handler/recon/recon_check_heal_handler.go
- Cập nhật logic gom `gpayIDs` (dòng 191) và các chỗ gọi tương ứng trong heal check handler để trỏ đến `staleObj.Mismatched` và `staleObj.MissingFromSrc`.

---

## 3. Quy trình Kiểm thử & Xác minh (DoD)
1. **Biên dịch:** Đảm bảo code compile thành công `go build ./...`.
2. **Unit Tests:** Chạy test suite `go test -v ./internal/service/recon/...` và `./internal/handler/recon/...` thành công.
3. **Static Analysis Check:** Chạy script `/Users/trainguyen/Documents/work/agent/memory/workspaces/ReconTracesDetail/check_segment_b_impact.py` phải trả về **SUCCESS (exit 0)** (Không còn phát hiện cấu trúc cũ nào sót lại).
4. **Integration Real-Test:** Bắn message NATS đối soát Segment B thật, verify report được lưu đúng format JSON, không có trường rác và chạy lệnh Heal Segment B thật thành công!
