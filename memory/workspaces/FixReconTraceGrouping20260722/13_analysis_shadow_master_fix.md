# 13 Analysis: Audit Luồng Recon Shadow-Master

## Phân Tích Gốc Rễ Từ Jaeger Trace Log:
Khi gửi `POST /api/reconciliation/check`:
Trace log thực tế:
- `ReconJobWorker.ProcessJob`
- `cdc.recon.chunk_stream_bucket: payment_bills`
- `recon.source.hash_window: payment-bills` (MongoDB Source)
- `pg.hash_window: shadow_testpbs.payment_bills` (Shadow PG)
- `pg.update_recon_job_status: COMPLETED`

**Phát hiện:**
1. Không hề có span nào cho Master DB (`master_payment_bill_service.payment_bills`).
2. `ReconJobCreatedEvent` thiếu field `Segment`.
3. `ChunkStreamBucketEngine` thiếu `masterAgent` và chỉ chạy duy nhất Segment A (`source_shadow`).

**Giải pháp:**
Wire `masterAgent` vào `ChunkStreamBucketEngine`, đóng gói `Segment` vào NATS event, và thực thi đối soát Segment B (`shadow_master`) khi `segment == "shadow_master"` hoặc `segment == "both"`.
