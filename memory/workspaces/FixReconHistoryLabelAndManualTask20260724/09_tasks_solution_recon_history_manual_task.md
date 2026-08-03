# 09_tasks_solution_recon_history_manual_task.md — Phương án kỹ thuật 5 Tabs Nhật ký Pipeline

## I. CẤU TRÚC 5 TABS GIAO DIỆN (`ReconPipelineGrid.tsx`)

Trong card `"Nhật ký đối soát (30 phiên gần nhất)"` tại `ReconPipelineGrid.tsx`, bổ sung cấu trúc 5 Tabs:

1. **Tab 1: "Đối soát tự động"** (Key: `'smoke'`):
   - Đổi tên nhãn từ `'Smoke'` thành `'Đối soát tự động'`.
   - Hiển thị danh sách 30 phiên Smoke Check tự động qua `HistoryTable`.

2. **Tab 2: "Đối soát thủ công"** (Key: `'recon'`):
   - Đổi tên nhãn từ `'Recon'` thành `'Đối soát thủ công'`.
   - Hiển thị danh sách 30 phiên đối soát thủ công (hash_window, full_diff, deep_check) qua `HistoryTable`.

3. **Tab 3: "Tiến trình đối soát thủ công"** (Key: `'active_jobs'`):
   - Gọi API `GET /api/reconciliation/jobs/active?table=:tableName`.
   - Hiển thị các `JobWorker` đang chạy (`RUNNING` / `PENDING`) kèm Progress bar %, mốc `checkpoint_ts`, thời gian chạy và số bản ghi chênh lệch tạm tính (`total_diff_count`).
   - Nếu không có job nào đang chạy, hiển thị *"Không có tiến trình đối soát thủ công nào đang thực thi"*.

4. **Tab 4: "Log Transmute"** (Key: `'log_transmute'`):
   - Gọi API `GET /api/activity-log?operation=transmute&target_table=:tableName`.
   - Hiển thị bảng nhật ký hoạt động Transmute worker (thời gian, status, thông điệp chi tiết) của pipeline.

5. **Tab 5: "Log Kafka Consumer"** (Key: `'log_kafka_consumer'`):
   - Gọi API `GET /api/activity-log?operation=kafka-consumer&target_table=:tableName` (hoặc `operation=sink-upsert`).
   - Hiển thị bảng nhật ký hoạt động Ingest/Kafka Consumer worker (thời gian, status, số records ingest) của pipeline.

## II. GIẢI THÍCH NATS RPC TOPIC `cdc.cmd.recon-job-status`
- **Mục đích:** `natsClient.Conn.Subscribe("cdc.cmd.recon-job-status", jobHandler.HandleGetJobStatusNATS)` tạo 1 NATS listener trong `centralized-data-service`.
- **Cơ chế hoạt động:** Cho phép `cdc-cms-service` hoặc các client NATS gửi Request lấy trạng thái `ReconJob` bất đồng bộ theo `job_id` thông qua NATS bus thay vì gọi qua đường HTTP REST.

## III. IMPLEMENTATION STEPS
1. Cập nhật `ReconPipelineGrid.tsx` render 5 Tabs với Ant Design `<Tabs>`.
2. Tạo component `ActivityLogMiniTable` hiển thị log gọn cho Tab 4 và Tab 5.
3. Bổ sung các hooks trong `useReconStatus.ts`.
