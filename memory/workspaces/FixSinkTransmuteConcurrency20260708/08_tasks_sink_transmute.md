# Danh sách Task Chi tiết: Concurrency & Batching Optimization

Dưới đây là danh sách đầu việc chi tiết để thực hiện tối ưu hóa luồng ghi song song và cơ chế gom lô vòng 2:

## Phase 1: Song song hóa Sink Flush (`BatchBuffer`)
- [ ] Tích hợp `golang.org/x/sync/errgroup` vào file `internal/handler/shadow/batch_buffer.go`.
- [ ] Thay đổi vòng lặp tuần tự trong `Flush()` thành gọi `g.Go(...)` song song với giới hạn concurrency bằng `g.SetLimit(20)`.
- [ ] Bảo vệ biến tích lũy `written` và biến lưu trữ lỗi `err` bằng `sync.Mutex`.
- [ ] Kiểm chứng compile thành công: `go build ./internal/handler/shadow/...`.

## Phase 2: Giới hạn Concurrency tại Transmute Handler
- [ ] Bổ sung trường `sem chan struct{}` vào cấu trúc `TransmuteHandler` trong `transmute_handler.go`.
- [ ] Khởi tạo semaphore channel trong `NewTransmuteHandler` với dung lượng được cấu hình động từ struct config (mặc định là 10).
- [ ] Cập nhật logic `HandleTransmute` để lấy slot semaphore trước khi thực thi `h.svc.Run` và giải phóng slot khi hoàn thành.
- [ ] Kiểm chứng compile thành công: `go build ./internal/handler/master/...`.

## Phase 3: Xây dựng Debounce Buffer & Late ACK
- [ ] Thiết kế struct `TableDebounceBuffer` gom nhóm `nats.Msg` theo bảng đích.
- [ ] Tích hợp vòng lặp gom nhóm trong RAM với timer 1s và giới hạn kích thước 500 tin nhắn.
- [ ] Triển khai cơ chế Late ACK: Chỉ gọi `msg.Ack()` sau khi mẻ ghi DB thành công; gọi `msg.Nak()` khi ghi DB thất bại.
- [ ] Tích hợp fallback ghi tuần tự từng dòng khi mẻ ghi bị lỗi cú pháp dữ liệu (Poison Pill).
