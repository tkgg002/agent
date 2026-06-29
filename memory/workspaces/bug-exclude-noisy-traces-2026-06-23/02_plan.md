# Plan: Exclude Noisy Traces from Telemetry

## Proposed Steps

### Phase 1: Research & Locate Spans
- [ ] Dùng `grep_search` để định vị chuỗi `cdc.batchbuffer.flush` và `cdc.cms.stuck_job_reaper` trong toàn bộ codebase.
- [ ] Phân tích các file chứa các span này để tìm cách loại bỏ việc khởi tạo span hoặc skip tracing mà không ảnh hưởng đến business logic của hàm.

### Phase 2: Implementation
- [ ] Loại bỏ span `cdc.batchbuffer.flush` khỏi `centralized-data-service` (hoặc module tương ứng).
- [ ] Loại bỏ span `cdc.cms.stuck_job_reaper` khỏi `cdc-cms-service` (hoặc module tương ứng).
- [ ] Đảm bảo giữ nguyên các logic xử lý nghiệp vụ đi kèm, chỉ loại bỏ phần `otel.Start` / `trace.Start` và kết thúc span.

### Phase 3: Compile & Verification
- [ ] Biên dịch lại các service bị ảnh hưởng (`go build ./...`) để đảm bảo không lỗi cú pháp hay thiếu import.
- [ ] Chạy tests (`go test ./...`) của các service bị ảnh hưởng để đảm bảo các thay đổi không làm hỏng test suites.
