# 04 — Architecture Decisions

> Mỗi quyết định nằm ở 1 ADR (Architecture Decision Record). Status mặc định
> **Proposed** — chuyển sang **Accepted** sau khi user approve plan.

---

## ADR-001 — Tách layout theo trục `schema/` vs `seed/`

**Status**: Proposed

**Context**:
Hiện tại `migrations/` chứa 9 sub-folder chức năng (core, ids, registry, ...).
3 file (worker/007, registry/020, cdc_system_model/029) trộn DDL với INSERT
seed business data. Production cold-boot bị ép apply 8 row seed dù không cần.

**Options considered**:
- **A. Renumber + flat folder**: rename tất cả về `001_*.sql ... NNN_*.sql`
  ở top-level. → Phá tracker compat (28 row đã ghi).
- **B. Giữ nguyên 9 folder, thêm comment "is_seed=true"** in file header,
  runner đọc comment để skip. → Phức tạp, parser fragile.
- **C. Tách 2 top-level folder `schema/` + `seed/`**, file SQL nằm trong từng
  folder dựa trên loại data. → Đơn giản, file basename không đổi, tracker
  compat OK.

**Decision**: Chọn **C**.

**Rationale**:
- Tracker `cdc_system.schema_migrations` dùng `path.Base()` (không gồm folder)
  → MOVE file không ảnh hưởng tracker compat.
- Folder name self-documenting: dev nhìn `schema/` biết là DDL safe-on-prod,
  nhìn `seed/` biết là dev-only.
- Trùng với pattern industry (golang-migrate, atlas, Rails db/migrate +
  db/seeds).

**Consequences**:
- `embed.go` phải split thành 2 var (SchemaFiles + SeedFiles).
- Runner phải nhận flag `includeSeeds`.
- Numbering schema (001-052) tách với seed (100+) → tránh basename collision.

---

## ADR-002 — Preserve basename, KHÔNG renumber

**Status**: Proposed

**Context**:
Numbering có 24 lỗ thủng do squash (004-006, 009, 012, 014-017, ...). Nhìn
không "đẹp". Có thể renumber `001 002 003 004 ... NNN` liên tục.

**Decision**: KHÔNG renumber. Giữ basename hiện tại.

**Rationale**:
- Tracker dùng `path.Base()` làm PK → renumber = phải mutate tracker manually
  → vi phạm REQ-4 "không cheat DB".
- Lỗ thủng có thể track qua `archive/README.md` (giải thích từng số bị bỏ).
- Industry pattern (Rails, Django) cũng giữ timestamp/number lịch sử dù squash.

**Consequences**:
- Reader thấy numbering "thưa" — mitigation qua `schema/README.md` ghi rõ.

---

## ADR-003 — Giữ `enum_types` INSERT trong schema/registry/020

**Status**: Proposed

**Context**:
Migration 020 vừa ALTER cdc_mapping_rules vừa INSERT 3 enum_types
(payment_state, api_type, currency_iso). Có thể tách INSERT sang seed/.

**Decision**: GIỮ enum_types INSERT trong schema (file 020 không split).

**Rationale**:
- enum_types là **domain config** (bất biến giữa env): mapping rules
  reference các enum này, prod cần.
- Không phải "demo data" hay "dev fixture".
- Tách ra seed/ sẽ break production cold-boot (mapping rules invalid khi
  enum chưa được seed).

**Consequences**:
- Documentation `seed/README.md` cần explain rule phân loại:
  - **schema/**: DDL + domain config (enum, lookup).
  - **seed/**: env-specific fixture (default schedules, demo connections).

---

## ADR-004 — Bóc seed `worker/007` (schedule rows) sang `seed/100`

**Status**: Proposed

**Context**:
Migration 007 tạo `cdc_worker_schedule` + INSERT 5 default schedule rows
(bridge, transform, field-scan, partition-check, airbyte-sync). Production
operator có thể tự config schedule qua CMS UI sau cold-boot.

**Decision**: Tách INSERT sang `seed/100_worker_schedules.sql`.

**Rationale**:
- Schedule interval là runtime config (60s, 90s, ...) có thể khác giữa env.
- Production deploy không nên có default schedule cứng → operator phải explicit.
- Dev/local cần để smoke test pipeline (không phải gõ tay).

**Consequences**:
- Production deploy lần đầu KHÔNG có schedule row → CMS UI phải có form
  "Insert default schedules" hoặc operator dùng `seed/100` manual SQL.
- Dev: skipSeeds=false → có schedule sau migrate.

---

## ADR-005 — Bóc seed `cdc_system_model/029` (legacy connections) sang `seed/101`

**Status**: Proposed

**Context**:
Migration 029 INSERT 3 row `legacy_system_db`, `legacy_shadow_default`,
`legacy_master_default`. V2SyncCommand.resolveShadowConnectionID cần ≥1 row
role_type='shadow' active.

**Decision**: Tách 3 INSERT sang `seed/101_v2_legacy_connections.sql`.

**Rationale**:
- `bootstrap.EnsureDefaultShadowConnection` (server.go:94) đã làm cùng việc
  này ở Go layer, đọc từ `cfg.ShadowDB` config.
- Production deploy nên dựa vào `cfg.ShadowDB` (env-based) thay vì hardcode
  `current_database()` + `'env:DB_SINK_URL'` ở SQL seed.
- SQL seed values (default_database, secret_ref) ép local context vào prod
  DB — không reusable.

**Consequences**:
- Production deploy lần đầu phụ thuộc vào `bootstrap.EnsureDefaultShadowConnection`
  chạy đúng → cần verify path Go code này.
- Nếu Go bootstrap fail, production fail cold-boot. → Cần `06_validation.md`
  ghi rõ phải check Go bootstrap trước khi disable SQL seed.

---

## ADR-006 — Numbering convention seed `100+`

**Status**: Proposed

**Context**:
Seed files cần basename unique để tracker không collide với schema. Schema
hiện tại cao nhất = 052.

**Decision**: Seed prefix bắt đầu từ `100`.

**Rationale**:
- Buffer `053-099` (~47 số) cho schema growth.
- `100+` signal rõ "ngoài chuỗi DDL chính".
- Sort theo basename: schema 001-052 < seed 100-NNN → seed apply sau schema
  (đúng dependency direction).

**Consequences**:
- Nếu schema grow vượt 099 (47 file mới) — số seed phải push lên 200+.
  Rất khó xảy ra trong 3-5 năm tới.

---

## ADR-007 — `embed.go` dùng `all:` modifier

**Status**: Proposed

**Context**:
Pattern hiện tại liệt kê 9 folder explicit:
```go
//go:embed core/*.sql ids/*.sql ... ops/*.sql
```
Mỗi khi thêm folder mới (vd: `schema/auth/`) phải sửa embed.go.

**Decision**: Dùng `all:schema` và `all:seed`.

**Rationale**:
- `all:` modifier kéo sub-folder + file ẩn (leading `.` hoặc `_`).
- Auto-include folder mới → giảm maintenance.
- Filter `.sql` trong code (walkEmbed) thay vì pattern selectivity.

**Consequences**:
- Phải có filter `strings.HasSuffix(p, ".sql")` trong runner.
- File README.md trong schema/seed sẽ bị embed nhưng filter skip → no harm.

---

## ADR-008 — Runner signature `Run(db, includeSeeds, logger)`

**Status**: Proposed

**Context**:
Cần truyền config toggle vào migrator. Options:
- A. Pass full `cfg *AppConfig` → tight coupling.
- B. Pass struct `Options{IncludeSeeds bool}` → extensible nhưng overkill.
- C. Pass bool primitive → simple.

**Decision**: Chọn **C** với positive name `includeSeeds`.

**Rationale**:
- Toggle hiện tại chỉ 1 boolean → primitive đủ.
- Tên `includeSeeds` (positive) tránh double negation in tests
  (`!skipSeeds`).
- Caller (`server.go`) negate từ `cfg.Migration.SkipSeeds` 1 lần.

**Consequences**:
- Tương lai nếu thêm toggle khác (vd: `DryRun`) phải refactor sang struct.
  → Defer, không over-engineer.

---

## ADR-009 — Rename `.archive/` → `archive/`

**Status**: Proposed

**Context**:
Folder hiện tại `.archive/` (leading dot) → file inside sẽ bị `go:embed`
ignore mặc định. Nhưng `all:archive` cũng có thể fix.

**Decision**: Rename `.archive/` → `archive/` (bỏ leading dot).

**Rationale**:
- Folder không có lý do hidden — đây là docs/history quan trọng.
- Symmetry với `cluster/`, `seed/`, `schema/` (đều không leading dot).
- `embed.go` KHÔNG embed `archive/` (chỉ embed `schema/` + `seed/`) → file
  trong archive vẫn không vào binary.

**Consequences**:
- File listing trên Mac Finder hiển thị folder rõ ràng (không hidden).
- Cần update bất kỳ `.gitignore` hoặc tooling reference `.archive`.

---

## ADR-010 — Config `migration.skipSeeds`, không phải `migration.seedMode`

**Status**: Proposed

**Context**:
Naming option:
- A. `migration.skipSeeds: true/false` (boolean negate).
- B. `migration.seedMode: "skip"|"apply"` (enum string).
- C. `migration.includeSeeds: true/false` (boolean positive).

**Decision**: Chọn **A** (`skipSeeds`).

**Rationale**:
- Production thường có nhiều "skip" toggle (skip_telemetry, skip_warmup, ...).
  → `skipSeeds` consistent với pattern.
- Default false = include seed (an toàn cho dev quên config).
- ENV `CMS_MIGRATION_SKIP_SEEDS=true` đọc semantic rõ ("skip in production").

**Consequences**:
- Runner internal dùng `!skipSeeds` để có positive name `includeSeeds`.
- Bool double negation chỉ xảy ra 1 chỗ (server.go gọi migrate.Run).

---

## ADR-011 — Tracker không bị mutate qua refactor

**Status**: Proposed

**Context**:
DB local đã ghi 28 row tracker. Có thể tempt manually update tracker để
"clean up" (vd: rename row `029_v2_connection_registry` → `029_v2_connection_registry_no_seed`).

**Decision**: TUYỆT ĐỐI KHÔNG mutate tracker.

**Rationale**:
- REQ-4 "không cheat DB".
- File basename giữ nguyên → tracker auto-compat.
- Idempotent IF NOT EXISTS / ON CONFLICT đảm bảo file 007/029 sau khi bóc
  seed re-run vẫn no-op trên DB đã có data.

**Consequences**:
- Phải verify local DB sau migrate: `SELECT count(*) FROM
  cdc_system.schema_migrations` = 30 (28 cũ + 100 + 101) sau lần đầu apply
  seed; = 28 nếu skipSeeds=true.

---

## ADR-012 — Workspace document set follow CLAUDE.md §7

**Status**: Accepted

**Context**:
CLAUDE.md §7 yêu cầu Full Doc Set: 00_context, 01_requirements, 02_plan,
03_implementation, 04_decisions, 05_progress, 06_validation, 08_tasks,
09_tasks_solution.

**Decision**: Tuân thủ đủ bộ. Bonus: thêm `report_refactor_2026-05-15.md`
ở repo root migrations folder (REQ-7).

**Rationale**:
- Governance rule mandatory.
- Future Brain/Muscle session có thể trace lý do từng quyết định.

**Consequences**:
- ~10 file workspace cần tạo (đã có 00, 01, 02; pending 03, 04, 05, 06, 08, 09).
- Memory file protection: 05_progress.md PHẢI append-only (CLAUDE.md §11).

---

## ADR-013 — Seed naming: descriptive, no number prefix (supersedes ADR-006)

**Status**: Accepted (2026-05-15 user correction)

**Context**:
ADR-006 đề xuất `seed/100_*.sql`, `seed/101_*.sql` để fit-style với chuỗi
schema 001-052. User feedback: "Seed numbering mày ko biết cái gì hợp lý
à. mấy cái migration & seed này nó chỉ chạy 1 lần".

**Decision**: Đổi sang **descriptive filename, không số prefix**:
- `seed/worker_schedules.sql`
- `seed/v2_default_connections.sql`

**Rationale**:
- Migration & seed CHẠY 1 LẦN (idempotent tracker skip) → numbering chỉ
  để "ordering trong cùng phase". Vì runner đã apply schema/ trước seed/
  (sort 2 list riêng), trong seed/ chỉ có vài file, ordering nội bộ không
  critical.
- Tracker dùng `path.Base()` làm PK → `worker_schedules` / `v2_default_connections`
  vẫn unique, không collide với chuỗi `001_*` → `052_*`.
- Tên descriptive = self-documenting, "trực quan, chuyên nghiệp" (REQ-1).
- Loại bỏ "fake migration numbers" gây hiểu lầm đó là tiếp nối schema sequence.

**Consequences**:
- Sort tổng hợp basename: schema (ASCII `0`-`9` prefix) < seed (ASCII `v`/`w`
  prefix). Đúng thứ tự apply. Nhưng runner sẽ sort SCHEMA list riêng và
  SEED list riêng (append theo phase) → naming trong seed/ tự do hoàn toàn.
- ADR-006 (numbering 100+) **SUPERSEDED**.

---

## ADR-014 — Bỏ luôn `legacy_shadow_default` seed (supersedes ADR-005)

**Status**: Accepted (2026-05-15 user correction)

**Context**:
ADR-005 đề xuất tách 3 legacy_* INSERT (system, shadow, master) sang
seed/. User feedback: "cdc_internal nó còn ko đc xài nữa, old lắm rồi" +
Go `bootstrap.EnsureDefaultShadowConnection` (server.go:94) đã INSERT
row `connection_code='default_shadow'` với `ON CONFLICT DO UPDATE` từ
`cfg.ShadowDB` env-driven.

**Decision**:
- BỎ HOÀN TOÀN `legacy_shadow_default` row (Go bootstrap cover).
- GIỮ `legacy_system_db` + `legacy_master_default` trong seed mới, nhưng
  đổi `default_schema='cdc_internal'` → `'cdc_system'` (cdc_internal đã
  bị DROP CASCADE ở migration 038 line 234).
- Seed file mới: `seed/v2_default_connections.sql` (2 row thay vì 3).

**Rationale**:
- Tránh redundant: 2 row shadow (Go `default_shadow` + SQL `legacy_shadow_default`)
  cùng role_type='shadow' sẽ làm V2SyncCommand resolve không deterministic.
- Pointer đến schema không tồn tại (`cdc_internal`) là data drift bug —
  fix forward thay vì carry legacy.
- `legacy_system_db` (role=system) + `legacy_master_default` (role=master)
  vẫn cần để V2SyncCommand resolve source/master target có row active.

**Consequences**:
- Local DB đã apply migration 029 cũ → có 3 row `legacy_*` với
  `default_schema='cdc_internal'`. Khi seed mới apply, INSERT WHERE NOT
  EXISTS sẽ skip (row tồn tại theo connection_code). Drift cũ giữ
  nguyên trên local DB. Production fresh sẽ có data đúng ngay từ đầu.
- Để fix local drift cho thẩm mỹ, thêm UPDATE statement đầu seed file:
  ```sql
  UPDATE cdc_system.connection_registry
     SET default_schema='cdc_system'
   WHERE default_schema='cdc_internal'
     AND connection_code IN ('legacy_system_db','legacy_master_default');
  ```
- ADR-005 (3-row split) **SUPERSEDED**.

---

## ADR-015 — Worker schedule column names: bám sát actual schema

**Status**: Accepted (2026-05-15 user correction)

**Context**:
Plan draft ban đầu của tôi tự suy diễn column names sai (`job_name/
interval_seconds`). Reading file `worker/007_worker_schedule.sql` thực tế:
- Column: `operation` (VARCHAR(50)), `target_table` (VARCHAR(200)),
  `interval_minutes` (INT), `is_enabled` (BOOLEAN), `notes` (TEXT).
- UNIQUE: `(operation, target_table)`.
- 5 default rows: `bridge|transform|airbyte-sync (5 min)`, `field-scan
  (60 min)`, `partition-check (1440 min)`.

**Decision**: Seed file mới copy EXACT từ block INSERT line 30-36 của
007, KHÔNG re-design tên column.

**Rationale**:
- Tránh schema/seed drift.
- Tránh break consumer code đang đọc `operation` / `target_table` /
  `interval_minutes` (xem `internal/app/queries/list_worker_schedules.go`,
  `internal/persistence/worker_schedule_read_repo.go`).

**Consequences**:
- Plan documents (02/03/08/09) phải dùng đúng tên cột.
- Lesson: ALWAYS đọc file SQL gốc TRƯỚC khi viết spec; không suy diễn.

---

## ADR-016 — Skip in-place renumber file SQL; chỉ tách physical layout

**Status**: Accepted (re-confirm ADR-002)

**Context**:
User nói "migration & seed này nó chỉ chạy 1 lần". Tempt: tại sao không
renumber file luôn cho gọn (001-030 liên tục)?

**Decision**: KHÔNG renumber. Giữ basename hiện tại 001-052 (có 24 lỗ
thủng).

**Rationale**:
- Local DB tracker đã ghi 28 row theo basename hiện tại. Renumber → tracker
  miss → re-apply file đã apply → có thể fail trên DDL không idempotent.
- Production cold-boot sau renumber: tracker fresh, apply 001-030 mới
  từ đầu — OK. Nhưng KHÔNG có production cold-boot trong scope task này
  (xem out-of-scope §4 trong 00_context.md).
- User goal: layout "trực quan" qua FOLDER (schema/seed/cluster/archive),
  KHÔNG yêu cầu renumber file individual.

**Consequences**:
- archive/README.md liệt kê 24 lỗ thủng + lý do từng số bị bỏ.
- schema/README.md note rằng numbering không liên tục là di sản squash
  ops, không bug.
