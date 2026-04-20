# Implementation — v4 FE Recon UX (DataIntegrity rewrite)

> **Date**: 2026-04-17
> **Author**: Muscle (claude-opus-4-7)
> **ADR**: `04_decisions_recon_systematic_v4.md` §2.2 + §2.3 + §2.7 + §2.8
> **Scope**: FE-only. Backend tasks (timestamp detector, full-count aggregator, `/detect-timestamp-field` endpoint, error_code stamping) are tracked separately and are tolerated as "not yet populated" by this FE rollout (all new fields are optional with graceful fallbacks).

---

## 1. Files touched

| Kind    | Path                                                                | Summary                                                                                   |
|---------|---------------------------------------------------------------------|-------------------------------------------------------------------------------------------|
| NEW     | `cdc-cms-web/src/constants/reconErrorMessages.ts`                   | VI translation map + severity map + `lookupReconError` helper                             |
| NEW     | `cdc-cms-web/src/components/ReDetectButton.tsx`                     | Per-row button → POST `/api/registry/:id/detect-timestamp-field`                          |
| MODIFY  | `cdc-cms-web/src/hooks/useReconStatus.ts`                           | Add `ReconRow` + `ReconStatus` types; extend `ReconReport` (superset) for back-compat     |
| MODIFY  | `cdc-cms-web/src/pages/DataIntegrity.tsx`                           | Replace `reportColumns` with v4 layout; add status/engine lookup tables; wire `ReDetectButton` |

## 2. Column changes (ADR §2.8)

Old columns:

```
Bảng | Source DB | Sync Engine | Source(7d+tooltip) | Dest | Chênh lệch | Thiếu | Trạng thái | Tier | Kiểm tra lúc | Đã chữa | Thao tác
```

New columns:

```
Bảng | Sync Engine | Total Source | Total Dest | Source (7d window) | Dest (7d window) | Drift % | Trạng thái | Timestamp field | Kiểm tra lúc | Thao tác
```

Rationale (per ADR §2.7):

- **Full counts first**. Operators have repeatedly asked "is the full table in sync?" — window counts mislead because snapshot `_source_ts` can diverge from Mongo `updated_at`. The two leftmost numeric columns now surface the daily-aggregated absolute truth. Window counts kept (as "7d window") for drift-recent detection only.
- **Drift % is the primary status signal**, bounded 0-100% and unsigned (ADR §2.2). Colour threshold: `<0.5%` = default, `<5%` = gold, `≥5%` = red. A `null` drift renders as `-` so rows produced by legacy backend (no drift_pct yet) don't break.
- **Source (7d window)** renders `Query fail` tag when `source_count` is NULL — the old FE would show `0` here and the user couldn't tell if that meant "Mongo is empty" or "the query blew up". Migration 017 already made `cdc_reconciliation_report.source_count` NULLABLE to let the backend propagate this distinction.
- **Diff / Missing / Tier / Healed columns dropped**. They were summary-noise; detailed history still lives on `TableHistory` endpoint (unchanged).
- **Timestamp field column added**. Shows the currently-used field name as `<code>`, a `Manual` tag when admin-pinned, and hovering surfaces `timestamp_field_confidence + timestamp_field_source`. Operators can now tell at a glance whether a value is auto-detected vs overridden — closes the "why is source 0?" loop (ADR §1.1).

## 3. Error translation (ADR §2.3)

`reconErrorMessages.ts` ships three exports:

- `ERROR_MESSAGES_VI` — flat `Record<code, string>` covering the 10 backend error codes.
- `ERROR_SEVERITY` — maps each code to `critical | warning | info` → drives tag colour (red / orange / default).
- `lookupReconError(code)` — safe helper. Unknown codes fall back to `UNKNOWN` severity `warning` but show the raw code string so new backend codes don't get hidden behind a generic "Lỗi không xác định".

The `Trạng thái` column delegates to `lookupReconError` only when `status === 'error'`. Non-error statuses use the `STATUS_COLOR` + `STATUS_LABEL_VI` lookup tables (green / gold / red / orange / default) — no more per-status ternary cascade.

## 4. Re-detect button (ADR §2.1)

`ReDetectButton` wraps `useAsyncDispatch` with:

- `endpoint = /api/registry/:id/detect-timestamp-field`
- `operation = detect-timestamp-field` (audit-log subject)
- `targetTable` for multi-tenant filtering on the status poll
- `invalidateKeys = [['registry'], ['recon-report']]` — so the recon table refreshes as soon as the worker persists the newly-detected field.

Button is hidden on rows where `registry_id` is missing (defensive: backend may not have joined that row). The `reason` field carries `Re-detect timestamp field cho <table>` — traceable via the governance `X-Action-Reason` header.

## 5. Tooltip placement rationale

All new tooltips sit on `InfoCircleOutlined` inside a `<Space size={4}>` next to the column title:

```
Total Source [i]
```

- Keyboard accessibility: `tabIndex={0}` + `aria-label` make each info icon focus-navigable; AntD's Tooltip hooks `aria-describedby` on focus/blur natively.
- Placement on the icon (not the whole header) avoids the "Antd filter-hover vs tooltip-hover" race that the old code hit on the Source column.
- Tooltip text is flat strings (not nested `<div>` blobs like the old Source tooltip) — simpler to translate and screen-reader-friendly.

## 6. Type hardening

`ReconRow` is the new source of truth. It narrows status to a discriminated union (`ReconStatus`), marks source_count / drift_pct / full_*_count as nullable, and carries the new detector metadata. `ReconReport` extends `ReconRow` to keep existing mutation callers (heal, check-all, backfill) compiling without touching them.

No `any` introduced. The only `unknown` is in `render: (_: unknown, record) => ...` for the actions column, matching AntD's typing idiom.

## 7. Build verification

```
$ npx tsc --noEmit -p tsconfig.app.json
(exit 0)

$ npm run build
...
dist/assets/DataIntegrity-CV9u-7It.js  16.07 kB │ gzip: 5.48 kB
✓ built in 361ms
```

Chunk size 5.48 kB gzip — well under the 20 kB budget called out in the brief.

## 8. Runtime expectations

Until the backend ships (a) `LatestReport` joining `cdc_table_registry` for `full_source_count / full_dest_count / timestamp_field_confidence / timestamp_field_source / registry_id`, and (b) `drift_pct` / `error_code` on the report model, the FE will render:

- Total Source / Total Dest → `—` (em-dash, muted text)
- Drift % → `-`
- Timestamp field confidence tooltip → `Confidence: - | Source: -`
- Re-detect button → hidden (registry_id null)

None of these break the page; they degrade visibly so operators know what's not yet wired.

## 9. Follow-up (not in this task)

- Backend: finalize `cdc_reconciliation_report` model fields (`error_code`, `drift_pct`) and the LatestReport SELECT to surface registry metadata (`full_*_count`, `timestamp_field_confidence`, `timestamp_field_source`, `registry_id`). Current handler already joins on `cdc_table_registry` but SELECTs only three registry columns.
- Backend: `/api/registry/:id/detect-timestamp-field` endpoint + worker subject `cdc.cmd.detect-timestamp-field`.
- FE: optional `HealButton` / `CheckButton` component extraction (currently inline in DataIntegrity) if we want them reused on TableRegistry as well.
