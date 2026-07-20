# 08_tasks_phase_2.md - Danh sách Task chi tiết Phase 2

Checklist này chia nhỏ các hạng mục Phase 2 thành các task cụ thể cho Muscle thực thi:

## 1. Hạng mục P2-1: Tối ưu hóa Concurrency
- [x] Refactor `BatchBuffer.Flush` sử dụng `golang.org/x/sync/errgroup` song song 20 bảng.
- [x] Sửa đổi lifecycle context trong `BatchBuffer.Flush` sử dụng background context timeout 10s để tránh bị cancel khi shutdown.
- [x] Viết struct `TableDebouncer` trong package master hỗ trợ idleTimer + maxTimer.
- [x] Tích hợp Backpressure hãm tốc pull tin nhắn trong `TableDebouncer.Add()`.
- [x] Viết hàm `binarySearchSplit` chia để trị Poison Pill.
- [x] Định kỳ gọi `msg.InProgress()` trong `binarySearchSplit` để kéo dài AckWait (đối với NATS Core: reply lỗi Poison Pill isolated).
- [x] Phân loại transient errors để `Nak()` nhanh (đối với NATS Core: reply lỗi transient_db_error).
- [x] Tích hợp `TableDebouncer` vào `TransmuteHandler` thay thế logic xử lý cũ.

## 2. Hạng mục P2-2: Dọn dẹp bản ghi mồ côi (Flatten Orphan Cleanup)
- [x] Sửa đổi file `internal/service/master/transmute/flatten.go` để track các ID cũ của parent (gộp logic vào transmuter.go sau bulkUpsertMaster).
- [x] Thực hiện đối chiếu và sinh câu lệnh soft-delete cho các ID dư thừa mồ côi.
- [x] Viết unit tests kiểm thử logic dọn dẹp mảng co rút (5 -> 2 phần tử).

## 3. Hạng mục P2-3: Đối soát tự động (Kafka vs Shadow Recon) - HOÃN LẠI (POSTPONED)
- [ ] (Sẽ chuyển sang phase sau theo yêu cầu của User)

## 4. Hạng mục P2-4: Giải phóng Scheduler kẹt
- [x] Thêm method `cleanupStuckSchedules` vào `TransmuteScheduler`.
- [x] Thực thi câu lệnh SQL reset status của các schedules bị kẹt `running` quá 10 phút.
- [x] Gọi `cleanupStuckSchedules` ở đầu mỗi hàm `tick()`.
