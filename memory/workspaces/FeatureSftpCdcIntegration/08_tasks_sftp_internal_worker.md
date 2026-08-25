# 08_tasks_sftp_internal_worker.md — Danh sách Task Chi tiết

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## Phase A: Dependencies & Config

- [ ] **A1** — Kiểm tra `go.mod` xem `github.com/pkg/sftp` đã có chưa.
- [ ] **A2** — `go get github.com/pkg/sftp` (nếu chưa có).
- [ ] **A3** — Kiểm tra `golang.org/x/crypto` đã có trong `go.mod` chưa (dep của pkg/sftp).
- [ ] **A4** — Thêm `SFTPWorkerConfig` struct vào `config/config.go`.
- [ ] **A5** — Thêm field `SFTPWorker SFTPWorkerConfig` vào `AppConfig` struct.
- [ ] **A6** — Thêm section `sftpWorker:` vào `config/config-local.yml`.

---

## Phase B: Core Worker Implementation

- [ ] **B1** — Tạo file `internal/handler/shadow/sftp_worker.go`.
- [ ] **B2** — Implement `SFTPPollingWorker` struct + constructor `NewSFTPPollingWorker`.
- [ ] **B3** — Implement `Start(ctx)` goroutine với `time.NewTicker`.
- [ ] **B4** — Implement `Stop()` via context cancel.
- [ ] **B5** — Implement `pollOnce(ctx)` — entry point cho mỗi poll cycle.
- [ ] **B6** — Implement `connectSFTP()` — SSH + pkg/sftp dial.
- [ ] **B7** — Implement `discoverFiles()` — ReadDir + filter by FilePattern regexp.
- [ ] **B8** — Implement `processFile()` — Open, read, parse CSV.
- [ ] **B9** — Implement `parseCSVRows()` — header detection + row → flat JSON.
- [ ] **B10** — Implement `publishRows()` — kafka-go writer, push messages.
- [ ] **B11** — Implement `moveFile()` — SFTP Rename (processed/error).

---

## Phase C: Wiring

- [ ] **C1** — Thêm import `handlershadow` vào `server_setup.go` (đã có, verify).
- [ ] **C2** — Thêm block `if cfg.SFTPWorker.Enabled { ... }` sau kafkaConsumer block.
- [ ] **C3** — `RegisterOnStart` + `RegisterOnStop` cho `SFTPPollingWorker`.

---

## Phase D: Tests

- [ ] **D1** — Tạo `internal/handler/shadow/sftp_worker_test.go`.
- [ ] **D2** — Test UT-01: `TestParseCSVRows_HappyPath`.
- [ ] **D3** — Test UT-02: `TestParseCSVRows_EmptyFile`.
- [ ] **D4** — Test UT-04: `TestFilePatternMatch_Valid`.
- [ ] **D5** — Test UT-05: `TestFilePatternMatch_Invalid`.
- [ ] **D6** — `go test ./internal/handler/shadow/... -run TestParseCSV -v`.

---

## Phase E: Verification

- [ ] **E1** — `go build ./...` — 0 errors.
- [ ] **E2** — `go vet ./...` — 0 warnings.
- [ ] **E3** — Manual E2E: upload CSV → verify shadow DB row.
