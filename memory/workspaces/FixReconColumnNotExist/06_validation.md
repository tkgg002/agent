# 06_validation: Kịch Bản Kiểm Thử & Xác Minh Sửa Lỗi Đối Soát

Tài liệu này ghi lại danh mục các kịch bản kiểm thử (Test Cases) và bằng chứng xác minh kết quả chạy thực tế cho việc đồng bộ lọc cửa sổ thời gian (Window Filtering) và sửa lỗi Heal Segment B.

## 1. Kịch Bản Kiểm Thử Tự Động (Automated Test Cases)

### Kịch bản 1: Lọc cửa sổ với cột không tồn tại trên Postgres Shadow
* **Mục tiêu**: Đảm bảo hàm `resolveSourceTSField` phát hiện cột không tồn tại, tự động fallback về các ứng viên khả thi và không gây lỗi cú pháp SQL GORM.
* **Bằng chứng**: Chạy thành công bộ unit tests trong `recon_tier_a_test.go` và `recon_dest_agent_test.go`.
* **Lệnh chạy**:
  ```bash
  go test -v ./internal/service/recon/... -run TestResolveSourceTSField
  ```

### Kịch bản 2: Trích xuất Timestamp động từ MongoDB document
* **Mục tiêu**: Đảm bảo hàm `extractSourceTsFromDoc` ưu tiên trích xuất trường thời gian được chỉ định từ registry thay vì fix cứng `"updated_at"`.
* **Bằng chứng**: Kịch bản test `TestExtractSourceTsFromDoc_CustomField` chạy PASS.
* **Lệnh chạy**:
  ```bash
  go test -v ./internal/service/governance/... -run TestExtractSourceTsFromDoc
  ```

### Kịch bản 3: Sửa lỗi stale report fallback
* **Mục tiêu**: Báo cáo đối soát đã quá 5 phút sẽ được coi là stale, tự động kích hoạt chạy lại Tier 2 thay vì tái sử dụng báo cáo cũ.
* **Bằng chứng**: Test case `TestHealSegmentA_StaleReportFallback` hoạt động đúng như thiết kế.
* **Lệnh chạy**:
  ```bash
  go test -v ./internal/handler/recon/... -run TestHealSegmentA_StaleReportFallback
  ```

---

## 2. Kịch Bản Xác Minh Trên Hệ Thống Thật (Manual Verification)

### Kịch bản 4: Chữa lành dữ liệu bảng `payment_bills` bị lệch
* **Điều kiện đầu vào**: 
  * Cấu hình shadow object registry của bảng `payment_bills` có `timestamp_field` là `lastUpdatedAt`.
  * Có 68 bản ghi bị lệch dữ liệu giữa MongoDB source và Postgres shadow.
* **Hành động**:
  * Restart `cdc-worker` thành công.
  * Publish message heal Segment A qua NATS.
* **Kết quả quan sát**:
  * Logs hệ thống ghi nhận `heal-a: latest report is stale, running tier 2 first`.
  * `resolveSourceTSField` xác minh cột `lastUpdatedAt` tồn tại trên Postgres shadow bảng `shadow_test1111.payment_bills` thông qua phương thức `ColumnExists` và sử dụng nó để lọc.
  * Đối soát Tier 2 phát hiện chính xác 68 bản ghi lệch (missing from dest: 50, missing from src: 18).
  * Dispatch thành công 68 debezium snapshot signals qua NATS với filter ID tương ứng.
* **Log chi tiết minh họa**:
  ```json
  {"level":"info","ts":1782786827.642465,"msg":"tier2 hash_window","table":"payment_bills","windows":672,"drifted_windows":22,"missing_from_dest":50,"missing_from_src":18,"mismatched":0}
  {"level":"info","ts":1782786827.652669,"msg":"recon heal-a dispatched snapshot signal","table":"payment_bills","ids":68}
  ```
