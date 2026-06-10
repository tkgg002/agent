# 13_proposed_structure — Đề xuất sắp xếp lại (Vertical Slice 8 BC)

> Kế thừa hướng v2 (`feature-cdc-cms-hexagonal-refactor-2026-06-01`). Nguyên tắc: **mỗi file phải thuộc 1 Bounded Context** → file nào không có BC + không có caller = **rơi vào DEAD bucket** (chính là cách "sắp xếp lại làm lộ file chưa dùng").

## Cấu trúc đích
```
cdc-cms-service/
├── cmd/server/                      # 1 entrypoint (bỏ cmd/sync_v2 — one-shot, archive)
├── internal/
│   ├── modules/                     # VERTICAL SLICE theo BC (api+command+query+domain+repo co-located)
│   │   ├── source/                  # discovery + registration + provisioning lifecycle
│   │   ├── mapping/                 # mapping rule + schema-drift approval
│   │   ├── master/                  # master binding (gộp master_swap nếu còn dùng — hiện DEAD)
│   │   ├── transform/               # transmute schedule + snapshot + backfill
│   │   ├── reconciliation/          # reconcile + self-healing + DLQ
│   │   ├── wizard/                  # source→master saga (cross-BC orchestration)
│   │   ├── system_control/          # connector (debezium/kafka) + worker schedule
│   │   └── observability/           # health + alerts + activity + audit/qa
│   ├── platform/                    # hạ tầng dùng chung (gom infra/ + pkgs/)
│   │   ├── persistence/  messaging/  cache/  http/  observability/
│   │   ├── config/  naming/  migrate/  middleware/
│   └── server/                      # composition root MỎNG (wire modules) + router
└── migrations/  docs/
```
Mỗi module: `handler_*.go` (từ api) · `cmd_*.go` (từ app/commands) · `qry_*.go` (từ app/queries) · `domain.go` (từ domain/) · `repo_*.go` (từ infra/persistence) · `*_test.go` **co-located** (kéo từ cây `test/` về → xoá hết `*ForTest`).

## Map current → target (theo nhóm)

| Hiện tại | Đích | Cách |
|---|---|---|
| `internal/api/*.go` (52) | `modules/<bc>/handler_*.go` | tách theo BC của route |
| `internal/app/commands/*.go` (32) | `modules/<bc>/cmd_*.go` | tách theo BC; đồng thời khử 18 raw `*gorm.DB` → repo port |
| `internal/app/queries/*.go` (30) | `modules/<bc>/qry_*.go` | tách theo BC; xoá `.Type()` hoặc wire query-bus |
| `internal/infra/persistence/*.go` (27) | `modules/<bc>/repo_*.go` + `platform/persistence` (base) | repo theo BC; base/conn để platform |
| `internal/domain/*` (8) | `modules/<bc>/domain.go` | nhập vào module; cân nhắc thêm method (hết anemic) |
| `internal/model/*` (14, V1 GORM) | `modules/<bc>/` nếu còn dùng; 4 file trùng V2 → **DEAD bucket** | xem `12` mục D |
| `internal/infra/{http,messaging,observability,cache}` | `platform/*` | `cache/` → **xoá** (dead) |
| `internal/{middleware,migrate,naming,config}` | `platform/*` | giữ nguyên chức năng |
| `internal/{router,server}` | `server/` | composition root mỏng lại (server.go 343→≤80 LOC) |
| `pkgs/*` (utils,database,natsconn,rediscache,observability) | `platform/*` | `utils/hash.go`+`type_inference.go` → **DEAD bucket** |
| `cmd/sync_v2` | **archive/** ngoài build | one-shot xong |

## 🗑️ DEAD bucket (lộ ra khi sắp xếp — KHÔNG có BC nào nhận)
Toàn bộ nhóm A (8 file/575 LOC) + B (sync_v2, registry_handler_read 2 endpoint) trong `12_unused_files.md`:
`infra/cache/doc.go` · `persistence/master_swap.go` · `bootstrap/registry_mirror.go` · `model/cdc_event.go` · `app/ports/{query_bus,publisher}.go` · `pkgs/utils/{hash,type_inference}.go` · `cmd/sync_v2/`.

## Vì sao reorg này "làm lộ file chưa dùng"
1. **Ép mỗi file vào 1 BC** → file không gọi/không được gọi bởi BC nào = orphan hiện ra ngay (master_swap, registry_mirror, cdc_event...).
2. **Co-locate test** → mọi `*ForTest` mất lý do tồn tại → 1 lớp dead-shim biến mất.
3. **Khử God Interface `ports/`** → 2 interface chết (query_bus, publisher) lộ ra vì không ai implement.
4. **Hợp nhất model/↔domain/** → 4 file V1 GORM trùng lặp lộ là legacy.

## Thực thi (KHÔNG làm trong workspace này — chỉ đề xuất)
- Đây là plan Brain-style. Thực thi cần: user duyệt → Muscle làm theo phase (xem roadmap 4-phase ở v2) → mỗi phase verify `go build ./...` + 53 route + 26 NATS subject + go test.
- Xoá file DEAD: gom thành 1 PR riêng "remove dead code" (575 LOC), chạy `go build ./...` + `deadcode` lại để xác nhận 0 regression.
