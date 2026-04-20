# Implementation v3 — FE Async Dispatch Hooks (Phase 7.1)

> **Date**: 2026-04-17
> **Scope**: FE (cdc-cms-web) polling hooks for 202-Accepted backend endpoints
> **Trigger**: Backend refactor (`10_gap_analysis_scan_fields_boundary_violation.md`) moves long-running ops CMS HTTP handler → Worker NATS handler. CMS now returns `202 Accepted`; FE must poll status instead of blocking on sync response.
> **Status**: Delivered — tsc clean + build PASS + dev smoke PASS.

---

## 1. Hook design

### 1.1 File layout
- `src/hooks/useAsyncDispatch.ts` — generic hook (NEW).
- `src/hooks/useRegistry.ts` — specialized wrappers (NEW).
- `src/components/DispatchStatusBadge.tsx` — shared visual indicator (NEW).
- `src/pages/TableRegistry.tsx` — refactor scan-fields / sync / refresh-catalog buttons to dispatch hooks + badge + `ConfirmDestructiveModal`.

### 1.2 `useAsyncDispatch(opts)`
Top-level generic. Wraps React Query `useMutation` (dispatch) + `useQuery` (polling) so consumers get a single state object. Maintains `DispatchState` machine in component state.

**Options**
```ts
interface UseAsyncDispatchOptions {
  endpoint: string;
  statusEndpoint?: string;      // default `${endpoint}/dispatch-status`
  operation: string;            // activity-log subject filter
  targetTable?: string;         // multi-row endpoint filter
  pollInterval?: number;        // default 3000ms
  maxPollDuration?: number;     // default 5 min (hard cutoff → timeout state)
  invalidateKeys?: string[][];  // default [['registry'], ['mapping-rules']]
}
```

**Returns** `{ state, dispatch, dispatchAsync, isPending, reset }`.

### 1.3 State machine

```
              dispatch()
 idle  ────────────────────►  dispatching
                                    │ POST returns 202
                                    ▼
                                accepted ──┐
                                    │      │ poll returns running
                                    │      ▼
                                    │   running ──┐
                                    │      │      │
                  latest.status=success   │      │
                                    ▼      ▼      ▼
                                      success
                                       OR
                                      error
                                       OR
                                  timeout (maxPollDuration)
```

Transitions:
- `dispatch()` calls axios POST with `Idempotency-Key` + `X-Action-Reason` (governance pattern reused from `useReconStatus`/`useSystemHealth`).
- On 202 response: sets `sinceTs = now.toISOString()` and starts poll. `setTimeout(maxPollDuration)` arms the timeout guard.
- Poll query `['dispatch-status', endpoint, operation, sinceTs, targetTable]` runs every `pollInterval`ms while status ∈ {accepted, running}. Query is disabled otherwise (React Query auto-stops polling on terminal state).
- Latest entry in response drives transition. `success` → clear timer + invalidate caches. `error` → clear timer + populate `error`. `running` → advance state if not already.
- `reset()` restores `idle` + clears `sinceTs` + clears timer.

### 1.4 Polling strategy
| Aspect | Value | Rationale |
|:-------|:------|:----------|
| Interval | 3000 ms | Balances UI responsiveness vs backend load (Activity Log query cost). |
| Max duration | 300_000 ms (5 min) | Most dispatched ops finish < 2 min; 5 min covers slow Airbyte discover. After that: `timeout` status, UI prompts user to check Activity Log. |
| Stop trigger | Terminal state (success/error/timeout) OR `reset()` | React Query `enabled` + `refetchInterval` combo. |
| Dedup | `sinceTs` filter | Guarantees we only see entries from THIS dispatch, not prior runs of same operation. |
| Auth | `cmsApi` axios instance | Reuses JWT interceptor + 401 redirect. |

---

## 2. Endpoint ↔ hook mapping

| Hook | Endpoint | Operation | Target |
|:-----|:---------|:----------|:-------|
| `useScanFields(id, table)` | `POST /api/registry/:id/scan-fields` | `scan-fields` | per-table |
| `useSyncAirbyte(id, table)` | `POST /api/registry/:id/sync` | `airbyte-sync` | per-table |
| `useRefreshCatalog(id, table)` | `POST /api/registry/:id/refresh-catalog` | `refresh-catalog` | per-table |
| `useScanSource()` | `POST /api/registry/scan-source` | `scan-source` | global |
| `useRestartDebezium()` | `POST /api/tools/restart-debezium` | `restart-debezium` | global |
| `useBulkSyncFromAirbyte()` | `POST /api/registry/sync-from-airbyte` | `bulk-sync-from-airbyte` | global |

All share status endpoint `{endpoint}/dispatch-status?subject={operation}&since={ISO}&target_table={optional}`.

---

## 3. UI component — `DispatchStatusBadge`

AntD `Tag` with color per status:
| Status | Color | Label (vi) | Icon |
|:-------|:------|:----|:----|
| idle | default | Sẵn sàng | — |
| dispatching | processing | Đang gửi... | Spin |
| accepted | processing | Đã nhận — đang xử lý | Spin |
| running | processing | Đang chạy | Spin |
| success | success | Hoàn tất | — |
| error | error | Lỗi: \<msg\> | — |
| timeout | warning | Quá thời gian chờ | — |

Accessibility: wrapped in `<span tabIndex={0} aria-live="polite">` for keyboard nav + screen-reader announcements. Tooltip shows `Dispatched at ${localTime}`.

---

## 4. TableRegistry refactor

- Removed the sync `handleScanFields` imperative function.
- Introduced `AsyncRowActions` subcomponent per-row (hooks must be at top-level; one set of hooks per row).
- Buttons: **Quét field** (always), **Sync** + **Refresh catalog** (airbyte/both engines only).
- Each button triggers `ConfirmDestructiveModal` with audit reason (min 10 chars). `Sync` flagged `danger=true` due to Airbyte quota impact.
- Badges show live dispatch state per action (idle hidden — only show once user has dispatched).
- Success → `message.success` + `fetchData()` to refresh table.
- Error/timeout → `message.error`/`message.warning` pointing user to Activity Log.
- Preserved untouched: `handleBridge`, `handleTransform`, `handleCreateTable`, `handleCreateDefaultFields`, `handleBulkImport`, `updateEntry`.

---

## 5. Before / After UX

| | Before (sync block) | After (async dispatch) |
|:--|:--|:--|
| User clicks button | Button spinner ~5-30s | Modal opens → user enters reason |
| Backend | Handler blocks until scan done | Handler publishes NATS, returns 202 in ~100ms |
| UI feedback | Single success toast at end | Badge updates: dispatching → accepted → running → success |
| Audit trail | Partial (reason missing) | Full (reason + idempotency-key) |
| Failure mode | HTTP 500 after timeout | Worker error surfaced via Activity Log poll |
| Retry safety | Duplicate triggers possible | Idempotency-Key prevents duplicate execution |

---

## 6. Verification

```
✓ tsc --noEmit -p tsconfig.app.json → exit 0 (strict mode, no any leak)
✓ npm run build                     → exit 0 in 430ms
  └ TableRegistry chunk: 17.77 KB raw / 5.93 KB gzip (< 15 KB budget)
✓ npm run dev + curl /              → HTTP 200, <title>cdc-cms-web</title>
```

---

## 7. Open items / follow-ups

1. Backend `/dispatch-status` contract must match `{ entries: [{ status, error_message, details, timestamp }] }`. If backend returns different shape, consumer narrowing at `useAsyncDispatch.ts:statusQuery` needs adjustment.
2. `useScanSource` / `useRestartDebezium` / `useBulkSyncFromAirbyte` hooks are wired but not yet used by pages (Phase 7.2).
3. Consider extracting `AsyncRowActions` into shared component when SchemaChanges / DataIntegrity pages adopt the pattern.
4. Add Playwright E2E covering dispatch → badge transition → success (Phase 7.3).

---

## 8. Files delivered

- NEW `cdc-cms-web/src/hooks/useAsyncDispatch.ts` (~200 LOC)
- NEW `cdc-cms-web/src/hooks/useRegistry.ts` (~55 LOC)
- NEW `cdc-cms-web/src/components/DispatchStatusBadge.tsx` (~75 LOC)
- EDIT `cdc-cms-web/src/pages/TableRegistry.tsx` — added `AsyncRowActions`, 3 new buttons with badge + confirm modal. Legacy sync-handler helpers retained for non-dispatch ops.
