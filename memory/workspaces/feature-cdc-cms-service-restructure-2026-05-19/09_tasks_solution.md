# 09_tasks_solution.md — Đề xuất giải pháp

## Tổng quát

4 hướng đi đã được cân nhắc. Mỗi hướng kèm trade-off, effort estimate, và đánh giá bám với 4 yêu cầu mental model A/B/C/D của user.

| Hướng | Tên | Effort | Risk | Match user mental model | Khuyến nghị |
|-------|-----|--------|------|------------------------|------------|
| 1 | **Status quo + cleanup** | 1 ngày | Thấp | ❌ Không cải thiện cấu trúc | ❌ |
| 2 | **Sub-folder hexagonal** | 3-5 ngày | Vừa | 🟡 Cải thiện một phần | 🟡 |
| 3 | **Vertical Slice / Modular Monolith** | 7-10 ngày | Vừa-Cao | ✅ Khớp 100% với A/B/C/D | ⭐ **KHUYẾN NGHỊ** |
| 4 | **Multi-service tách physical** | 3-4 tuần | Rất cao | ❌ Overkill | ❌ |

---

## Hướng 1 — Status quo + cleanup

### Mô tả
Giữ nguyên hexagonal layer dọc, chỉ:
- Tách 3 file > 500 LOC thành nhiều file.
- Sửa 17 file `api/ → infra/persistence` bằng port interface.
- Đổi tên `model/` → `persistence/gormmodel/` để giảm chồng chéo.

### Ưu
- Effort thấp (1 ngày).
- Risk thấp — đổi tên + thêm interface, không di chuyển package.
- Không cần đào tạo lại team.

### Nhược
- KHÔNG giải quyết "ko trực quan" — vẫn flat 52 file trong `api/`.
- Mental model user (A/B/C/D) vẫn không nhìn thấy được trong cây thư mục.
- User đã phàn nàn → giữ nguyên = không respond gì cả.

### Verdict
❌ **Không khuyến nghị** — không đáp ứng FR-2.

---

## Hướng 2 — Sub-folder trong layer hexagonal

### Mô tả
Giữ hexagonal nhưng group con theo bounded context BÊN TRONG mỗi layer:

```
internal/
├── api/
│   ├── mapping/         (mapping_rule_handler*.go → 5 file)
│   ├── registry/        (registry_handler*.go → 9 file)
│   ├── provisioning/    (provisioning_handler.go)
│   ├── alerts/          (alerts_handler.go)
│   ├── reconciliation/  (reconciliation_handler*.go)
│   ├── connectors/      (system_connectors_handler.go)
│   ├── health/          (health_handler.go, system_health_handler.go)
│   └── system/          (introspection_handler.go, …)
├── app/
│   ├── commands/
│   │   ├── mapping/
│   │   ├── registry/
│   │   └── …
│   └── queries/
│       ├── mapping/
│       └── …
├── domain/        (giữ nguyên — đã group rồi)
└── infra/
    └── persistence/
        ├── mapping/
        ├── registry/
        └── …
```

### Ưu
- Vẫn theo Clean/Hexagonal — dev có background DDD quen thuộc.
- Migration mượt: chỉ `git mv` theo group, không đổi semantic.
- Domain anemic vẫn được sửa song song.

### Nhược
- Dev mở 1 feature mapping vẫn phải nhảy **4 thư mục** (api/mapping + app/commands/mapping + app/queries/mapping + infra/persistence/mapping). User mental model "1 folder = 1 nhóm" KHÔNG đạt.
- Bytes giảm flat-namespace nhưng folder count tăng (32 sub-folder).

### Verdict
🟡 **Cải thiện một phần** — FR-2 chưa đạt. Tốt nếu muốn dùng làm bước đệm trước Hướng 3.

---

## Hướng 3 — Vertical Slice / Modular Monolith ⭐

### Mô tả

Chuyển từ **layer-first** sang **feature-first**. Mỗi business capability = 1 module độc lập, mọi thứ liên quan đến nó (HTTP + command + query + repo + domain + DTO) ở chung 1 folder.

```
internal/
├── modules/                       ← bounded context layer
│   ├── mapping/                   ← A. Table mapping CRUD
│   │   ├── handler.go             HTTP routes + handlers
│   │   ├── routes.go              Mount subgroup vào Fiber app
│   │   ├── commands.go            Create/Update/Delete logic
│   │   ├── queries.go             List/Get logic
│   │   ├── repo.go                GORM persistence
│   │   ├── domain.go              Rule entity + behaviors (Validate, CanApply…)
│   │   ├── dto.go                 Request/response payloads
│   │   └── *_test.go
│   ├── registry/                  ← A+B. Table registry + trigger
│   ├── provisioning/              ← B. Provisioning orchestrator (target home for 729-LOC)
│   ├── alerts/                    ← (target home for 446-LOC alert_manager)
│   ├── reconciliation/            ← Reconciliation actions
│   ├── connectors/                ← C. Kafka Connect connectors (display)
│   ├── sources/                   ← C. Source connections registry
│   ├── health/                    ← D. /health, /ready, /system/health
│   └── jobs/                      ← Async job tracking (sync_v2, etc.)
├── platform/                      ← cross-cutting infrastructure
│   ├── bus/                       NATS command bus + publisher (từ infra/messaging)
│   ├── db/                        GORM init, connection pool, migrate guard
│   ├── http/                      Kafka Connect HTTP client (từ infra/http)
│   ├── observability/             Logger, OTEL, probes
│   ├── middleware/                JWT, RBAC, audit
│   └── eventbus/                  In-process pub/sub (nếu cần cross-module)
├── server/                        ← composition root + main wiring
│   ├── wire.go                    DI container — assemble modules + platform
│   └── app.go                     Fiber app bootstrap
├── router/                        ← root router (≤ 50 LOC, chỉ delegate)
│   └── router.go
└── pkgs/                          (giữ — utilities công khai)
```

**Quy ước**:
1. Module **KHÔNG** import module khác. Cross-module communication qua `platform/eventbus/`.
2. Module **CHỈ** import từ `platform/`, `pkgs/`, hoặc thư viện ngoài.
3. Mỗi module export 1 hàm constructor `New(deps) *Module` — chấp nhận deps qua interface.
4. Domain entity sống bên trong module → có behavior (validate, transition). KHÔNG còn anemic.

### Mapping cấu trúc cũ → mới

| Old | New |
|-----|-----|
| `api/mapping_rule_handler*.go` (5 file) | `modules/mapping/handler.go` |
| `app/commands/{create,update,delete}_mapping_rule.go` | `modules/mapping/commands.go` |
| `app/queries/list_mapping_rules.go` | `modules/mapping/queries.go` |
| `domain/mapping/{rule,errors}.go` | `modules/mapping/domain.go` |
| `infra/persistence/mapping_rule_*_repo.go` | `modules/mapping/repo.go` |
| `api/dto/mapping_rule_dto.go` | `modules/mapping/dto.go` |
| `api/registry_handler*.go` (9 file) | `modules/registry/handler.go` |
| `infra/persistence/provisioning_orchestrator.go` (**729 LOC**) | `modules/provisioning/orchestrator.go` |
| `infra/persistence/provisioning_state_machine.go` | `modules/provisioning/state_machine.go` |
| `infra/persistence/approval_service.go` | `modules/provisioning/approval.go` |
| `infra/persistence/alert_manager.go` (**446 LOC**) | `modules/alerts/manager.go` |
| `api/alerts_handler.go` | `modules/alerts/handler.go` |
| `api/system_connectors_handler.go` (367 LOC) | `modules/connectors/handler.go` |
| `infra/http/kafka_connect.go` | `modules/connectors/client.go` HOẶC `platform/http/kafka_connect.go` (nếu nhiều module dùng) |
| `api/health_handler.go` + `api/system_health_handler.go` | `modules/health/handler.go` |
| `api/sources_handler.go` + `model/source.go` | `modules/sources/{handler,repo,domain}.go` |
| `infra/messaging/*` | `platform/bus/` |
| `infra/observability/*` + `probes/` | `platform/observability/` |
| `middleware/*` | `platform/middleware/` |
| `model/*` (GORM) | `modules/<X>/repo.go` (di chuyển GORM struct vào repo của từng module) HOẶC giữ `platform/db/gormmodel/` nếu shared |
| `bootstrap/*` | `server/wire.go` |
| `app/ports/{repository,command_bus,publisher,query_bus}.go` | Inline trong từng module (port nội bộ), hoặc xóa nếu chỉ dùng cho 1 module |
| `router/router.go` (408 LOC) | `router/router.go` ≤ 50 LOC + `modules/<X>/routes.go` |

### Ưu
- ✅ **Khớp 100% mental model user**: mở 1 folder = thấy 1 nhóm A/B/C/D đầy đủ.
- ✅ Số file mỗi folder ≤ ~10 → trực quan.
- ✅ Domain entity nằm cạnh business logic → tự nhiên gắn behavior (hết anemic).
- ✅ Cross-module isolation enforce qua import check.
- ✅ Dev thêm tính năng mới = thêm 1 module = template hóa.
- ✅ Long-term: dễ tách microservice nếu cần (mỗi module ≈ 1 candidate service).

### Nhược
- Effort cao hơn (7-10 ngày Muscle).
- Risk cross-module dependency cycle → cần kỷ luật `eventbus/` từ ngày 1.
- Wire DI ở `server/wire.go` phức tạp ban đầu — cần phase 0 setup riêng.
- Khi 2 module thật sự cần chia sẻ logic chung (vd: `naming/`) phải tách ra `pkgs/`.

### Verdict
⭐ **KHUYẾN NGHỊ** — đáp ứng đầy đủ FR-1..FR-10 và mental model user A/B/C/D.

---

## Hướng 4 — Multi-service tách physical

### Mô tả
Tách `cdc-cms-service` thành 4-5 service riêng (`mapping-svc`, `registry-svc`, `provisioning-svc`, …) — mỗi service deploy K8s riêng.

### Ưu
- Mỗi service nhỏ, đội ngũ có thể scale độc lập.
- Boundary cứng → không thể bypass.

### Nhược
- Effort khổng lồ (3-4 tuần).
- Tăng độ phức tạp infra (5 deployment, 5 secret, 5 image).
- DB cần tách hoặc giữ chung → cả 2 đều có nhược điểm lớn.
- Service mesh / discovery cần đầu tư.
- User đang complain "ko trực quan" → tách service KHÔNG giải quyết, chỉ chuyển vấn đề.

### Verdict
❌ **Không khuyến nghị** — overkill, không match user complaint.

---

## Khuyến nghị cuối: **Hướng 3 (Vertical Slice)** ⭐

### Lý do

1. **Khớp mental model user** — 4 nhóm A/B/C/D thành 4 (+ phụ) folder tách bạch.
2. **Đụng tối thiểu logic** — phần lớn là `git mv` + sửa import path. Không refactor business.
3. **Có pattern reference** — modular monolith là chuẩn Go community (DDD-lite, ports nội bộ).
4. **Phase nhỏ** — mỗi phase = di chuyển 1 module, dễ revert, dễ review.
5. **Tương thích lessons** — đáp ứng `lessons.md:2181` (CQRS Phase 3 scope discipline) và `lessons.md:2755` (refactor có 7 ràng buộc rõ).

### Phase rollout đề xuất (chi tiết trong `02_plan.md`)

| Phase | Mục tiêu | File chạm | Effort | Risk |
|-------|----------|-----------|--------|------|
| **P0** | Setup skeleton `modules/`, `platform/`, `server/wire.go`. KHÔNG di chuyển logic. | ~5 file mới | 0.5 ngày | Thấp |
| **P1** | Move `infra/messaging/`, `infra/observability/`, `middleware/` → `platform/` | 23 file | 0.5 ngày | Thấp |
| **P2** | Module `health/` (đơn giản nhất, kiểm test pattern) | 3 file | 0.5 ngày | Thấp |
| **P3** | Module `mapping/` (table CRUD — module mẫu pattern) | ~10 file | 1 ngày | Vừa |
| **P4** | Module `registry/` | ~12 file | 1 ngày | Vừa |
| **P5** | Module `connectors/` + `sources/` | ~8 file | 1 ngày | Vừa |
| **P6** | Module `alerts/` (tách 446 LOC alert_manager) | ~5 file | 1 ngày | Cao |
| **P7** | Module `provisioning/` (tách 729 LOC orchestrator — phase nhạy nhất) | ~6 file | 1.5 ngày | Cao |
| **P8** | Module `reconciliation/` + `jobs/` | ~8 file | 1 ngày | Vừa |
| **P9** | Xóa `internal/{api,app,domain,model,infra,bootstrap}/`, thu router ≤ 50 LOC | cleanup | 0.5 ngày | Thấp |
| **P10** | Smoke test + production canary + report | - | 0.5 ngày | Vừa |

**Tổng**: ~9-10 ngày người làm tập trung, có review gate sau mỗi phase.

### Câu hỏi user cần chốt trước khi Muscle thực thi

1. **OK với Hướng 3 (Vertical Slice)?** — Hay user muốn xem xét Hướng 2 (sub-folder) làm bước đệm?
2. **`internal/model/` GORM struct**: di chuyển vào từng module (`modules/<X>/repo.go`) hay giữ `platform/db/gormmodel/` shared?
3. **`platform/eventbus/`**: có cần ngay từ phase 0 không, hay chỉ tạo khi gặp use case cross-module thực tế?
4. **Naming**: `internal/modules/` hay `internal/feature/` hay `internal/bc/` (bounded context)?
5. **Migration giữ ở đâu**: tiếp tục `migrations/` root hay di chuyển vào `platform/db/migrations/`?
6. **Review gate**: pause sau mỗi phase chờ user approve, hay chạy liên tục với checkpoint ở 3 mốc (sau P1, P5, P10)?

→ Khi user trả lời 6 câu này → Muscle có đủ context thực thi.
