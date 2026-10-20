tier: T2

# 01. Requirements — Audit Lỗ Hổng Rớt Update Từ Shadow Sang Master (Mongo -> Postgres)

## 1. Bối cảnh & Triệu chứng Vận hành (Symptom)
- **Nguồn phản ánh:** Đội Vận hành (Operations) báo cáo luồng đồng bộ dữ liệu từ **MongoDB $\rightarrow$ PostgreSQL** với lưu lượng khoảng **~15,000 bản ghi/ngày** xuất hiện hiện tượng **bị rớt (miss/drop) các bản cập nhật (UPDATE)** ở chặng **Shadow $\rightarrow$ Master** với tỉ lệ khoảng **3–4 bản ghi/ngày** (~0.02% – 0.03%).
- **Yêu cầu của User:**
  1. Thực hiện Audit chuyên sâu theo đúng **Master Workflow Agent mới (`S0 → S1 → S2 → S3 → DG1/DG2/DG3 → Propose-Only chờ APPROVE`)**.
  2. Truy vết toàn bộ đường đi của bản ghi UPDATE từ **MongoDB CDC (Debezium/Kafka) $\rightarrow$ Shadow Ingestion (`event_handler.go`, `batch_buffer.go`, `writer_v2.go`) $\rightarrow$ Trigger NATS JetStream / Catch-up Scan $\rightarrow$ Transmute Consumer & `TableDebouncer` $\rightarrow$ `Transmuter.RunEvents` / `Transmuter.Run` $\rightarrow$ Master Upsert & OCC Guard (`_source_ts`, `_gpay_id`, `_version`)**.
  3. Xác định mọi lỗ hổng (`F1..Fn`) có bằng chứng `file:line` xác thực (`VERIFIED`) giải thích tại sao ở quy mô ~15k records/ngày lại rớt đúng một tỷ lệ nhỏ (3–4 bản ghi update) khi chuyển từ Shadow sang Master.
  4. Kiểm định toàn bộ phát hiện và giải pháp đề xuất qua Decision Gate `/laya_router` (`DG1`, `DG2`, `DG3`).
  5. Giữ kỷ luật **Propose-Only** (tuyệt đối không tự ý sửa source code khi chưa có lệnh `APPROVE` của User).

## 2. Các Yêu cầu Cụ thể (`R1` – `R4`)
- **R1 (End-to-End Code & Runtime Audit):** Rà soát toàn diện các mắt xích từ lúc Shadow ghi nhận bản UPDATE (`batch_buffer.go`, `writer_v2.go`) đến lúc kích hoạt và thực thi Transmute sang Master (`transmute_publisher.go`, `transmute_consumer_pool.go`, `table_debouncer.go`, `transmute_handler.go`, `transmuter.go`, `transmuter_helpers.go`, `reconcile`).
- **R2 (Evidence-Based Root Cause Analysis):** Chỉ rõ từng sơ hở kiến trúc/logic/concurrency kèm `file:line` cụ thể gây mất hoặc bỏ qua (skip/drop) sự kiện UPDATE ở chặng `Shadow -> Master`.
- **R3 (Single Best Technical Solution & Code Demo):** Đề xuất **1 phương án khắc phục tối ưu nhất** trong `09_tasks_solution_shadow_master_update_drop.md` kèm Code Demo chuẩn xác và báo cáo phân tích `13_analysis_shadow_master_update_drop.md`.
- **R4 (Laya Decision Gate Verification):** Chạy và đạt `PASS` trên `laya_gate.py` cho `DG3` (kiểm chứng các phát hiện Audit) và `DG1` + `DG2` (kiểm chứng giải pháp & code demo đề xuất) trước khi trình User duyệt.
