# 06_test_cases_sftp_internal_worker.md — Kế hoạch Kiểm thử

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## Unit Tests (sftp_worker_test.go)

| TC-ID | Tên Test | Mô tả | Expected |
|:---|:---|:---|:---|
| UT-01 | `TestParseCSVRows_HappyPath` | Đọc CSV 3 rows với header → 3 flat JSON maps | 3 maps đúng key-value |
| UT-02 | `TestParseCSVRows_EmptyFile` | CSV chỉ có header, không có data row | 0 rows, no error |
| UT-03 | `TestParseCSVRows_MissingHeader` | CSV rỗng hoàn toàn | Error hoặc 0 rows |
| UT-04 | `TestFilePatternMatch_Valid` | `reconcile_final_20260811.csv` match pattern `^reconcile_final_.*\.csv$` | true |
| UT-05 | `TestFilePatternMatch_Invalid` | `other_file.txt` không match pattern | false |
| UT-06 | `TestSFTPWorker_Stop` | Gọi Stop() khi worker chưa Start → không panic | No panic |
| UT-07 | `TestBuildKafkaMessage` | Build message từ flat JSON row | Message value đúng format |

---

## Integration Tests (Manual — E2E)

| TC-ID | Bước | Expected |
|:---|:---|:---|
| IT-01 | Start Docker SFTP container (`docker compose up -d`) | Container `sftp-host` Running |
| IT-02 | Start `centralized-data-service` với `sftpWorker.enabled: true` | Log: `SFTP polling worker started` |
| IT-03 | Copy file CSV mới vào `data-hub/docker/data/reconcile_final/` | File xuất hiện trong SFTP `/goopay/reconcile_final/` |
| IT-04 | Đợi 30s (1 poll cycle) | Log: `SFTP poll: found N files`, `pushed M rows to Kafka` |
| IT-05 | Kiểm tra Kafka topic `sftp.reconcile.final` (qua Kafka UI) | N messages xuất hiện |
| IT-06 | Kiểm tra Shadow DB: `SELECT * FROM shadow_reconcile_final` | Rows với data từ CSV |
| IT-07 | Đợi TransmuteScheduler chạy (5m) | Master DB có rows tương ứng |
| IT-08 | Kiểm tra file trong SFTP `processed/` | File CSV đã được move sang `processed/` |
| IT-09 | Upload file bị malformed → kiểm tra `error/` | File CSV lỗi được move sang `error/` |

---

## Edge Cases

| EC-ID | Kịch bản | Expected |
|:---|:---|:---|
| EC-01 | SFTP server tắt giữa chừng | Worker log error, bỏ qua cycle hiện tại, retry ở cycle sau |
| EC-02 | Kafka broker unreachable | Worker log error, không crash, retry ở cycle sau |
| EC-03 | File CSV rỗng (chỉ có header) | Worker log warning, move sang processed/, không push message |
| EC-04 | File CSV có row thiếu column | Parse tiếp các row hợp lệ, log warning cho row lỗi |
| EC-05 | Worker Stop() khi đang giữa pollOnce | Context cancel → dừng gracefully |
