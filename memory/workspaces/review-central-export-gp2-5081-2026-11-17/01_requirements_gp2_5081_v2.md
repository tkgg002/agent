# 01. Requirements — Re-Review Merger `feat/GP2-5081 -> develop` (`7f4e9a7`, Pass 2)

tier: T2
workspace: review-central-export-gp2-5081-2026-11-17
suffix: gp2_5081_v2
mode: audit-propose-only
project: centralized-export-service
repo_path: /Users/trainguyen/Documents/work/centralized-export-service
source_branch: origin/feat/GP2-5081 (7f4e9a7e149dabd75674f9fb3737c81bbfa21794)
target_branch: origin/develop (3eef80790b319e4d0f188e3d12cef9f0ac90b302)

## 1. Yêu cầu của User (Requirements)
- **R1 (Re-verify Previous Findings `F1`–`F6`):** Kiểm tra lại toàn bộ 4 commit sửa lỗi mới (`51c8898`, `0928958`, `28460e0`, `7f4e9a7`) trên nhánh `origin/feat/GP2-5081` so với `origin/develop` (tổng 20 files changed, `+1420 / -4`) xem đã xử lý triệt để các vấn đề từ vòng review đầu tiên chưa.
- **R2 (Deep Audit New Code — Postgres Compiler, PG Handler & Bill Helper):** Rà soát kỹ các file mới phát sinh và thay đổi sâu ở tầng SQL/Postgres (`models/trans-his-pg/trans-his-pg.query.ts`, `GetAllPartnerCollectionTransExportPgHandler.ts`, `partner-collection-trans-bills.ts`, `partner-collection-trans-export.pure.ts`) về SQL correctness, index usage trên `public.transaction_history`, schema compatibility (`cd_flow`, `biz_type` — migration 011), pagination/cursor, edge cases và backward compatibility với các export `trans-his` hiện có.
- **R3 (Real Test, Typecheck & ESLint Verification):** Chạy kiểm chứng toàn bộ unit tests, integration tests, `tsc --noEmit` và `eslint` trên 20 files thay đổi qua `evidence_run.py` lấy biên lai `EV-...` mới nhất.
- **R4 (Propose-Only & Decision Gate):** Tuân thủ tuyệt đối chế độ *Audit Propose-Only* (không sửa source code trước khi User `APPROVE`), chạy đủ `DG0`, `DG1`/`DG2` (nếu còn đề xuất chỉnh sửa), `DG3` và `verify_governance.py`.

## 2. Ràng buộc Thực thi Trích xuất từ Global Lessons (Technical Constraints)
- **TC1 (`G2` / Rule #1A, #13):** Chế độ *Propose-Only* — không sửa bất kỳ file source code nào của dự án.
- **TC2 (`L076` — Zero Runtime DDL & No Full-Table Scans):** Kiểm tra SQL sinh ra bởi `trans-his-pg.query.ts` có bám đúng index của `public.transaction_history` hay vô tình dùng biểu thức làm gãy index (như `COALESCE` hoặc thiếu cột phân loại), và kiểm tra giới hạn tra cứu `payment-bills`.
- **TC3 (`L078` & `L080` — Schema Verification & BSON/Timezone Consistency):** Đối chiếu tên cột SQL (`core.cd_flow`, `core.biz_type`, `payload.info`, `payload.receiver`) với schema thực tế của `core-trans-proxy-history-service-v2` (migration `011` — nếu DB chưa chạy migration `011` thì sao?!) và kiểm tra chuyển đổi kiểu dữ liệu/múi giờ.
- **TC4 (`L042`, `G3`, `G4`):** Tách lệnh đơn sạch khi gọi `evidence_run.py` và `laya_gate.py`; báo cáo trung thực trạng thái cổng kiểm định.
