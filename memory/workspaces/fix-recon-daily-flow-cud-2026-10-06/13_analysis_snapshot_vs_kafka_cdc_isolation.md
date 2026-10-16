# Báo cáo Phân tích: Tách bạch Bản chất Luồng Kafka CDC Streaming và Snapshot Nạp Mẻ Trong Bảng Đối Soát

## 1. Bản chất Kỹ thuật & Bối cảnh Thực tế

Mục đích sống còn của bảng thống kê đối soát theo ngày trong `ReconPipelineGrid.tsx` là:
> **ĐỐI SOÁT TOÀN VẸN LUỒNG KAFKA CDC TỪ LÚC BẮN VÀO HỆ THỐNG ĐẾN KHI GHI VÀO MASTER THEO TỪNG NGÀY.**

Tuy nhiên, trong hệ thống CDC phân tán tồn tại hai luồng dữ liệu độc lập với cơ chế hoạt động hoàn toàn khác nhau:

### Luồng A: Kafka CDC Streaming (Sự kiện thời gian thực)
- **Cơ chế:** Debezium CDC bắt các biến động từ cơ sở dữ liệu nguồn (MongoDB, MySQL, PostgreSQL) và đẩy vào Kafka topic theo từng event (`c`: create, `u`: update, `d`: delete).
- **Trạm 1 (Kafka Fetch & Buffer):** `BatchBuffer` tiêu thụ từ Kafka topic. Ghi nhận `operation = 'kafka-consumer'`, `triggered_by = 'kafka-consumer'`, `buffer_type = 'cdc-stream'`, `is_snapshot = false`.
- **Trạm 2 (Shadow Ingest):** Thực hiện `INSERT ... ON CONFLICT (pk) DO UPDATE` vào PostgreSQL Shadow.
  - Ví dụ: 1 insert và 3 updates trên cùng một bản ghi trong cùng một batch sẽ tính là **4 Ops**, nhưng chỉ sinh ra **1 dòng DB vật lý**.
- **Trạm 3 (Master Transmute):** `batch_buffer_fanout` bắn trigger qua NATS `cdc.cmd.transmute`. `transmute_handler` đọc các ID thay đổi từ Shadow, ánh xạ schema, áp quy tắc nghiệp vụ và ghi vào Master DB.
  - Ghi nhận `operation = 'transmute'`, `triggered_by = 'kafka-consumer-hook'`.
  - Số dòng Master bốc lên (`scanned`) chính xác bằng số dòng DB Shadow đã ghi. Master chia thành: `inserted` (bản ghi mới) và `updated` (bản ghi đã có sẵn trên Master được cập nhật).

### Luồng B: Snapshot Runner / Re-sync (Nạp mẻ ban đầu hoặc bù trừ)
- **Cơ chế:** Snapshot Runner đọc trực tiếp hàng ngàn / hàng chục ngàn bản ghi từ DB nguồn, nạp thẳng vào DB Shadow qua đệm `snapshot buffer`.
- **Đặc điểm nhật ký (`cdc_activity_log`):**
  - Ghi nhận `operation = 'snapshot.v2'` (tổng kết mẻ) hoặc `operation = 'kafka-consumer'` với `triggered_by = 'snapshot'`, `buffer_type = 'snapshot'`, `is_snapshot = true`.
  - Khi xả sang Master: transmute được kích hoạt từ snapshot hoặc từ quản trị viên (`triggered_by = 'admin'` hoặc mang `trace_id` của mẻ snapshot).

---

## 2. Giải mã Dữ liệu Thực tế từ PostgreSQL `cdc_dw`

Đối chiếu trực tiếp trên database `cdc_dw` ngày 09/10/2026 cho thấy rõ hai hiện tượng:

### Trường hợp 1: Bảng `export_jobs` (Ngày 09/10/2026 - Chỉ có Snapshot, KHÔNG CÓ Kafka CDC)
- **Kafka CDC Stream:** 0 tin nhắn. Debezium không bắn bất kỳ sự kiện nào.
- **Snapshot Run:** Chạy mẻ snapshot nạp 539 bản ghi vào `export_jobs`.
  - Log 1: `operation = 'snapshot.v2'`, `rows_affected = 539`.
  - Log 2: `operation = 'kafka-consumer'`, `triggered_by = 'snapshot'`, `batch_size = 539`, `rows_affected = 0` (do bản ghi đã tồn tại trong shadow).
  - Log 3: `operation = 'transmute'`, `triggered_by = 'kafka-consumer-hook'`, `trace_id = '82b9d2dc52f84ca88aa03646ec5b3bd4'` (trùng trace ID snapshot), `scanned = 539`, `inserted = 0`, `updated = 539`.
- **Hậu quả khi code cũ xử lý lem nhem:**
  - Cột 1 (KAFKA CDC): Hiện `0 Ops` (0 c, 0 u, 0 d).
  - Cột 2 (SHADOW): Cộng 539 vào `row.shadow.total` -> Hiện `539 Ops (0 dòng DB)`.
  - Cột 3 (MASTER): Cộng 539 vào `row.master.total` -> Hiện `539 Ops (539 ghi DB)`.
  - **Nhận định sai lầm của người xem:** Nhìn vào tưởng Kafka bị mất sạch 539 tin nhắn, hoặc hệ thống bị lỗi nghiêm trọng!

### Trường hợp 2: Bảng `payment_bills` (Ngày 09/10/2026 - Luồng Kafka CDC Streaming thuần túy)
- **Truy vấn thực tế từ `cdc_activity_log`:**
  - `kafka-consumer` (`triggered_by = 'kafka-consumer'`): 104 logs, tổng `batch_size = 322` ($131 c + 191 u$), tổng `written = 157` dòng DB.
  - `transmute` (`triggered_by = 'kafka-consumer-hook'`): 104 logs, tổng `scanned = 157`, `inserted = 131`, `updated = 26`, `skipped = 0`, tổng `written = 157` ghi DB.
- **Đối soát 3 trạm:**
  - Kafka CDC: Nhận **322 Ops** ($131 c + 191 u$).
  - Shadow: Xử lý **322 Ops**, qua ON CONFLICT cô đọng thành **157 dòng DB**.
  - Master: Bốc đúng **157 dòng** từ Shadow, ghi thành **131 Inserts + 26 Updates = 157 dòng ghi DB**!
  - Hai trạm Shadow và Master khớp nhau 100% về số dòng DB vật lý ($157 = 157$).

---

## 3. Kiến trúc Cấu trúc lại Bảng Đối soát (Resolution Architecture)

Để trả về đúng bản chất đối soát của Kafka CDC và minh bạch hóa toàn bộ các hoạt động nạp mẻ:

1. **Khối 1: SNAPSHOT / RE-SYNC (NẠP MẺ BAN ĐẦU)**
   - Đặt riêng ở vị trí đầu tiên hoặc tách biệt hoàn toàn.
   - Thể hiện:
     + `Shadow (Nạp)`: Số bản ghi nạp vào Shadow từ Snapshot.
     + `Master (Sync)`: Số bản ghi đồng bộ sang Master từ Snapshot.
   - Nếu ngày nào không có Snapshot: Hiển thị `—`.
   - Nếu ngày nào có Snapshot: Hiển thị số bản ghi rõ ràng (màu tím `#722ed1`), không cộng lẫn vào Kafka CDC!

2. **Khối 2: KAFKA CDC BẮN VỀ (STREAMING)**
   - Chỉ tính các sự kiện stream Debezium:
     + `Tổng (Ops)` = $c + u + d$
     + `Insert (c)`
     + `Update (u)`
     + `Delete (d)`

3. **Khối 3: GHI VÀO DB SHADOW (TỪ CDC)**
   - Chỉ tính các mẻ do Kafka CDC ghi vào Shadow:
     + `Tổng xử lý`: `<total> Ops` kèm `(<written> dòng DB)`
     + `Insert (c)`
     + `Update (u)`
     + `Delete (d)`

4. **Khối 4: GHI VÀO DB MASTER (TỪ CDC)**
   - Chỉ tính các mẻ transmute kích hoạt từ luồng CDC:
     + `Tổng xử lý`: `<total> Ops` kèm `(<written> ghi DB)`
     + `Inserted`
     + `Updated`
     + `Skipped`

5. **Khối 5: ĐỐI SOÁT & DLQ (LỖI / TỒN ĐỌNG)**
   - `DLQ / Lỗi`: Số lỗi batch hoặc DLQ
   - `Lệch thiếu`: Số bản ghi missing từ phiên đối soát cuối của ngày
   - `Lệch stale`: Số bản ghi stale từ phiên đối soát cuối của ngày
   - `Đã heal`: Số bản ghi đã tự động sửa chữa

6. **ĐÁNH GIÁ (FSM)**
   - Đánh giá trạng thái ngày dựa trên:
     + Có lỗi DLQ hoặc Missing > 0: `CÓ LỆCH` (Đỏ).
     + Có Stale > 0 hoặc Kafka bắn tin mà Shadow CDC không nhận: `CẢNH BÁO` (Vàng).
     + Khớp toàn diện: `CÂN BẰNG` (Xanh).
   - Ngày chỉ có Snapshot thành công không bị báo động giả!
