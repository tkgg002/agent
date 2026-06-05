# 09_tasks_solution_phase_ui — Hồ sơ giải pháp UI Admin Audit

## Yêu cầu (từ user)
> "trên cms-fe có nơi để admin audit."

## Solution overview
4 section trong 1 page `/audit` (admin-only):
1. **Composite Score**: big number Statistic (56/64 hoặc real-time).
2. **Metric Health Cards**: 4 card (consumer_lag, e2e_p99, dlq_rate, recon_drift) với status color.
3. **Rating Matrix**: 16 criterion × rating L0..L4.
4. **Gap List**: Tabs P0/P1/P2 với status open/in_progress/closed.

## Design choices

### Backend
- **Lưu DB thay vì YAML** (ADR-003): real-time UPDATE state khi Muscle resolve gap.
- **3 endpoint riêng** thay vì 1 endpoint /audit/all:
  - Refetch interval khác nhau (summary 30s, metric-health 15s).
  - Permission granular (sau này tách operator-read-only nếu cần).
- **Query layer thuần Go** (không qua repository ORM heavy): SELECT đơn giản, không cần CQRS deep cho read-only.
- **Prometheus Client gọi từ Go**: query 4 metric → map status (healthy < threshold, warning 1-2x, critical > 2x).

### Frontend
- **React Query polling** (ADR-004): đủ cho admin page, không cần SSE.
- **Lazy import**: AuditPage không load nếu user không vào.
- **Ant Design pure**: align cdc-cms-web hiện có, không thêm dep mới.
- **Tag color L0=red..L4=green**: align convention rating đỏ→xanh.

### Test
- **Backend**: fiber test trên 3 endpoint với mock repo + mock Prom client.
- **Frontend**: skip unit (theo pattern hiện có không có Jest setup), verify bằng browser manual.

## Permission model (ADR-005)
- Route mount `/api/v1/admin/audit/*` dưới `adminGroup` với `RequireRole("admin")`.
- FE menu item ẩn nếu user không có admin role (check `useUser().roles`).

## Tổng impact UI
- Score: 0 (visibility-only, không vá criterion).
- Value: Operator có 1 nơi single-source-of-truth về QA state, không phải đọc Grafana + memory file riêng lẻ.
- Side benefit: Khi gap closed (UPDATE qa_gap_state SET status='closed'), composite score recompute tự động trên UI.
