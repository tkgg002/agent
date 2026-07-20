# Walkthrough — Fix Audit Sink & Transmute Issues (Phase 0, 1 & 2)

Đã hoàn thành và xác minh thành công toàn bộ các tasks trong kế hoạch hành động. Code đã được biên dịch thành công và vượt qua tất cả unit tests của shadow handler, master handler, và master service.

---

## Các thay đổi chính trong Phase 2

### 1. [MODIFIED] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go) (P2-1.A Concurrency & Detached Context)
- Refactor hàm `Flush()` để thực hiện DB upsert song song cho các bảng khác nhau bằng `golang.org/x/sync/errgroup` với giới hạn concurrency là 20.
- Tách biệt context trong `Flush()` bằng cách sử dụng `context.WithTimeout(context.Background(), 10*time.Second)` thay vì context hủy liên đới của parent, đảm bảo dữ liệu đợt cuối khi shutdown luôn được ghi DB thành công (best-effort drain) và không bị rollback do cancel signal.

### 2. [NEW] [debounce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/debounce.go) (P2-1.B Transmute Debouncer)
- Xây dựng `TableDebouncer` để gom lô tin nhắn transmute theo idle timeout (100ms) và max timeout (1s).
- Khống chế concurrency xử lý transmute theo từng master table bằng semaphore giới hạn 10 luồng song song.
- Tích hợp cơ chế Backpressure: Tạm dừng pull tin nhắn 10ms nếu bộ đệm hàng đợi trong RAM vượt quá `2 * maxSize`.

### 3. [MODIFIED] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) (P2-1.B Debouncer Integration & Poison Pill Split)
- Tích hợp `TableDebouncer` map theo master table trong `TransmuteHandler`.
- Định tuyến tất cả các incremental transmute requests qua `TableDebouncer` để gom mẻ.
- Triển khai thuật toán **Binary Search Split (chia để trị)** đệ quy độ phức tạp $O(\log N)$ để tự động cô lập Poison Pill (bản ghi lỗi dữ liệu), trả lỗi về vĩnh viễn cho task chứa record lỗi đó để ghi DLQ/cảnh báo, và tự động ACK/xử lý trôi chảy các bản ghi thành công khác.
- Phân loại lỗi transient (lỗi mạng/kết nối DB) để reply lỗi transient nhanh để caller thực hiện retry sau.

### 4. [MODIFIED] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) (P2-2 Flatten Orphan Cleanup)
- Sửa đổi hàm `Run()` để tự động dọn dẹp các dòng master mồ côi (stale rows) khi mảng document bị co rút kích thước (ví dụ 5 phần tử xuống 3 phần tử).
- Cơ chế: Quét các index từ $N$ đến $N+500$ (với $N$ là kích thước mảng mới) và tự động soft-delete các dòng có `_gpay_id` tương ứng tồn tại trong DB.

### 5. [MODIFIED] [transmute_scheduler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmute_scheduler.go) (P2-4 stuck Job Recovery)
- Thêm phương thức `cleanupStuckSchedules()` chạy ở đầu mỗi chu kỳ `tick()`.
- Tự động reset trạng thái các transmute schedule job bị kẹt ở `'running'` quá 10 phút (2x interval) về trạng thái `'failed'` kèm theo log detail để cron scheduler tự động trigger lại ở chu kỳ tiếp theo.

---

## Kết quả kiểm thử

Đã chạy thành công toàn bộ unit tests và build binaries thành công 100%:
- `go test -v ./internal/handler/shadow/...` → **PASS**
- `go test -v ./internal/service/master/...` → **PASS**
- `go test -v ./internal/handler/master/...` → **PASS**
- `go build -o /dev/null ./cmd/...` → **SUCCESS** (biên dịch sạch không lỗi)
