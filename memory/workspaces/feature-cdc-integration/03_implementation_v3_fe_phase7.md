# 03 — FE Phase 7 Implementation (React Query + ConfirmDestructiveModal applied)

| Metadata   | Value                               |
|:-----------|:------------------------------------|
| Operator   | Muscle (Chief Engineer)             |
| Model      | claude-opus-4-7-1m                  |
| Date       | 2026-04-17                          |
| Scope      | `cdc-cms-web/` only (no Go touch)   |
| Depends on | Phase 0 FE (React Query bootstrap)  |

---

## 1. Tasks

| #  | Task                                                            | Files                                                                                      | Status |
|:---|:----------------------------------------------------------------|:-------------------------------------------------------------------------------------------|:------:|
| T1 | Refactor `SystemHealth.tsx` → `useSystemHealth()` + per-section | `src/pages/SystemHealth.tsx`                                                               | DONE   |
| T2 | Restart Connector → `ConfirmDestructiveModal` + invalidate cache| `src/pages/SystemHealth.tsx`                                                               | DONE   |
| T3 | Refactor `DataIntegrity.tsx` + NEW hook `useReconStatus`        | `src/pages/DataIntegrity.tsx`, `src/hooks/useReconStatus.ts`                               | DONE   |
| T4 | `QueryErrorBoundary` global wrap                                | `src/components/QueryErrorBoundary.tsx`, `src/App.tsx`                                     | DONE   |
| T5 | Skeleton + Empty states                                         | SystemHealth + DataIntegrity                                                               | DONE   |

---

## 2. File Inventory

### NEW

- `src/hooks/useReconStatus.ts` — `useReconReport`, `useFailedLogs`, `useCheckAllMutation`,
  `useCheckTableMutation`, `useHealMutation`, `useRetryFailedMutation`. All mutations
  attach `Idempotency-Key` (UUID) + `X-Action-Reason` via shared `auditHeaders` helper.
- `src/components/QueryErrorBoundary.tsx` — React class boundary wrapped in
  `QueryErrorResetBoundary`; default fallback renders Ant Design `<Alert>` + retry button
  that calls the reset hook so React Query refetches pending queries on click.

### REWRITTEN

- `src/pages/SystemHealth.tsx` (196 → 460 LOC).
  - Removed `useEffect` + `setInterval(fetchHealth, 30000)` → `useSystemHealth()`.
  - Added `resolveSections(snapshot)` adapter: prefers `data.sections.*` (Plan v3 §7),
    falls back to legacy top-level keys so FE ships ahead of backend rewrite.
  - Per-section `<HealthSection>` handles status `unknown` → `<Empty>`, `down` → `<Alert error>`,
    `degraded` → yellow warning + still render data, `ok` → data.
  - Loading state → `<Skeleton active>`. Full-page error → `<Alert>` + retry via `refetch()`.
  - `cache_age_seconds > 60` → stale-data banner above all sections.
  - `<DebeziumFailurePanel>` reads `sections.pipeline.data.debezium.tasks`, renders
    Restart button; clicking opens `<ConfirmDestructiveModal>` with
    `targetName = debezium.connector || 'debezium-connector'`, `danger`, `actionLabel="Restart Connector"`.
  - Success `onConfirm` → `restart.mutateAsync({reason, connectorName})` →
    `queryClient.invalidateQueries({queryKey: ['system-health']})`.
  - `<LatencyBody>` surfaces `latency.source` (`prometheus` | `fallback_worker_metrics` |
    `unknown`) via colour-coded tag + tooltip explaining data provenance.

- `src/pages/DataIntegrity.tsx` (191 → 430 LOC).
  - Removed four `useCallback`/`useState` + `useEffect` + `setTimeout(fetchX)` patterns →
    React Query hooks.
  - All 4 destructive actions (Check-all, Check-table, Heal, Retry) route through a single
    `<ConfirmDestructiveModal>` instance driven by a `ModalPlan` state (discriminated union
    on `kind`). Keeps modal a singleton and avoids state explosion.
  - `Heal` is `danger=true` (writes to dest). Check/Retry are `danger=false` but still
    require a reason (audit trail).
  - Tabs migrated to Ant Design 6 `items=[]` prop (previous code used deprecated `<TabPane>`
    — silent build warning on AntD 6, addressed alongside refactor).
  - Per-tab skeleton + empty + error fallback (with retry button).

### EDITED

- `src/App.tsx` — wrap `<Routes>` inside `<QueryErrorBoundary>` so any query error that
  escapes local boundaries still gives the user a retry button instead of a blank page.

### UNCHANGED (intentional)

- `src/main.tsx` — `<QueryClientProvider>` already set up Phase 0.
- `src/hooks/useSystemHealth.ts` — Phase 0 hook reused as-is.
- `src/components/ConfirmDestructiveModal.tsx` — Phase 0 component reused as-is.
- `src/services/api.ts` — Phase 0 axios instance reused.

---

## 3. Design Notes

### 3.1 Backward compatibility during backend v3 rollout

Backend `/api/system/health` currently returns **legacy shape**:
`{ overall, infrastructure, cdc_pipeline, reconciliation, latency, failed_sync, alerts, recent_events }`.

Plan v3 §7 calls for a new shape:
`{ timestamp, cache_age_seconds, sections: { infrastructure, pipeline, reconciliation, latency, alerts, recent_events } }`.

`resolveSections()` consumes either by preferring `data.sections.*` and falling back to
top-level keys. This lets FE Phase 7 ship without blocking the backend rewrite, and
when backend flips to the new shape no FE change is required.

### 3.2 Modal singleton in DataIntegrity

Using one `<ConfirmDestructiveModal>` instance with a `ModalPlan` discriminated union:

```ts
type ModalAction =
  | { kind: 'check-all' }
  | { kind: 'check-table'; table: string; tier: string }
  | { kind: 'heal'; table: string }
  | { kind: 'retry'; id: number; table: string };
```

Avoids N modals mounted simultaneously, reduces DOM churn, keeps focus management
simple (one `<Modal>` with `destroyOnHidden` resets reason field on every open — Phase 0
contract).

### 3.3 Audit headers helper

`useReconStatus.ts` centralises the `Idempotency-Key` + `X-Action-Reason` injection so
every destructive call inherits the same contract. Keeps the mutation hook body small
and prevents drift across call sites.

### 3.4 Accessibility

- Ant Design `<Modal>` traps focus by default (verified against ARIA-Dialog pattern).
- `<ConfirmDestructiveModal>` OK button disabled until reason valid → no accidental Enter.
- `destroyOnHidden` resets reason each open (verified by Phase 0 component).

---

## 4. Verification

### 4.1 Build (`npm run build`)

```
> cdc-cms-web@0.0.0 build
> tsc -b && vite build

✓ 3125 modules transformed.
dist/index.html                     0.46 kB │ gzip:   0.29 kB
dist/assets/index-BeTX8X1x.css      1.78 kB │ gzip:   0.80 kB
dist/assets/index-ChbqmnKl.js   1,262.87 kB │ gzip: 399.00 kB
✓ built in 453ms
```

Only outstanding warning: bundle chunk > 500 kB (pre-existing tech debt, tracked since
Phase 0 — to be addressed by `build.rolldownOptions.output.codeSplitting`).

### 4.2 TypeScript (`tsc -b`)

No errors. `tsc -b` runs as first step of `npm run build`; a failure would abort the
pipeline before `vite build`. Since the vite build ran to completion, tsc passed.

### 4.3 Runtime (`npm run dev`)

```
VITE v8.0.3  ready in 176 ms
➜  Local:   http://localhost:5173/
```

Transform checks (HTTP 200 each):

- `GET /src/pages/SystemHealth.tsx` → 200
- `GET /src/pages/DataIntegrity.tsx` → 200
- `GET /src/hooks/useReconStatus.ts` → 200
- `GET /src/components/QueryErrorBoundary.tsx` → 200
- `GET /` → 200 (index.html served)

### 4.4 Manual navigation (pending — requires backend running)

Runtime behaviour documented for manual spot-check when backend is up:

- `/system-health` → page renders with 6 sections; DevTools Network tab should show
  `/api/system/health` re-fire every 30s (hook `refetchInterval`). Cache age banner
  appears when `cache_age_seconds > 60`.
- `/data-integrity` → Overview tab shows recon table; clicking "Kiểm tra tất cả" opens
  modal; entering reason (≥10 chars) enables OK; OK triggers `POST /api/reconciliation/check`
  with `Idempotency-Key` + `X-Action-Reason` headers (verifiable in Network tab).
- `/system-health` with Debezium FAILED → red panel with "Restart Connector" button →
  modal opens → reason enforced → `POST /api/tools/restart-debezium` with headers.

---

## 5. Known Gaps / Follow-ups

| # | Item                                                                     | Phase    |
|:--|:-------------------------------------------------------------------------|:---------|
| 1 | Chunk-size warning > 500 kB → code-split per route                       | Phase 8  |
| 2 | Section `status` enum still widened with legacy 'up'/'healthy'/'FAILED'  | Phase 8 / backend rewrite |
| 3 | E2E test for confirm-modal focus trap (manual for now)                   | Phase 8 QA |
| 4 | Playwright smoke test for `/system-health` auto-refresh                  | Phase 8 QA |
| 5 | Retry button on failed log currently `danger=false` — reconsider if replay mutates prod data | Follow-up |

---

## 6. Security Gate Self-Review

- No secret/token leak introduced (axios JWT interceptor unchanged).
- All destructive APIs go through modal + reason + Idempotency-Key.
- `<QueryErrorBoundary>` surfaces `error.message` only — no raw stack to user.
- Mutation `retry: 0` prevents accidental double-execution on network hiccup.
- No new env vars, no new external URLs, no bundled secrets.
- No server-side changes in this phase (FE-only scope).

---

## 7. Skills / Tools Used

- Read / Write / Edit / Grep / Bash (curl, npm, pkill)
- React Query 5 (`useQuery`, `useMutation`, `useQueryClient`, `QueryErrorResetBoundary`)
- Ant Design 6 (`Modal`, `Skeleton`, `Empty`, `Alert`, `Tabs.items`, `Tooltip`)
- TypeScript discriminated unions for `ModalPlan` / `UnifiedSections`
- `vite build` + `tsc -b` verification
