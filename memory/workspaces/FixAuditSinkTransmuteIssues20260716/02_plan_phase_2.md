# 02_plan_phase_2.md - Lộ trình triển khai Phase 2

Tài liệu này xác định các mốc lộ trình, thứ tự triển khai, và kế hoạch verify cho các tasks của Phase 2.

---

## 1. Thứ tự thực thi đề xuất
Kế hoạch triển khai được chia làm 2 bước tuần tự để kiểm soát rủi ro:

```
Bước 1: P2-4 (Scheduler Stuck Cleanup) & P2-2 (Flatten Orphan Cleanup)
  ↳ Task ít rủi ro nhất, tác động cục bộ trên logic transmute scheduler và flatten transformer.
                                ↓
Bước 2: P2-1 (Concurrency Optimization cho Sink & Transmute)
  ↳ Task lớn nhất, thay đổi lõi flush & NATS consumer debounce, yêu cầu test concurrency kỹ lưỡng.
```

---

## 2. Kế hoạch triển khai chi tiết

### Giai đoạn 1: Sửa chữa logic Orphan & Stuck Job (Ưu tiên an toàn dữ liệu và phục hồi)
- **Task P2-2 (Flatten Orphan):** 
  - Sửa đổi file `internal/service/master/transmute/flatten.go` để query các row cũ của parent ID trước khi bulk upsert, đối chiếu và soft-delete các dòng thừa.
- **Task P2-4 (Scheduler Stuck Cleanup):**
  - Thêm phương thức cleanup vào `TransmuteScheduler` chạy ở đầu mỗi `tick()`.

### Giai đoạn 2: Tối ưu hóa Concurrency (Sink & Transmute)
- **Task P2-1.A (Sink Concurrency):**
  - Refactor `BatchBuffer.Flush` với `errgroup` song song 20 bảng.
- **Task P2-1.B (Transmute Concurrency & Debounce):**
  - Viết struct `TableDebouncer` trong package master, thay thế logic handler cũ bằng debouncer mới.
  - Tích hợp backpressure và thuật toán `binarySearchSplit` chia để trị.

### Giai đoạn 3: Tác vụ hoãn lại (Deferred/Postponed)
- **Task P2-3 (Auto Reconciliation Kafka vs Shadow):** Hoãn lại theo yêu cầu của User. Sẽ lập kế hoạch và triển khai ở phase sau.


---

## 3. Kế hoạch xác minh (Verification Plan)

### Kiểm thử tự động (Unit / Integration Tests)
- **Test P2-2:** Viết test case transmute mảng với kích thước thay đổi (5 -> 2) -> assert chỉ còn 2 dòng hoạt động trong master table, 3 dòng còn lại có `_deleted = true`.
- **Test P2-4:** Mock schedule kẹt trạng thái `running` với `last_run_at` quá hạn -> trigger scheduler tick -> assert job tự động reset về `failed`.
- **Test P2-1:**
  - Viết test song song ghi dữ liệu cho 5 bảng khác nhau, giả lập 1 bảng bị lỗi DB -> assert 4 bảng còn lại vẫn ghi thành công (Sink).
  - Viết test case gửi mẻ tin nhắn NATS có chứa 1 tin nhắn lỗi (Poison Pill) -> assert debouncer tự động phân tách và cô lập được tin nhắn lỗi, term tin nhắn đó và gửi DLQ, các tin nhắn khác được ACK thành công (Transmute).

### Kiểm thử thủ công trên Staging
- Deploy code Concurrency lên Staging, chạy load test với TPS cao (2000 - 5000 msg/s) -> monitor connection pool, CPU/RAM, metrics `cdc_sink_events_dropped_total` và `cdc_dlq_write_failures_total`.
- Kill cưỡng bức container worker đang chạy -> verify scheduler tự giải phóng job kẹt sau timeout.
