# 01_requirements.md — Transmute Job Realtime Tracking & Progress Bar (50M-500M Records)

## Yêu cầu Bắt buộc (Boundary Isolation):
1. Transmute Oplog / CDC Realtime bình thường: KHÔNG sinh job_id, KHÔNG ghi job record, KHÔNG ảnh hưởng hiệu năng (15-50ms).
2. Khi bấm nút "Transmute Now" trên CMS UI: CMS sinh job_id, gửi NATS payload kèm job_id, kích hoạt Live Tracking & Progress Bar trên UI.
3. DDL: cdc_system.transmute_jobs lưu vết async job transmute (job_id, master_table, status, rows_affected, cancel_requested, ...).
4. Worker Engine (centralized-data-service): TransmuterModule.Run chỉ heartbeat rows_affected & check cancel_requested khi job_id != "".
5. CMS Service (cdc-cms-service): Endpoints /transmute-job-status & /transmute-cancel.
6. CMS Web UI (cdc-cms-web): Render TransmuteJobStatus component trong MasterRegistry.tsx khi có active job_id.
