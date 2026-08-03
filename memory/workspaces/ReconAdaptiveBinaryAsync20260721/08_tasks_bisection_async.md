# 08 — Danh Sách Công Việc (Task Checklist): Refactor Adaptive Binary & Async Job

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  

---

## Task Checklist Theo Phase

- [ ] **Phase 1: DB Schema & Binary Drill-Down Core Engine**
  - [ ] `TASK-1.1`: Tạo DDL migration bảng `cdc_system.recon_jobs`
  - [ ] `TASK-1.2`: Tạo repository `ReconJobRepository` với các phương thức Create, UpdateState, GetByID
  - [ ] `TASK-1.3`: Khởi tạo `BinaryDrillDownEngine` (`recon_bisection_engine.go`) với Merkle Tree Bisection & Parallel `errgroup`
  - [ ] `TASK-1.4`: Tạo Unit tests `recon_bisection_engine_test.go` (TC-BDD-01 -> TC-BDD-05)

- [ ] **Phase 2: Async Worker State Machine & Event Consumer**
  - [ ] `TASK-2.1`: Khai báo NATS Subject `cdc.event.recon.job_created` và Payload DTO
  - [ ] `TASK-2.2`: Viết `ReconJobWorker` tiêu thụ NATS message và quản lý vòng đời Job (`PENDING` -> `RUNNING` -> `COMPLETED`)
  - [ ] `TASK-2.3`: Tích hợp Checkpointing mốc thời gian và tính toán `progress_percent`

- [ ] **Phase 3: Control Plane APIs Integration**
  - [ ] `TASK-3.1`: Thêm Handler `POST /api/reconciliation/check-async`
  - [ ] `TASK-3.2`: Thêm Handler `GET /api/reconciliation/jobs/:job_id`
  - [ ] `TASK-3.3`: Thêm Router routes trong `internal/router/router.go`

- [ ] **Phase 4: Full Verification & Benchmark**
  - [ ] `TASK-4.1`: Chạy `go test -v ./internal/service/recon/...` PASS 100%
  - [ ] `TASK-4.2`: Integration test trên container Docker với 30 ngày dữ liệu test
