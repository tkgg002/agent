# 04_decisions.md — Architecture Decision Records (ADR)

> 8 ADR cho các quyết định khó / tranh cãi trong refactor v2. Mỗi ADR theo format: Context → Options → Decision → Consequences.

---

## ADR-01: Vertical Slice (8 BC) thay vì Horizontal layering

### Context
Cấu trúc hiện tại flat layer: `internal/app/commands/` (32 file), `internal/app/queries/` (30 file), `internal/api/` (52 file). Tìm logic 1 BC phải nhảy 3-4 folder. 8 BC đã được định danh từ 18 capability.

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Giữ flat layer | KHÔNG break | Khó tìm logic, BC bị scatter |
| B | Vertical slice `internal/bc/<bc>/{commands,queries,api}` | Co-location, BC-isolation rõ | Phải `git mv` 80 file, risk import path |
| C | Hybrid: vertical cho domain mới, flat cho legacy | Migrate dần | Tech debt 2 pattern song song lâu |

### Decision
**Option B — Vertical Slice** (Phase 4, OPTIONAL sau khi Phase 1-3 xong).

### Rationale
- Conway's Law: team backend đã ngầm chia ownership theo BC.
- Phase 1-3 standalone value 80% — nếu user reject Phase 4 vẫn giữ được lợi ích chính.
- `git mv` preserve blame (NFR-8).
- Linter `go-arch-lint` enforce → tự động ngăn drift.

### Consequences
- ✅ Mỗi BC self-contained → ownership rõ, onboarding nhanh.
- ✅ Cross-BC import bị linter chặn → cycle import gần như impossible.
- ⚠ Phase 4 cao risk (R2): move folder phá CI nếu sai thứ tự.
- ⚠ Wizard BC phải có ngoại lệ (xem ADR-08).

### Liên quan
- `09_tasks_solution.md` §2 (8 BC mapping)
- `02_plan.md` §7 (Phase 4 detail)

---

## ADR-02: REJECT Shared Kernel cho domain

### Context
Đề xuất ban đầu (user): trích xuất `internal/domain/shared/` chứa common enum/value object dùng chung giữa BC (vd: `SchemaStatus`, `JobState`).

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Tạo `internal/domain/shared/` với enum chung | Đỡ duplicate | False Cognate — 2 BC nghĩ nghĩa khác nhau cho cùng symbol |
| B | Mỗi BC tự define enum riêng, accept duplicate code | Decoupled, evolve độc lập | Code lặp ~ 5-10% |
| C | `internal/shared/` CHỈ chứa technical primitive (`naming/`, `pg/`, `id/`) — KHÔNG domain | Tránh False Cognate, vẫn share infra | Phải nghiêm cấm domain enum vào |

### Decision
**REJECT Option A. Chọn Option B + Option C kết hợp.**

### Rationale
- **False Cognate risk** (DDD anti-pattern): `JobState` của `transform` BC khác semantics với `JobState` của `reconciliation` BC dù trùng tên.
- Coupling Shared Kernel = nghĩa vụ "thay đổi đồng bộ giữa các team" — phá vỡ BC autonomy.
- Lesson #1294 chỉ ra: rename map → typed struct phá JSON contract → cùng nguyên lý, share type = share invariant.
- Duplicate enum 5-10% << cost của Shared Kernel evolution lock.

### Consequences
- ✅ Mỗi BC evolve độc lập.
- ✅ KHÔNG có file `internal/domain/shared/`.
- ✅ `internal/shared/` chỉ chứa: `naming/` (schema/table convention), `pg/` (postgres helper), `id/` (UUID gen) — technical, không domain.
- ⚠ Phải có code review gắt: nếu PR thêm domain enum vào `internal/shared/` → reject.
- ⚠ Linter `go-arch-lint` add rule: `internal/shared/**` cannot depend `internal/domain/**`.

### Liên quan
- `01_requirements.md` FR-4.7
- `03_implementation.md` Demo 4 (linter config)

---

## ADR-03: Pure Function Composition Root (KHÔNG receiver-state)

### Context
`server.go` hiện 333 LOC dùng pattern receiver-state mutation: `s.initDB()`, `s.initBus()`, ... → khó test sub-step, order matter ngầm.

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Giữ receiver-state + tách thành sub-method | Refactor nhẹ | Vẫn implicit dep, order matter |
| B | Pure function với explicit input/output | Test dễ, dep rõ | Nhiều function param hơn |
| C | DI framework (Wire/Fx) | Auto-resolve | Thêm dependency mới, magic, học cost |

### Decision
**Option B — Pure Function.**

### Rationale
- Go idiom: explicit dependency tốt hơn magic.
- Pure func test riêng: `buildInfra(cfg)` không phụ thuộc `buildBus()` đã chạy chưa.
- Wire/Fx thêm complexity không tương xứng với scale dự án (8 BC, không phải 80).
- Receiver-state ẩn order matter → nếu Muscle đổi thứ tự call sẽ nil panic.

### Consequences
- ✅ Mỗi sub-builder pure, test isolation tốt.
- ✅ Không thêm dependency.
- ✅ Code review dễ thấy thiếu/thừa dependency.
- ⚠ `server.go` có vẻ "verbose" hơn (gọi tuần tự từng builder).
- ⚠ Phải kỷ luật KHÔNG capture state qua closure — code review enforce.

### Liên quan
- `03_implementation.md` Demo 2
- `01_requirements.md` FR-2.2

---

## ADR-04: Phase 0 (Coverage Baseline) là MANDATORY, không optional

### Context
Workspace v1 không có Phase 0. Refactor đẩy thẳng vào port split → risk silent regression khi test coverage thấp.

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Skip coverage gate, refactor luôn | Nhanh start | High risk silent regression |
| B | Soft gate: chỉ đo, không enforce | Có visibility | Không cản được Phase 1 khởi động khi coverage thấp |
| C | Hard gate ≥ 60% (app+server), ≥ 70% (bootstrap) | Safe refactor | Có thể tốn 3-5d viết test bổ sung |

### Decision
**Option C — Hard gate 60%/70%.**

### Rationale
- `internal/bootstrap/` chạy mỗi boot, có sync logic V1→V2 → cao risk bug ẩn. 0% coverage hiện tại = roulette.
- Refactor không có test = "thay tim không có ECG". Risk R1/R6 trong `02_plan.md` chỉ giảm được nếu có test.
- 3-5d Phase 0 vs 2 tuần debug regression production → ROI rõ.

### Consequences
- ✅ Safe refactor net.
- ✅ Bootstrap module được kiểm tra, có thể discover bug ẩn (Risk R4) → fix riêng workspace.
- ⚠ Có thể trễ start Phase 1 nếu coverage < 60% (mitigation: re-estimate 1 tuần max).

### Liên quan
- `01_requirements.md` FR-0
- `02_plan.md` §3

---

## ADR-05: 1 commit / 1 command trong Phase 3

### Context
18 commands refactor raw gorm → port. Risk silent regression cao (R1).

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | 1 PR refactor tất cả 18 cmd | Ít overhead PR | Khó revert 1 command, review nặng |
| B | 1 commit / 1 command, 1 PR / batch (3 PR cho 3 batch) | Granular revert, review nhẹ | Nhiều commit, có thể chậm |
| C | 1 PR / command (18 PR) | Review siêu nhỏ | Overhead quá cao |

### Decision
**Option B — 1 commit / 1 command, gom thành 3 PR theo batch (đơn giản → phức tạp).**

### Rationale
- Granular revert: nếu command X gây regression → revert riêng commit X, không phải revert cả Phase 3.
- Stop rule: 3 commit fail liên tiếp → STOP, escalate Brain re-plan.
- Batch giảm overhead PR (3 PR thay vì 18).

### Consequences
- ✅ Revert dễ, blame rõ.
- ✅ Test riêng từng cmd → fail thấy ngay.
- ⚠ Git history dài hơn — chấp nhận được.

### Liên quan
- `01_requirements.md` FR-3.4 + Risk R1
- `02_plan.md` §6

---

## ADR-06: KHÔNG dùng DI Framework (Wire / Fx / Dig)

### Context
Composition root phình → có thể dùng Wire để auto-generate DI graph.

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Pure function (đã chọn ADR-03) | KHÔNG thêm dep | Code verbose hơn |
| B | Google Wire (codegen) | Compile-time check, performant | Codegen step, learning curve |
| C | Uber Fx (runtime) | Powerful, lifecycle hook | Runtime magic, error khó debug |
| D | Dig (Uber, reflection) | Linh hoạt | Runtime, reflection cost |

### Decision
**Option A — Pure Function. EXPLICITLY REJECT B/C/D.**

### Rationale
- Scale dự án: 8 BC, ~ 40 wiring → pure func đủ readable.
- Wire codegen step thêm build complexity + Muscle phải learn.
- Fx runtime magic conflict với rule "Simplicity First" (CLAUDE.md §6).
- Lesson: framework thường giải bài "không có" cho dự án nhỏ.

### Consequences
- ✅ Build process không đổi.
- ✅ Onboarding mới đọc `buildInfra → buildRepos → buildBus` là hiểu ngay.
- ⚠ Nếu dự án scale lên 50 BC → có thể re-evaluate (workspace mới).

### Liên quan
- `00_context.md` §5.2 (out-of-scope: migrate sang DI framework)

---

## ADR-07: `internal/shared/` cho technical primitives, KHÔNG domain

### Context
Liên kết ADR-02. Sau khi reject Shared Kernel cho domain, cần định nghĩa rõ `internal/shared/` chứa gì.

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Không có `internal/shared/` chút nào | Triệt để | Lặp helper UUID/naming ở mọi BC |
| B | `internal/shared/` chứa CHỈ: `naming/`, `pg/`, `id/`, `httperr/` | Đỡ duplicate technical | Phải nghiêm cấm domain leak |
| C | `internal/shared/` chứa cả technical + 1 vài domain enum "rõ ràng common" | Practical | Slippery slope → quay lại Shared Kernel |

### Decision
**Option B — CHỈ technical primitives.**

### Rationale
- Technical (UUID gen, postgres helper, naming convention) không thay đổi semantics giữa BC → safe share.
- Domain enum (Status, State) phải BC-specific dù trùng tên (False Cognate).
- Linter enforce: `internal/shared/` cannot import `internal/domain/` hay `internal/bc/X/domain/`.

### Consequences
- ✅ DRY cho technical, decoupled cho domain.
- ✅ Linter rule chặn drift.
- ⚠ Code review phải cẩn thận: nếu PR thêm `type Status int` vào `internal/shared/` → reject ngay.

### Liên quan
- ADR-02
- `01_requirements.md` FR-4.7
- `03_implementation.md` Demo 4 (linter component `shared`)

---

## ADR-08: Wizard BC được phép cross-BC import (Saga exception)

### Context
Wizard BC là **Source→Master automation flow** — bản chất orchestrate nhiều BC khác. Nếu áp rule "0 cross-BC import" → wizard không hoạt động.

### Options
| Option | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Wizard import trực tiếp các BC khác (cross-BC OK) | Code straightforward | Vi phạm BC isolation |
| B | Wizard chỉ gọi qua CommandBus (async event-driven Saga) | Pure BC isolation | Async complex, debug khó |
| C | Wizard exception: được phép import `bc_source/commands`, `bc_mapping/commands`, `bc_master/commands`, `bc_transform/commands` | Pragmatic, code rõ | Phải document rõ "wizard là exception" |

### Decision
**Option C — Wizard exception, document rõ trong linter config + ADR.**

### Rationale
- Wizard là **Orchestration Saga** — pattern này yêu cầu biết các step concrete.
- Pure async (Option B) overhead lớn, debug khó, chậm hơn.
- Linter `go-arch-lint` cho phép define exception explicit → KHÔNG silent slip.
- Wizard là **đỉnh** của dependency graph (không có BC nào import wizard) → vẫn DAG, không cycle.

### Consequences
- ✅ Wizard code rõ ràng, dễ hiểu.
- ✅ Linter enforce: chỉ wizard mới cross-BC, các BC khác bị chặn.
- ⚠ Nếu wizard logic phình → có thể tách thành "Saga Engine" trong `internal/platform/` (workspace tương lai).

### Liên quan
- `00_context.md` §3 (Wizard BC)
- `03_implementation.md` Demo 4 §4.2 (linter exception)
- `09_tasks_solution.md` §2 (BC mapping)

---

## 9. Tóm tắt 8 ADR

| ADR | Quyết định | Status |
|---|---|---|
| ADR-01 | Vertical Slice (Phase 4 optional) | ✅ ACCEPT |
| ADR-02 | REJECT Shared Kernel cho domain | ✅ ACCEPT (negative decision) |
| ADR-03 | Pure Function Composition Root | ✅ ACCEPT |
| ADR-04 | Phase 0 Coverage Gate MANDATORY ≥60% | ✅ ACCEPT |
| ADR-05 | 1 commit / 1 command Phase 3 | ✅ ACCEPT |
| ADR-06 | KHÔNG DI Framework | ✅ ACCEPT (negative decision) |
| ADR-07 | `internal/shared/` technical only | ✅ ACCEPT |
| ADR-08 | Wizard Saga cross-BC exception | ✅ ACCEPT |

---

## 10. Quy trình thay đổi ADR

Nếu trong quá trình thực thi cần thay đổi ADR:
1. KHÔNG sửa ADR cũ (immutable like `05_progress.md`).
2. Tạo ADR mới: `ADR-XX-REVISED-ADR-YY.md` trong workspace.
3. Append vào `05_progress.md` với rationale.
4. User approve mới được áp dụng.
