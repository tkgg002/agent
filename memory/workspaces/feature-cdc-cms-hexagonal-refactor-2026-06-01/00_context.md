# 00_context.md — Bối cảnh refactor `cdc-cms-service` (Iteration v2)

| Field | Value |
|-------|-------|
| **Workspace** | `feature-cdc-cms-hexagonal-refactor-2026-06-01` |
| **Iteration** | **v2** (tiếp nối `feature-cdc-cms-service-restructure-2026-05-19` — PENDING REVIEW 13 ngày) |
| **Owner Brain** | Antigravity / Claude (GEMINI.md §1) |
| **Scope** | **Brain-only** — Plan & document. KHÔNG chạm 1 dòng code (§12). |
| **Date** | 2026-06-01 |
| **Target service** | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service` |
| **Hiện trạng** | 76 .go file / ~14,000 LOC / 333-LOC `server.go` / 32 commands / 30 queries / 53 routes / 26 NATS subjects |

---

## 1. Lý do tạo workspace v2 (không reuse workspace cũ)

### 1.1 Workspace cũ
- Path: `agent/memory/workspaces/feature-cdc-cms-service-restructure-2026-05-19/`
- Output: 11 file / 89 KB doc
- Hướng đã chọn: **Hướng 3 — Vertical Slice Modular Monolith** (`internal/modules/` + `internal/platform/`)
- Roadmap: 11 phase rollout
- Status: ⏳ **PENDING USER REVIEW** từ 2026-05-19 (chưa approve, chưa Muscle thực thi)

### 1.2 Vì sao cần v2
| Lý do | Workspace cũ (v1) | Workspace mới (v2) |
|---|---|---|
| Số phase | 11 phase (quá phức tạp, blocker review) | **4 phase** (gọn, có Phase 0 chuẩn bị) |
| Phân tích bounded context | 9-10 modules đặt tên ad-hoc | **8 BC theo capability map thực tế** từ 18 chức năng đã list |
| 18 commands raw `*gorm.DB` | Không nhắc tới | **Phase 3 đặc thù** để khử |
| God Interface (`ports/repository.go`) | Có nhắc nhưng không tách thành phase riêng | **Phase 1 (priority cao nhất)** |
| Composition root | "wire.go pattern" generic | **Pure-func split-file** (rejection receiver-state) |
| Shared Kernel | Không bàn | **Rejection explicit** (False Cognate risk) |
| Test coverage gate | Không có | **Phase 0 mandatory** ≥ 60% baseline |
| Linter enforce cycle import | Không | **`go-arch-lint` / `depguard`** |
| Wizard cross-BC | Không xử lý | **Tách rõ "Wizard as Orchestration Saga"** |

### 1.3 Quan hệ với workspace cũ
- KHÔNG ghi đè / xóa workspace cũ.
- 00_context (file này) reference workspace cũ chỗ phù hợp.
- Nếu user approve v2 → workspace cũ chuyển trạng thái **SUPERSEDED**.
- Nếu user prefer v1 → workspace v2 tạm dừng, không có data destruction.

---

## 2. User input nguyên văn (round 2026-06-01)

> 1. *"xuất ra cấu trúc file chi tiết cho repo cdc-cms-service. có des chi tiết các pattern, chức năng"* → output: 7 nhóm pattern + cây thư mục đầy đủ.
> 2. *"review 1. Vertical Slicing... 2. Xóa bỏ God Interface... 3. Tách nhỏ Composition Root... 4. Triết xuất Shared Kernel"* → Claude review 4 đề xuất với verdict 7/10, 9/10, 6/10, **3/10 (reject Shared Kernel)**.
> 3. *"list ra các chức năng chính của repo này"* → output: 18 capability gom thành 8 BC.
> 4. *"đề xuất refactor cấu trúc lại như nào cho hợp lý đây"* → output: structure target + 4-phase roadmap + anti-pattern warning.
> 5. *"tạo 1 cái workspace giúp anh, đầy đủ tài liệu và giải pháp"* → workspace này.

### Ràng buộc user (sticky)
1. Đọc `agent/memory/global/lessons.md` trước tất cả.
2. Đọc `work/agent/GEMINI.md` để hiểu role/skill.
3. **Simplicity First, minimal impact**.
4. Chỉ làm đúng yêu cầu, theo đúng pattern của source, tối ưu nhất.
5. Luôn hướng core systems, KHÔNG cheat DB / config.
6. Plan phải rõ ràng, có code demo chi tiết.
7. Report dựa kết quả tính toán thực tế, note file thay đổi, không láo.
8. Kết thúc PHẢI check service work mới báo done.
9. Có `report_*.md` ghi rõ files thay đổi + số dòng code.

---

## 3. Mental model — 8 Bounded Context (mới so với v1)

User đã định danh ngầm 4 nhóm (Table CRUD / Trigger / Connection / Health). Round 2026-06-01 thảo luận sâu hơn → **8 BC chi tiết**:

| # | BC | Chức năng gom vào | Commands | Queries |
|---|---|---|---|---|
| 1 | `source` | Discovery + Registration + Provisioning Lifecycle | 8 | 6 |
| 2 | `mapping` | Mapping Rule + Schema Drift Approval | 4 | 4 |
| 3 | `master` | Master Binding | 5 | 1 |
| 4 | `transform` | Transmute Schedule + Snapshot + Backfill | 4 | 3 |
| 5 | `reconciliation` | Reconciliation + Self-Healing | 3 | 4 |
| 6 | `wizard` | Source→Master automation flow (cross-BC saga) | 3 | 2 |
| 7 | `system_control` | Connector (Debezium/Kafka Connect) + Worker Schedule | 4 | 3 |
| 8 | `observability` | Health + Alerts + Activity + Audit/QA | 2 | 5 |

→ Xem chi tiết mapping ở `09_tasks_solution.md` §2.

---

## 4. Bối cảnh kiến trúc hiện tại (snapshot 2026-06-01)

### 4.1 Số liệu thực tế (đo trực tiếp từ repo)
```
internal/app/commands/   : 32 files
internal/app/queries/    : 30 files
internal/app/ports/      : 4 files (command_bus, publisher, query_bus, repository)
internal/api/            : 52 files (split-file pattern theo verb)
internal/infra/persistence/ : 18+ *_repo_gorm.go
internal/server/server.go: 333 LOC (composition root)
internal/router/router.go : 428 LOC
ports/repository.go      : 151 LOC chứa 10+ interface (GOD INTERFACE)
Commands xài raw *gorm.DB : 18/32 (56% bypass port layer)
```

### 4.2 Pain points (cập nhật so với gap_analysis cũ)

| # | Pain | Bằng chứng đo được | Severity |
|---|---|---|---|
| P1 | **God Interface** `repository.go` | 151 LOC / 10 interface trong 1 file → mọi thay đổi đụng file chung | 🔴 HIGH |
| P2 | **18 commands ăn raw `*gorm.DB`** | 18/32 = 56% bypass hexagonal port | 🔴 HIGH |
| P3 | **Composition root phình** | `server.go` 333 LOC, khối `cmdBus.Register*` 100 dòng | 🟡 MED |
| P4 | **Flat layer**, không vertical slice | API 52 file / Commands 32 / Queries 30 → khó tìm logic 1 BC | 🟡 MED |
| P5 | **Bootstrap V1→V2 sync** chạy mỗi boot, 0% coverage | `bootstrap.SyncLegacyToV2Bootstrap` không có test | 🟠 RISK |
| P6 | **Naming chồng** `model/` vs `domain/` | TableRegistry ở 2 chỗ với 2 schema khác nhau | 🟡 MED |
| P7 | **Audit logic xen lẫn middleware + persistence + queries** | `audit.go` + `activity_logger.go` + `audit_handler.go` không cùng bounded context | 🟢 LOW |

---

## 5. Scope refactor (v2)

### 5.1 IN-SCOPE
- 4-phase roadmap: **Phase 0 (coverage) → Phase 1 (port split) → Phase 2 (composition root) → Phase 3 (refactor 18 commands) → [Phase 4 optional: vertical slice]**.
- Code demo Go đầy đủ trong `03_implementation.md` (3 module: `master` BC, port split, composition root pure-func).
- Linter rule `go-arch-lint` config để enforce cycle import.
- Review gate sau mỗi phase.
- `report_*.md` tổng kết.

### 5.2 OUT-OF-SCOPE
- KHÔNG sửa schema DB / migration.
- KHÔNG sửa YAML deployment / config-production.yml.
- KHÔNG thêm tính năng business.
- KHÔNG động vào `centralized-data-service`, `cdc-auth-service`, `cdc-cms-web`.
- **KHÔNG chạy `mv` / `git mv` / `Edit` / `Write` lên source `.go`** — Brain task §12.
- KHÔNG dùng DI framework (Wire/Fx) — chỉ pure function.

---

## 6. Tham chiếu liên quan (đã đọc round này)

| File | Lý do |
|---|---|
| `agent/GEMINI.md` | 14 rule + workflow list |
| `agent/memory/global/lessons.md` | grep keyword: hexagonal/CQRS/refactor/bounded context — 30+ lesson liên quan |
| `agent/memory/global/project_context.md` | cdc-system domain glossary + 4 service overview |
| `agent/memory/global/active_plans.md` | 50+ workspace history |
| `agent/memory/workspaces/feature-cdc-cms-service-restructure-2026-05-19/*` | Workspace v1 — diff & non-duplicate |
| `data-hub/cdc-cms-service/README.md` | Service overview + tech stack |
| `data-hub/cdc-cms-service/cmd/server/main.go` | Bootstrap entrypoint |
| `data-hub/cdc-cms-service/internal/server/server.go` | Composition root 333 LOC |
| `data-hub/cdc-cms-service/internal/router/router.go` | 53 routes |
| `data-hub/cdc-cms-service/internal/app/ports/*.go` | God Interface evidence |
| `data-hub/cdc-cms-service/internal/app/commands/*.go` | 32 commands + grep 18 ăn `*gorm.DB` |

---

## 7. Lesson áp dụng từ `lessons.md` (relevant)

| Lesson | Áp dụng vào phase nào |
|---|---|
| #1294 JSON serialization order khi migrate map → typed struct (CQRS Q-side) | Phase 4 — khi move queries phải byte-identical contract test |
| #1295 Hybrid command bus cần ResultBody | Đã có sẵn trong bus, giữ nguyên |
| Anti-over-abstraction (audit log không qua bus) | Phase 1 ports — chỉ tách port cho domain command, không cho audit |
| CQRS lift-and-shift verification | Phase 4 — symbol grep, không chỉ directory grep |
| "User đặt rule absolute → tuân thủ literal" | KHÔNG đề xuất exception cho rule §12 (Brain không sửa code) |
| Metadata Integrity log format | Mọi entry `05_progress.md` dùng `[Timestamp] [Agent:Model] Action` |

---

## 8. Definition of DONE workspace v2

✅ Đủ 12 file vật lý: 00, 01, 02, 03, 04, 05, 06, 07, 08, 09, 10 + `report_*.md`.
✅ Tham chiếu rõ workspace v1 (không duplicate).
✅ 8 BC mapping có data đo thực tế (số commands/queries chính xác).
✅ Code demo Go đầy đủ cho 3 case: BC slice / port split / composition root pure-func.
✅ ADR mỗi quyết định khó (vì sao reject Shared Kernel, vì sao pure-func, vì sao Phase 0 mandatory).
✅ `05_progress.md` append-only, có timestamp + model ID hợp lệ.
✅ `report_*.md` ghi số dòng + danh sách file vật lý.
✅ KHÔNG đụng 1 file `.go` nào.
✅ KHÔNG overwrite `lessons.md` / `04_decisions.md` global.
✅ Append 1 dòng vào `active_plans.md`.
