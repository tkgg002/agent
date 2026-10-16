# 10 — Gap Analysis & Adversarial Review: Phản biện Kế hoạch Tách biệt Luồng CDC và Snapshot

> **Mục tiêu**: Phản biện gay gắt kế hoạch phân tách pipeline CDC và Snapshot, vạch trần các điểm mù về khóa dữ liệu (Row Lock Contention), phân bổ tài nguyên và tính toàn vẹn OCC.

---

## I. MA TRẬN PHẢN BIỆN ADVERSARIAL

### Phản biện 1: Tranh chấp Khóa Hàng (Row-level Lock Contention) trên Cùng Một Bảng
- **Vấn đề**: Kể cả khi đã tách thành 2 buffer riêng và 2 pool riêng, nếu Snapshot và CDC cùng ghi vào **cùng một bảng vật lý** (ví dụ `shadow.orders`), chúng vẫn sẽ đụng độ trực tiếp tại tầng PostgreSQL engine!
- **Kịch bản rủi ro**:
  - Snapshot đang chạy bulk transaction 2000 records trên bảng `orders`. Transaction này giữ lock trên một loạt các hàng.
  - Cùng lúc đó, một sự kiện CDC realtime cập nhật trạng thái đơn hàng của một trong các hàng đó.
  - Lệnh UPDATE của CDC bị block chờ lock của Snapshot giải phóng.
  - Nếu thời gian chờ vượt ngưỡng hoặc xảy ra vòng tròn phụ thuộc: Kích hoạt **Deadlock (`SQLSTATE 40P01`)** hoặc **Lock Timeout**!
- **Biện pháp giải trừ suy diễn**:
  1. **Thu nhỏ Chunk Size của Snapshot**: Giảm chunk transaction của Snapshot từ 2000 xuống **200–500 records/tx**, cam kết mỗi transaction của Snapshot kết thúc trong vòng dưới 50ms để giải phóng row locks tức thì.
  2. **Fast-fail Lock Timeout cho CDC**: CDC thiết lập `SET LOCAL lock_timeout = '1s'` để nếu bị vướng lock quá 1 giây, CDC sẽ retry ngay lập tức thay vì bị treo pipeline.
  3. **OCC Time Guard (`_source_ts`)**: Đảm bảo điều kiện `WHERE NEW._source_ts >= shadow_table._source_ts` hoạt động chuẩn xác, ngăn chặn Snapshot (dữ liệu cũ) ghi đè lên dữ liệu CDC (dữ liệu mới) vừa cập nhật.

---

### Phản biện 2: Sự Lãng phí của Hạn ngạch Kết nối Tĩnh (Static Connection Quota)
- **Vấn đề**: Nếu tách cứng thành 2 pool vật lý độc lập (CDC 70 conns, Snapshot 30 conns):
  - 95% thời gian trong ngày **không hề có Snapshot chạy** (chỉ chạy định kỳ hoặc khi có yêu cầu re-sync).
  - 30 connections dành riêng cho Snapshot sẽ bị lãng phí, trong khi CDC lúc cao điểm lại không thể mượn 30 connections này.
- **Biện pháp giải trừ suy diễn**:
  - **Không tách thành 2 pool vật lý riêng biệt**.
  - Duy trì một Connection Pool chung tối ưu (`maxOpenConn: 100`).
  - Sử dụng **Dynamic Semaphore (Soft/Hard Concurrency Cap)**:
    + Cấp cho Snapshot một `sync.Semaphore` hoặc unbuffered/buffered channel với dung lượng tối đa **25 permits**.
    + Snapshot chỉ được phép mở tối đa 25 concurrent DB queries tại mọi thời điểm.
    + CDC không bị giới hạn permit, được quyền sử dụng toàn bộ dung lượng còn lại của pool (75–100 conns). Khi Snapshot không chạy, CDC tự do sử dụng 100% tài nguyên pool.

---

### Phản biện 3: Quên Đồng bộ Schema Cache giữa Hai Buffer
- **Vấn đề**: Trước đây chỉ có 1 buffer, khi có sự kiện reload schema (`nats: schema.config.reload`), hệ thống gọi `batchBuffer.FlushSchemaCache()`.
- **Rủi ro**: Nếu tách thành 2 buffer mà quên cập nhật NATS subscriber, `snapshotBatchBuffer` có thể giữ schema cache cũ, dẫn đến việc snapshot sau khi bảng thêm cột mới bị lỗi `SQLSTATE 42703 (column does not exist)`.
- **Biện pháp giải trừ**: Bắt buộc cập nhật NATS handlers trong `server_setup.go` để invalidate cache đồng thời trên cả hai buffer.
