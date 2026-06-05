# Implementation Plan: Audit Observability Dashboard (Kế hoạch Triển khai: Bảng Điều khiển Giám sát)

*(Tiếng Việt)*
Theo đúng yêu cầu của bạn, trang `/audit` (`AdminAudit.tsx`) hiện tại quá "sáo rỗng" (chỉ hiển thị test gaps tĩnh) và sai mục đích của một công cụ Audit Pipeline thực chiến. Tôi sẽ đập bỏ giao diện cũ và xây dựng lại thành một **Real-time Observability Dashboard** (bảng điều khiển giám sát thời gian thực) đáp ứng trọn vẹn 5 nhóm kiểm thử mà bạn yêu cầu, sử dụng dữ liệu thực tế từ Prometheus và API hiện có.

*(English)*
Per your exact request, the current `/audit` page (`AdminAudit.tsx`) is too "hollow" (only showing static test gaps) and defeats the purpose of a practical Pipeline Audit tool. I will tear down the old UI and rebuild it into a **Real-time Observability Dashboard** that fully satisfies the 5 testing categories you requested, utilizing actual data from Prometheus and existing APIs.

## User Review Required (Cần User Phê duyệt)

> [!IMPORTANT]
> **Phương pháp vẽ Chart 30s (30s Chart Method)**: 
> *Việt*: Thay vì sửa core backend để hỗ trợ PromQL `query_range` (rủi ro ảnh hưởng performance), frontend React sẽ **poll API `/api/v1/audit/metric-health` mỗi 2 giây** và tích lũy dữ liệu vào một mảng local state (rolling window 30 điểm). Cách này giúp vẽ Live Chart (Lag, TPS, DLQ) mượt mà, đúng yêu cầu "30s qua chart nó như nào" mà không chạm vào core backend. Bạn có đồng ý với phương án Frontend-polling này không?
> *En*: Instead of modifying the core backend to support PromQL `query_range` (risking performance impact), the React frontend will **poll the `/api/v1/audit/metric-health` API every 2 seconds** and accumulate the data into a local state array (a rolling window of 30 points). This allows drawing smooth Live Charts (Lag, TPS, DLQ), perfectly meeting your "how is the chart for the past 30s" request without touching the core backend. Do you agree with this Frontend-polling approach?
>
> **Thêm Metrics Mới (Add New Metrics)**: 
> *Việt*: Backend `get_metric_health.go` hiện chỉ có Lag, Latency, DLQ, Recon. Tôi sẽ thêm các metric:
> - `throughput_tps`: `sum(rate(cdc_batches_flushed_total[1m]))` hoặc tương đương
> - `cpu_usage` & `memory_usage`: Lấy từ `process_cpu_seconds_total` và `go_memstats_alloc_bytes` của worker.
> Bạn xác nhận cho phép sửa file `get_metric_health.go` để lấy thêm số liệu này chứ?
> *En*: The backend `get_metric_health.go` currently only exposes Lag, Latency, DLQ, and Recon. I will add the following metrics:
> - `throughput_tps`: `sum(rate(cdc_batches_flushed_total[1m]))` or equivalent
> - `cpu_usage` & `memory_usage`: Extracted from `process_cpu_seconds_total` and `go_memstats_alloc_bytes` of the worker.
> Do you approve modifying `get_metric_health.go` to extract these additional figures?

## Proposed Changes (Thay đổi Đề xuất)

---

### 1. Nâng cấp Backend Metrics (Upgrade Backend Metrics)
**Mục tiêu (Goal)**: Cung cấp đủ KPI thực tế cho Dashboard (TPS, CPU, Memory).
#### [MODIFY] `cdc-cms-service/internal/app/queries/get_metric_health.go`
- Bổ sung query PromQL (Add PromQL queries for):
  - `throughput_tps` (Throughput/TPS).
  - `cpu_usage` (CPU process).
  - `memory_usage` (Memory bytes).
- Đảm bảo trả về list `MetricHealth` đầy đủ cho frontend (Ensure full `MetricHealth` list is returned to frontend).

---

### 2. Xây dựng Real-time Audit Dashboard (Build Real-time Audit Dashboard)
**Mục tiêu (Goal)**: Thay thế giao diện rỗng bằng UI giám sát động, bám sát 5 kịch bản test (Replace the hollow interface with a dynamic monitoring UI, closely following the 5 test scenarios).

#### [MODIFY] `cdc-cms-web/src/pages/AdminAudit.tsx`
- **Xóa bỏ (Remove)**: Các bảng/chart tĩnh về QA Gaps cũ (Old static tables/charts about QA Gaps).
- **Thêm Hook Local Accumulation (Add Local Accumulation Hook)**: Sử dụng `useEffect` với `setInterval(2000ms)` để fetch `/api/v1/audit/metric-health` và push vào một mảng `timeSeriesData` giới hạn 30 phần tử (tương đương 60s qua) (Use `useEffect` with `setInterval(2000ms)` to fetch data and push to a 30-item array for rolling 60s history).
- **Layout 5 Nhóm (5-Category Layout)**:
  1. **Functional & Correctness (Data Integrity)**:
     - Component `Reconciliation Panel`: Call API `/api/reconciliation/report` hiển thị bảng (display table): Source Count vs Dest Count vs Drift %.
     - Trạng thái Schema Drift (Schema Drift status).
  2. **Stability & Resilience**:
     - Thống kê Dead Letter Queue (DLQ Rate) realtime bằng `Gauge` chart.
     - Worker Status (Up/Down) & Failover events.
  3. **Performance & Scalability**:
     - `Live LineChart`: Replication Lag (events) theo thời gian thực (rolling 30s) (Real-time rolling 30s).
     - `Live AreaChart`: Throughput (TPS) - Số lượng bản ghi được xử lý mỗi giây (Records processed per second).
  4. **Kiểm thử Tài nguyên (Resource Limits)**:
     - `Live LineChart`: Memory Usage (MB) để theo dõi Memory Leak trong suốt quá trình Soak test.
     - `Gauge`: CPU Usage.
  5. **Snapshot & Downstream Progress**:
     - Bảng `Activity Stats` (gọi từ `/api/activity-log/stats`): Hiển thị tiến trình Insert/Update/Delete đang nhận và flush xuống downstream, tỷ lệ thành công/thất bại (Display receiving and downstream flush progress, success/fail rates).

#### [NEW] `cdc-cms-web/src/hooks/useLiveMetrics.ts`
- Hook chuyên dụng để fetch và tích lũy chuỗi thời gian (time-series) cho các metrics từ `/api/v1/audit/metric-health` (Dedicated hook to fetch and accumulate time-series metrics).

## Verification Plan (Kế hoạch Kiểm định)

### Manual Verification (Kiểm tra Thủ công)
1. **Load Dashboard**: Mở http://localhost:5173/audit, xác nhận giao diện mới đã thay thế hoàn toàn giao diện cũ (Confirm the new UI fully replaced the old one).
2. **Live Charts**: Quan sát các biểu đồ Lag, TPS, Memory liên tục dịch chuyển (cập nhật mỗi 2 giây) (Observe Lag, TPS, Memory charts moving continuously every 2 seconds).
3. **Data Integrity**: Bảng Đối soát (Reconciliation) hiển thị số liệu thực tế số bản ghi Source vs Dest (Reconciliation table shows actual Source vs Dest record counts).
4. **Stress Test**: Dùng script sinh data bắn vào Kafka, theo dõi TPS Chart vọt lên và Lag Chart tăng/giảm theo đúng thực tế, chứng minh tính "Real-time" của hệ thống (Use data generation script to push to Kafka, monitor TPS and Lag charts responding accurately in real-time).
