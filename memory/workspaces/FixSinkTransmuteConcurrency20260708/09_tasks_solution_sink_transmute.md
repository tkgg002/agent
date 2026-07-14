# Hồ sơ Giải pháp Kỹ thuật: Concurrency & Batching Optimization

Tài liệu này tổng hợp thiết kế giải pháp cho hai bài toán tối ưu:

## 1. Giải pháp Parallel Flush
*   Tài liệu thiết kế chi tiết: [12_implementation_plan_sink_transmute.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/FixSinkTransmuteConcurrency20260708/12_implementation_plan_sink_transmute.md) phần 1.
*   Công nghệ sử dụng: `golang.org/x/sync/errgroup` phối hợp với Semaphore để bảo vệ kết nối DB.

## 2. Giải pháp Debounce Buffer & Poison Pill Fallback
*   Tài liệu thiết kế chi tiết: [12_implementation_plan_sink_transmute.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/FixSinkTransmuteConcurrency20260708/12_implementation_plan_sink_transmute.md) phần 2.
*   Công nghệ sử dụng: Mutex map, timer Go, NATS JetStream Pull (`ManualAck`, `Fetch`, `Ack`, `Nak`, `Term`), và `failed_sync_logs` DB table làm DLQ.
