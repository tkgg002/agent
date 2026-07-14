# Kịch bản Kiểm thử Thủ công Thực tế dành cho QC (Manual QC Test Cases)

Tài liệu này hướng dẫn chi tiết các kịch bản test case và câu lệnh thực tế mà QC cần thực thi để kiểm chứng giải pháp **Global Hash Check** và **Smart Tracing** hoạt động đúng đắn trên môi trường thực tế (không phải mock).

---

## Chuẩn bị môi trường (Pre-requisites)
1. Máy QC đã cài đặt Homebrew và **nats** CLI (`brew install nats-server/nats/nats`).
2. Docker local của dự án đang chạy (`docker ps` hiển thị đầy đủ `gpay-postgres-cdc`, `gpay-mongo`, `gpay-nats`).
3. Dịch vụ `cdc-worker` đang hoạt động local hoặc trên container.

---

## Kịch bản 1: Kiểm thử đối soát khớp (Global Hash Match - Fast Path)

### Mục tiêu
Kiểm chứng khi dữ liệu giữa Source (MongoDB) và Dest (Postgres Shadow) khớp hoàn toàn, hệ thống thực thi Global Hash Check nhanh và trả về báo cáo thành công ngay lập tức (thời gian chạy dưới 100ms thay vì 30 giây).

### Các bước thực hiện
1. **Gửi lệnh đối soát qua NATS CLI**:
   Chạy lệnh sau trên terminal của máy QC:
   ```bash
   nats req -s nats://cdc_worker:worker_secret_2026@localhost:14222 cdc.cmd.recon-check '{"table":"export_jobs","segment":"source_shadow","legacy":false}'
   ```

2. **Kết quả mong đợi (Expected Output)**:
   Phản hồi JSON nhận về phải hiển thị status `ok`, `diff` bằng 0 và có ID báo cáo hợp lệ (đã được lưu vào DB):
   ```json
   {
     "id": 123,  // Phải lớn hơn 0 (chứng tỏ đã gọi stampA lưu database)
     "target_table": "shadow_testexp.export_jobs",
     "source_db": "shadow_testexp",
     "source_count": 5,
     "dest_count": 5,
     "diff": 0,
     "check_type": "hash_window",
     "status": "ok",
     "segment": "source_shadow",
     "duration_ms": 45  // Thời gian chạy cực nhanh (thường < 100ms)
   }
   ```

---

## Kịch bản 2: Kiểm thử đối soát lệch (Global Hash Mismatch - Fallback to Window Check)

### Mục tiêu
Kiểm chứng khi có sự sai lệch dữ liệu (drift), Global Hash Check phát hiện không khớp và tự động fallback về chia nhỏ 672 windows 15-phút để quét tuần tự và drill down định vị bản ghi lỗi.

### Các bước thực hiện
1. **Tạo dữ liệu drift giả lập**:
   Chạy lệnh SQL sau để chèn 1 bản ghi chỉ có ở shadow Postgres (không có ở Mongo source) để tạo lệch count và hash:
   ```bash
   docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
       "INSERT INTO shadow_testexp.export_jobs (id, status, created_at, updated_at) VALUES ('999999', 'SUCCESS', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour') ON CONFLICT (id) DO NOTHING;"
   ```

2. **Gửi lệnh đối soát qua NATS**:
   ```bash
   nats req -s nats://cdc_worker:worker_secret_2026@localhost:14222 cdc.cmd.recon-check '{"table":"export_jobs","segment":"source_shadow","legacy":false}'
   ```

3. **Kết quả mong đợi (Expected Output)**:
   * Hệ thống đối soát chạy fallback và phát hiện ra bản ghi lệch.
   * Phản hồi JSON nhận về hiển thị status `drift`, `diff` âm (-1 do Dest thừa 1 bản ghi) và danh sách missing/stale IDs chứa `999999`:
     ```json
     {
       "id": 124,
       "target_table": "shadow_testexp.export_jobs",
       "diff": -1,
       "missing_count": 0,
       "stale_count": 1,
       "stale_ids": {
         "missing_from_dest": [],
         "missing_from_src": ["999999"],
         "mismatched": []
       },
       "check_type": "hash_window",
       "status": "drift"
     }
     ```

4. **Dọn dẹp dữ liệu drift**:
   Sau khi hoàn tất test, chạy lệnh SQL sau để xóa bản ghi giả lập và khôi phục trạng thái sạch cho database:
   ```bash
   docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "DELETE FROM shadow_testexp.export_jobs WHERE id = '999999';"
   ```

---

## Kịch bản 3: Phân mảnh Block (> 7 ngày)

### Mục tiêu
Kiểm chứng khi dải thời gian kiểm tra lớn (ví dụ 10 ngày), hệ thống tự động chia nhỏ thành các block tối đa 7 ngày/block để quét Global Hash lần lượt, tránh Full Table Scan gây treo database.

### Các bước thực hiện
1. **Gửi lệnh đối soát với lookback lớn**:
   (Lưu ý: Chạy lệnh đối soát chỉ định range thời gian `lo` và `hi` xa nhau > 7 ngày, ví dụ từ 2026-06-25 đến 2026-07-09).
   ```bash
   nats req -s nats://cdc_worker:worker_secret_2026@localhost:14222 cdc.cmd.recon-check '{"table":"export_jobs","segment":"source_shadow","legacy":false,"lo":"2026-06-25T00:00:00Z","hi":"2026-07-09T00:00:00Z"}'
   ```

2. **Kết quả mong đợi (Expected Output)**:
   * Nếu dữ liệu trong 14 ngày này sạch hoàn toàn, hệ thống trả về status `ok` nhanh chóng.
   * QC kiểm tra log của `cdc-worker` sẽ thấy in ra dòng log:
     `[tier2] global blocks hash match — no drift detected in range`
     chứng tỏ hệ thống đã chạy qua block partitioning thành công.
