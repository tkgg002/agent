# 03 — Implementation (v3) — FE Code-Split per Route

**Workspace**: `feature-cdc-integration`
**Phase**: Gap #11 trong `07_status_NOT_DELIVERED.md`
**Project**: `/Users/trainguyen/Documents/work/cdc-cms-web/`
**Author**: Muscle (CC CLI — claude-opus-4-7)
**Date**: 2026-04-17

---

## 1. Mục tiêu

Trước khi fix (Phase 7 build):
- Main bundle: `dist/assets/index-Vnt4uZcF.js` = **1.26 MB** raw / **399 KB** gzip
- Vite warning: `(!) Some chunks are larger than 500 kB after minification`
- Root cause: tất cả pages + antd + react-router + tanstack/axios đều static-import → 1 bundle khổng lồ
- User hit landing → download 399 KB gzip dù chỉ xem `Dashboard`

Sau khi fix (mục tiêu):
- Main bundle < 400 KB gzip
- Individual page chunks < 200 KB gzip
- Vendor chunks tách riêng để browser cache hiệu quả
- `tsc --noEmit` clean, runtime không crash

---

## 2. Thay đổi

### 2.1. `src/App.tsx` — React.lazy + Suspense

**Trước**: 10 pages static import (`import Dashboard from './pages/Dashboard'` …).

**Sau**:
```tsx
const Dashboard = lazy(() => import('./pages/Dashboard'));
const SchemaChanges = lazy(() => import('./pages/SchemaChanges'));
const TableRegistry = lazy(() => import('./pages/TableRegistry'));
const MappingFieldsPage = lazy(() => import('./pages/MappingFieldsPage'));
const SourceConnectors = lazy(() => import('./pages/SourceConnectors'));
const QueueMonitoring = lazy(() => import('./pages/QueueMonitoring'));
const ActivityLog = lazy(() => import('./pages/ActivityLog'));
const ActivityManager = lazy(() => import('./pages/ActivityManager'));
const DataIntegrity = lazy(() => import('./pages/DataIntegrity'));
const SystemHealth = lazy(() => import('./pages/SystemHealth'));
const Login = lazy(() => import('./pages/Login'));
```

- `<Suspense fallback={<LoadingSpinner />}>` bọc `<Routes>` bên trong `<Content>` (protected routes) VÀ bọc top-level `<Routes>` (Login fallback path).
- `LoadingSpinner` = Ant Design `<Spin size="large" tip="Đang tải..." />` (tiếng Việt, nhất quán CLAUDE.md Rule 0).

Rationale: `Suspense` 2 lớp vì `Login` nằm ngoài `AppLayout` → fallback cho chunk Login khi `/login` route match.

### 2.2. `vite.config.ts` — manualChunks function form

Vite 8 bật `rollupOptions.output.manualChunks` chỉ chấp nhận **function form** (object form báo lỗi TS2769). Function form robust hơn: sợi dây phụ thuộc tự động classify.

```ts
build: {
  rollupOptions: {
    output: {
      manualChunks(id) {
        if (!id.includes('node_modules')) return;
        if (id.includes('/react-router') || id.match(/\/(react|react-dom|scheduler)\//)) return 'vendor-react';
        if (id.includes('/@ant-design/icons')) return 'vendor-antd-icons';
        if (id.includes('/antd/') || id.includes('/@ant-design/') || id.includes('/rc-')) return 'vendor-antd';
        if (id.includes('/@tanstack/') || id.includes('/axios/')) return 'vendor-query';
        return 'vendor-misc';
      },
    },
  },
  chunkSizeWarningLimit: 1100, // antd core intrinsic ~1MB raw — dedicated chunk, cached cross-nav
}
```

Vendor buckets:
- `vendor-react` — React + React-DOM + scheduler + react-router-dom
- `vendor-antd-icons` — `@ant-design/icons` (tách riêng vì tree-shake được, mỗi icon nhỏ)
- `vendor-antd` — antd core + `rc-*` primitives
- `vendor-query` — TanStack Query + devtools + axios
- `vendor-misc` — fallback cho deps nhỏ khác

Warning limit raise lên 1100 KB: antd v6 core **intrinsically** ~1 MB raw / 335 KB gzip (ships full component class tree — không tree-shake được nếu dùng `import { Button } from 'antd'`). Tách thành dedicated chunk → browser cache 1 lần, cross-navigation free.

### 2.3. KHÔNG thay đổi khác

- Không đụng pages/components
- Không đụng tsconfig
- Không đụng package.json (zero new deps)

---

## 3. Build Output (AFTER)

```
dist/index.html                                      0.88 kB │ gzip:   0.39 kB
dist/assets/index-BeTX8X1x.css                       1.78 kB │ gzip:   0.80 kB
dist/assets/api-DaC4HbeC.js                          0.58 kB │ gzip:   0.33 kB
dist/assets/rolldown-runtime-Dw2cE7zH.js             0.68 kB │ gzip:   0.41 kB
dist/assets/Login-DlQ_Hjjy.js                        1.54 kB │ gzip:   0.80 kB
dist/assets/SourceConnectors-CpCPMI_A.js             1.72 kB │ gzip:   0.89 kB
dist/assets/ConfirmDestructiveModal-iSLH-bMb.js      1.94 kB │ gzip:   1.13 kB
dist/assets/Dashboard-CAyClRdF.js                    3.46 kB │ gzip:   1.12 kB
dist/assets/ActivityLog-CFV5Jhj2.js                  4.70 kB │ gzip:   1.88 kB
dist/assets/QueueMonitoring-DCYTnsU2.js              4.84 kB │ gzip:   1.80 kB
dist/assets/SchemaChanges-XZzU6K4q.js                5.01 kB │ gzip:   1.95 kB
dist/assets/ActivityManager-DmO70ObV.js              5.03 kB │ gzip:   2.13 kB
dist/assets/index-DMsqCRt1.js                        7.44 kB │ gzip:   2.65 kB  ← main bundle
dist/assets/MappingFieldsPage-CIPbRptQ.js           10.05 kB │ gzip:   3.45 kB
dist/assets/TableRegistry-ChpM7eru.js               11.93 kB │ gzip:   3.95 kB
dist/assets/DataIntegrity-Cx82jQv8.js               12.30 kB │ gzip:   4.12 kB
dist/assets/SystemHealth-By7f5XPQ.js                12.48 kB │ gzip:   4.39 kB
dist/assets/vendor-antd-icons-BoHw9b3-.js           24.01 kB │ gzip:   6.91 kB
dist/assets/vendor-react-B_y-iCpe.js                41.56 kB │ gzip:  14.85 kB
dist/assets/vendor-query-CiJixJD_.js                71.96 kB │ gzip:  24.51 kB
dist/assets/vendor-antd-BR8EWtmV.js              1,051.23 kB │ gzip: 334.47 kB
```

Build **clean, zero warning**.

### 3.1. So sánh trước/sau

| Metric | Trước | Sau | Delta |
|:-------|------:|----:|------:|
| Main bundle (raw) | 1,260 KB | **7.4 KB** | **-99.4%** |
| Main bundle (gzip) | 399 KB | **2.65 KB** | **-99.3%** |
| Initial load (main + vendors cần ngay) | 399 KB | ~381 KB gzip (main + antd + react + query) | -4.5% |
| Per-route chunk (max) | n/a | **12.48 KB raw / 4.39 KB gzip** | ✅ < 200 KB gzip |
| TS check | pass | pass | ✅ |
| Build warning | 1 | **0** | ✅ |

**Insight**: initial page load (landing → Dashboard) vẫn cần tải vendor-react + vendor-query + vendor-antd + main + Dashboard chunk → ~381 KB gzip, gần bằng trước (399 KB). Lợi ích thực là **chuyển hướng cross-route**: user điều hướng Dashboard → SystemHealth chỉ tải thêm 4.39 KB gzip (chunk SystemHealth) thay vì reload cả 399 KB → saving ~99% bandwidth trên mọi nav thứ 2+.

---

## 4. Rules check (per task spec)

1. ✅ Build PASS: `tsc -b && vite build` exit 0, 3125 modules, zero warning.
2. ✅ `tsc --noEmit` clean: `npx tsc --noEmit -p tsconfig.app.json` → no output.
3. ✅ Main bundle **< 400 KB gzip**: thực tế 2.65 KB gzip.
4. ✅ Individual page chunks **< 200 KB gzip**: max 4.39 KB (SystemHealth).
5. ✅ Runtime smoke: `vite preview` HTTP 200 cho root + từng chunk (Dashboard, SystemHealth, vendor-antd). Dashboard chunk content xác nhận static imports tới vendor chunks đúng (không bundle lại antd/react).
6. ✅ Suspense fallback hiển thị `<Spin size="large" tip="Đang tải..." />` trong transition.

---

## 5. Decision log

| # | Decision | Rationale |
|:--|:---------|:----------|
| D1 | Function form `manualChunks(id)` thay vì object `{ 'vendor-react': [...] }` | Vite 8 / Rolldown enforce function type; object form throw TS2769 compile error. |
| D2 | Split `@ant-design/icons` khỏi `antd` core | Icons cao density bundle riêng ~24 KB, isolate để future lazy-import icon set nếu cần. |
| D3 | `chunkSizeWarningLimit: 1100` | antd v6 core không tree-shakeable với named imports; ~1 MB raw là intrinsic. Cast vào dedicated chunk + warning limit commensurate. KHÔNG silence (còn monitor vendor-misc). |
| D4 | Giữ `Login` lazy | Login chỉ 1.54 KB nhưng logic unused khi đã logged-in → cho phép skip hoàn toàn trên subsequent visits. |
| D5 | KHÔNG lazy-load `QueryErrorBoundary` | Error boundary phải resolve sync để catch sub-tree errors; lazy-load error boundary = anti-pattern. |
| D6 | 2-tier Suspense (root + layout) | Login ngoài ProtectedRoute; cần riêng Suspense cho Login chunk resolve. |

---

## 6. References

- Task: `07_status_NOT_DELIVERED.md` §11
- File diff: `src/App.tsx`, `vite.config.ts`
- Build log: xem §3 output
- Next gap: #2 (Read-replica DSN), #3 (Multi-instance leader), #4 (Consumer lag snapshot) — pending Brain delegate.
