# Báo cáo Phân tích Metrics Prometheus & SigNoz Dashboard

## 1. Tổng quan
- **File code Go:** `centralized-data-service/pkgs/metrics/prometheus.go` định nghĩa tổng cộng **49 metrics**.
- **File cấu hình Dashboard SigNoz:** `centralized-data-service/deployments/signoz-dashboard-recon.json` sử dụng **10 metrics** để hiển thị trên 11 panels/widgets.
- **Mục tiêu:** Rà soát sự trùng lặp, thiếu sót và dư thừa giữa code Go (nơi định nghĩa & cập nhật metric) và Dashboard SigNoz (nơi trực quan hóa).

---

## 2. Các phát hiện chi tiết

### Nhóm A: Metrics định nghĩa trong code nhưng HOÀN TOÀN KHÔNG được sử dụng (Dead Metrics)
Đây là các metrics đã được khởi tạo trong file `prometheus.go` nhưng không được gọi `.Set()`, `.Inc()`, `.Observe()` hay bất kỳ thao tác nào khác ở codebase Go (không xuất hiện trong production code lẫn test code):

1. **`pending_fields_count` (`PendingFieldsCount`)**
   - *Mô tả:* Pending fields by status.
   - *Trạng thái:* Dead code, không được sử dụng ở bất kỳ đâu.
2. **`registered_tables_total` (`RegisteredTables`)**
   - *Mô tả:* Registered CDC tables.
   - *Trạng thái:* Dead code, không được sử dụng ở bất kỳ đâu.
3. **`cdc_snapshot_partial_done_total` (`SnapshotPartialDoneTotal`)**
   - *Mô tả:* Snapshot mark-done attempts blocked by completeness guard, by reason.
   - *Trạng thái:* Dead code, không được sử dụng ở bất kỳ đâu.

### Nhóm B: Metrics định nghĩa nhưng chỉ dùng trong Unit Test (Test-only Metrics)
1. **`mapping_rules_loaded` (`MappingRulesLoaded`)**
   - *Mô tả:* Current loaded mapping rules count.
   - *Trạng thái:* Chỉ được gọi `.Set(5)` trong unit test `metrics_test.go` nhằm mục đích kiểm tra khả năng đăng ký metric. Code production không thực sự cập nhật metric này khi nạp mapping rules.

### Nhóm C: Metrics bị trùng lặp/dư thừa về mặt ngữ nghĩa (Semantic Duplicates)
1. **`cdc_recon_mismatch_count` (`ReconMismatchCount`)** vs **`cdc_recon_drift_count` (`ReconDrift`)**
   - *Mô tả:* Cả hai metrics này đều ghi nhận số lượng bản ghi bị lệch (drift/mismatch) của mỗi bảng theo từng Tier (0, 1, 2, 3, 4).
   - *Điểm trùng lặp:*
     - `cdc_recon_drift_count` được cập nhật chi tiết trong từng logic của các Tier:
       - Tier 0: Trong `recon_smoke.go` (độ lệch giữa source và shadow).
       - Tier 1, 2, 3: Trong `recon_tier_a.go` (lệch theo số lượng, hash, hoặc bucket).
       - Tier 4 (Segment B): Trong `recon_tier_b.go` (lệch shadow vs master).
       - Cảnh báo alert rule `ReconDriftPersistent` trong `deployments/prometheus/alerts/cdc.yml` cũng sử dụng `cdc_recon_drift_count`.
     - Trong khi đó, `cdc_recon_mismatch_count` chỉ được cập nhật ở cuối hàm chạy của engine (`recon_engine_run.go`) với cùng giá trị:
       `metrics.ReconMismatchCount.WithLabelValues(h.table, fmt.Sprintf("%d", h.tier)).Set(float64(h.mismatches))`
     - Metric `cdc_recon_mismatch_count` **không được dùng ở bất kỳ panel nào trên Dashboard hay trong bất kỳ alert rule nào**.
   - *Kết luận:* `cdc_recon_mismatch_count` hoàn toàn dư thừa vì đã có `cdc_recon_drift_count` chịu trách nhiệm theo dõi chi tiết này.

### Nhóm D: Metrics hoạt động bình thường trong code nhưng chưa có trên Dashboard (Code-only Metrics)
Đây là các metrics đang chạy thực tế, được cập nhật giá trị đúng trong code Go nhưng hiện tại chưa được đưa lên cấu hình trực quan hóa trên SigNoz Dashboard:

1. **`cdc_events_processed_total` (`EventsProcessed`)**
   - *Vị trí cập nhật:* Được tăng giá trị (`.Inc()`) tại `kafka_consumer.go` (khi consume tin nhắn từ Kafka thành công/thất bại) và `event_handler.go` (khi xử lý xong event CDC).
   - *Trạng thái:* Hoạt động bình thường trong code Go, nhưng **không hiển thị trên Dashboard** (Dashboard hiện dùng `cdc_sink_events_total` để đo throughput ở tầng Sink).
2. Các metrics vận hành khác (ví dụ: `cdc_processing_duration_seconds`, `cdc_kafka_consumer_lag`, `cdc_recon_run_duration_seconds`, `cdc_recon_heal_actions_total`, `cdc_e2e_latency_seconds`, `cdc_pipeline_paused_total`, `cdc_dlq_write_failures_total`, ...).

---

## 3. Rà soát Dashboard SigNoz (`signoz-dashboard-recon.json`)

### Độ chính xác và đồng bộ:
- **100% metrics trên Dashboard đều hợp lệ:** Mọi metric được gọi trên Dashboard đều tồn tại trong `prometheus.go` và được cập nhật chính xác trong code Go. Cụ thể:
  1. `cdc_source_table_row_count` -> Panel `w01` (Source Row Count)
  2. `cdc_shadow_active_row_count` -> Panel `w02` (Shadow Active Row Count)
  3. `cdc_master_active_row_count` -> Panel `w03` (Master Active Row Count)
  4. `cdc_sink_events_total` -> Panel `w04` (Sink Events Rate)
  5. `cdc_batches_flushed_total` -> Panel `w05` (Sink Batch Flush Rate)
  6. `cdc_transmute_ops_total` -> Panels `w06` & `w11` (Transmute Ops)
  7. `cdc_transmute_duration_ms` -> Panel `w07` (Transmute Duration)
  8. `cdc_pipeline_table_status` -> Panel `w08` (Pipeline Health Status)
  9. `cdc_dlq_depth` -> Panel `w09` (DLQ Depth)
  10. `cdc_recon_cycle_drift_detected` -> Panel `w10` (Tables Drifting)

### Lưu ý về Row Count:
- Code Go định nghĩa cả:
  - `cdc_shadow_table_row_count` (Ước lượng tổng số dòng gồm cả dòng đã xoá mềm)
  - `cdc_shadow_active_row_count` (Chỉ tính dòng active thực tế: total - deleted)
- SigNoz Dashboard sử dụng đúng đắn phiên bản `active` (`cdc_shadow_active_row_count` và `cdc_master_active_row_count`). Điều này là hoàn toàn chính xác để so sánh số liệu với MongoDB (vốn không có cơ chế soft-delete mặc định của shadow/master). Hai metrics `cdc_shadow_table_row_count` và `cdc_master_table_row_count` chỉ tồn tại trong code như các metrics phụ trợ, không cần đưa lên dashboard để tránh nhiễu thông tin.

---

## 4. Đề xuất cải tiến (Recommendation)
1. **Dọn dẹp code Go (Clean up Dead/Unused Metrics):**
   - Loại bỏ các biến metric không sử dụng: `PendingFieldsCount`, `RegisteredTables`, `SnapshotPartialDoneTotal`.
   - Xem xét loại bỏ `ReconMismatchCount` (`cdc_recon_mismatch_count`) vì bị trùng lặp với `ReconDrift` (`cdc_recon_drift_count`).
   - Cập nhật logic code nạp mapping rules để thực sự set dữ liệu cho `MappingRulesLoaded` (nếu muốn giữ metric này), hoặc xoá bỏ nếu không cần thiết.
2. **Dashboard bổ sung (Tùy chọn cho SRE):**
   - Các metrics như `cdc_kafka_consumer_lag` (độ trễ Kafka) và `cdc_e2e_latency_seconds` (thời gian đi từ Kafka đến shadow DB) rất quan trọng cho hiệu năng hệ thống. Mặc dù đã có alert rules cảnh báo, SRE có thể cân nhắc thêm chúng vào Dashboard để theo dõi trực quan hơn.
