# 08. TASKS — Audit & Giải Pháp Khắc Phục Lệch `-2` Record Ở Master Cho `payment-bills` (`payment_bills_missing_2`)

## Danh sách Task

- [ ] **TASK-01 (S2 Investigate — Truy vết DB & Codebase):**
  - **Mô tả & Context:** Tìm chỗ render chuỗi `ingest: 0`, `transmute: -2 (thiếu)`, `Lệch , ở master bị thiếu 2 record` trong `cdc-cms-web` và API tương ứng trong `cdc-cms-service` / `centralized-data-service`. Đồng thời kiểm tra kết nối DB (local/dev/testing trong config) xem có bảng `shadow_traitestpbs.payment_bills` và `master_payment_bill_service.payment_bills` (hoặc cấu hình binding/mapping/log của `payment_bills`) để truy vết chính xác 2 bản ghi bị thiếu.
  - **Logs/Test:** Truy vấn read-only trên PostgreSQL/MongoDB và grep codebase `cdc-cms-web`, `cdc-cms-service`, `centralized-data-service`.
  - **Definition of Done (DoD):** Xác định rõ 100% cơ chế tính con số `56,108 / 56,108 / 56,106` và nguyên nhân gốc rễ khiến Master thiếu 2 bản ghi (`VERIFIED` bằng `file:line` và DB query).

- [ ] **TASK-02 (S2.5 & S3 — Phản biện DG0, Thiết kế Giải pháp & Kiểm định DG1/DG2):**
  - **Mô tả & Context:** Chạy `laya_gate.py run --stage dg0`, lập báo cáo phân tích `13_analysis_payment_bills_missing_2.md` và giải pháp kỹ thuật `09_tasks_solution_payment_bills_missing_2.md` kèm Code Demo, chạy `laya_gate.py run --stage dg1`.
  - **Logs/Test:** `06_laya_summary_dg0_*.json`, `06_laya_summary_dg1_*.json`, `06_laya_summary_dg2_*.json` đạt `PASS`.
  - **Definition of Done (DoD):** Hồ sơ S1–S3 đầy đủ, trình bày rõ nguyên nhân gốc rễ + phương án xử lý và dừng chờ User `APPROVE`.
