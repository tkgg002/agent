# 01. Requirements — Design Pattern & Architecture Audit `feat/GP2-5081 -> develop` (`7f4e9a7`)

tier: T2
workspace: review-central-export-gp2-5081-2026-11-17
suffix: gp2_5081_dp
mode: audit-propose-only
project: centralized-export-service
repo_path: /Users/trainguyen/Documents/work/centralized-export-service
source_branch: origin/feat/GP2-5081 (7f4e9a7e149dabd75674f9fb3737c81bbfa21794)
target_branch: origin/develop (3eef80790b319e4d0f188e3d12cef9f0ac90b302)

## 1. Yêu cầu của User (Requirements)
- **R1 (Design Pattern & Project DNA Audit):** Kiểm toán chuyên sâu việc tuân thủ Design Patterns và Clean Architecture của nhánh `feat/GP2-5081` (`7f4e9a7`) đối chiếu với toàn bộ 37 handlers và 13 cross-model exporters hiện hữu trong `centralized-export-service` (`BaseExportProcessor`, `PayoutRequestExport`, `PaymentHistoryExport`, `TransHisExport`, `TransHisCollectionExport`, `InternalTransferExport`).
- **R2 (Single Optimal Refactoring Proposal):** Đề xuất 1 phương án cấu trúc lại chuẩn chỉnh nhất bám sát 100% Design Patterns sẵn có của dự án (Template Method hooks `async getConfig(..., queryExecutor)` + `subQueryClass`/`buildSubQueryParams`/`mergeData`, Clean Architecture unidirectional dependency, Stateless `IQuery`, tái sử dụng `GetAllTransHisExportQuery`), kết hợp xử lý gọn điểm `N1`/`N3` từ vòng trước.
- **R3 (Mid-Session Self-Improvement & Propose-Only Governance):** Ghi bài học chuẩn Global vào `lessons.md` + đồng bộ `INDEX.md` do thiếu trục kiểm toán Design Pattern ở 2 lượt đầu (Rule #5); giữ nguyên chế độ *Propose-Only* (không sửa source code dự án cho tới khi User `APPROVE`), chạy đủ `DG0`, `DG1`/`DG2`, `DG3` và `verify_governance.py`.

## 2. Ràng buộc Thực thi Trích xuất từ Global Lessons (Technical Constraints)
- **TC1 (`L001` — Mandatory Design Pattern, Layer Dependency & Template Hook Audit In Code Review):** Bắt buộc kiểm tra (a) chiều phụ thuộc một chiều (`domain/` cấm import ngược `logics/`), (b) tính bất biến của `IQuery` (cấm gắn state runtime `resolvedBillIds` vào DTO Query), (c) trách nhiệm đơn mô hình của từng `IQueryHandler`, và (d) tái sử dụng móc mở rộng `queryExecutor` + `subQueryClass` / `buildSubQueryParams` / `mergeData` của `BaseExportProcessor` cùng `GetAllTransHisExportQuery`.
- **TC2 (`G2` / Rule #1A, #13):** Tuyệt đối không sửa source code dự án trước khi có lệnh `APPROVE`.
- **TC3 (`L077` / `L079` — Partial Index & Schema Parity):** Giữ điều kiện `isDelete: { $ne: true }` (`core.is_delete IS DISTINCT FROM TRUE`) trong thiết kế đề xuất để khớp Partial Index `idx_th_cd_flow_created_at_id`.
