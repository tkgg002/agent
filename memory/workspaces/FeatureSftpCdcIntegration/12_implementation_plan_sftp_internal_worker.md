# 12_implementation_plan_sftp_internal_worker.md — Kế hoạch Triển khai Chi tiết của AI

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## Mô hình Thực thi

```
Brain (Planner) → Muscle (Executor)
```

- **Brain** đã hoàn thành: Audit, thiết kế kiến trúc, lập plan, Full Doc Set.
- **Muscle** sẽ thực thi: Code, go get, wire, test, build verify.

---

## Thứ tự Thực thi của Muscle

### Step 1 — Kiểm tra go.mod
```bash
grep "pkg/sftp" /Users/trainguyen/Documents/work/data-hub/centralized-data-service/go.mod
grep "golang.org/x/crypto" /Users/trainguyen/Documents/work/data-hub/centralized-data-service/go.mod
```

### Step 2 — go get (nếu chưa có)
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service
go get github.com/pkg/sftp@latest
```

### Step 3 — Thêm SFTPWorkerConfig vào config/config.go
- Thêm struct `SFTPWorkerConfig` vào `config/config.go`.
- Thêm field `SFTPWorker SFTPWorkerConfig` vào `AppConfig` struct.

### Step 4 — Cập nhật config/config-local.yml
- Thêm section `sftpWorker:` với thông số Docker SFTP Host.

### Step 5 — Tạo sftp_worker.go
- Tạo file `internal/handler/shadow/sftp_worker.go` theo spec trong `09_tasks_solution_sftp_internal_worker.md`.

### Step 6 — Wire server_setup.go
- Thêm block `if cfg.SFTPWorker.Enabled { ... }` sau kafkaConsumer block.

### Step 7 — Tạo sftp_worker_test.go
- Unit tests theo `06_test_cases_sftp_internal_worker.md`.

### Step 8 — Verify
```bash
go build ./...
go test ./internal/handler/shadow/... -run TestParseCSV -v
go vet ./...
```

---

## Điều kiện Báo Done

- `go build ./...` exit code 0.
- Unit tests PASS.
- Log: `sftp_worker: started` khi service khởi động (nếu test chạy service).
