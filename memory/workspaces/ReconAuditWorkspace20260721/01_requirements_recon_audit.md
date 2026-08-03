# 01 — Yêu Cầu Audit (Requirements Audit Spec)

> **Workspace:** `ReconAuditWorkspace20260721`  
> **Workspace Đích:** `ReconAdaptiveBinaryAsync20260721`  

---

## I. YÊU CẦU AUDIT THEO CÁC CỔNG DOD (G1–G8)

| Mã Gate | Tên Cổng Kiểm Kiểm | Tiêu Chí Bắt Buộc | Trạng Thái Audit |
| :--- | :--- | :--- | :--- |
| **G1** | Requirement Traceability | 100% các tính năng thiết kế (Chunk stream, Async Worker, Adaptive Endpoint, CMS Report Sync) đều có code và test tương ứng. | PASSED |
| **G2** | Red → Green Verification | Các bug phát sinh (như giới hạn biên sub-window, lỗi interface mock test) phải được kiểm thử tái hiện trước khi sửa. | PASSED |
| **G3** | Real Test Execution | Toàn bộ unit test trong gói `service/recon` và `handler/recon` phải thực thi THẬT và đạt PASS 100%. | PASSED |
| **G4** | Edge-Case & Boundaries | Áp dụng nửa khoảng $[start, end)$, làm tròn phút `:00` cho Lag Buffer 120s, và cắt biên $min(subStart+15m, dayEnd)$ chính xác. | PASSED |
| **G5** | Anti-Regression | Giữ nguyên tính tương thích ngược với `ReconCore` và các API handler hiện hữu. | PASSED |
| **G6** | Output Correctness | Giá trị đếm và XOR hash khớp tuyệt đối giữa MongoDB và Postgres shadow table. | PASSED |
| **G7** | Adversarial Self-Review | Rà soát mã nguồn khắt khe, xóa bỏ file rỗng/rác (`test_write.go`), sửa bù interface test. | PASSED |
| **G8** | Physical Workspace Integrity | Mọi báo cáo audit, tiến độ và minh chứng phải được lưu vết thành tệp tin vật lý trong workspace này. | PASSED |
