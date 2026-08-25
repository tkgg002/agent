# 02_plan_sftp_internal_worker.md — Roadmap Cao tầng

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## Khai báo Skill sử dụng (Pre-flight Declaration)

1. **golang-patterns**: Idiomatic Go goroutine, context cancellation, struct patterns.
2. **clean-code**: Minimal impact, follow existing architecture DNA.
3. **systematic-debugging**: Verify từng bước trước khi báo Done.

---

## Roadmap Cao tầng

```
Phase A: Dependencies & Config
  └─ go get github.com/pkg/sftp
  └─ Thêm SFTPWorkerConfig vào config.go
  └─ Cập nhật config-local.yml

Phase B: Core Worker
  └─ Implement SFTPPollingWorker (sftp_worker.go)
      ├─ SFTP connection manager (SSH + pkg/sftp)
      ├─ File discovery & pattern matching
      ├─ CSV parser → flat JSON rows
      ├─ Kafka producer (segmentio/kafka-go writer)
      ├─ File lifecycle (move to processed/error)
      └─ Graceful shutdown via context

Phase C: Wiring & Registration
  └─ server_setup.go: RegisterOnStart/RegisterOnStop

Phase D: Tests
  └─ sftp_worker_test.go (unit + mock)

Phase E: Verification
  └─ go build ./... pass
  └─ Manual E2E test với Docker SFTP Host
```

---

## Thứ tự Ưu tiên

| Priority | Task |
|:---:|:---|
| P0 | Phase A: Config (prerequisite) |
| P1 | Phase B: `sftp_worker.go` core |
| P2 | Phase C: Wiring vào WorkerServer |
| P3 | Phase D: Unit Tests |
| P4 | Phase E: E2E Verification |

---

## Rủi ro & Giảm thiểu

| Rủi ro | Giảm thiểu |
|:---|:---|
| `github.com/pkg/sftp` chưa có trong go.mod | `go get` trước khi code |
| SSH host key verification thất bại | Disable strict host key check trong SSH config cho local dev |
| File bị đọc nhiều lần nếu move thất bại | Atomic check: chỉ process file nếu chưa tồn tại trong processed/ |
| Kafka writer timeout | Set WriteTimeout hợp lý, log error và skip (không crash) |
