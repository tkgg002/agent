# 02_plan_frontend — Roadmap Frontend Dashboard V2

> **Track**: `cdc-cms-web`
> **Effort tổng**: ~16h Muscle. Chia 4 phase.

---

## Phase F1 — Foundation (service + hook + routing)

**Mục tiêu**: layer dữ liệu hoạt động (mock layer pass), navigation đầy đủ.

**Tasks**:
1. T-FE-01: Tạo `src/types/dashboard.ts` — định nghĩa interface cho 5 response (R-FE-9).
2. T-FE-02: Tạo `src/services/dashboard.ts` — 5 fetcher function.
3. T-FE-03: Tạo `src/hooks/useDashboard.ts` — wrap react-query (R-FE-8).
4. T-FE-04: Thêm route + menu item `/dashboard-v2` (R-FE-10).
5. T-FE-05: Tạo skeleton page `DashboardV2.tsx` (chỉ tab nav + placeholder text).
6. T-FE-06: Env `VITE_SIGNOZ_BASE_URL` (R-FE-11).

**Effort**: ~3h.
**Verify**: navigate đến `/dashboard-v2`, 3 tab placeholder hiện đúng; F5 giữ tab.

---

## Phase F2 — Tab 1 Snapshot Commander

**Tasks**:
1. T-FE-07: Stat row (4 statistic).
2. T-FE-08: Active snapshots table với progress bar.
3. T-FE-09: Pending queue collapse panel.
4. T-FE-10: Prioritize action với confirm modal + mutation.
5. T-FE-11: View Trace button → SigNoz URL.

**Effort**: ~4h.
**Verify**: 1 snapshot kick → table show; prioritize button đổi thứ tự queue ở BE.

---

## Phase F3 — Tab 2 Streaming Real-time (heart of dashboard)

**Tasks**:
1. T-FE-12: `TtcWidget.tsx` — 4 trạng thái + blink animation (R-FE-5).
2. T-FE-13: `StreamExpiryPair.tsx` (R-FE-6).
3. T-FE-14: `UnifiedCrosshairChart.tsx` — synced cursor (R-FE-7).
4. T-FE-15: Reconciliation status card (cuối page).

**Effort**: ~5h.
**Verify**: 
- TTC widget 4 state mock đúng màu.
- Hover chart → 3 cursor sync.
- Banner critical hiển thị khi TTC=red_blink.

---

## Phase F4 — Tab 3 DLQ & Schema Drift

**Tasks**:
1. T-FE-16: List card DLQ + drift.
2. T-FE-17: `PayloadViewerModal.tsx` — JSON viewer + trace link.
3. T-FE-18: Deep-link drift → SchemaProposals page.

**Effort**: ~3h.
**Verify**: 1 DLQ row click → modal có payload + trace link mở SigNoz tab mới.

---

## Phase F5 — Polish + accessibility

**Tasks**:
1. T-FE-19: Empty + Skeleton states (R-FE-12).
2. T-FE-20: `prefers-reduced-motion` cho blink (R-FE-13).
3. T-FE-21: I18n VN-first.
4. T-FE-22: Unit test cho hook + widget logic (TTC formula + crosshair sync).

**Effort**: ~1h.
**Verify**: 
- Tắt BE → empty render OK, không crash.
- OS bật Reduce Motion → animation tắt.
- `npm run test` PASS.

---

## Phụ thuộc & thứ tự thực thi

```
F1 ──> F2 ──┐
        F3 ─┼─> F5
        F4 ─┘
```

F2/F3/F4 có thể parallel nếu chia người. F5 cuối cùng.

---

## Phụ thuộc backend

| Tab | Endpoint cần |
|-----|--------------|
| Tab 1 | R-BE-9, R-BE-10 |
| Tab 2 | R-BE-8 + system health pipeline (đã có) |
| Tab 3 | R-BE-11, R-BE-12 (cần B4 trace_id) |

→ FE có thể dev với MSW mock layer song song, chỉ block ở smoke test cuối.

---

## Verify gate

- [ ] `npm run typecheck` PASS (no `any`)
- [ ] `npm run lint` PASS
- [ ] `npm run build` PASS
- [ ] `npm run test` PASS (theo R-FE-13: hook + widget unit tests)
- [ ] Manual smoke 4 scenario:
  - Mở Dashboard V2 → 3 tab navigable.
  - Kick snapshot → Tab 1 update.
  - Backpressure traffic → Tab 2 TTC chuyển màu.
  - DLQ row inject → Tab 3 list update + payload modal mở.
- [ ] Lighthouse a11y > 90.

---

## Risk mitigation

| Risk | Mitigation |
|------|-----------|
| Recharts synced cursor không có SDK | Lift state lên parent + custom Tooltip — PoC sớm ở T-FE-14 |
| Bundle size tăng do react-json-view | Lazy import `<Suspense>` cho modal |
| Polling 5s × N user → BE load | BE có cache 10s (R-BE-8) — verify chiều ngược lại từ FE qua devtools Network tab |
| AntD v5 breaking change so với code hiện có | Audit imports — đã verify `useSystemHealth.ts` dùng AntD v5 patterns OK |
