# 08_tasks_frontend — Checklist Muscle thực thi (Frontend)

> Mỗi task = 1 PR nhỏ. DoD bắt buộc trước khi tick.

---

## Phase F1 — Foundation (~3h)

### T-FE-01 — Types
- [ ] Tạo `cdc-cms-web/src/types/dashboard.ts` (Section 1 impl).
- **DoD**: `npm run typecheck` PASS, không có `any`.

### T-FE-02 — Service
- [ ] Tạo `src/services/dashboard.ts` — 5 fetcher (Section 2 impl).
- **DoD**: TS compile + axios instance reuse `cmsApi`.

### T-FE-03 — Hook
- [ ] Tạo `src/hooks/useDashboard.ts` — 5 hook (Section 3 impl).
- **DoD**: `npm run lint` PASS.

### T-FE-04 — Route + menu
- [ ] Edit `src/App.tsx` thêm route `/dashboard-v2`.
- [ ] Thêm menu item "Dashboard V2" sider.
- **DoD**: click menu navigate được; URL deep-link `?tab=streaming` mở đúng tab.

### T-FE-05 — Skeleton container
- [ ] Tạo `src/pages/DashboardV2.tsx` chỉ với `<Tabs>` + 3 `<TabPane>` placeholder text.
- **DoD**: 3 tab clickable; URL state work.

### T-FE-06 — Env config
- [ ] Thêm `VITE_SIGNOZ_BASE_URL=http://localhost:3301` vào `.env.example`.
- [ ] Tạo helper `src/utils/signoz.ts` với `buildSignozUrl(trace)`.
- **DoD**: unit test 2 case (with trace / without trace).

---

## Phase F2 — Tab 1 Snapshot (~4h)

### T-FE-07 — Stat row
- [ ] 4 `<Statistic>` AntD cho active slots / pending / throughput / avg ETA.
- **DoD**: render mock data đúng.

### T-FE-08 — Active table
- [ ] AntD `<Table>` với cột: ID (rút gọn + tooltip), Table, Progress bar, Throughput, ETA, Actions.
- **DoD**: render 3 mock row; progress bar animate đúng.

### T-FE-09 — Pending queue
- [ ] Collapse panel + table tương tự.
- **DoD**: 0 pending → Empty state.

### T-FE-10 — Prioritize action + confirm
- [ ] Button + `Modal.confirm` + mutation `usePrioritizeSnapshot`.
- [ ] On success: invalidate query.
- **DoD**: click → modal → confirm → POST → list refetch.

### T-FE-11 — View Trace button (per row)
- [ ] Nếu snapshot có `trace_id` → button mở SigNoz URL (defer if BE chưa expose).
- **DoD**: button enable đúng khi có trace_id.

---

## Phase F3 — Tab 2 Streaming (~5h)

### T-FE-12 — `TtcWidget.tsx`
- [ ] Component + CSS blink (Section 5 impl).
- [ ] Unit test `ttc.test.ts` (Section 4 unit test).
- **DoD**: 4 state render đúng màu; `prefers-reduced-motion` tắt animation.

### T-FE-13 — `StreamExpiryPair.tsx`
- [ ] 2 column statistic với divider, tag color theo threshold.
- **DoD**: render mock data 3 case (ok / yellow / red).

### T-FE-14 — `UnifiedCrosshairChart.tsx`
- [ ] 3 Recharts `<LineChart>` stacked.
- [ ] PoC `syncId` — nếu hover sync OK → ship; nếu KHÔNG → fallback state lift (xem ADR-007).
- **DoD**: hover chart 1 → 3 cursor sync visible; manual smoke trên Chrome + Firefox.

### T-FE-15 — Reconciliation card
- [ ] Reuse `health.reconciliation` từ `useSystemHealth`.
- [ ] Render table per-table với drift count + recon_last_success_ts tag color.
- **DoD**: section render khi health endpoint có data.

---

## Phase F4 — Tab 3 DLQ + Drift (~3h)

### T-FE-16 — List cards
- [ ] AntD `<List>` cho DLQ + Drift (Section 9 impl).
- **DoD**: render 5 mock row mỗi card; click → mở modal (DLQ) hoặc deep-link (drift).

### T-FE-17 — `PayloadViewerModal.tsx`
- [ ] Modal width 780, JSON viewer trong `<pre>` (HTML-escape vì security).
- [ ] Button "View trace in SigNoz" → mở tab mới với URL từ `buildSignozUrl`.
- **DoD**: 1 DLQ row click → modal có payload + button enable nếu trace_id; click button → tab mới mở đúng URL.

### T-FE-18 — Drift deep-link
- [ ] Click drift row → navigate `/schema-proposals?field=<name>` (page hiện có).
- **DoD**: navigate work; query param parse đúng ở SchemaProposals page (defer enhance ở SchemaProposals).

---

## Phase F5 — Polish + a11y (~1h)

### T-FE-19 — Empty + Skeleton states
- [ ] Toàn bộ tab: wrap với `<QueryErrorBoundary>` + Skeleton khi `isLoading` + Empty khi data empty.
- **DoD**: tắt backend → empty render, không crash; loading → skeleton.

### T-FE-20 — Reduced motion
- [ ] CSS `prefers-reduced-motion` cho `.ttc-blink`.
- **DoD**: OS Reduce Motion bật → animation off (outline đỏ thay thế).

### T-FE-21 — I18n (VN-first)
- [ ] Strings VN: "Không có snapshot đang chạy", "Đang tải metric…", "Không thể bắt kịp", "Pipeline đang lag".
- **DoD**: visual smoke pass; không có text EN fallback ở Tab 1 + Tab 2.

### T-FE-22 — Unit + render test
- [ ] `ttc.test.ts` (4 case).
- [ ] `TtcWidget.test.tsx` (4 state render snapshot).
- [ ] `useDashboard.test.ts` (MSW mock cho 5 hook).
- **DoD**: `npm run test` PASS, coverage > 70% cho `utils/ttc.ts` + `components/dashboard/*`.

---

## Cross-cutting / final

### T-FE-99 — Verify gate (theo §3)
- [ ] `npm run typecheck` PASS
- [ ] `npm run lint` PASS
- [ ] `npm run build` PASS (bundle size delta < 100KB gz)
- [ ] `npm run test` PASS
- [ ] Manual 4 scenario:
  - Mở Dashboard V2 → 3 tab navigable.
  - Kick snapshot từ BE → Tab 1 update < 10s.
  - Tạo backpressure (chậm DB) → Tab 2 TTC chuyển vàng/đỏ.
  - Inject DLQ row → Tab 3 list update + payload modal mở + trace link work.
- [ ] Lighthouse a11y > 90.
- [ ] `/security-agent` review (XSS từ JSON viewer + URL injection ở SigNoz link).
- [ ] Append `05_progress.md`.
