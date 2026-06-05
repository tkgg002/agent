# 01_requirements — Fix CDC Testing Residual (2026-06-01)

## Mục tiêu
Đẩy điểm CDC QA Maturity từ **L3 (78.1%)** → **L4 (≥93%)** bằng cách
đóng 8 gap residual mà audit-rerun phát hiện.

## Yêu cầu chức năng
1. **G1-RES** Phát ra metric `cdc_kafka_consumer_offset{topic,partition}`
   trong cùng vòng metricsTicker để failover verify resume position.
2. **G2-RES** WAL monitor tự trigger snapshot resume khi slot `inactive`
   quá ngưỡng hoặc lag vượt threshold; phát metric
   `cdc_wal_snapshot_resume_total{reason}`.
3. **G3-RES** Unit test cho GORM callback `RegisterQueryMetrics`:
   verify histogram `cdc_source_query_duration_seconds` thực sự tăng
   sample count sau khi DB query.
4. **G4-RES** k6 script chạy được CDC data path thật:
   INSERT source DB → đọc shadow row trong < 5s, tỷ lệ thành công ≥99%.
5. **G5-RES** 2 case validation message của UpdateMappingRule trả về
   đúng chuỗi "status or data_type is required".
6. **G6-RES** Chaos script dùng pumba (Docker container pause/loss)
   thay iptables (yêu cầu root, không reproducible CI).
7. **G7-RES** AdaptiveBatcher giảm batchSize về baseBatchSize khi
   destination unhealthy; expose metric `cdc_dest_throttled_total`.
8. **G8-RES** APPEND lessons vào workspace plan-cdc-qa-gap-fix-2026-05-27.

## Yêu cầu phi chức năng
- §0 Không cheat DB/config.
- §6 Minimal impact, không over-engineer.
- §8 Mỗi gap done → /security-agent gate (chỉ scope code mới).
- Reproducible: ai cũng chạy lại được, không phụ thuộc machine state.

## Out-of-scope
- Không refactor module ngoài scope gap.
- Không chuyển đổi sang framework test mới.
- Không thay đổi behavior pipeline runtime ngoài 2 patch G1/G2/G7.
