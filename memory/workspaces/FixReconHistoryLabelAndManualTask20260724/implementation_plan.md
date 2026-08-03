# Kế hoạch Cập nhật 5 Tabs Nhật ký Pipeline & Tiến trình đối soát thủ công (JobWorker)

Cập nhật card **"Nhật ký đối soát (30 phiên gần nhất)"** trong giao diện pipeline thành hệ thống 5 Tabs đầy đủ, đồng thời phân tích rõ luồng kiến trúc song song giữa REST API và NATS RPC.

## User Review Required

> [!IMPORTANT]
> - **Cấu trúc 5 Tabs Giao diện (`ReconPipelineGrid.tsx`):**
>   - **Tab 1 ("Đối soát tự động"):** Lịch sử 30 phiên Smoke Check tự động (đổi từ nhãn cũ `Smoke`).
>   - **Tab 2 ("Đối soát thủ công"):** Lịch sử 30 phiên đối soát thủ công hash_window / full_diff / deep_check (đổi từ nhãn cũ `Recon`).
>   - **Tab 3 ("Tiến trình đối soát thủ công"):** Tiến trình `JobWorker` đang đối soát ngầm (`RUNNING` / `PENDING`), hiển thị % progress, mốc checkpoint, số chênh lệch tạm tính.
>   - **Tab 4 ("Log Transmute"):** Nhật ký hoạt động Transmute worker (`/api/activity-log?operation=transmute`).
>   - **Tab 5 ("Log Kafka Consumer"):** Nhật ký hoạt động Ingest/Kafka Consumer worker (`/api/activity-log?operation=kafka-consumer`).
>
> - **Giải thích Kiến trúc: Sự kết hợp giữa REST API và NATS RPC:**
>   - **Phần 1 — REST API (`GET /api/reconciliation/jobs/active`):** Dùng cho **Frontend (`cdc-cms-web`)** kết nối trực tiếp với backend `cdc-cms-service` qua HTTP REST để lấy danh sách tất cả các JobWorker đang thực thi (`status IN ('PENDING', 'RUNNING')`) hiển thị lên **Tab 3**.
>   - **Phần 2 — NATS RPC (`cdc.cmd.recon-job-status`):** Dùng cho **Inter-service (Giữa các Microservices với nhau)**. Khi `cdc-cms-service` hoặc service khác cần hỏi nhanh trạng thái 1 `job_id` cụ thể qua NATS bus (không qua HTTP), nó gửi tin nhắn NATS request đến topic này và `centralized-data-service` trả lời trực tiếp.
>   - **Kết luận:** Hai cơ chế bổ trợ cho nhau: REST API phục vụ UI Web rendering; NATS RPC phục vụ giao tiếp ngầm giữa 2 dịch vụ Go backend.

## Architecture Flow

```
[ cdc-cms-web (React FE) ]
      │
      ├─ (1) HTTP GET /api/reconciliation/jobs/active ─────────┐
      ├─ (2) HTTP GET /api/activity-log?operation=transmute ───┤
      └─ (3) HTTP GET /api/activity-log?operation=kafka-consumer
                                                                ▼
                                                [ cdc-cms-service (CMS API) ]
                                                                │
                                    ┌───────────────────────────┴───────────────────────────┐
                                    ▼                                                       ▼
                    [ DB: cdc_system.recon_jobs ]                            [ NATS Bus ]
                                    ▲                                                       │
                                    │ (Updates progress)         cdc.cmd.recon-job-status   │
                                    │                                                       ▼
                                    └──────────────────────────── [ centralized-data-service ]
                                                                   (Recon Engine & JobWorker)
```

## Proposed Changes

---

### UI Frontend Component (`cdc-cms-web`)

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Cập nhật Card *"Nhật ký đối soát (30 phiên gần nhất)"* render 5 Tabs:
  - **Tab 1 (Key `'smoke'`):** Label `'Đối soát tự động'` $\rightarrow$ `HistoryTable` (smoke data).
  - **Tab 2 (Key `'recon'`):** Label `'Đối soát thủ công'` $\rightarrow$ `HistoryTable` (recon data).
  - **Tab 3 (Key `'active_jobs'`):** Label `'Tiến trình đối soát thủ công'` $\rightarrow$ Render tiến độ % Progress bar, status, `checkpoint_ts`, `total_diff_count` của JobWorker đang chạy ngầm từ API `/api/reconciliation/jobs/active`.
  - **Tab 4 (Key `'log_transmute'`):** Label `'Log Transmute'` $\rightarrow$ Render mini table logs từ `/api/activity-log?operation=transmute`.
  - **Tab 5 (Key `'log_kafka_consumer'`):** Label `'Log Kafka Consumer'` $\rightarrow$ Render mini table logs từ `/api/activity-log?operation=kafka-consumer`.

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Bổ sung hook `useActiveReconJobs(table)` (poll `GET /api/reconciliation/jobs/active` mỗi 3s).
- Bổ sung hook `usePipelineActivityLog(table, operation)` (fetch `GET /api/activity-log`).

---

### Backend API Services (`centralized-data-service` & `cdc-cms-service`)

#### [MODIFY] [recon_job_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon_job_repo.go)
- Bổ sung `GetActiveJobs(ctx context.Context, targetTable string) ([]ReconJob, error)` để truy vấn các job `PENDING` / `RUNNING` từ DB `cdc_system.recon_jobs`.

#### [MODIFY] [recon_job_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_job_handler.go)
- Bổ sung handler `HandleGetActiveJobs` phục vụ REST endpoint `GET /api/reconciliation/jobs/active`.
- Giữ nguyên handler `HandleGetJobStatusNATS` phục vụ NATS Subject `cdc.cmd.recon-job-status`.

---

## Verification Plan

### Automated Tests
- Chạy `go test ./internal/handler/recon/...` verify backend job handlers (cả REST và NATS).
- Chạy `npm run build` trong `cdc-cms-web` kiểm tra TypeScript / Syntax UI build.
- Chạy `python3 agent/tooling/verify_governance.py` verify linter quy trình.

### Manual Verification
- Mở CMS dashboard, kiểm tra card *"Nhật ký đối soát (30 phiên gần nhất)"* hiển thị đúng 5 Tabs.
- Kiểm tra dữ liệu từng Tab:
  1. Tab 1: Phiên Smoke tự động.
  2. Tab 2: Phiên Recon thủ công.
  3. Tab 3: Tiến trình JobWorker đang chạy realtime.
  4. Tab 4: Logs Transmute worker.
  5. Tab 5: Logs Kafka Consumer / Ingest worker.
