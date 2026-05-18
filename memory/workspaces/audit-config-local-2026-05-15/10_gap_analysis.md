# 10_gap_analysis — Phase 2 GAP: `config.go` (Go side) chưa được clean

> **Date**: 2026-05-15
> **Trigger**: User: "`config.go` mày đang làm rất vớ vẩn mấy file này".
> **Root cause**: Round 1+2 chỉ sửa YAML. Struct/method/env-override mirror các key đã xoá vẫn còn nguyên trong code → mismatch giữa YAML hiện tại (114 dòng, clean) và `config.go` (622 dòng, full legacy).

## Bằng chứng (grep hard evidence)

| # | Item | Vị trí | Caller (ngoài định nghĩa) | Verdict |
|---|------|--------|----------------------------|---------|
| 1 | `WorkerConfig.FetchSize` | `config.go:189` | **0** | DEAD |
| 2 | `WorkerConfig.TransformInterval` | `config.go:190` | **0** | DEAD |
| 3 | `WorkerConfig.ScanInterval` | `config.go:191` | **0** | DEAD |
| 4 | `JWTConfig.Expiration` | `config.go:198` | **0** (grep `\.Expiration\b` → 0 match) | DEAD |
| 5 | `DebeziumConfig.ConnectorName` | `config.go:70` | Chỉ assign tại `config.go:379` (env). Handler `command_handler.go:2018-2022` hardcode `"goopay-mongodb-cdc"`, KHÔNG đọc cfg field. Comment line 2019 thừa nhận "Allow an environment-injected default later if needed". | DEAD |
| 6 | Env override `DEBEZIUM_CONNECTOR_NAME` | `config.go:378-380` | Set field DEAD (#5) | DEAD |
| 7 | Env override `SOURCE_DSN_POSTGRES_PRIMARY` | `config.go:400-405` | Set `Sources["postgres_primary"]`. `SourceURL()` 0 caller (#8) → field dead downstream. | DEAD |
| 8 | Method `(cfg *AppConfig) SourceURL(name)` | `config.go:591-597` | **0** (grep `SourceURL\(` → chỉ định nghĩa) | DEAD |
| 9 | `DBConfig.Host/Port/UserName/Password/Database/SSLMode` | `config.go:133-138` | `cfg.DB.Host` + `cfg.DB.Database` đọc tại `validateConfig:430` (hasLegacy check). `PgxDSN()` đọc tất cả để compose DSN. Sau Round 2 YAML KHÔNG còn ship các field này → empty → `PgxDSN()` trả `postgres://:@:0/?sslmode=` (GARBAGE). | DEAD-AT-VALUE, ACTIVE-AT-CODE |
| 10 | `DBConfig.URL` | `config.go:139` | Đọc tại `validateConfig:429` + env `DB_SINK_URL:300`. Nếu xoá legacy path thì cũng xoá. | DEAD-AT-VALUE |
| 11 | `DBConfig.DSN()` method (key=value form) | `config.go:150-158` | 1 caller: `pkgs/database/postgres.go:15` (`OpenDB(cfg)`). Cần check ai gọi `OpenDB`. | ⚠️ Cần verify |
| 12 | `applyDBFallbacks` line 462-473 (ShadowDB/MasterDB ← legacy) | `config.go:462-473` | Sau Round 2, `legacy = "postgres://:@:0/?sslmode="`. Nếu ShadowDB.URLs empty → fallback ghi GARBAGE vào `ShadowDB.URLs["default"]`. MasterDB cũng vậy nhưng validator chặn empty trước, nên không bao giờ chạy. | DANGEROUS-FALLBACK (DEAD-IF-NOT-EMPTY-OR-DANGEROUS-IF-EMPTY) |

## Phân loại quyết định

### A. DEAD chắc chắn — xoá an toàn (0 caller):
- #1-#4: `Worker.FetchSize`, `Worker.TransformInterval`, `Worker.ScanInterval`, `JWT.Expiration` → xoá field khỏi struct.
- #5-#6: `Debezium.ConnectorName` field + env override → xoá cả hai. Sửa comment `command_handler.go:2018-2022` để loại bỏ TODO "env-injected default later".
- #7-#8: `SourceURL()` method + env `SOURCE_DSN_POSTGRES_PRIMARY` → xoá.

### B. Legacy DSN path — quyết định: GIỮ hay XOÁ?
- #9-#12: Toàn bộ `DBConfig.{Host,Port,UserName,Password,Database,SSLMode,URL}` + `DBConfig.DSN()` + `PgxDSN()` + nửa dưới `applyDBFallbacks` + env `DB_SINK_URL` + validator `hasLegacy` branch.

**Option B1 — XOÁ legacy hoàn toàn**:
- Pro: Single-source-of-truth (`systemDb.url` / `shadowDb.urls` / `masterDb.urls`). Không còn fallback garbage. Code gọn.
- Con: Mất khả năng deploy cũ bằng `DB_SINK_URL=postgres://...` 1-env-var setup. Phá `pkgs/database/postgres.go:OpenDB` nếu còn caller. Phá `pkgs/database/pgx_pool.go` nếu nó vẫn dùng `PgxDSN()`.

**Option B2 — GIỮ legacy (status quo)**:
- Pro: Backward-compat. Production có thể đang dùng `DB_SINK_URL`.
- Con: 50% struct chỉ tồn tại để feed fallback chain mà YAML mới không dùng. ShadowDB fallback có khả năng ghi garbage.

**Option B3 — GIỮ field, BỎ fallback nguy hiểm** (recommended):
- Giữ `DBConfig.{Host..URL}` + `PgxDSN()` cho backward-compat path qua env.
- Xoá `applyDBFallbacks` line 462-473 (ShadowDB/MasterDB từ legacy) — nếu user không config thì FAIL loud thay vì ghi garbage.
- Update validator: shadowDb cũng bắt buộc nếu mode worker dùng shadow path (cần thêm check).

### C. Tài liệu/comment dài dòng
- Comment `DebeziumConfig` (config.go:54-63) — dài 10 dòng kể lịch sử Phase v3 §7 Heal via Signal. Có thể rút ngắn thành 2-3 dòng.
- Comment `Sources` (config.go:25-28) — OK.
- Comment `ControlPlane` (config.go:30-38) — sau Round 2 YAML đã giải thích plane → comment trong code có thể rút.

## Đề xuất plan cho Phase 2

### Scope tối thiểu (chỉ xoá DEAD chắc chắn — Option A):
- Xoá 4 field struct (#1-#4).
- Xoá field `Debezium.ConnectorName` (#5) + env override (#6).
- Xoá env override `SOURCE_DSN_POSTGRES_PRIMARY` (#7) + method `SourceURL()` (#8).
- Sửa comment `command_handler.go:2019` (xoá dòng "Allow an environment-injected default later").
- **NET delete**: ~25 dòng `config.go` + 1 dòng comment trong `command_handler.go`.
- **Build/test verify**: `go build ./...` + `go test ./config/...`.

### Scope mở rộng (thêm Option B3):
- Toàn bộ Option A.
- Xoá `applyDBFallbacks:462-473` (legacy → Shadow/Master fallback).
- Thêm validator check: nếu `cfg.ShadowDB.URLs` empty → error loud (thay vì silent garbage).
- **NET delete**: ~40 dòng `config.go`. Behavior change: fail-fast nếu YAML thiếu shadowDb.

### KHÔNG làm trong phase này:
- Option B1 (xoá hoàn toàn legacy DSN path) — risky, cần biết production có dùng `DB_SINK_URL` không.
- Refactor `detectConnectorName` để đọc cfg (out-of-scope cleanup).
- Touch `command_handler.go` ngoài 1 dòng comment.

## Câu hỏi cho User (chỉ 1 — chọn scope)

1. **Scope A (an toàn, chỉ DEAD)** — xoá 4 field + 2 env + 1 method, không động fallback. Build pass chắc chắn.
2. **Scope A+B3 (recommended)** — A cộng thêm bỏ fallback garbage cho Shadow/Master, thêm validator fail-loud.

Mặc định tôi sẽ làm **Scope A** trừ khi user nói khác. Sau khi user duyệt sẽ tạo `08_tasks_phase2_go_cleanup.md` + `09_tasks_solution_phase2_go_cleanup.md` rồi Muscle thực thi.
