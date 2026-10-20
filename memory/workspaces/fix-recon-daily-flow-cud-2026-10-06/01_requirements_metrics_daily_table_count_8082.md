tier: T2

# 01 — Requirements: Kiểm Tra & Chuẩn Hóa Metrics Count Table Theo Ngày Trên `:8082/metrics`

- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **Task Suffix:** `metrics_daily_table_count_8082`
- **Ngày lập:** `2026-10-16`
- **Phân tầng quy mô (ADR-03):** `tier: T2`

---

## 1. Yêu Cầu Cốt Lõi Từ User (Requirements `R1` – `R4`)

- **`R1` — Kiểm định độ chính xác của Metrics đếm bảng theo ngày trên `:8082/metrics`:**
  - Kiểm tra thực chứng xem số đếm bản ghi tạo mới theo ngày (`report_date = YYYY-MM-DD`) tại các trạm đã được đẩy ra `:8082/metrics` chính xác chưa, đối chiếu trực tiếp với dữ liệu thật trong DB và mã nguồn.
- **`R2` — Cấm tuyệt đối các tác vụ đếm tổng toàn bảng / toàn hệ thống:**
  - Tuyệt đối không thực hiện và không đưa các chỉ số quét tổng toàn bảng (`source_total`, `source_active`, `full_source_count`, `window_source_count`, `shadow_total`, `master_total`) vào thiết kế vận hành. Chỉ làm việc với **tín hiệu luồng thời gian thực (Stream Throughput, Lag, Error)** và **số đếm bản ghi theo lát cắt ngày (`Daily Sliced Count`)**.
- **`R3` — Làm rõ toàn bộ thông số của 4 thực thể (`Kafka`, `1 Bảng Source`, `1 Bảng Shadow`, `1 Bảng Master`) và không đoán mò cột ngày tạo:**
  - Bóc tách rõ mỗi thực thể đang có những thông số cấu hình/định danh nào trong mã nguồn và những chỉ số Prometheus `/metrics` nào.
  - Chỉ rõ lỗ hổng: Bảng Source (`source_object_registry`) hiện chưa có chỗ cấu hình cột ngày tạo bản ghi (`creation_date_field` / `created_at_field`), dẫn tới việc đoán mò tên cột khi đếm theo ngày. Tuyệt đối không dùng cơ chế đoán mò tên cột.
- **`R4` — Tuân thủ kỷ luật `/agent` (`Propose-Only` trước khi có `APPROVE`):**
  - Làm rõ đặc tả 4 trạm và thiết kế giải pháp chuẩn (`09_tasks_solution_metrics_daily_table_count_8082.md`), qua kiểm định `DG1 / DG2 / DG3` (`laya_gate.py`), và chờ User chốt / `APPROVE` trước khi chạm vào source code dự án.
