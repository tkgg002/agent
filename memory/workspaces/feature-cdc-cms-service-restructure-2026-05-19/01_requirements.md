# 01_requirements.md — Yêu cầu refactor

## 1. Functional (FR)

| ID | Yêu cầu | Nguồn |
|----|---------|-------|
| FR-1 | Cấu trúc đích phải làm nổi rõ 4 nhóm A/B/C/D của mental model user (table CRUD / trigger / connection / health) | `00_context.md §2` |
| FR-2 | Mỗi nhóm chức năng = 1 folder, dev mở folder thấy đủ handler + command + query + repo + domain entity của nhóm đó | User: *"ko trực quan để dev debug"* |
| FR-3 | Loại bỏ hoặc thống nhất chồng chéo `model/` ↔ `domain/` — chỉ giữ 1 thuật ngữ duy nhất | `10_gap_analysis §B.5` |
| FR-4 | Tách `provisioning_orchestrator.go` (729 LOC) khỏi `infra/persistence/` ra application service tại module riêng | `10_gap_analysis §B.3 #12` |
| FR-5 | Tách `alert_manager.go` (446 LOC) tương tự | `10_gap_analysis §B.3 #15` |
| FR-6 | Loại bỏ `api/` import `infra/persistence` trực tiếp (hiện 17 file) | `10_gap_analysis §B.6` |
| FR-7 | Loại bỏ `app/commands/` import `infra/persistence` (hiện 4 file) | `10_gap_analysis §B.6` |
| FR-8 | Loại bỏ `api/` import `pkgs/natsconn` trực tiếp (hiện 7 file) — bus duy nhất | `10_gap_analysis §B.6` |
| FR-9 | Business rule "register → restart debezium" và "transform" KHÔNG nằm trong HTTP handler | `10_gap_analysis §B.7` |
| FR-10 | Router file (`router.go` 408 LOC) phải tách theo module — mỗi module tự khai báo route subgroup | `10_gap_analysis §B.1 #5` |

## 2. Non-functional (NFR)

| ID | Yêu cầu | Đo lường |
|----|---------|----------|
| NFR-1 | **Zero downtime** — service đang chạy production phải tiếp tục hoạt động qua mỗi phase | `kubectl rollout status` không lỗi sau mỗi phase deploy |
| NFR-2 | **Backward-compat API** — tất cả route HTTP hiện có giữ nguyên path + method + payload | Smoke test 100% endpoint pass sau mỗi phase |
| NFR-3 | **Build sạch** — `go vet ./...` + `go build ./...` + `go test ./...` không cảnh báo sau mỗi phase | CI gate |
| NFR-4 | **Không cheat DB / config** — không sửa schema / migration / config file để tránh fix code | User constraint |
| NFR-5 | **Refactor pure structural** — không thêm tính năng business mới | Diff sau refactor không chứa logic mới |
| NFR-6 | **Audit trail** — mỗi phase commit + git mv để git lịch sử rename | `git log --follow` hoạt động trên file đổi chỗ |
| NFR-7 | **Phase nhỏ** — mỗi phase ≤ 5 file đổi tên/move, dễ revert | User: *"task lớn cẩn review"* |
| NFR-8 | **Review gate** — sau mỗi phase, user review trước khi sang phase tiếp | User: *"task lớn cẩn review"* |
| NFR-9 | **Số LOC chạm tối thiểu** — chỉ thay đường import, không refactor logic | Diff = move + import path |
| NFR-10 | **Test coverage giữ nguyên** — không xóa test hiện có, chỉ di chuyển | `go test ./...` pass count = trước |

## 3. Acceptance Criteria (AC) — Tổng thể

| ID | Tiêu chí | Verify |
|----|----------|--------|
| AC-1 | Folder `internal/modules/` có ≤ 10 thư mục con, mỗi thư mục là 1 bounded context | `ls internal/modules \| wc -l` |
| AC-2 | Mỗi module tự chứa: `handler.go` (HTTP), `commands.go` hoặc folder commands/, `queries.go` hoặc folder queries/, `repo.go` (gorm), `domain.go` (entity + behavior), `routes.go` (subgroup mount) | grep convention |
| AC-3 | Không file nào trong `internal/modules/<X>/` import `internal/modules/<Y>/` (cross-module isolation) | `grep -rE 'cdc-cms-service/internal/modules/(?!<self>)'` mỗi module = 0 |
| AC-4 | `internal/platform/` chỉ chứa cross-cutting: bus (NATS), db init, observability, middleware, http client | Convention enforced trong PR review |
| AC-5 | `internal/api/` BỊ XÓA. Không còn handler ở đó | `ls internal/api 2>&1 \| grep "No such"` |
| AC-6 | `internal/model/` BỊ XÓA, hoặc đổi tên `internal/persistence/gormmodel/` để rõ vai trò | `ls internal/model 2>&1` |
| AC-7 | `internal/app/{commands,queries,ports}/` BỊ XÓA — di chuyển vào module | `ls internal/app 2>&1` |
| AC-8 | `internal/router/router.go` còn ≤ 50 LOC — chỉ delegate sang module route | `wc -l internal/router/router.go` |
| AC-9 | Smoke test endpoint chính 100% pass: `GET /health`, `GET /api/v1/sources`, `GET /api/system/health`, `GET /api/v1/mapping-rules`, `POST /api/v1/mapping-rules` (mock data), `GET /api/system/connectors` | Bash curl script trong `06_test.md` |
| AC-10 | Service binary chạy được `./cms-server` không panic, log "ready" trong 5s | Manual run check |

## 4. Definition of Done (DoD)

Một module được coi là **DONE** khi tất cả các check sau pass:

- [ ] Folder `internal/modules/<name>/` tồn tại, có đủ 6 file canonical (xem AC-2).
- [ ] Toàn bộ file gốc liên quan đã `git mv` hoặc tạo mới — file gốc đã xóa.
- [ ] Import bên ngoài đều chỉ tới `internal/modules/<name>/` qua interface công khai (constructor) — không có import nội bộ struct riêng tư.
- [ ] `go vet`, `go build`, `go test ./internal/modules/<name>/...` PASS.
- [ ] Smoke test endpoint của module PASS bằng curl.
- [ ] `git log --follow` còn theo dõi được lịch sử file gốc.
- [ ] User review + approve trong commit message hoặc workspace status.

## 5. Risk register

| ID | Risk | Mức | Mitigation |
|----|------|-----|-----------|
| R-1 | Vòng import giữa modules (vd: mapping cần gọi registry) | Cao | Đặt `internal/platform/eventbus/` làm trung gian; module chỉ publish/subscribe — không gọi trực tiếp |
| R-2 | Quên di chuyển 1 helper → build fail nửa chừng | Vừa | Mỗi phase = 1 module, build + test sau mỗi phase |
| R-3 | Test fixture path hardcode → fail sau move | Vừa | Grep `testdata/` + `_test.go` trước mỗi phase |
| R-4 | Wire DI ở `bootstrap/` đụng nhiều file | Cao | Wire-up tập trung `internal/server/wire.go` — tách rõ DI từ phase 0 |
| R-5 | `git mv` không track rename nếu nội dung file đổi quá 50% | Thấp | Đầu phase chỉ `git mv` rỗng → commit → sang phase nhỏ sửa import |
| R-6 | User phát hiện regression production sau deploy | Cao | NFR-1 + NFR-2 + AC-9 — smoke test trước rollout. Có feature flag rollback. |
| R-7 | Refactor dài → conflict với feature branch khác | Vừa | Phase nhỏ (NFR-7) + rebase liên tục |
| R-8 | `provisioning_orchestrator.go` 729 LOC chứa state ẩn → di chuyển làm vỡ | Cao | Phase riêng dành cho provisioning, có code review kỹ |

## 6. Stakeholder

| Vai trò | Người | Trách nhiệm |
|---------|-------|------------|
| User (Product Owner) | Train Nguyen | Approve plan + review gate sau mỗi phase |
| Brain | Antigravity | Plan, document, gap analysis, no code |
| Muscle | Claude Code CLI | Thực thi phase 0 → N theo plan, có verify gate |
| Reviewer | x2 (ưu tiên, lock 2026-05-07) | Cross-review nếu rảnh |

## 7. Non-requirements (sẽ KHÔNG làm)

- ❌ Không đổi schema DB / không thêm migration.
- ❌ Không sửa `config-production.yml` hay K8s manifest.
- ❌ Không refactor logic NATS subjects.
- ❌ Không upgrade thư viện (Fiber, GORM, NATS client).
- ❌ Không tạo unit test mới (giữ nguyên test có sẵn).
- ❌ Không động `centralized-data-service`, `cdc-auth-service`, `cdc-cms-web`.
