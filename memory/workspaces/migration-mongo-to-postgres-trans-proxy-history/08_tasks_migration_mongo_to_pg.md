# 📋 Task List CDC System — Migration MongoDB → PostgreSQL (core-trans-proxy-history-service)

> **Phạm vi**: Dành riêng cho **Hệ thống CDC (CDC Platform / CDC CMS)**
> **Phân loại Task**: 
> - 🖥️ **[Vận hành CMS]**: Thao tác cấu hình, trigger job, kích hoạt, monitor trên CDC CMS
> - 🛠️ **[Code thêm / Dev]**: Viết code DDL, config connector, custom pipeline driver, đối soát

---

## Track 1 — CDC Data Sync (MongoDB → Postgres)

### T1.1 — Dựng Schema & Cấu hình Metadata Mapping
- [ ] 🛠️ **[Code thêm / Dev]** Thiết kế & chạy DDL Migration tạo Postgres Destination Schema/Tables cho `core-trans-proxy-history-service`.
- [ ] 🖥️ **[Vận hành CMS]** Khai báo DB Connections (MongoDB Source & Postgres Shadow/Dest) trên CDC CMS.
- [ ] 🖥️ **[Vận hành CMS]** Tạo mới Source Object Registry & Shadow Binding trên CMS cho `core-trans-proxy-history-service`.
- [ ] 🖥️ **[Vận hành CMS]** Cấu hình Field Mappings (MongoDB JSON/BSON → Postgres Data Types: UUID, TIMESTAMPTZ, NUMERIC, JSONB, TEXT).
- [ ] 🖥️ **[Vận hành CMS]** Approve Mapping Rules & Activate Table Registry trên CDC CMS.

### T1.2 — Backfill & Thiết lập CDC Pipeline Real-time (Mongo → Postgres)
- [ ] 🛠️ **[Code thêm / Dev]** Cấu hình MongoDB Change Streams (`capture.mode: change_streams_with_full_update` để không rớt payload update).
- [ ] 🛠️ **[Code thêm / Dev]** Cấu hình CDC Pipeline / Worker Consumer (Debezium/Worker engine) đọc stream MongoDB đẩy vào NATS/Kafka.
- [ ] 🖥️ **[Vận hành CMS]** Trigger Batch Transform / Full Backfill Job từ Mongo sang Postgres Shadow trên CMS.
- [ ] 🖥️ **[Vận hành CMS]** Kích hoạt Streaming CDC Pipeline (Mongo → Postgres) trên CMS.
- [ ] 🖥️ **[Vận hành CMS]** Giám sát CDC Consumer Lag, Status & Dead-Letter Queue (DLQ) trên Dashboard.

### T1.3 — Đối soát Dữ liệu (Reconciliation)
- [ ] 🖥️ **[Vận hành CMS]** Khai báo Recon Profile / Job trên CMS cho cặp MongoDB Source ↔ Postgres Dest.
- [ ] 🖥️ **[Vận hành CMS]** Trigger Manual Recon Job (đối soát record count + checksum hash sub-window 15m).
- [ ] 🛠️ **[Code thêm / Dev]** Kiểm tra lỗi đối soát (nếu có): Sửa cast expr/date variant hoặc trigger Bridge Sync bù bản ghi rớt.
- [ ] 🖥️ **[Vậnhanh CMS]** Thiết lập Lịch đối soát tự động (Schedule Recon Job) định kỳ trên CMS.

---

## Track 2 — Hỗ trợ v2 Read & Canary (CDC Monitoring)

- [ ] 🖥️ **[Vận hành CMS]** Giám sát liên tục CDC Streaming Lag (< 1s) trong suốt quá trình v2 Canary Read.
- [ ] 🖥️ **[Vận hành CMS]** Kiểm tra log audit transform & DLQ trên CMS nếu v2 phát hiện sai lệch dữ liệu đọc.

---

## Track 3 — Switch Write & CDC Chiều Ngược (Postgres → Mongo)

- [ ] 🛠️ **[Code thêm / Dev]** Thiết lập CDC Pipeline chiều ngược (Postgres Logical Replication / Debezium Postgres → Mongo BSON converter).
- [ ] 🖥️ **[Vận hành CMS]** (Thời điểm Switch): Tắt CDC Pipeline chiều Mongo → Postgres trên CMS.
- [ ] 🖥️ **[Vận hành CMS]** (Thời điểm Switch): Kích hoạt CDC Pipeline chiều ngược Postgres → Mongo trên CMS.
- [ ] 🖥️ **[Vận hành CMS]** Giám sát CDC chiều ngược trên CMS: Đảm bảo mọi giao dịch ghi mới vào Postgres được sync về Mongo backup.
- [ ] 🖥️ **[Vận hành CMS]** (Sau thời gian theo dõi an toàn): Tắt hẳn CDC Pipeline chiều ngược trên CMS.
