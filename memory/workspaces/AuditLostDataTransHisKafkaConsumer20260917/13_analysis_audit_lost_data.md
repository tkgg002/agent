# 13 Phân Tích Chuyên Sâu: Nguyên Nhân Mất Dữ Liệu trans-his và Kafka Consumer Không Chạy

## 1. Vấn Đề 1: Mất Dữ Liệu Source (trans-his) -> Shadow (shadow_traitestctphs.trans_his) [10:07:00 -> 10:22:00]

### A. Chuỗi Sự Kiện (Execution Chain)
1. **Debezium Source**:
   - Connector `traitestctphs` cấu hình `"topic.prefix": "cdc.goopaylocal"`.
   - Message đẩy vào topic Kafka: `cdc.goopaylocal.core-trans-proxy-history-service.trans-his`.
   - Payload Debezium có `source.name = "cdc.goopaylocal"`.
2. **Kafka Consumer (`kafka_consumer.go:L671-L687`)**:
   - Cắt tiền tố:
     ```go
     sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.goopay.") // KHÔNG khớp vì sau goopay là chữ l (goopaylocal)
     sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")         // Thành "goopaylocal"
     ```
   - Kiểm tra: `source.IsGenericConnectionCode("goopaylocal")`
     Trong `metadata_registry_utils.go:L300-L306`:
     ```go
     switch strings.ToLower(strings.TrimSpace(code)) {
     case "goopay", "gpay", "mongodb", "postgres", "postgresql", "mysql", "mariadb", "sftp":
         return true
     }
     ```
     `"goopaylocal"` trả về `false`!
   - Không được reset về rỗng `""`. Biến `sourceConnCode` bị gán cứng là `"goopaylocal"`.
3. **ResolveSourceRoutes (`metadata_registry_service.go:L553-L596`)**:
   - Nhận `sourceConn = "goopaylocal"`.
   - Vì `conn != ""` và không phải generic, hàm thực hiện lọc:
     `rConn == conn || rConn == cleanConn` -> tìm route có connection key là `"goopaylocal"`.
   - Trong Database `source_object_registry`:
     `id = 181`: `connection_code = "traitestctphs"`, `source_database = "core-trans-proxy-history-service"`, `source_object_name = "trans-his"`.
     Không có bất kỳ route nào mang connection key `"goopaylocal"`.
   - `len(filtered) == 0` -> Hàm trả về `nil` (Dòng 595).
4. **EventHandler Drop & Offset Commit (`event_handler.go:L273-L289`)**:
   - Thấy `routes == nil`, EventHandler ghi log:
     `"event skipped: source not in registry cache — check is_active flags or registry reload"`
   - Tăng metric:
     `metrics.EventsDropped.WithLabelValues("source_not_registered", "trans-his").Inc()`
   - Trả về `return 0, nil`.
   - Kafka Consumer nhận `err == nil` nên tiến hành **COMMIT OFFSET** lên Kafka broker!
   - Kết quả: 490/490 bản ghi của `trans-his` bị drop sạch sẽ mà không ghi bất kỳ dòng nào vào `shadow_traitestctphs.trans_his`!

### B. Bằng Chứng Thực Tế (Prometheus Metrics Live)
```text
cdc_events_processed_total{operation="kafka",table="cdc.goopaylocal.core-trans-proxy-history-service.trans-his"} 490
cdc_sink_events_dropped_total{reason="source_not_registered",topic="trans-his"} 490
cdc_sink_events_dropped_total{reason="source_not_registered",topic="payment-bills"} 175
cdc_recon_smoke_drift_count{segment="source_shadow",table="shadow_traitestctphs.trans_his"} 212
```
Số lượng processed = Số lượng dropped (490 = 490). Không có 1 bản ghi nào lọt qua!

---

## 2. Vấn Đề 2: Vì Sao Từ Hôm Qua Tới Nay Không Có Bất Cứ 1 Kafka-Consume Nào Chạy?

1. **Kafka Consumer Lag = 0**:
   - Toàn bộ 490 messages đã bị đọc hết từ đợt trước và đã bị commit offset.
   - `cdc_kafka_consumer_lag{partition="-1",topic=""} = 0`.
   - Trong topic Kafka `cdc.goopaylocal.core-trans-proxy-history-service.trans-his` hiện tại không còn message nào chưa đọc.
2. **Debezium Chạy Chế Độ `"snapshot.mode": "no_data"`**:
   - Cấu hình connector `traitestctphs`:
     `"snapshot.mode": "no_data"`
   - Debezium chỉ bắt các lệnh INSERT/UPDATE/DELETE mới phát sinh trên MongoDB replica set oplog kể từ lúc start.
   - Nếu từ hôm qua tới nay MongoDB không có phát sinh giao dịch mới nào vào collection `trans-his`, Debezium hoàn toàn KHÔNG đẩy thêm bất kỳ message nào vào Kafka.
3. **Cơ Chế Polling Im Lặng Của Worker (`kafka_consumer.go:L308-L313`)**:
   - Goroutine tiêu thụ chạy lệnh:
     `msg, err := currentReader.FetchMessage(fetchCtx)` (timeout 200ms).
   - Khi topic không có message mới, `FetchMessage` trả về `context.DeadlineExceeded`.
   - Code bắt lỗi này và `continue` ngay lập tức:
     ```go
     if errors.Is(err, context.DeadlineExceeded) {
         continue
     }
     ```
   - Span OTel `kafka.consume: <topic>` (dòng 392) chỉ được khởi tạo khi fetch thành công 1 message.
   - Do đó, khi topic không có message mới, worker vẫn đang chạy poll bình thường nhưng không sinh bất cứ log hay span `kafka.consume` nào!

---

## 3. Vấn Đề 3: Vì Sao Không Có Chạy Lại (Auto-Heal) Hay Thông Báo Gì?

1. **Recon Engine Là Passive Detection**:
   - Trong `ReconJobWorker.ProcessJob` (`recon_job_worker.go:L280-L345`):
     Khi phát hiện drift giữa Source và Shadow, nó drill-down lấy danh sách `stale_ids`, lưu vào DB `cdc_reconciliation_report` và `recon_jobs`, rồi cập nhật status = `COMPLETED`.
   - Hoàn toàn KHÔNG có logic tự động gửi command NATS `cdc.cmd.recon-heal`.
   - Chức năng Heal hiện tại hoàn toàn phụ thuộc vào việc Admin vào CMS UI bấm nút thủ công.
2. **Thiếu Tích Hợp Kênh Cảnh Báo (Alert Notification)**:
   - Khi `cdc_sink_events_dropped_total` tăng hoặc khi `cdc_recon_smoke_drift_count > 0`, hệ thống chỉ xuất metric Prometheus và log Warn cục bộ.
   - Chưa có module webhook / bot gửi alert sang Slack, Telegram hoặc Email.
