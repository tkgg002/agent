# 08_tasks_phase_ui — Checklist Muscle Phase UI Admin Audit

> Reference: `03_implementation_phase_ui.md`. Parallel với P1 OK.

## Backend (`cdc-cms-service`) — 8h

### Migration
- [ ] Tạo NEW `cdc-cms-service/migrations/0060_qa_gap_state.sql`:
  - Bảng `cdc_system.qa_gap_state(id text PK, code text, title text, priority text, status text, score_impact int, criterion text, created_at timestamptz, updated_at timestamptz)`.
  - Bảng `cdc_system.qa_criterion_rating(code text PK, group_name text, criterion text, rating text, weight int, updated_at timestamptz)`.
  - INSERT 16 gap (G-1..G-16) với priority P0/P1/P2 + status='open'.
  - INSERT 16 criterion với rating L0..L4.
- [ ] Verify: `psql -c "SELECT COUNT(*) FROM cdc_system.qa_gap_state"` returns 16.

### DTO + Query + Handler
- [ ] Tạo NEW `internal/api/dto/audit_dto.go`: structs `QASummaryResponse`, `GapCount`, `CriterionRow`, `GapRow`, `MetricHealthResponse`, `MetricHealth`.
- [ ] Tạo NEW `internal/app/queries/get_qa_summary.go`: query rating + gap counts → compute composite (L0=0..L4=4, sum × weight / total).
- [ ] Tạo NEW `internal/app/queries/list_gaps.go`: filter `priority`, `status`, pagination.
- [ ] Tạo NEW `internal/app/queries/get_metric_health.go`: gọi Prom client query 4 metric (consumer_lag, e2e_latency_p99, dlq_rate, recon_drift) → map status (healthy/warning/critical).
- [ ] Tạo NEW `internal/infra/persistence/gap_state_repo_gorm.go`: implement repo interface đọc 2 bảng.
- [ ] Tạo NEW `internal/api/audit_handler.go`: 3 method `Summary`, `Gaps`, `MetricHealth`.

### Router + DI
- [ ] Sửa `internal/router/router.go`: dưới `adminGroup` (đã có `RequireRole("admin")`) thêm:
  - `dualGet(adminGroup, "/audit/qa-summary", h.Audit.Summary)`
  - `dualGet(adminGroup, "/audit/gaps", h.Audit.Gaps)`
  - `dualGet(adminGroup, "/audit/metric-health", h.Audit.MetricHealth)`
- [ ] Sửa `internal/server/server.go` inject `AuditHandler` vào handler struct.

### Test
- [ ] Tạo NEW `internal/api/audit_handler_test.go` (fiber + httptest):
  - Test composite score calc.
  - Test filter gaps.
  - Test metric health status mapping.
- [ ] Verify: `go test ./internal/api -run TestAuditHandler -v` PASS.

## Frontend (`cdc-cms-web`) — 8h

### Types + Hook
- [ ] Tạo NEW `src/types/audit.ts`: types `Rating` (L0..L4), `Priority` (P0/P1/P2), `GapStatus`, `MetricStatus`, `QASummary`, `Gap`, `MetricHealth`.
- [ ] Tạo NEW `src/hooks/useAuditStatus.ts`:
  - `useQASummary()` React Query, key `['audit-summary']`, `refetchInterval: 30000`.
  - `useGaps(filter)` key `['audit-gaps', filter]`, `refetchInterval: 30000`.
  - `useMetricHealth()` key `['audit-metric-health']`, `refetchInterval: 15000`.

### Components
- [ ] Tạo NEW `src/components/audit/RatingMatrix.tsx`: Ant Design Table với column [Criterion, Group, Rating, Weight], cell Rating render Tag 5 màu L0=red..L4=green.
- [ ] Tạo NEW `src/components/audit/GapList.tsx`: Tabs P0/P1/P2 + Table mỗi tab với [Code, Title, Status, Score Impact].
- [ ] Tạo NEW `src/components/audit/MetricHealthCards.tsx`: Row/Col 4 Card với Statistic + color theo status.

### Page + Route
- [ ] Tạo NEW `src/pages/AuditPage.tsx`: 4 section
  1. Composite Score Statistic (big number).
  2. Metric Health Cards.
  3. Rating Matrix.
  4. Gap List.
- [ ] Sửa `src/App.tsx`:
  - `const AuditPage = lazy(() => import('./pages/AuditPage'))`.
  - Route `<Route path="/audit" element={<AuditPage />} />` trong protected layout.
  - Menu item nhóm "Operate" với icon + label "Audit".

### Verify FE
- [ ] `pnpm lint` PASS.
- [ ] `pnpm typecheck` PASS.
- [ ] `pnpm build` PASS.
- [ ] Browser navigate `/audit`, observe 3 API call 200 + render đủ 4 section.
- [ ] Network tab: qa-summary refetch 30s, metric-health 15s.

## Post-phase
- [ ] /security-agent scan PASS.
- [ ] APPEND `05_progress.md` "UI phase executed".
- [ ] Composite score unchanged (visibility-only), 87.5% retain.
