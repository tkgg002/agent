# 00_context.md — Bối cảnh refactor cấu trúc `cdc-cms-service`

| Field | Value |
|-------|-------|
| **Workspace** | `feature-cdc-cms-service-restructure-2026-05-19` |
| **Owner Brain** | Antigravity (CLAUDE.md §1) |
| **Scope** | **Brain-only** — Plan & document. KHÔNG chạm 1 dòng code (§12). |
| **Date** | 2026-05-19 |
| **Target service** | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service` (76 .go file, ~14,000 LOC) |

## 1. User input nguyên văn

> "đang rất khó chịu về patten & cấu trúc của cdc-cms-service. ko trực quan để dev debug, theo dõi và quản lý tiến trình."
>
> "nó chả có con mẹ gì. quản lý mấy cái table, thêm xóa sửa xong gọi triger, có thêm mấy cái connect để show, health service."
>
> "mày lên 1 plan để sắp xếp cấu trúc. ko làm 1 dòng code nào, task lớn cẩn review."

### Ràng buộc user note rõ ràng

1. Đọc `agent/memory/global/lessons.md` trước tất cả.
2. Đọc `work/agent/GEMINI.md` để hiểu role/skill.
3. Chỉ làm đúng những gì được yêu cầu.
4. Theo hướng **core systems** — KHÔNG cheat DB hay thay đổi config để đạt kết quả.
5. Plan phải **rõ ràng + có code demo** tới từng chi tiết.
6. Report phải **dựa trên kết quả tính toán thực tế** — review cẩn thận, note file thay đổi, KHÔNG report láo.
7. Kết thúc PHẢI kiểm tra service work mới báo done.
8. Phải có 1 file `report_*.md` ghi lại thay đổi.

## 2. Mental model user mong muốn

User mô tả 4 nhóm chức năng cốt lõi:

| # | Nhóm | Câu nói user |
|---|------|--------------|
| A | **Table CRUD** | "quản lý mấy cái table, thêm xóa sửa" |
| B | **Trigger dispatch** | "thêm xóa sửa xong gọi triger" |
| C | **Connection display** | "có thêm mấy cái connect để show" |
| D | **Health service** | "health service" |

→ Cấu trúc đích phải làm nổi 4 nhóm này. Dev mở project là thấy ngay 4 vùng tách bạch.

## 3. Bối cảnh kiến trúc hiện tại

### 3.1 Lịch sử

- **Task #19** (workspace `feature-cdc-system-refactor`) closed `2026-05-07` bởi agent x2: refactor `cdc-cms-service` sang hexagonal pattern → `app/{commands,queries,ports}` + `domain/` + `infra/{cache,http,messaging,persistence,observability}` + `api/` + `server/` + `middleware/` + `router/` + `model/`.
- Build sạch, 3 endpoint smoke test PASS, binary live tại `/tmp/cdc-cms-service-postJ` (PID 52079).
- → **Hexagonal có tồn tại nhưng user vẫn không hài lòng**.

### 3.2 Vì sao user khó chịu (gap so với 4 nhóm A/B/C/D)

| Pain | Vị trí | Bằng chứng |
|------|--------|------------|
| File quá nhiều trong 1 package | `internal/api/` 52 file flat | Không group theo feature — phải dò qua filename prefix |
| Logic CRUD bị tán mỏng | 1 use case = 6 file qua 4 layer (router → middleware → api → bus → command → domain) | `mapping_rule` CRUD đụng 3 DTO struct ở 3 file khác nhau |
| Tên thư mục chồng chéo | `internal/model/` (11 file GORM) + `internal/domain/` (7 file clean) | Cùng concept "source" tồn tại ở `model.TableRegistry` + `domain/source.Object` + `model/source.go` |
| Application service nhét vào persistence | `infra/persistence/provisioning_orchestrator.go` (729 dòng) | Comment line 1 ghi `// Package service —` nhưng `package persistence` |
| Domain anemic | 7 entity, chỉ 1 method `job.New()` | Logic transition nằm ở `infra/persistence/provisioning_state_machine.go` |
| Bypass abstraction | 17 file `api/` import `infra/persistence` trực tiếp | Hexagonal port bị skip |
| Trigger logic rò vào HTTP | `api/registry_handler_register.go:82` tự gọi `Dispatch(RestartDebezium)` | Business rule out of place |
| Raw NATS bypass bus | 7 file `api/` import `pkgs/natsconn` trực tiếp | Không qua CommandBus, mất idempotency |

→ Hexagonal **đúng về layer nhưng sai về granularity**. Layer dọc (api/app/domain/infra) làm flat namespace 52/32/27 file — không thấy được "domain mapping nằm đâu" trong 1 click.

## 4. Scope refactor

### IN-SCOPE

- Đề xuất **cấu trúc thư mục mới** (modular monolith / vertical slice).
- Phân bổ lại **17 file vi phạm import** vào đúng module.
- Tách `provisioning_orchestrator.go` (729 dòng) khỏi `persistence`.
- Code demo `internal/modules/mapping/` đầy đủ trong `03_implementation.md`.
- Phase rollout có **review gate** sau mỗi phase (user yêu cầu "task lớn cẩn review").
- File `report_*.md` tổng kết.

### OUT-OF-SCOPE (non-goal)

- KHÔNG sửa schema DB.
- KHÔNG sửa file YAML deployment / config-production.yml.
- KHÔNG thêm tính năng business mới.
- KHÔNG refactor logic — chỉ di chuyển + đổi tên + đóng gói.
- KHÔNG động vào `centralized-data-service` (đã refactor xong tuần trước).
- KHÔNG động vào `cdc-auth-service` hoặc `cdc-cms-web`.
- **KHÔNG chạy 1 lệnh `mv` / `git mv` / `Edit` / `Write` lên source `.go`** — Brain task §12.

## 5. Tham chiếu liên quan (đã đọc)

| File | Lý do đọc |
|------|-----------|
| `agent/GEMINI.md` | 14 rule + workflow list |
| `agent/memory/global/project_context.md` | profile cdc-cms-service hiện tại |
| `agent/memory/global/tech_stack.md` | Fiber v2 + CQRS pattern note |
| `agent/memory/global/active_plans.md` | Task #19 hexagonal đã closed |
| `agent/memory/global/lessons.md` | grep keyword "cdc-cms-service" — 30+ lesson liên quan (file 325KB > giới hạn read full, dùng grep) |

## 6. Định nghĩa "DONE" cho workspace này (Brain scope)

✅ Đủ 11 file doc (00, 01, 02, 03, 04, 05, 07, 08, 09, 10 + report).
✅ Tất cả pain point trong §3.2 có proposed location ở `02_plan.md`.
✅ Code demo Go đầy đủ cho 1 module trong `03_implementation.md`.
✅ Service `cdc-cms-service` vẫn chạy bình thường sau khi plan xong (Brain không sửa code).
✅ User approve `09_tasks_solution.md` trước khi giao Muscle thực thi.
