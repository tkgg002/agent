# 03 — Implementation: Observability v3 FE Phase 0 (Infrastructure Quick Wins)

**Phase**: Phase 0 — FE Infrastructure Quick Wins
**Source plan**: `02_plan_observability_v3.md` §12 (FE v3)
**Scope**: `/Users/trainguyen/Documents/work/cdc-cms-web/`
**Executor**: Muscle (Chief Engineer) — CC CLI
**Date**: 2026-04-17
**Status**: DONE (build pass, dev server pass)

---

## 1. Objective

Chuẩn bị hạ tầng FE cho v3 observability mà KHÔNG refactor page hiện tại. 3 deliverables:
1. Cài `@tanstack/react-query` + wire provider.
2. Hook `useSystemHealth` + `useRestartConnector` tái sử dụng cho các page sắp refactor.
3. Shared `ConfirmDestructiveModal` với audit-reason enforcement.

Không touch Go services, không refactor `pages/SystemHealth.tsx`/`pages/DataIntegrity.tsx` (Phase 1 job).

---

## 2. Task Breakdown & File Citations

### TASK 1 — React Query Setup

| Artifact | Path | Change |
|----------|------|--------|
| Deps | `/Users/trainguyen/Documents/work/cdc-cms-web/package.json` | +`@tanstack/react-query@^5.59.0`, +`@tanstack/react-query-devtools@^5.59.0` |
| Entry | `/Users/trainguyen/Documents/work/cdc-cms-web/src/main.tsx` | Wrap `<App/>` với `<QueryClientProvider>` + DevTools ở dev mode |

**QueryClient config** (align với v3 cache policy):
```ts
defaultOptions: {
  queries: { staleTime: 25_000, retry: 2 }
}
```
→ `staleTime: 25s` khớp backend cache TTL (tránh thundering-herd với reconcile snapshot).

**Install manager**: `npm` (dự án đã có `package-lock.json`, không yarn.lock).

### TASK 2 — Hooks cho System Health

| Artifact | Path |
|----------|------|
| Hook file | `/Users/trainguyen/Documents/work/cdc-cms-web/src/hooks/useSystemHealth.ts` (NEW) |

**Exports**:
- `useSystemHealth()` — `useQuery` wrapper GET `/api/system/health`, `refetchInterval: 30s`, `staleTime: 25s`, `retry: 2`.
- `useRestartConnector()` — `useMutation` POST `/api/tools/restart-debezium` với headers:
  - `Idempotency-Key`: UUID sinh từ `crypto.randomUUID()` (fallback `Date.now()+random` cho env cũ).
  - `X-Action-Reason`: reason text để audit log.
  - Body `{ reason, connector }`.
  - `retry: 0` (destructive — không tự động lặp).

**Types** (forward-compat):
```ts
type SectionStatus = 'ok' | 'degraded' | 'down' | 'unknown';
interface SectionResult { status: SectionStatus; error?: string; data?: unknown }
interface SystemHealthSnapshot {
  timestamp: string;
  cache_age_seconds: number;
  sections: {
    infrastructure?: SectionResult;
    pipeline?: SectionResult;
    reconciliation?: SectionResult;
    latency?: SectionResult;
    alerts?: SectionResult;
    recent_events?: SectionResult;
  };
  [key: string]: unknown;  // backend v3 đang refactor, giữ flexible
}
```

**Axios**: reuse `cmsApi` instance từ `src/services/api.ts` (JWT interceptor + 401 handling sẵn). KHÔNG tạo axios instance mới → consistent với codebase.

### TASK 3 — ConfirmDestructiveModal

| Artifact | Path |
|----------|------|
| Component | `/Users/trainguyen/Documents/work/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx` (NEW) |

**Props**:
```ts
interface ConfirmDestructiveModalProps {
  open: boolean;
  title: string;
  description: string;
  targetName: string;
  actionLabel: string;
  danger?: boolean;
  onConfirm: (reason: string) => Promise<void> | void;
  onCancel: () => void;
  loading?: boolean;
}
```

**Behavior**:
- AntD `<Modal>` với optional `<Alert type="warning">` khi `danger=true`.
- `<Input.TextArea>` cho "Lý do" — required, min 10 chars (trim trước khi validate).
- OK button `disabled` cho tới khi `reason.trim().length >= 10`, và `danger` flag → button đỏ.
- `useEffect(open)` reset reason state mỗi lần mở → không carry stale text.
- Trong khi `loading`/`submitting`: disable OK/Cancel, disable `maskClosable` → tránh đóng giữa chừng.
- `destroyOnHidden` → unmount form mỗi lần đóng.

Usage example in-file (comment block ở đầu file) — copy-paste ready cho Phase 1 refactor.

---

## 3. Verification (Rule 3 — Test Before Done)

| Check | Command | Result |
|-------|---------|--------|
| Install | `npm install` | +4 packages, no errors |
| TypeScript | `tsc --noEmit -p tsconfig.app.json` | PASS (0 errors trong files mới) |
| Build | `npm run build` | PASS — `tsc -b && vite build` → `dist/index-*.js` 1.24 MB gz 392 KB |
| Dev start | `npm run dev` | PASS — Vite ready on `:5174`, re-optimized deps OK |

**Build warnings** (không phải lỗi): chunk >500 KB — tech debt, ghi chú cho Phase 1 (code-split routes).

---

## 4. Side Effects (pre-existing tech debt cleanup)

Để build pass với `noUnusedLocals: true` + `noUnusedParameters: true`, phải xử lý 5 lỗi TS pre-existing ở page CŨ (không phải tôi tạo):

| File | Line | Fix |
|------|------|-----|
| `src/pages/ActivityLog.tsx` | L3 | Bỏ `ForwardOutlined` khỏi import list (chưa dùng) |
| `src/pages/ActivityLog.tsx` | L52 | Đổi `const [recentErrors, setRecentErrors]` → `const [, setRecentErrors]` (state được set nhưng chưa render) |
| `src/pages/DataIntegrity.tsx` | L2 | Bỏ `Modal` khỏi import list (chưa dùng) |
| `src/pages/TableRegistry.tsx` | L32 | Thêm `// @ts-expect-error` comment trước `handleForceSync` (declared for future wiring) |
| `src/pages/TableRegistry.tsx` | L158 | Thêm `// @ts-expect-error` comment trước `handleStandardize` (declared for future wiring) |

Không thay đổi runtime behavior. Đánh dấu TODO(phase1) cho 2 function sẽ wire vào row action menu.

---

## 5. File Inventory

**NEW**:
- `/Users/trainguyen/Documents/work/cdc-cms-web/src/hooks/useSystemHealth.ts`
- `/Users/trainguyen/Documents/work/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx`

**MODIFIED**:
- `/Users/trainguyen/Documents/work/cdc-cms-web/package.json`
- `/Users/trainguyen/Documents/work/cdc-cms-web/src/main.tsx`
- `/Users/trainguyen/Documents/work/cdc-cms-web/src/pages/ActivityLog.tsx` (unused-cleanup)
- `/Users/trainguyen/Documents/work/cdc-cms-web/src/pages/DataIntegrity.tsx` (unused-cleanup)
- `/Users/trainguyen/Documents/work/cdc-cms-web/src/pages/TableRegistry.tsx` (unused-cleanup)

**UNTOUCHED** (theo scope):
- `src/pages/SystemHealth.tsx` — giữ nguyên, Phase 1 sẽ refactor để dùng `useSystemHealth`.
- `src/pages/DataIntegrity.tsx` business logic — chỉ clean unused import.
- 2 Go services (`cdc-cms-service`, `centralized-data-service`).

---

## 6. Next (Phase 1 — Refactor Pages)

- Thay `useState + useEffect + cmsApi.get` trong `SystemHealth.tsx` bằng `useSystemHealth()`.
- Thay restart button trong `SystemHealth.tsx` bằng `<ConfirmDestructiveModal>` + `useRestartConnector()`.
- Apply cùng pattern cho `DataIntegrity.tsx` (reconcile trigger, reload-cache).
- Split routes thành dynamic `import()` để giảm chunk.
- Xoá `// @ts-expect-error` trong `TableRegistry.tsx` khi wire `handleForceSync`/`handleStandardize`.
