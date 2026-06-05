# 01_requirements.md — Yêu cầu chi tiết refactor

## 1. Mục tiêu (Goals)

| ID | Goal |
|---|---|
| G1 | Khử **Gap A** (God Interface) — tách `ports/repository.go` thành port-per-aggregate, mỗi file ≤ 4 interface hẹp ngữ nghĩa |
| G2 | Khử **Gap B** (18 commands raw gorm) — mọi command chỉ phụ thuộc port hẹp, không import `gorm.io/gorm` |
| G3 | Khử **Gap C** (composition root phình) — `server.go` ≤ 80 LOC, wiring tách thành 4-5 file pure-function cùng package |
| G4 | (Optional) Khử **Gap D** (flat layer) — vertical slice theo 8 BC |
| G5 | Bảo vệ **Gap E** (bootstrap test) — thêm test coverage cho `internal/bootstrap/` trước Phase 4 |
| G6 | Lint enforce — `go-arch-lint` hoặc `depguard` chặn cycle import giữa BC |

---

## 2. Functional Requirements

### 2.1 Phase 0 — Baseline (FR-0)
| ID | FR |
|---|---|
| FR-0.1 | Đo coverage baseline `make test-cover`, output `coverage_baseline_2026-06-01.txt` |
| FR-0.2 | Yêu cầu coverage `internal/app/` + `internal/server/` ≥ **60%**. Nếu < 60%, viết test bổ sung TRƯỚC khi bắt đầu Phase 1 |
| FR-0.3 | Thêm test cho `internal/bootstrap/registry_mirror.go` + `shadow_connection.go` (target ≥ 70% — high-risk module) |
| FR-0.4 | Setup `go-arch-lint` config với rule: chỉ `internal/infra/persistence/` được import `gorm.io/gorm` |

### 2.2 Phase 1 — Port Split (FR-1)
| ID | FR |
|---|---|
| FR-1.1 | Tạo `internal/app/ports/master_port.go`, `mapping_port.go`, `source_port.go`, `master_port.go`, `job_port.go`, `recon_port.go`, `wizard_port.go`, `system_connector_port.go`, `registry_port.go` |
| FR-1.2 | Mỗi port file chứa 2-4 interface ngữ nghĩa (Reader / Writer / Approver / Swapper / …), KHÔNG quá 80 LOC |
| FR-1.3 | XÓA `internal/app/ports/repository.go` sau khi mọi caller đã chuyển sang port hẹp |
| FR-1.4 | Mọi handler/command/query phải import interface hẹp tương ứng, KHÔNG import legacy `ports.MasterRepo` |
| FR-1.5 | Chạy `go build ./... && go test ./test/... -short` PASS |

### 2.3 Phase 2 — Composition Root (FR-2)
| ID | FR |
|---|---|
| FR-2.1 | Tách `internal/server/server.go` thành: `server.go` (orchestrate ≤80 LOC) + `infra.go` + `repos.go` + `bus.go` + `routes.go` + `workers.go` (cùng package `server`) |
| FR-2.2 | Mọi sub-builder là **pure function** — input/output explicit, KHÔNG receiver-state |
| FR-2.3 | `buildRepos(db)` trả `Repos` struct typed (KHÔNG `map[string]any`) |
| FR-2.4 | `registerCommandHandlers(bus, repos, infra)` tách hẳn — chứa 100 dòng `RegisterSubject` + `RegisterSync` |
| FR-2.5 | Test integration `make test-integration` PASS |

### 2.4 Phase 3 — Refactor 18 commands raw gorm (FR-3)
| ID | FR |
|---|---|
| FR-3.1 | Mỗi command trong list 18 đổi từ `db *gorm.DB` sang port hẹp (vd `MasterApprover`, `RegistryWriter`) |
| FR-3.2 | Implementation port hẹp đặt trong `internal/infra/persistence/<aggregate>_repo_gorm.go` (file đã có sẵn) |
| FR-3.3 | KHÔNG đụng business logic — chỉ wrap raw SQL thành method có ngữ nghĩa |
| FR-3.4 | Mỗi command refactor đi kèm test PASS — 1 commit / 1 command |
| FR-3.5 | Sau Phase 3: `grep -l "db \*gorm" internal/app/commands/*.go` = empty |

### 2.5 Phase 4 (OPTIONAL) — Vertical Slice (FR-4)
| ID | FR |
|---|---|
| FR-4.1 | Tạo `internal/bc/{source,mapping,master,transform,reconciliation,wizard,system_control,observability}/` |
| FR-4.2 | Move file theo BC mapping (xem `09_tasks_solution.md` §2): mỗi BC chứa `domain/`, `ports.go`, `commands/`, `queries/`, `infra/`, `api/` |
| FR-4.3 | Dùng `git mv` (KHÔNG delete+create) để giữ git blame |
| FR-4.4 | Linter enforce: `internal/bc/X/` KHÔNG import `internal/bc/Y/` (trừ `wizard` được phép) |
| FR-4.5 | Move thứ tự: source → master → mapping → transform → reconciliation → wizard → system_control → observability. Sau mỗi BC: full test PASS |
| FR-4.6 | Tạo `internal/platform/` chứa: `bus/` (CommandBus + Job), `auth/`, `ratelimit/`, `audit_mw/`, `deprecation/`, `observability/` |
| FR-4.7 | Tạo `internal/shared/` chứa CHỈ technical primitives (`naming/`, `pg/`, `id/`) — KHÔNG domain enum |

---

## 3. Non-Functional Requirements

| ID | NFR |
|---|---|
| NFR-1 | KHÔNG regression coverage — sau mỗi phase, coverage ≥ baseline |
| NFR-2 | KHÔNG đổi public API surface (53 routes giữ nguyên path + method + response schema) |
| NFR-3 | KHÔNG đổi NATS subject names (26 subject giữ nguyên) |
| NFR-4 | KHÔNG đổi DB schema |
| NFR-5 | Service start time ≤ baseline + 100ms |
| NFR-6 | Memory footprint ≤ baseline + 5% |
| NFR-7 | Mỗi PR phase ≤ 500 LOC delta (trừ Phase 4 move folder) |
| NFR-8 | Git blame KHÔNG bị break (Phase 4 dùng `git mv`) |
| NFR-9 | Lint `go vet` + `golangci-lint` PASS sau mỗi phase |
| NFR-10 | Swagger doc regenerate `make swagger` PASS, không drift |

---

## 4. Acceptance Criteria

| ID | AC | Verify command |
|---|---|---|
| AC-1 | Phase 0: coverage báo cáo lưu vật lý | `cat coverage_baseline_2026-06-01.txt` |
| AC-2 | Phase 1: 0 file import legacy `ports.MasterRepo` etc. | `grep -rn "ports.MasterRepo\|ports.MappingRuleRepo\|ports.SourceRepo" internal/` → empty |
| AC-3 | Phase 1: `wc -l internal/app/ports/repository.go` → File DELETED | `test ! -f internal/app/ports/repository.go` |
| AC-4 | Phase 2: `wc -l internal/server/server.go` ≤ 80 | `[ $(wc -l < internal/server/server.go) -le 80 ]` |
| AC-5 | Phase 2: tồn tại `infra.go`, `repos.go`, `bus.go`, `routes.go`, `workers.go` trong `internal/server/` | `ls internal/server/{infra,repos,bus,routes,workers}.go` |
| AC-6 | Phase 3: 0 command import `gorm.io/gorm` | `grep -rn "gorm.io/gorm" internal/app/commands/` → empty |
| AC-7 | Phase 4 (nếu làm): 0 cross-BC import (trừ wizard) | linter rule PASS |
| AC-8 | Mọi phase: `make test` PASS + `make test-integration` PASS | exit code 0 |
| AC-9 | Mọi phase: 53 routes phản hồi đúng status & schema | smoke test script `scripts/smoke_routes.sh` |
| AC-10 | Mọi phase: service start ≤ baseline + 100ms | `time ./bin/cms` (3 lần lấy trung bình) |

---

## 5. Definition of Done (mỗi Phase)

✅ Code change theo FR tương ứng.
✅ Test PASS (`make test` + `make test-integration`).
✅ Coverage ≥ baseline (không regression).
✅ `go vet` + `golangci-lint` PASS.
✅ Swagger regenerate PASS (`make swagger`).
✅ Smoke 53 routes PASS.
✅ `05_progress.md` append đầy đủ entry phase đó.
✅ `report_*.md` cập nhật danh sách file thay đổi + số LOC.
✅ User review approve phase trước khi sang phase tiếp.
✅ **Security review** `/security-agent` PASS (rule §8).

---

## 6. Risk Register

| ID | Risk | Probability | Impact | Mitigation |
|---|---|---|---|---|
| R1 | Phase 3 refactor 18 commands gây regression silent | MED | HIGH | 1 commit / 1 command + test riêng từng cái. Stop khi 3 commit fail liên tiếp (rule §8 escalation). |
| R2 | Phase 4 move folder phá import path → CI fail kéo dài | HIGH | MED | Move BC một, test sau mỗi BC. Dùng `git mv` để giữ blame. |
| R3 | Linter `go-arch-lint` config sai → false positive | LOW | LOW | Test config trên repo nhỏ trước, sample 1 BC |
| R4 | Bootstrap test phát hiện bug ẩn → blocker Phase 0 | MED | MED | Bug fix riêng workspace, KHÔNG nhập vào refactor. Reschedule timeline. |
| R5 | User reject Phase 4 sau khi Phase 1-3 xong → wasted scope | LOW | LOW | Phase 4 mark `OPTIONAL`. Phase 1-3 là **standalone value** (80% benefit). |
| R6 | NATS subject hoặc route đổi do typo → break worker/FE | LOW | HIGH | NFR-2/3 enforce. Test integration cover cả 2. |
| R7 | Phase 2 split file gây cycle giữa server_*.go | LOW | MED | Pure func, KHÔNG receiver-state — cycle khó xảy ra. `go vet ./...` detect. |
| R8 | Coverage không đạt 60% → Phase 0 kéo dài | MED | MED | Estimate 3 ngày Phase 0. Nếu > 1 tuần → re-plan |

---

## 7. Out-of-Scope (KHÔNG làm trong workspace này)

- ❌ Sửa schema DB / migration / SQL
- ❌ Sửa YAML deployment, k8s manifest, config-production.yml
- ❌ Thêm tính năng business / endpoint mới
- ❌ Refactor logic — chỉ di chuyển + đóng gói + đổi tên interface
- ❌ Đụng `centralized-data-service`, `cdc-auth-service`, `cdc-cms-web`
- ❌ Performance tuning (workspace khác đang xử lý)
- ❌ Migrate sang DI framework (Wire/Fx)
- ❌ Tạo `internal/domain/shared/` (Shared Kernel — xem ADR-04 reject)

---

## 8. Stakeholder & Approval gate

| Stakeholder | Vai trò | Approval gate |
|---|---|---|
| User (Owner) | Approve mỗi phase trước khi bắt đầu phase tiếp | After AC verify |
| Brain (Antigravity) | Plan + document, không sửa code | Already done (workspace này) |
| Muscle (Claude Code CLI) | Thực thi từng phase | After user approve plan |
| Security agent | Review trước khi merge | Mỗi PR phase |
