# Analysis - Audit Sink & Transmute Risks

## Phân tích kiến trúc Sink

### Kết luận chính
- **Kafka Consumer** (`internal/handler/shadow/`) = PRIMARY path, chạy cả local + production
- **Sink Worker** (`cmd/sinkworker/` + `internal/sinkworker/`) = Legacy V1, KHÔNG chạy ở bất kỳ đâu

### Bằng chứng
| Source | Kafka Consumer (PRIMARY) | Sink Worker (LEGACY) |
|--------|-------------------------|---------------------|
| Activity log | action=`kafka-consumer`, detail=`{"written":N, "batch_size":N}` | action=`kafka-consumer-sw`, detail=`{"topic","partition","offset","snap"}` |
| Prod activity | ✅ Có records | ❌ Không có records |
| Config | `kafka.enabled: true` + env vars inject brokers | Binary riêng, phải start thủ công |
| K8s | Embedded trong `cmd/worker` deployment | Không có deployment |
| Git history | Thêm 30/06 (commit `ac4d82c`, 282 files) | Có từ init 13/05 (commit `94aa71c`) |

### Lỗi phân tích ban đầu
- **Sai:** Đọc `config-production.yml` thấy `brokers: []` → kết luận Kafka Consumer không chạy ở prod
- **Đúng:** Production inject brokers qua env vars, `brokers: []` chỉ là template mặc định
- **Lesson ghi:** #config-assumption #env-vars-override

## Phân tích rủi ro

### Top 2 nguyên nhân trực tiếp gây miss data (PRODUCTION)

**1. SINK-C1: CommitInterval=1s (auto-commit)**
- File: `kafka_consumer.go:151`
- Code: `CommitInterval: time.Second`
- Impact: Mỗi lần crash/restart → buffer chưa flush bị mất vĩnh viễn
- Fix: 1 dòng code → `CommitInterval: 0`

**2. SINK-C2: Flush timing gap**
- File: `kafka_consumer.go:474` + `batch_buffer.go:155`
- Mechanism: offset auto-committed → 5s gap → data ghi DB
- Impact: Crash trong gap = data loss không recoverable
- Fix: ~50 LOC refactor (flush trước commit)

### Design Pattern Issue
- Toàn bộ Kafka consumer logic nằm sai layer (`handler/shadow/`)
- Vi phạm Single Responsibility: 1 package ôm transport + business + infrastructure
- 12 files, ~3000 LOC trong handler package
- Khuyến nghị tách: `consumer/kafka/` + `service/shadow/` + `handler/shadow/` (thin adapter)

## Thống kê rủi ro

| Severity | Kafka Consumer (PROD) | Sink Worker (Legacy) | Transmute (PROD) | Tổng |
|----------|----------------------|---------------------|-----------------|------|
| 🔴 Critical | 2 | 1 | 4 | 7 |
| 🟠 High | 6 | 1 | 6 | 13 |
| 🟡 Medium | 3 | 4 | 7 | 14 |
| 🟢 Low | 0 | 3 | 3 | 6 |
| **Tổng** | **11** | **9** | **20** | **40** |

## Historical Incidents
- 13 incidents trong 2 tháng (05-07/2026)
- 6 recurring patterns (BSON mismatch, V1↔V2 lệch, index hỏng, fire-and-forget, safety gate cứng, schema drift)
- 1 concurrency optimization đã thiết kế (34KB) nhưng CHƯA implement
