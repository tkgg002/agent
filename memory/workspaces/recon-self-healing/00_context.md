# Context: Recon Self-Healing & Upstream Flow Investigation

## Scope
Điều tra và sửa lỗi cơ chế tự chữa lành (Self-Healing) qua Debezium Snapshot đối với các record bị thiếu (ví dụ: ID 41063) từ MongoDB nguồn sang Postgres đích, đảm bảo core system hoạt động ổn định và không làm sai lệch luồng upstream.

## Kiến trúc hệ thống
1. **Source**: MongoDB (`payment-bill-service`).
2. **Upstream Pipeline**: MongoDB -> Debezium Connector -> Kafka (`cdc.goopaylocal.payment-bill-service.payment-bills`) -> CDC Worker (SinkWorker) -> Postgres (`cdc_shadow`, table `shadow_test.payment_bills`).
3. **Recon Engine**: Quét so khớp định kỳ giữa MongoDB và Postgres Shadow. Khi phát hiện missing record, gửi tín hiệu snapshot (execute-snapshot) đến Kafka topic `cdc.signal.commands`.
4. **Debezium Signaling**: Debezium lắng nghe topic `cdc.signal.commands` và thực hiện incremental snapshot các record chỉ định dựa trên filter.
