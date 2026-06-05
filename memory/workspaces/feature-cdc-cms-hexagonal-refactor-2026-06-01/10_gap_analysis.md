# 10_gap_analysis.md — Phân tích lỗ hổng kiến trúc

> Đo trực tiếp từ repo `cdc-cms-service` ngày 2026-06-01. Tất cả số liệu có lệnh reproducible.

## 1. Số liệu nền (measurable)

| Metric | Giá trị | Cách đo |
|---|---|---|
| Total `.go` file | 76 | `find . -name "*.go" \| wc -l` |
| `internal/api/` | 52 files | `ls internal/api/*.go \| wc -l` |
| `internal/app/commands/` | 32 files | `ls internal/app/commands/*.go \| wc -l` |
| `internal/app/queries/` | 30 files | `ls internal/app/queries/*.go \| wc -l` |
| `internal/app/ports/` | 4 files | `ls internal/app/ports/*.go` |
| `internal/infra/persistence/` | 22 files | `ls internal/infra/persistence/*.go \| wc -l` |
| `internal/domain/` | 7 aggregate files | `find internal/domain -name "*.go"` |
| `internal/model/` | 11 GORM models | `ls internal/model/*.go \| wc -l` |
| `server.go` LOC | 333 | `wc -l internal/server/server.go` |
| `router.go` LOC | 428 | `wc -l internal/router/router.go` |
| `ports/repository.go` LOC | 151 | `wc -l internal/app/ports/repository.go` |
| Interface trong repository.go | 10+ | `grep -c "^type.*interface" internal/app/ports/repository.go` |
| Commands ăn raw `*gorm.DB` | **18/32 (56%)** | `grep -l "db \*gorm" internal/app/commands/*.go \| wc -l` |
| NATS subjects registered | 26 | `grep -c "RegisterSubject" internal/server/server.go` |
| Routes Fiber | 53 | `grep -cE "\.(Get\|Post\|Patch\|Delete)\(\"/" internal/router/router.go` |

---

## 2. Gap A — God Interface (`ports/repository.go`) 🔴 HIGH

### 2.1 Triệu chứng
File `internal/app/ports/repository.go` (151 LOC) chứa **10 interface** trong 1 file:
- `MappingRuleRepo` (6 method)
- `SourceRepo` (5 method)
- `MasterRepo` (4 method)
- `JobRepo` (4 method)
- `ReconReportRepo` (2 method)
- `FailedSyncLogRepo` (3 method)
- `SchemaLogRepo` (2 method)
- `PendingFieldRepo` (3 method)
- `WizardRepo` (4 method)
- `SystemConnectorRepo` (5 method)
- `RegistryRepo` (3 method)

### 2.2 Vì sao là Gap
- **File chung = merge conflict magnet**. Bất kỳ thay đổi nào (add method, rename, đổi signature) đều đụng 1 file → conflict song song.
- **Vi phạm Interface Segregation Principle**. `CreateMasterHandler` chỉ cần 1 method `Save()` nhưng phải mock cả 6 method của `MappingRuleRepo` để compile test (nếu đặt mock cùng package).
- **Comment "groups everything for review" là rationalize** sau khi đã viết — không phải kiến trúc đúng. Go idiom: *consumer defines the interface*.

### 2.3 Bằng chứng impact
- Test file `master_swap_test.go` phải implement 4 method `MasterRepo` chỉ để test `Swap()` → 75% mock code là dead weight.
- Khi Phase 2 cũ định tách `cdc_jobs` ra package `platform/bus/`, `JobRepo` trong file God → kéo theo phải import cả file → cycle.

### 2.4 Root cause
- Hexagonal refactor lúc đầu (Task #19) dồn mọi interface vào 1 file cho dễ review → quên break ra khi codebase phình.

---

## 3. Gap B — 18 commands raw `*gorm.DB` 🔴 HIGH

### 3.1 Triệu chứng
**18/32 (56%)** command file trong `internal/app/commands/` nhận `db *gorm.DB` trực tiếp:
```
approve_master.go, approve_schema_proposal.go, bulk_register_registry.go,
create_master.go, create_mapping_rule.go, create_schedule.go,
create_worker_schedule.go, mark_failed_log_retrying.go, register_registry.go,
reject_master.go, reject_schema_proposal.go, toggle_master_active.go,
toggle_schedule.go, update_mapping_rule.go, update_registry.go,
update_schedule.go, update_shadow_binding.go, update_source_object_v2.go
```

### 3.2 Vì sao là Gap
- **Phá vỡ Hexagonal**: app layer KHÔNG ĐƯỢC chạm `*gorm.DB`. Comment trong `internal/infra/persistence/doc.go` nói rõ: *"Upper layers (api, app, domain) must not touch *gorm.DB directly."*
- **Test khó**: phải spin up testcontainers Postgres để test 1 command → unit test chạy 10s/test.
- **Coupling kép**: command vừa biết ORM, vừa biết business → khi đổi ORM (vd sang sqlc) phải sửa cả command lẫn repo.

### 3.3 Root cause
- Khi refactor sang CQRS, command handler được tạo nhanh = lift-and-shift từ old handler → giữ nguyên `db *gorm.DB` thay vì wrap port.
- Không có lint rule cấm import `gorm.io/gorm` ngoài `internal/infra/persistence/`.

---

## 4. Gap C — Composition Root phình 🟡 MED

### 4.1 Triệu chứng
`internal/server/server.go` = **333 LOC** với:
- 53 dòng wire repos + queries (dòng 93–146)
- **100 dòng `cmdBus.RegisterSubject` + `RegisterSync`** (dòng 149–246) — khối phình to nhất
- 83 dòng init handlers (dòng 172–255)
- 27 dòng background workers + shutdown

### 4.2 Vì sao là Gap
- Merge conflict cao khi 2 PR cùng add command mới (cùng đụng vùng `RegisterSync`).
- Khó scan: dev mới nhìn 333 dòng không phân biệt được "đâu là infra setup, đâu là business wiring".
- Hidden order dependency: `cmdBus` phải có trước `RegisterSubject`, `RegisterSubject` phải trước `RegisterSync`, `alertMgr` phải trước `healthCollector.SetAlertManager(alertMgr)` — không có doc.

### 4.3 Root cause
- Refactor cũ đặt mọi wiring vào 1 hàm `New()` → đúng pattern Composition Root nhưng không split file.

---

## 5. Gap D — Flat Layer, không Vertical Slice 🟡 MED

### 5.1 Triệu chứng
- 52 file `internal/api/*.go` flat trong 1 folder.
- 32 file `internal/app/commands/` flat.
- 30 file `internal/app/queries/` flat.
- Để đổi 1 business rule của Master, dev phải:
  1. Mở `internal/api/master_registry_handler_*.go` (5 file split-verb)
  2. Mở `internal/app/commands/{create,approve,reject,toggle,swap}_master.go` (5 file)
  3. Mở `internal/app/queries/list_masters.go`
  4. Mở `internal/app/ports/repository.go` (đụng MasterRepo)
  5. Mở `internal/infra/persistence/master_read_repo_gorm.go` + `master_swap.go`
  → **12+ file** ở 5 folder khác nhau.

### 5.2 Vì sao là Gap
- **Cohesion thấp**: code thay đổi cùng lý do (business Master) nằm rải rác.
- **Ownership mờ**: team không thể chia "team A owns Master, team B owns Mapping" — sẽ luôn đụng chung folder.
- **Onboarding chậm**: dev mới phải học toàn bộ 4 layer trước khi hiểu 1 feature.

### 5.3 Root cause
- Refactor cũ (Task #19) optimize cho "lớp ngang" (api / app / infra / domain) — đúng Hexagonal layer nhưng sai về **granularity** ở mức codebase phình.

---

## 6. Gap E — Bootstrap không có test 🟠 RISK

### 6.1 Triệu chứng
`internal/bootstrap/` 2 file:
- `registry_mirror.go` (sync V1 legacy registry → V2 schema)
- `shadow_connection.go` (seed default shadow connection)

**Cả 2 chạy mỗi lần boot, 0 test file** trong `test/internal/bootstrap/`.

### 6.2 Vì sao là Gap
- Bootstrap fail = service không start. Risk cao khi data V1 corrupt.
- Refactor sẽ đụng (Phase 4 move folder) → không có test = không phát hiện regression.

### 6.3 Root cause
- Code bootstrap viết 1 lần lúc migrate V1→V2, không ai add test sau.

---

## 7. Gap F — Naming chồng `model/` vs `domain/` 🟡 MED

### 7.1 Triệu chứng
- `internal/model/source.go` chứa GORM struct `Source` (raw row).
- `internal/domain/source/object.go` chứa domain entity `Object` (clean Go).
- `internal/model/table_registry.go` (`TableRegistry`) vs `internal/domain/source/object.go` (cùng concept source object nhưng schema khác).

### 7.2 Vì sao là Gap
- Dev mới hỏi "import model nào?" → mơ hồ.
- Cùng business concept (source) tồn tại 2 struct → drift schema không kiểm soát.

### 7.3 Root cause
- Migrate V1→V2 chưa kết thúc — V1 vẫn dùng `model/`, V2 đã chuyển `domain/`.

---

## 8. Gap G — Audit logic xen lẫn 🟢 LOW

### 8.1 Triệu chứng
Audit/Activity concern rải qua:
- `internal/middleware/audit.go` — Fiber middleware intercept request.
- `internal/infra/persistence/activity_logger.go` — goroutine batch insert.
- `internal/api/activity_log_handler.go` — query endpoint.
- `internal/app/queries/list_activity_logs.go`, `get_activity_stats.go` — read model.

### 8.2 Vì sao là Gap (mức nhẹ)
- 4 vị trí khác nhau cho 1 concept "activity log" → khó tìm.
- Khi vertical slice → middleware sẽ ở `platform/` (cross-cutting), còn domain query sẽ ở BC `observability/` → tách rõ.

### 8.3 Root cause
- Middleware ra đời sớm (audit là cross-cutting), domain query ra đời sau khi cần endpoint UI.

---

## 9. Bảng tổng hợp Gap

| Gap | Severity | Phase target | Tool detect |
|---|---|---|---|
| A. God Interface | 🔴 HIGH | **Phase 1** | `wc -l ports/repository.go`, `grep -c "^type.*interface"` |
| B. 18 commands raw gorm | 🔴 HIGH | **Phase 3** | `grep -l "db \*gorm" commands/*.go` |
| C. Composition root phình | 🟡 MED | **Phase 2** | `wc -l server/server.go` |
| D. Flat layer | 🟡 MED | **Phase 4 (optional)** | manual review |
| E. Bootstrap 0 test | 🟠 RISK | **Phase 0 (mandatory)** | `find test/internal/bootstrap` → empty |
| F. Naming chồng model/domain | 🟡 MED | **Phase 4** | manual review |
| G. Audit logic xen lẫn | 🟢 LOW | **Phase 4** | manual |

---

## 10. Anti-pattern phát hiện thêm (so với workspace v1)

| Anti-pattern | Trigger | Mitigation |
|---|---|---|
| **False Cognate Shared Kernel** | User đề xuất `domain/shared/` cho cross-aggregate enum | REJECT — xem ADR-04 |
| **Receiver-state Composition Root** | User đề xuất `s.setupRepositories()` mutate s | REJECT — pure function thay thế |
| **Migration mid-refactor** | Phase 4 move folder mà chưa tách port (Gap A) | Phase 1 PHẢI trước Phase 4 (dependency cứng) |
| **Skip coverage baseline** | Refactor mà coverage < 60% | Phase 0 mandatory, không bypass |
| **CQRS lift-and-shift không verify** | Lesson cũ trong `lessons.md` #2719 | Phase 4 dùng symbol grep, không directory grep |
| **Over-migrate audit log qua CommandBus** | Lesson cũ trong `lessons.md` #2278 | KHÔNG đụng audit middleware writer trong refactor này |

---

## 11. Out-of-scope của gap analysis

- Schema DB migration → KHÔNG đụng.
- Performance (slow SQL, N+1) → workspace khác đã/đang xử lý (`bug-cms-slow-sql-probes-2026-05-26`, `fix-slow-sql-queries-2026-05-29`).
- Bug fix logic → KHÔNG refactor logic, chỉ di chuyển + đóng gói.

---

## 12. Đối chiếu với gap_analysis cũ (workspace 05-19)

Workspace cũ liệt 14 mismatch. Workspace v2 này:
- ✅ **Giữ + đo lại** Gap A, C, D, F, G (đã có trong v1).
- 🆕 **Thêm mới** Gap B (18 commands raw gorm — v1 không nhắc).
- 🆕 **Thêm mới** Gap E (bootstrap 0 test — v1 không nhắc).
- ❌ **Loại bỏ** một số pain nhỏ v1 đã có patch sau 2 tuần (vd "raw NATS bypass bus" — đã có CommandBus thay thế).
