# 01_requirements_recon_history_manual_task.md — Yêu cầu đối soát & Kiến trúc 5 Tabs

## I. TỔNG QUAN YÊU CẦU (BUSINESS REQUIREMENTS)
Cập nhật card "Nhật ký đối soát (30 phiên gần nhất)" tại `ReconPipelineGrid.tsx` thành hệ thống 5 Tabs chuẩn hóa:
1. **Tab 1 ("Đối soát tự động"):** (Key `'smoke'`) Lịch sử 30 phiên Smoke Check tự động.
2. **Tab 2 ("Đối soát thủ công"):** (Key `'recon'`) Lịch sử 30 phiên đối soát thủ công (hash_window / full_diff / deep_check).
3. **Tab 3 ("Tiến trình đối soát thủ công"):** (Key `'active_jobs'`) Tiến trình các `JobWorker` đang đối soát ngầm (`status IN ('PENDING', 'RUNNING')`), hiển thị % progress, mốc checkpoint, khoảng thời gian và số chênh lệch tạm tính.
4. **Tab 4 ("Log Transmute"):** (Key `'log_transmute'`) Nhật ký hoạt động Transmute worker (`/api/activity-log?operation=transmute`).
5. **Tab 5 ("Log Kafka Consumer"):** (Key `'log_kafka_consumer'`) Nhật ký hoạt động Ingest/Kafka Consumer worker (`/api/activity-log?operation=kafka-consumer`).

## II. PHÂN TÍCH KIẾN TRÚC & MỐI QUAN HỆ BE/FE

```
[ cdc-cms-web (React FE) ]
      │
      ├─ HTTP GET /api/reconciliation/jobs/active ──────────────┐
      ├─ HTTP GET /api/activity-log?operation=transmute ────────┤
      └─ HTTP GET /api/activity-log?operation=kafka-consumer ───┤
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

1. **Vì sao cần `GET /api/reconciliation/jobs/active` (`GetActiveJobs`)?**
   - Khi giao diện CMS Web (`cdc-cms-web`) mở Tab 3 **"Tiến trình đối soát thủ công"**, Web gọi HTTP REST `GET /api/reconciliation/jobs/active` tới `cdc-cms-service`.
   - `cdc-cms-service` truy vấn bảng `cdc_system.recon_jobs` (lọc `status IN ('PENDING', 'RUNNING')`) và trả danh sách các JobWorker đang thực thi cho FE hiển thị thanh Progress bar %.

2. **NATS RPC Topic `cdc.cmd.recon-job-status` đóng vai trò gì?**
   - NATS RPC Topic `cdc.cmd.recon-job-status` là kênh giao tiếp giữa các **Microservices với nhau** qua NATS bus.
   - Khi `cdc-cms-service` (hoặc service khác) muốn tra cứu thông tin 1 job cụ thể theo `job_id` thông qua NATS (không dùng REST HTTP), nó gửi tin nhắn NATS request đến topic này và `centralized-data-service` sẽ phản hồi kết quả qua NATS bus.
   - **Tóm lại:** REST API `GET /api/reconciliation/jobs/active` dùng cho **Frontend (Web UI)** poll danh sách job active. NATS Topic `cdc.cmd.recon-job-status` dùng cho **Inter-service RPC (Service to Service)** qua NATS bus.
