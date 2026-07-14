# Phân tích Kỹ thuật: Khắc phục lệch pha đối soát thời gian thực

## 1. Vấn đề Hiện tại
Các hàm đối soát toàn bộ bảng (`FullIDDiffMissingFromShadow`, `RunOrphanPrune`) và đối soát fingerprint toàn bảng (`RunDeepCheck` thông qua `BucketHash`) hiện đang quét toàn bộ dữ liệu từ MongoDB và Postgres shadow mà không có giới hạn trên về thời gian (upper bound).
Đối với các bảng có dữ liệu ghi liên tục, do độ trễ đồng bộ tự nhiên của CDC (Debezium/Airbyte), dữ liệu mới phát sinh trong vài giây/phút gần đây chỉ có ở MongoDB mà chưa kịp đồng bộ sang Postgres shadow. Việc đối soát dữ liệu mới này dẫn đến:
- `FullIDDiffMissingFromShadow` báo cáo hàng loạt ID bị thiếu ở shadow (false drift) và cố gắng chữa lành (heal) thừa, gây conflict ghi.
- `RunOrphanPrune` có thể nhầm lẫn các record bị xóa ở source nhưng CDC event delete chưa tới, dẫn đến việc prune sớm không an toàn.
- `RunDeepCheck` báo cáo lệch hash ở các bucket chứa dữ liệu mới, gây ra false alarm liên tục.

## 2. Giải pháp Đề xuất
Áp dụng mốc chặn trên thời gian `upper` (now - lag time) động dựa trên `adaptiveFreeze(ingestLagMs)`. Dữ liệu mới hơn mốc này sẽ bị loại trừ khỏi quá trình đối soát toàn bảng.

### 2.1. Cập nhật `FullIDDiffMissingFromShadow` & `RunOrphanPrune`
- Phân giải trường timestamp `srcTS` và `dstTS`.
- Tính toán replication lag hiện tại và mốc chặn trên:
  `upper = time.Now().UTC().Add(-rc.adaptiveFreeze(ingestLagMs))`
- Tại Postgres shadow, lọc các ID có `dstTS < upper` (dùng `resolvePostgresTimeParams` để convert `upper` sang dạng dữ liệu phù hợp của cột).
- Tại MongoDB source, thay vì `StreamAllIDs`, sử dụng `StreamIDsInTimeRange` từ mốc thời gian 0 (`time.Time{}`) đến `upper`.

### 2.2. Cập nhật `BucketHash` của Source và Destination
- Đổi signature của `BucketHash` để nhận thêm tham số `upper time.Time`.
- **MongoDB Source (`recon_hash.go`)**:
  Thêm filter vào câu query `Find`:
  `tsField < upper` (hỗ trợ cả dạng ISODate và Unix Epoch Milliseconds).
- **Postgres Destination (`recon_dest_hash.go`)**:
  Thêm mệnh đề `WHERE` vào câu SQL query:
  `tsCol < ?` (với value được convert qua `resolvePostgresTimeParams`).
- **`RunDeepCheck` (`recon_tier_a.go`)**:
  Tính toán `upper` tương tự như trên và truyền vào cả 2 hàm `BucketHash`.

## 3. Đánh giá Hiệu năng & Rủi ro
- **MongoDB query performance**: Việc dùng `StreamIDsInTimeRange` với keyset pagination sort `_id` sẽ dùng index chính `_id` làm scan order, do đó không bị lỗi 32MB in-memory sort limit của MongoDB. Hơn nữa, vì khoảng scan bao phủ 99.9% dữ liệu, index scan này rất hiệu quả.
- **Postgres query performance**: Postgres sử dụng index trên timestamp (nếu có) hoặc sequential scan. RDBMS tối ưu tốt mệnh đề `WHERE timestamp < ?` nên không có rủi ro hiệu năng.
