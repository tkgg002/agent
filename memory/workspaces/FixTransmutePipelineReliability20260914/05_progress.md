# Progress — Fix Transmute Pipeline Reliability

## [2026-09-08] [Agent:Gemini] Phiên 1 — Audit + Fix EnsureMaster
- Audit 6 drop points trong transmute pipeline
- Comment out EnsureMaster block trong transmuter.go L208-232
- Comment out delete(ensuredMasters) trong transmuter.go L652
- Build pass: `go build ./internal/...` ✅
- Xác nhận: index thiếu updatedAt trên shadow + master bank_requests_bvb
- Xác nhận: CountRows full scan timeout trên smoke recon

## [2026-09-14] [Agent:Gemini] Phiên 2 — Lên Plan chi tiết
- Tạo implementation plan chi tiết cho 5 fix còn lại
- Tạo workspace FixTransmutePipelineReliability20260914
- Đã khởi tạo đầy đủ bộ tài liệu workspace: 00_context.md, 01_requirements.md, 02_plan.md, 08_tasks.md, 09_tasks_solution_transmute_reliability.md, 12_implementation_plan_transmute_reliability.md, 13_analysis_transmute_drops.md
- Chờ user approve plan trước khi uỷ quyền Muscle thực thi code

## [2026-09-14 13:54:00] [Agent:Gemini] Phiên 3 — Audit chuyên sâu & hoàn thiện Plan chi tiết 100% dòng code
- Đã đọc GEMINI.md và lessons.md.
- Hoàn thành audit tận gốc:
  1. Tab Log Transmute rỗng: `ReconPipelineGrid.tsx:282` truyền dư `sourceDb` kích hoạt `needJoins = true`; trong `activity_log_read_repo_gorm.go` LATERAL join `tm` thiếu `source_object_id` và join `tm_so` bị NULL dẫn đến filter `COALESCE(...) = source_database` luôn false.
  2. Rớt Trigger Transmute: `transmuter.go:949-952` nuốt lỗi DB trong `bulkUpsertMaster` (im lặng `continue` và trả về `err = nil`), khiến caller báo `success` và vô hiệu hóa hoàn toàn cơ chế `binarySearchSplit` Poison Pill; dung sai OCC clock skew 2s quá hẹp làm drop update khi có độ trễ; và join `shadow_binding_id` thiếu fallback `source_object_id`.
- Đã ghi nhận toàn bộ mã code cụ thể vào `09_tasks_solution_transmute_reliability.md` và `implementation_plan.md`. Trình duyệt User phê duyệt trước khi sửa code.

## [2026-09-14 14:30:00] [Agent:Antigravity] Phiên 4 — Thiết kế giải pháp kiến trúc NATS Core -> JetStream cho Transmute Pipeline
- Đã đọc GEMINI.md và lessons.md.
- Tiếp thu chỉ đạo quyết liệt từ User:
  1. Loại bỏ hoàn toàn các đề xuất chạm vào CMS Web / CMS Service (Tab Log Transmute đã được fix).
  2. Không dùng thủ thuật chắp vá/cheat code.
  3. Tập trung 100% vào giải pháp kiến trúc gốc rễ theo lệnh của User: Chuyển đổi toàn bộ Transmute Trigger Pipeline từ NATS Core sang JetStream.
- Audit & giải quyết dứt điểm 5 Failure Modes do User chỉ ra:
  1. Worker crash/restart/OOM: Chuyển sang JetStream FileStorage + ManualAck(60s). Message CHỈ ĐƯỢC Ack sau khi Master Upsert commit thành công vào DB. Nếu worker crash giữa chừng, JetStream tự động redeliver cho worker sau khi restart -> không mất event.
  2. DB chậm/timeout 5s lookup: Khi DB quá tải lookup fail, transmute-shadow handler trả lỗi và gọi `msg.NakWithDelay(2s)` để JetStream retry sau khi DB hạ nhiệt -> không rớt event.
  3. NATS disconnect tạm thời: Chuyển publisher ở `batch_buffer_fanout.go` sang `js.PublishMsg()` với 3 lần retry exponential backoff và dedup ID -> đảm bảo ghi nhận thành công trên Stream.
  4. Debouncer overflow / Mutex Deadlock: Sử dụng cơ chế backpressure tự nhiên của JetStream PullSubscribe (Fetch) thay vì nhồi nhét vào RAM; sửa deadlock ở `triggerFlushLocked` (không block channel khi đang giữ Mutex).
  5. Scheduler stuck running: Thêm ticker định kỳ 5 phút cleanup schedules stuck `running` > 30 phút; tăng LIMIT từ 10 lên 50.
## [2026-09-14 16:05:00] [Agent:Antigravity] Phiên 5 — Tiếp thu phản biện của User & Chuẩn hóa 6 giải pháp kiến trúc Go Concurrency & JetStream
- Đã đọc GEMINI.md và lessons.md.
- Nhận phản biện từ User (Principal Engineer) chỉ ra 6 lỗi sai kiến trúc và vận hành trong code demo:
  1. Deadlock fix bằng `go func` phá vỡ thứ tự FIFO và gây rò rỉ goroutine -> Chuyển sang Idiomatic Go: tách extraction dưới Lock, sau đó blocking send ngoài Lock.
  2. Heartbeat `msg.InProgress()` vô hạn tạo Zombie Worker -> Bổ sung `context.WithTimeout` (deadline trần) cho tác vụ; dừng ticker và `Nak` nếu quá deadline.
  3. `reqCorrelationID` gây drop dữ liệu âm thầm khi batch kế tiếp cùng ID trong 2m window -> Sửa `Nats-Msg-Id` phản ánh nội dung duy nhất của batch: `ts-<table_name>-<batch_uuid>-<seq>`.
  4. Thiếu `FilterSubject` khiến Consumer kéo nhầm dữ liệu của nhau -> Cấu hình tường minh `FilterSubject` riêng cho từng Consumer (Shadow vs Master).
  5. Vòng lặp Poison Pill vô tận -> Khai báo `MaxDeliver: 5` trên Consumer Config, kiểm tra `meta.NumDelivered >= 5` để gọi `msg.Term()` và đẩy vào DLQ/log.
  6. Scheduler reset cứng 30 phút mâu thuẫn với Full Transmute bảng lớn -> Chuyển sang Heartbeat Cột `updated_at = NOW()` mỗi 1 phút từ worker; Scheduler chỉ reset khi `updated_at < NOW() - INTERVAL '10 minutes'`.
- Ghi nhận bài học kinh nghiệm mới vào `agent/memory/global/lessons.md`.
- Cập nhật tài liệu kỹ thuật `09_tasks_solution_transmute_reliability.md` và artifact `implementation_plan.md`.

## [2026-09-14 17:05:00] [Agent:Antigravity] Phiên 6 — Phản biện bậc cao (Deep Adversarial Audit): 5 bẫy ngầm phân tán & edge cases
- Đã đọc GEMINI.md và lessons.md.
- Tự rà soát phản biện sâu toàn bộ hệ thống phát hiện 5 bẫy ngầm rủi ro cao:
  1. `isTransientError` thiếu Postgres Deadlock (`40P01`), Statement Timeout (`57014`) và Deadline Exceeded -> Dẫn đến coi nhầm transient lock contention là Poison Pill và gọi `msg.Term()` làm mất data hợp lệ.
  2. Startup `cleanupStuckSchedules` reset mù quáng `WHERE last_status = 'running'` -> Phá vỡ Rolling Update trên K8s (Pod 1 khởi động sẽ giết nhầm job đang chạy của Pod 2).
  3. `processSubBatch` trong `binarySearchSplit` thiếu `t.Msg.Ack()` cho nửa batch thành công -> JetStream redeliver gây duplicate transmutation.
  4. Heartbeat thiếu fallback theo `master_table` khi `ScheduleID == 0` (chạy thủ công/CLI) -> Khiến job hợp lệ bị scheduler dọn dẹp nhầm.
  5. Vòng lặp `time.After(10ms)` busy-wait trong `debounce.go` là mã thừa/chết sau khi đã có `flushBatch` blocking send và `MaxAckPending`.
- Cập nhật giải pháp hoàn thiện vào `09_tasks_solution_transmute_reliability.md` và artifact `implementation_plan.md`.

