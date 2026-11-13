---
tier: T2
workspace: fix-masters-cms-grouping-2026-11-10
suffix: masters_cms_grouping_20261110
date: 2026-11-10
status: S1_INTAKE
---

# 01. Requirements — Gom nhóm hiển thị trang `/masters` trên CMS theo DB Master -> Schema/Database -> Table

## 1. Yêu cầu của User
> "/masters fix hiển thị trên cms. gom nhóm theo db master trước, trong đó mới là schema rồi table (hoặc gì tương ứng với mỗi loại sql, postgress, clickhouse) cho a cái icon ở nhóm luôn"

## 2. Mục tiêu
1. Khảo sát toàn diện trang `/masters` trên `cdc-cms-web` (`src/pages/MasterRegistry.tsx`, các component con trong `src/components/master/`, hook `useMasterRegistry.ts`, `src/types/index.ts`) và dữ liệu trả về từ API `GET /api/v1/masters` & `GET /api/v1/master-connections` trên `cdc-cms-service`.
2. Tổ chức lại cấu trúc hiển thị danh sách Master Tables trên trang `/masters` theo phân cấp 3 tầng rõ ràng:
   - **Tầng 1 (Nhóm DB Master / Master Connection):** Gom nhóm theo kết nối DB Master đích (`connection_name` / DB Master, bao gồm cả nhóm mặc định/local nếu `connection_name` trống), kèm **Icon đặc trưng của từng loại DB Engine** (`PostgreSQL`, `MySQL`, `ClickHouse`, `MongoDB`) và thông tin tổng quan nhóm.
   - **Tầng 2 (Nhóm Schema / Database / Namespace tương ứng với từng Engine):**
     - Với `postgresql`: Gom theo **Schema** (ví dụ `public`, `master`, ...).
     - Với `mysql` / `clickhouse`: Gom theo **Database** (tương ứng trường `master_schema` hoặc database từ connection).
     - Với `mongodb`: Gom theo **Database** (tương ứng `master_schema` / database đích, bên trong là các Collection).
     - Kèm icon cho nhóm Schema/Database.
   - **Tầng 3 (Table / Collection):** Hiển thị các bảng `master_table` (hoặc collection với MongoDB) nằm bên trong nhóm Schema/Database tương ứng, bảo toàn 100% các cột thông tin, trạng thái, bộ lọc và các nút thao tác hiện có.
3. Thiết kế phương án chuẩn hóa (S3) kèm bản mẫu trực quan UI/UX & Code Demo nguyên văn 1-1 tại HEAD, qua cổng kiểm định `DG0` và `DG1/DG2`, trình User duyệt (`Propose-Only` chờ `APPROVE`).

## 3. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L011 & L028 (Strict 1-to-1 Code Demo Parity):** Code Demo trong `09_tasks_solution` và trên chat phải trích dẫn nguyên văn từ HEAD, tuyệt đối không dùng dấu `...` rút gọn.
- **L014 (Respect Live Production Baseline & Backward Compatibility):** Bảo toàn 100% tương thích ngược cho dữ liệu Master Bindings và các thao tác CRUD/Transmute/DDL/Schedule trên trang `/masters`.
- **L019 (Frontend Call-Site Tracing):** Truy vết đầy đủ từ UI (`cdc-cms-web`) -> API (`cdc-cms-service`).
- **L037 (Propose-Only Gate):** Tuyệt đối không sửa source code trước khi User `APPROVE`.
- **L080 (Visual UI/UX Prototype First):** Trình bày rõ bố cục phân cấp, màu sắc ngữ nghĩa và icon từng nhóm trước khi sửa code.
- **L081 (React Rules of Hooks & TS Build):** Mọi hook React (`useState`, `useMemo`, `useEffect`...) bắt buộc khai báo vô điều kiện ở top-level của Function Component; đảm bảo `tsc -b` pass sạch và không còn unused imports.
- **L084 (No Ellipsis on Entity Names):** Không dùng `ellipsis` cắt cụt tên DB/schema/table.
