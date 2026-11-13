---
tier: T2
mode: standard
suffix: mr_recon_heal_develop
workspace: audit-mr-recon-heal-develop-2026-11-09
created_at: 2026-11-09T11:04:30+07:00
---
# 01 Requirements — Audit Merge Request `recon-heal -> develop` (2 Repos)

## 1. Yêu cầu của User
- Kiểm tra Merge Request từ nhánh `recon-heal` sang nhánh `develop` trên 2 repository:
  1. `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`
  2. `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`
- Rà soát toàn bộ các thay đổi (committed trên `recon-heal` so với `develop` VÀ kiểm tra luôn tình trạng working tree chưa commit nếu có) xem có thay đổi nào tiềm ẩn rủi ro (regression, phá vỡ tương thích ngược, xung đột merge, lỗi runtime/SQL/concurrency, lệch contract giữa CMS và Worker, hoặc code fix đang nằm ngoài commit chưa đưa vào MR).

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L001 (Respect Live Production Baseline):** Tôn trọng trạng thái vận hành thực tế của hệ thống; kiểm tra 100% tương thích ngược (Open-Closed) và đúng Design Pattern chuẩn của từng service (`system_overview_and_design_pattern.md`).
- **L002 (Source-Engine Dispatch In Heal Fetch):** Kiểm tra luồng kéo dữ liệu phục hồi (Heal Fetch) có phân nhánh đúng theo hệ quản trị nguồn (MongoDB vs PostgreSQL/MySQL) hay không.
- **L004 (Nil-Safe Field Access & Route Deduplication):** Kiểm tra truy cập con trỏ nil-safe và khử trùng tuyến đích đa bảng.
- **L012 (Branch-Aware Repository Resolution & Git-Diff Grounded):** Đọc trực tiếp git branch, merge-base, commit log, `git diff develop...recon-heal`, `git diff recon-heal..develop` (xung đột ngược), và `git status` working tree của cả 2 repo.
- **L015 (Verbatim Current-File Code Demo):** Mọi trích dẫn và Code Demo (nếu đề xuất sửa) phải trích nguyên văn từ file thực tế kèm `file:line`.
- **L017 (Executive Issues Summary Table):** Bắt buộc có Bảng Tổng Hợp Issues & Tổng Kết Hành Động rõ ràng ở đầu báo cáo.
- **L024 (Propose-Only Discipline):** Giữ chế độ 100% Read-Only đối với source code, trình bày phân tích + giải pháp + Code Demo trên chat và chờ `APPROVE`.
- **L052/L053/L055/L064/L065:** Kiểm tra rủi ro SQL/DDL runtime, multi-tenant 5-tuple identity, GORM transaction/slice `IN (?)`, spec-driven PK/timestamp resolution, và Recon Window Alignment.
