# 01. Requirements — Code Review `feat/GP2-5081 -> develop` (`centralized-export-service`)

tier: T2
workspace: review-central-export-gp2-5081-2026-11-17
suffix: gp2_5081
mode: audit-propose-only
project: centralized-export-service
repo_path: /Users/trainguyen/Documents/work/centralized-export-service
source_branch: origin/feat/GP2-5081 (2304b3b9a2fa80d23abdc9adf58e7634fbdc8ac6)
target_branch: origin/develop (3eef80790b319e4d0f188e3d12cef9f0ac90b302)

## 1. Yêu cầu của User (Requirements)
- **R1 (Diff & Architecture Review):** Rà soát toàn bộ thay đổi trên nhánh `feat/GP2-5081` chuẩn bị merge vào nhánh đích `develop` (`dev` theo cách gọi của User) tại dự án `centralized-export-service` (`#patch [GP2-4552][GP2-5081][CMS] Add PartnerCollectionTransExport for collection payment transaction list`).
- **R2 (Correctness, Query Performance, Timezone & Edge Cases):** Kiểm tra tính đúng đắn của DTO/Params, Query, QueryHandler, Pure Transform, Export Flow, index/full-scan risk trên MongoDB/DB lớn, xử lý timezone/date-range, phân trang/streaming/memory footprint, null-safety và backward compatibility với `origin/develop`.
- **R3 (Test Quality & Security):** Đánh giá chất lượng Unit Test & Integration Test (tránh mock giả tạo che giấu bug thực tế), kiểm tra rò rỉ PII/Secret/Injection, và chạy kiểm chứng test suite hiện có ở chế độ Read-Only.
- **R4 (Propose-Only & Decision Gate):** Tuân thủ nghiêm ngặt chế độ *Audit Propose-Only* (không tự ý chỉnh sửa bất kỳ file source code nào của dự án cho đến khi User `APPROVE`), kiểm định các phát hiện qua Decision Gate (`DG0`, `DG1`/`DG2` nếu có đề xuất sửa, và `DG3`), lưu báo cáo vật lý đầy đủ trong workspace.

## 2. Ràng buộc Thực thi Trích xuất từ Global Lessons (S0 Step 3 — Technical Constraints)
- **TC1 (`G2` / Rule #1A, #13 — Audit Propose-Only):** Đang ở chế độ Review/Audit chưa có `APPROVE` — tuyệt đối cấm sửa bất kỳ file `.ts`, `.js`, `.json` nào trong `centralized-export-service`.
- **TC2 (`L074` — Zero Full-Table Scans & Index Alignment):** Kiểm tra chặt chẽ mọi filter query trong `GetAllPartnerCollectionTransExportHandler.ts` xem có nguy cơ quét toàn bảng (thiếu bắt buộc khoảng thời gian `createdAt`/date window, dùng regex không neo đầu hoặc điều kiện làm gãy B-Tree/MongoDB index) trên bảng giao dịch lớn hay không.
- **TC3 (`L078` — BSON/Number Coercion & Timezone Drift):** Kiểm tra việc ép kiểu số tiền/Decimal128/BigInt, trạng thái giao dịch, và chuyển đổi múi giờ (`UTC` vs `Asia/Ho_Chi_Minh` `+07:00`) trong cả tầng lọc `params -> handler` lẫn tầng format xuất file `partner-collection-trans-export.pure.ts`.
- **TC4 (`G3` & `G4` — Decision Gate & Real Test Evidence):** Mọi nhận định phải kèm `file:line` xác thực trên đĩa; chạy test thật qua `evidence_run.py` lấy biên lai `EV-...`; kiểm định qua `laya_gate.py` (`DG0`, `DG3`, và `DG1` cho đề xuất khắc phục nếu phát hiện lỗi).
- **TC5 (`G1` & Rule #20 — Memory Hygiene & Secret Masking):** Không dùng `Overwrite: true` trên file memory đã tồn tại; mask mọi thông tin nhạy cảm/URI nếu trích dẫn config/env.
