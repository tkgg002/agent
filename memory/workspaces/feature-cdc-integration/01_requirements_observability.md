# Requirements: Observability — Mọi tính năng phải có nơi check

> Date: 2026-04-16
> Phase: observability
> Priority: High
> Source: Gap Analysis `10_gap_analysis_observability.md`

## Bối cảnh

Hệ thống CDC có 105 items (54 API, 13 NATS, 5 Schedule, 4 Background, 9 FE pages). Phân tích phát hiện 10 gaps — nhiều thành phần quan trọng không có nơi kiểm tra health/status. User phải check từng page, từng container log riêng lẻ.

## Yêu cầu

### R1: System Health Page — 1 nơi xem tổng quan toàn hệ thống
- 1 page duy nhất hiện status TẤT CẢ components
- Worker, Kafka, Debezium, NATS, Postgres, Redis, Airbyte
- Pipeline metrics: throughput, latency, lag, failed count
- Recon status: matched/drifted per table
- Auto-refresh mỗi 30 giây

### R2: Kafka Consumer Events → Activity Log
- Mỗi batch events consumed → ghi 1 entry Activity Log
- Details: topic, count, duration, success/failed
- Không ghi từng event (quá nhiều) — ghi tổng hợp per batch

### R3: NATS Command Results → Activity Log
- Mọi command handler `publishResult` → cũng ghi Activity Log
- Bao gồm: rows_affected, duration, error nếu có

### R4: Debezium + Kafka Connect Health
- Poll Kafka Connect REST API (`/connectors/status`)
- Hiện trên System Health page
- Alert nếu connector FAILED hoặc PAUSED

### R5: E2E Latency Measurement
- Đo thời gian từ Kafka message timestamp đến Postgres insert timestamp
- Hiện trung bình trên System Health page

### R6: Worker Log Persistence
- Worker structured logs (zap) → gửi qua OTel Logs exporter → SigNoz
- Hoặc fallback: log file rotation
