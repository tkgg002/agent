# 03_implementation_pattern.md — Transform Strategy Registry (DONE)

> 2026-06-03 | Muscle:Claude-Opus-4.8 | Build + vet + test PASS, security gate PASS.

## Files (mới)
- `centralized-data-service/internal/service/transmute/strategy.go` — interface `Strategy`, `RunContext`, `Emit`, `BuildStats`, registry (`Register/Get/IsWhitelisted/List`), fallback `copy_1_to_1`.
- `.../transmute/copy_1_to_1.go` — strategy 1:1 (tách từ `buildMasterRow`, parity).
- `.../transmute/flatten.go` — strategy array-explode; spec `{"explode_path":"after.items[*]"}`; fan-out key `parentID + "::idx::" + n`.
- `.../transmute/strategy_test.go` — 6 test (registry, fallback, copy, flatten explode/skip/index, validate spec, sorted). PASS.
- `.../transmute/README.md` — "cách thêm 1 loại sync mới".

## Files (sửa) — `internal/service/transmuter.go`
- import `internal/service/transmute`.
- `masterBindingRuntime` + `TransformSpec []byte`; query `loadMaster` thêm `COALESCE(mb.transform_spec,'{}'::jsonb)`.
- `processBatch`: dispatch `transmute.Get(binding.TransformType)` → `BuildEmits` → loop emit (hash trước, system cols, `_source_id=SourceID+KeySuffix`, upsert). Counters parity.
- `buildMasterRow` → đổi tên `extractColumns` (trả `ok` thay hash; engine tính hash). Logic 1:1 giữ nguyên 100%.
- thêm `toTransmuteRules`, `extractColumnsFn`, `extractArrayBytes` (reuse `extractArrayByPath`).

## Verify (DoD)
- `go build ./internal/service/... ./cmd/...` = EXIT 0.
- `go vet ./internal/service/transmute/` = EXIT 0 (cảnh báo `pkgs/idgen` là pre-existing, không phải code này).
- `go test ./internal/service/transmute/` = PASS (6/6).
- `go test ./internal/service/` (full suite) = PASS (không regression).

## Security Gate (§8) — kết quả
- **Code mới: CLEAN** — `upsertMaster` parameterized (`$N`) + `quoteTransmuteIdent` cho cột; `loadMaster` SQL tĩnh; `transform_spec` parse vào struct 1 field string (không code-exec/panic); registry chỉ ghi lúc `init()` (không race); closure không race.
- **Finding 2 (LOW, đã FIX)**: separator fan-out đổi `#` → `::idx::` (khó đụng PK). Test cập nhật, PASS.
- **Finding 1 (HIGH, PRE-EXISTING, NGOÀI scope)**: `child_explode.go:218-223` nội suy thẳng `r.DataType` vào DDL `ALTER TABLE ... ADD COLUMN ... <dt>` KHÔNG validate → SQLi (admin-controlled). KHÔNG nằm trong path của flatten (flatten dùng master upsert parameterized). **Đề xuất fix riêng**: validate `dt` qua `TypeResolver`/regex whitelist trước DDL. → Chờ User duyệt vì đụng subsystem khác (shadow ingest), cần test path child_explode.

## Còn lại (option sau, mỗi cái 1 file)
- `filter.go` (row-skip theo predicate), `aggregate.go`/`group_by.go`/`join.go` (cân nhắc mart SQL vs emit SQL — quyết định khi tới).
- flatten: orphan soft-delete khi array co lại (deferred, ghi rõ ở README).
