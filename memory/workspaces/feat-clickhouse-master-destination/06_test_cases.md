# 06 Test Cases: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`

---

## Danh mục Kịch bản Kiểm thử

| Mã TC | Tên Kịch bản | Mô tả Thực hiện | Kết quả Kỳ vọng |
|:---|:---|:---|:---|
| **TC-CH-01** | **ClickHouse Connection & Ping** | Kết nối ClickHouse container qua port 9000 bằng `pkgs/clickhouse/client.go`. | Trả về kết nối thành công, ping latency < 5ms. |
| **TC-CH-02** | **ClickHouse DDL Generation** | Chạy `MasterDDLGenerator` cho bảng có 15 cột đủ kiểu dữ liệu (`int`, `decimal`, `datetime64`, `nullable`). | Sinh câu lệnh `CREATE TABLE ... ENGINE = ReplacingMergeTree(_version, _deleted)` hợp lệ và execute thành công trên ClickHouse. |
| **TC-CH-03** | **Batch Append Transmute** | Chạy Transmute 10,000 dòng từ Shadow sang ClickHouse Master. | Dữ liệu được ghi thành công trong 1 batch duy nhất, thời gian thực thi < 200ms. |
| **TC-CH-04** | **Deduplication on Update** | Gửi 1 bản ghi mới có cùng `_gpay_id` nhưng `_source_ts` mới hơn. | Sau khi insert, câu lệnh `SELECT ... FINAL` trả về duy nhất 1 dòng với dữ liệu mới nhất. |
| **TC-CH-05** | **Soft-Delete Propagation** | Shadow table đánh dấu `_deleted = true`. Transmuter chạy đồng bộ. | Dòng mới được chèn vào ClickHouse với `_deleted = 1`. Mệnh đề `WHERE _deleted = 0` loại trừ chính xác bản ghi này. |
| **TC-CH-06** | **Recon Hash Check** | Kích hoạt Recon so sánh hash giữa Shadow (Postgres) và Master (ClickHouse). | Drift = 0. Khi có bản ghi lệch, hệ thống phát hiện chính xác `_gpay_id` bị lệch. |
