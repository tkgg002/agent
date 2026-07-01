# Architecture Decision Log (ADR) - Fix False Drift on Recon payment_bills

## ADR-001: Sử dụng hệ quy chiếu thời gian miền (Domain Timestamp) cho đối soát chéo Tier 1

### Bối cảnh (Context)
Đối soát Tier 1 thực hiện đối soát chéo giữa hai hệ cơ sở dữ liệu có công nghệ khác nhau: MongoDB (Source) và PostgreSQL (Shadow/Destination).
Ban đầu, phía `ReconDestAgent` (PostgreSQL) sử dụng metadata CDC timestamp `_source_ts` để phân chia window và tính toán fingerprint. Trong khi đó, Mongo Source sử dụng trường domain timestamp thực tế (ví dụ: `lastUpdatedAt` hoặc `updated_at`).
Khi xảy ra sự kiện backfill hoặc chạy snapshot lại dữ liệu từ phía Debezium, cột `_source_ts` ở Postgres sẽ được cập nhật theo thời gian snapshot mới, trong khi domain timestamp của Mongo vẫn giữ nguyên giá trị ở quá khứ. Điều này làm lệch hệ quy chiếu lọc cửa sổ dữ liệu giữa 2 bên, gây ra hiện tượng báo lệch ảo dữ liệu (drift ảo) mặc dù dữ liệu thực tế hoàn toàn khớp nhau.

### Quyết định (Decision)
1. **Tier 1 (Source vs Shadow)**: Bắt buộc phải sử dụng cột domain timestamp thực tế (ví dụ: `lastUpdatedAt` cấu hình trong mapping registry) làm cột lọc cửa sổ và tính toán hash cho cả hai bên Source và Destination.
2. **Tier 2 (Shadow vs Master)**: Vẫn giữ nguyên sử dụng `_source_ts` vì cả hai bên đều là Postgres và đối soát theo thời gian nhận luồng CDC stream.
3. **Quy đổi thời gian**: Khi sử dụng domain timestamp ở Postgres Shadow, kiểu dữ liệu `timestamp` hoặc `timestamptz` phải được quy đổi sang epoch milliseconds khi tính toán fingerprint đầu giờ để đồng bộ 100% với cách lưu trữ của MongoDB.

### Hệ quả (Consequences)
* Loại bỏ triệt để hiện tượng báo lệch ảo dữ liệu khi có sự kiện snapshot/backfill lại luồng CDC.
* Đòi hỏi shadow table phải có chỉ mục (index) thích hợp trên cột domain timestamp để đảm bảo hiệu năng truy vấn trong cửa sổ thời gian.
