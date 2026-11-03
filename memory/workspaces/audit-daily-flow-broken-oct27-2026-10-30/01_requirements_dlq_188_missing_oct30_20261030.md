# 01. REQUIREMENTS — `dlq_188_missing_oct30_20261030`

- **Workspace:** `audit-daily-flow-broken-oct27-2026-10-30`
- **tier:** T2
- **mode:** standard
- **Suffix:** `dlq_188_missing_oct30_20261030`
- **Ngày tạo:** `2026-10-30T14:07:30+07:00`

## 1. Yêu cầu của User
User cung cấp dòng dữ liệu ngày `2026-10-30` sau khi khôi phục bảng **Thống kê đối soát 3 trạm & DLQ theo ngày — Pipeline**:
- `1. SNAPSHOT / RE-SYNC`: Shadow Nạp `250,137` | Master Sync `—`
- `2. KAFKA CDC BẮN VỀ`: Tổng `539` (`232 c`, `307 u`, `0 d`)
- `3. GHI VÀO DB SHADOW`: `539 Ops (237 dòng DB)` (`232 c`, `307 u`, `0 d`)
- `4. GHI VÀO DB MASTER (Từ CDC Stream)`: `250,634 Ops (250,420 dòng DB)` (`250,409 Inserted`, `225 Updated`, `0 Delete`, `0 Skipped`)
- `5. ĐỐI SOÁT & DLQ (LỖI LUỒNG CDC)`: `DLQ / Lỗi = 188`, `Lệch thiếu = —`, `Lệch stale = —`, `Đã heal = 0`, `ĐÁNH GIÁ = CÓ LỆCH`
- Câu hỏi của User: **"có quăng ra mớ DLQ, rồi bọn nó đâu."**

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L005 (Propose-Only Scope Guard):** Tuyệt đối không sửa source code trước khi có `APPROVE` tường minh.
- **L020 (End-to-End Callchain Tracing):** Truy vết toàn trình cách cột `DLQ / Lỗi = 188` trong `GetDailyFlow` được tính toán so với nơi hiển thị/lưu trữ DLQ (`cdc_dlq_events`, modal click vào DLQ hoặc trang DLQ Management, và `transmute` / `kafka-consumer` error logs), đồng thời kiểm tra luôn vì sao `250,137` dòng Snapshot sang Master lại nằm ở cột `4. GHI VÀO DB MASTER (Từ CDC Stream)` (`250,634 Ops`) thay vì `1. SNAPSHOT / RE-SYNC -> Master Sync`.
- **L031 & L032 (Zero DB CLI / No Container Exec):** Tuyệt đối cấm chạy `docker exec ... psql`, `psql`, hay bất kỳ DB CLI nào.
