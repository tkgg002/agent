# 04_decisions.md — ADR (Architecture Decision Records)

> Mỗi quyết định = 1 ADR. Mô tả Context → Decision → Consequence → Alternative.

---

## ADR-001 — Chọn Vertical Slice (Modular Monolith) thay vì sub-folder hexagonal

**Status**: Proposed (chờ user approve)

**Context**:
Cấu trúc hexagonal hiện tại có 52 file `api/` + 32 file `app/commands/` + 24 file `app/queries/` + 27 file `infra/persistence/` flat. User phàn nàn "ko trực quan để dev debug, theo dõi và quản lý tiến trình". Mental model user: 4 nhóm A (table CRUD) / B (trigger) / C (connection) / D (health) — chứ không phải 4 layer.

**Decision**:
Refactor sang Vertical Slice: mỗi business capability = 1 module trong `internal/modules/<name>/` chứa toàn bộ stack (HTTP + write + read + persistence + domain) của capability đó.

**Consequence**:
- (+) Mental model user khớp 100% — mở folder = thấy 1 nhóm.
- (+) Mỗi module ≤ ~10 file → dễ scan.
- (+) Cross-module isolation enforce qua import rule.
- (+) Long-term: nếu cần tách microservice, mỗi module là 1 candidate.
- (–) Effort cao hơn (7-10 ngày Muscle).
- (–) Cần discipline: cấm import cross-module trực tiếp; phải qua eventbus hoặc qua port public của module.

**Alternative đã xem**:
- Hướng 1 (status quo + cleanup) → không giải quyết complaint user.
- Hướng 2 (sub-folder trong layer) → giảm flat nhưng dev vẫn jump 4 thư mục cho 1 feature.
- Hướng 4 (multi-service) → overkill, không match scope.

---

## ADR-002 — Domain entity sống bên trong module (KHÔNG còn `internal/domain/`)

**Status**: Proposed

**Context**:
Hiện 7 domain entity ở `internal/domain/{job,mapping,master,reconciliation,source}/` đều **anemic** (0 method), invariant logic nằm tại `infra/persistence/provisioning_state_machine.go`. Nguyên nhân: domain ở layer riêng → dev có xu hướng đẩy logic xuống persistence khi cần DB context.

**Decision**:
Di chuyển domain entity vào `internal/modules/<X>/domain.go` cùng package với command/query/repo. Entity được phép định nghĩa method (`Validate`, `CanTransitionTo`, `Apply`, …) vì cùng package có thể test mà không lo DB.

**Consequence**:
- (+) Hết anemic — domain entity có behavior gắn liền.
- (+) Test domain chạy in-memory, không DB.
- (+) Discover invariant dễ hơn: dev đọc 1 file domain.go thấy hết.
- (–) Mất "thuần khiết" của DDD layer truyền thống (domain layer riêng). Trade-off chấp nhận vì service nhỏ, không phải hệ thống lớn cần 4-layer cứng.

**Alternative**:
- Giữ `internal/domain/` riêng — đã thử qua Task #19, dẫn tới anemic domain. Loại.

---

## ADR-003 — `internal/model/` GORM bị xóa, GORM struct sống trong `repo.go` của module

**Status**: Proposed

**Context**:
`internal/model/` chứa 11 GORM struct chồng chéo với `internal/domain/` (vd: `model.TableRegistry` ↔ `domain/source.Object`). Dev mới hỏi: *"Mở model/ hay domain/ trước?"*.

**Decision**:
GORM struct di chuyển vào `repo.go` của module sử dụng nó. Đặt **unexported** (vd: `type ruleRow struct`) — không leak ra ngoài module.

Trường hợp 2 module thực sự dùng chung 1 GORM struct (rare), tạo `internal/platform/db/gormmodel/` shared.

**Consequence**:
- (+) Hết chồng chéo `model/` ↔ `domain/`.
- (+) GORM tag chỉ ảnh hưởng module sở hữu — cross-module không bị coupled.
- (+) Mapping `domain ↔ row` rõ ràng (`fromDomain`, `toDomain` helpers).
- (–) Duplicate field definition nếu 2 module cùng đọc 1 bảng. Mitigation: dùng `platform/db/gormmodel/` shared khi gặp.

**Alternative**:
- Giữ `model/` flat — không giải quyết chồng chéo.
- Đổi tên `model/` → `dbmodel/` — chỉ sửa cosmetics, vẫn coupled.

---

## ADR-004 — `internal/app/ports/` bị inline vào từng module

**Status**: Proposed

**Context**:
`internal/app/ports/{command_bus,publisher,query_bus,repository}.go` định nghĩa 4 interface global. Trong số đó `repository.go` ghi sẵn 5-6 repo port (mapping, registry, ...). Vấn đề: nếu module mới thêm port, phải sửa file global → coupling.

**Decision**:
- `bus.CommandBus`, `bus.Publisher`, `bus.QueryBus` → di chuyển sang `internal/platform/bus/` (cross-cutting, shared).
- `repository.RepoPort` của module → định nghĩa **inline** trong `internal/modules/<X>/repo.go` (xem ADR-002 + section 2 trong `03_implementation.md`).

**Consequence**:
- (+) Không cần file global "định nghĩa interface cho cả thiên hạ".
- (+) Mỗi module tự quản lý contract repo của mình.
- (–) Nếu 2 module thực sự share interface, phải tạo `internal/platform/shared/` (rare).

**Alternative**:
- Giữ `app/ports/` flat — gây phụ thuộc 1 file chung mỗi khi thêm module.

---

## ADR-005 — `internal/router/router.go` thin (≤ 50 LOC), module tự khai báo route

**Status**: Proposed

**Context**:
`router.go` hiện tại 408 LOC — mount tất cả route trong 1 file. Khi thêm endpoint mới, dev phải sửa file global → conflict với người khác.

**Decision**:
- `router.go` chỉ apply global middleware (recover, CORS, JWT).
- Mỗi module có `routes.go` định nghĩa `(m *Module) RegisterRoutes(app *fiber.App)`.
- Wire-up gọi `mappingMod.RegisterRoutes(app)` ở `server/wire.go`.

**Consequence**:
- (+) Router thin → dễ đọc.
- (+) Thêm route mới chỉ chạm 1 module.
- (+) Code review focus đúng phạm vi.
- (–) Khi cần tra "endpoint này được mount ở đâu", phải grep — bù lại bằng convention naming `RegisterRoutes`.

**Alternative**:
- Giữ router monolithic 408 LOC — tiếp tục conflict.
- Mount tự động qua reflection — quá magical, hard to debug.

---

## ADR-006 — Cross-module communication QUA `platform/eventbus/` HOẶC `platform/bus/`

**Status**: Proposed (defer — chỉ tạo khi gặp use case thực)

**Context**:
2 module có thể cần "biết" nhau (vd: registry hoàn tất → mapping invalidate cache). Nếu cho phép import trực tiếp, sẽ tạo dependency cycle dần dần.

**Decision**:
- Trong process: dùng `platform/eventbus/` — in-process pub/sub (channel-based hoặc lib `nats.go-events`). Module publish event, module khác subscribe.
- Cross-service hoặc cần durability: dùng `platform/bus/` (NATS CommandBus đã có).
- CẤM import trực tiếp `cdc-cms-service/internal/modules/<X>` từ `cdc-cms-service/internal/modules/<Y>`.

Trì hoãn `platform/eventbus/` tới khi có use case thực — không build trước.

**Consequence**:
- (+) Module isolation cứng.
- (+) Loose coupling — module có thể thay đổi internal mà không phá module khác.
- (–) Debug khó hơn 1 chút (event-driven). Mitigation: log đầy đủ + correlation ID.
- (–) Possible message order issue. Mitigation: bus.CommandBus sync; eventbus chỉ cho async notification.

**Alternative**:
- Cho phép import cross-module → quay về spaghetti.
- Module shared service trong `platform/` → khó scale khi module tăng.

---

## ADR-007 — Refactor PURE STRUCTURAL — không sửa logic, không sửa SQL, không sửa NATS subject

**Status**: Proposed (locked)

**Context**:
User yêu cầu rõ: "không cheat DB hay thay đổi config để đạt được kết quả". Refactor lớn → risk cao. Cần giảm risk bằng cách giới hạn scope.

**Decision**:
- KHÔNG sửa SQL query / migration.
- KHÔNG sửa NATS subject name.
- KHÔNG sửa JSON response shape.
- KHÔNG sửa route path / HTTP method.
- KHÔNG upgrade thư viện.
- CHỈ: `git mv` + đổi import path + tách file lớn theo trục module + đưa logic ra khỏi sai chỗ.

**Consequence**:
- (+) Diff dễ review (mostly rename + move).
- (+) Backward-compat 100% — client cũ vẫn hoạt động.
- (+) Risk regression giảm mạnh.
- (–) Hết phase refactor structural xong, vẫn còn nợ technical (vd: domain anemic chưa được fix triệt để ở Phase 0; cần phase tiếp theo "thicken domain"). Note vào lessons.

**Alternative**:
- Đồng thời refactor logic + structure → blast radius lớn, hard to revert.

---

## ADR-008 — Phase rollout có review gate sau mỗi phase (user explicit)

**Status**: Proposed

**Context**:
User yêu cầu: *"task lớn cẩn review"*. Đồng thời `lessons.md:2399` ghi nhận Task #19 ("đang làm rất lâu và rối kinh khủng") — minh chứng refactor mơ hồ thì fail.

**Decision**:
- 11 phase (P0..P10) trong `02_plan.md`.
- Sau mỗi phase: commit + tag + tự test + báo cho user → user approve trước phase tiếp.
- Critical gate: sau P2 (pilot module health) và sau P7 (provisioning sensitive).

**Consequence**:
- (+) User có quyền dừng / điều hướng giữa chừng.
- (+) Mỗi phase nhỏ → revert dễ.
- (+) Audit trail rõ.
- (–) Tiến độ chậm hơn nếu user busy. Mitigation: định nghĩa "auto-approve sau 24h không phản hồi" cho phase low-risk.

**Alternative**:
- Refactor 1 phát → risk cao + khó rollback.
- Continuous push không gate → user mất kiểm soát.

---

## Summary

| ADR | Tên | Status |
|-----|-----|--------|
| 001 | Vertical Slice | Proposed |
| 002 | Domain in module (no anemic) | Proposed |
| 003 | GORM struct in repo (no `model/`) | Proposed |
| 004 | Ports inline (no `app/ports/` global) | Proposed |
| 005 | Router thin + per-module routes | Proposed |
| 006 | Cross-module via eventbus (defer) | Proposed |
| 007 | Pure structural — no logic/SQL/NATS change | Locked |
| 008 | Phase rollout + review gate | Proposed |

**Đợi user approve toàn bộ ADR trước khi Muscle thực thi Phase 0**.
