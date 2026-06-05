# 04_decisions — ADRs cho Plan vá Gap CDC QA

## ADR-001: Adaptive metric scrape interval (15s/30s/60s)
- **Context**: Prometheus scrape mặc định 60s → lag detect chậm.
- **Decision**: Job `cdc-worker` scrape 15s (high-priority), job `kafka-exporter` 30s, job exporter khác 60s.
- **Trade-off**: Tăng load Prom server ~3x trên job worker, chấp nhận đổi lấy SLA detect lag < 1 phút.
- **Status**: Accepted.

## ADR-002: DLQ Circuit Breaker threshold — configurable vs hard-coded
- **Context**: Cần ngưỡng spike DLQ trigger pause pipeline.
- **Decision**: Configurable via `worker.dlqCircuitBreakerRPS` (default 100) + `dlqCircuitBreakerBurst` (default 200). Không hard-code.
- **Rationale**: Workload mỗi env khác nhau (prod vs staging). Cho operator tune mà không cần redeploy.
- **Status**: Accepted.

## ADR-003: Audit gap state lưu DB hay file YAML?
- **Context**: 16 gap state cần hiển thị trên UI Admin Audit.
- **Decision**: Lưu DB (`cdc_system.qa_gap_state` + `qa_criterion_rating`) thay vì YAML file.
- **Rationale**:
  - DB cho phép cập nhật state real-time khi muscle resolve gap (UPDATE status).
  - Query filter/sort/pagination tự nhiên qua SQL.
  - YAML phải redeploy mỗi khi thay đổi state.
- **Trade-off**: Thêm 2 table + 1 migration vs simplicity YAML.
- **Status**: Accepted.

## ADR-004: UI refresh interval — 30s polling vs SSE realtime
- **Context**: Audit page hiển thị metric health + gap status.
- **Decision**: React Query `refetchInterval` 30s (qa-summary, gaps) + 15s (metric-health).
- **Rationale**:
  - Audit không phải critical real-time như alerting (đã có Prom Alert).
  - SSE/WebSocket overhead cho 1 page admin-only không xứng.
  - Pattern này align với cdc-cms-web hiện có (không thấy SSE infrastructure).
- **Status**: Accepted.

## ADR-005: Admin-only audit page (RequireRole) vs operator readable
- **Context**: Audit page show internal gap/rating có thể nhạy cảm.
- **Decision**: Mount dưới `adminGroup` (route `/api/v1/admin/audit/*`) với `RequireRole("admin")`.
- **Rationale**:
  - Composite score + gap chưa fix là thông tin nội bộ.
  - Operator chỉ cần xem health metric — đã có Grafana cho mục đích đó.
- **Trade-off**: Operator muốn xem phải request admin role. Chấp nhận.
- **Status**: Accepted.

## ADR-006: Per-source pool semaphore vs separate pool instance
- **Context**: G-13 cần giới hạn concurrent connection per source.
- **Decision**: Semaphore (map[string]chan struct{}) trên shared pool, không tách pool riêng.
- **Rationale**:
  - Tách pool: phức tạp config, tăng connection footprint x N source.
  - Semaphore: cap concurrent acquire, share pool resource hiệu quả.
- **Status**: Accepted.

## ADR-007: goleak vs uber-go/goleak fork — chọn nguyên bản
- **Decision**: `go.uber.org/goleak v1.3.0` (đã có go.sum nếu cần verify).
- **Rationale**: Maintained, có `IgnoreTopFunction` API để skip kafka-go background goroutines.
- **Status**: Accepted.
