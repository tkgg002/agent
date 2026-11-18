---
tier: T2
mode: standard
suffix: target_db_connections_ui
workspace: fix-target-db-connections-ui-2026-11-17
created_at: 2026-11-17T15:40:30+07:00
---
# 01 Requirements — Tái Cấu Trúc Bảng UI Quản Lý Kết Nối Đích (`Target Database Connections`)

## 1. Yêu cầu của User
- **Màn hình / Khu vực mục tiêu:** Bảng **Quản Lý Kết Nối Đích (`Target Database Connections`)** trên giao diện `cdc-cms-web`.
- **Chi tiết thay đổi bố cục cột:**
  1. **Bỏ 2 cột rời:**
     - Bỏ cột **Engine** riêng lẻ.
     - Bỏ cột **Tên hiển thị** riêng lẻ.
  2. **Gộp chung `Mã kết nối` & `Tên hiển thị` vào 1 cột:**
     - **Hàng 1:** Mã kết nối (`connection_code`).
     - **Hàng 2:** Tên hiển thị (`display_name`).
  3. **Gộp chung `Engine` & `Host:Port` vào 1 cột:**
     - **Hàng 1:** Icon + Tên Engine (`icon + name engine`).
     - **Hàng 2:** `host:port`.

## 2. Ràng buộc Thực thi (Technical Constraints từ `GEMINI.md`, `lessons/INDEX.md` & `.agent-project.json`)
- **L094 (Visual UI/UX Prototype, Semantic Color Palette & Progress Preview First Before Coding):** Trình bày rõ bản mẫu bố cục trực quan (cột trước vs sau, hàng 1 / hàng 2, hệ màu ngữ nghĩa icon/tag Engine, typography) và Code Demo để User duyệt (`APPROVE`) trước khi chạm vào mã nguồn. Bảo toàn hệ màu ngữ nghĩa của từng Engine (`postgresql`, `mongodb`, `mysql`/`mariadb`, `clickhouse`, ...) và các cột/nút thao tác CRUD còn lại.
- **L095 (React Rules of Hooks, Strict Vite/TS Build & Minimal Diff):** Tuân thủ tuyệt đối React Rules of Hooks (100% hooks ở top-level), dọn sạch unused imports/variables để pass cả `npx tsc -b --noEmit`, `npm run lint` và `npm run build` (`tsc -b && vite build`).
- **L096 (CSS Full-Height Scroll Container & Unified App Shell):** Bảo toàn chiều rộng cột hợp lý, scroll ngang/dọc của Ant Design `Table`, không làm vỡ layout bảng trên màn hình hẹp.
- **L098 (Full CRUD, No Ellipsis & Truthful Tooltips):** Tuyệt đối KHÔNG dùng `ellipsis` (`...`) cắt cụt tên thực thể kỹ thuật (`connection_code`, `host:port`, `database_name`); bảo toàn 100% tính năng CRUD (Tạo mới, Chỉnh sửa, Kiểm tra kết nối, Xóa/Ngưng kích hoạt, Tìm kiếm/Lọc nếu có).
- **Ràng buộc Kiến trúc `cdc-cms-web` (`.agent-project.json` & `review_profile.md`):**
  - `INV-4`: UI/Table/Modal phức tạp tách ra `src/components/master/` đúng Single Responsibility Principle (`god_file_loc: 600`), giữ `NOW <= HEAD` trên file lớn (`src/pages/MasterRegistry.tsx` nằm trong `forbidden_ifelse_files` — tuyệt đối cấm nhồi thêm `if/else` vào `src/pages/MasterRegistry.tsx`!).
  - `INV-1` & `INV-2`: Bảo toàn 100% kiểu dữ liệu API (`MasterConnectionItem`), re-exports tương thích ngược và hành vi CRUD hiện hữu.
- **Rule #1, #9, #13 (Brain Propose-Only):** Hoàn tất điều tra `S2` (`file:line` thật), phản biện ngược `S2.5 (DG0)`, lập **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** kèm **Code Demo** trong `09_tasks_solution_target_db_connections_ui.md` và hiển thị trên khung chat, qua cổng `DG1/DG2` (`PASS`) rồi **DỪNG CHỜ `APPROVE`**.
