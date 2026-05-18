# 02_plan_phase3 — Fix "vớ vẩn" layout: rename `db:` → `dbPool:`

> **Trigger**: User: "mày vừa fix config-local.yml và tự nhiên chuyển `db: {maxOpenConn..}` + `systemDb: {url}` thành thữ vơ vẩn này".
>
> **Root cause**: Round 2 đã xoá legacy DSN fields trong `db:` block → còn lại `db:` chỉ chứa pool tuning. Tên block không phản ánh ngữ nghĩa (pool tuning áp cho ALL planes, không phải "the primary DB").

## Phân tích bằng grep (verified)

`cfg.DB.{MaxOpenConn, MaxIdleConn, ConnMaxLifetime}` được dùng tại:
- `pkgs/database/postgres.go:34-39, 63-68, 105-118` (3 entry points)
- `pkgs/database/multi.go:264-269, 280-281` (multi-DB registry, áp cho mọi plane)
- `pkgs/database/pgx_pool.go:22-23, 39` (pgx pool)

Kết luận: **3 field này là pool tuning toàn cục**, KHÔNG phải pool riêng của "db". Tên YAML phải phản ánh điều đó.

## Đề xuất — 2 option

### Option A (RECOMMENDED) — Rename YAML key + 1 struct tag

**YAML diff**:
```yaml
# TRƯỚC (vớ vẩn)
db:
  maxOpenConn: 50
  maxIdleConn: 25
  connMaxLifetime: 5m

# SAU
dbPool:
  maxOpenConn: 50
  maxIdleConn: 25
  connMaxLifetime: 5m
```

**Go diff** (`config/config.go:21`):
```go
// TRƯỚC
DB       DBConfig       `mapstructure:"db"`

// SAU
DB       DBConfig       `mapstructure:"dbPool"`
```

- Field name Go `cfg.DB` giữ nguyên → 50+ callers KHÔNG cần đổi.
- Chỉ tag mapstructure đổi → Viper sẽ đọc từ YAML key `dbPool:` thay vì `db:`.
- Backward-compat env override `DB_SINK_URL` vẫn chạy (vì env override set `cfg.DB.URL` trong Go, không qua YAML).

**Tradeoff**:
- Pro: minimal change (1 dòng Go + rename YAML key + cập nhật comment). Ngữ nghĩa rõ.
- Con: nếu có YAML production cũ dùng `db:` block → sẽ silently bị drop sau khi đổi tag. PHẢI cập nhật `config-production.yml` + `config-sample.yml` cùng lúc.

### Option B — Restructure to single `postgres:` parent

```yaml
postgres:
  pool:
    maxOpenConn: 50
    maxIdleConn: 25
    connMaxLifetime: 5m
  system:
    url: postgres://.../cdc_dw
  shadow:
    defaultKey: default
    urls: { default: postgres://.../cdc_shadow }
  master:
    defaultKey: default
    urls: { default: postgres://.../goopay_dest }
```

- Pro: tất cả Postgres-related gộp 1 block. Elegant nhất.
- Con: refactor lớn — đổi struct `AppConfig` (4 field cũ → 1 field `Postgres PostgresConfig`), đổi mapstructure path, đổi 50+ caller (`cfg.SystemDB.URL` → `cfg.Postgres.System.URL`, v.v.). Risk cao. Phá callers ngoài file config/.

### Option C — Inline pool vào từng plane (3x dup)

```yaml
systemDb:
  url: postgres://.../cdc_dw
  pool: { maxOpenConn: 50, ... }
shadowDb:
  urls: { default: ... }
  pool: { maxOpenConn: 50, ... }
masterDb:
  urls: { default: ... }
  pool: { maxOpenConn: 50, ... }
```

- Pro: pool tuning per-plane (future-proof khi cần tuning khác nhau).
- Con: 3x dup. Code refactor: mỗi caller `cfg.DB.MaxOpenConn` phải đổi thành `cfg.SystemDB.Pool.MaxOpenConn` (8+ caller) + logic chọn pool nào tương ứng plane nào. Hiện tại CHƯA có nhu cầu tuning per-plane.

## Recommend: **Option A**

- Lý do: 1 dòng Go thay đổi, không phá caller, ngữ nghĩa rõ ràng ngay lập tức.
- Phải đồng bộ 3 file YAML: `config-local.yml`, `config-sample.yml`, `config-production.yml`.

## Verification plan

1. `go build ./...` → EXIT=0.
2. `go test ./config/...` → PASS.
3. Smoke load `config.NewConfig()` với `config-local.yml` → confirm `cfg.DB.MaxOpenConn=50, MaxIdleConn=25, ConnMaxLifetime=5m`.
4. Grep `"db:"` trong YAML cũ — chỉ còn 0 match top-level (key sub-context như `db: 0` trong redis.db vẫn OK vì khác level).

## Rollback

- Đảo lại 1 dòng tag + rename 3 file YAML key về `db:`.

## Câu hỏi cho user

Có chấp nhận Option A (rename `db:` → `dbPool:` + 1 dòng Go) không? Nếu OK tôi sẽ:
1. Tạo `08_tasks_phase3_rename_dbpool.md` + `09_tasks_solution_phase3_rename_dbpool.md`.
2. Muscle thực thi: sửa `config.go` (1 line) + 3 file YAML (`config-local.yml`, `config-sample.yml`, `config-production.yml`).
3. Build + test + smoke verify.
4. Append log vào `05_progress.md`.

Nếu user muốn Option B/C → re-plan với scope lớn hơn.
