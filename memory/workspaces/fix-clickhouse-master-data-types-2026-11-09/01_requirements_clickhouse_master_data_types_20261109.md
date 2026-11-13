---
tier: T2
workspace: fix-clickhouse-master-data-types-2026-11-09
suffix: clickhouse_master_data_types_20261109
date: 2026-11-09
status: S1_INTAKE
---

# 01. Requirements — Kiểm tra & Chuẩn hóa Master Data Type cho ClickHouse

## 1. Yêu cầu của User
> "Master Data Type ở clickhouse có đúng không. hiện tại đang là bộ của postgress đó"

## 2. Mục tiêu
1. Kiểm tra toàn diện (`cdc-cms-web`, `cdc-cms-service`, `centralized-data-service`) nơi định nghĩa, hiển thị, đồng bộ (`sync-from-shadow`), validate và sinh DDL/Upsert cho **Master Data Type** khi đích (`engine_type`) là **ClickHouse** (cũng như đối chiếu với Postgres, MySQL, MongoDB).
2. Xác minh xem hiện tại trên giao diện (`cdc-cms-web`) và/hoặc Backend (`cdc-cms-service`, `centralized-data-service`) khi chọn Master thuộc ClickHouse thì danh sách kiểu dữ liệu (`Master Data Type`) có đang dùng nhầm bộ kiểu của PostgreSQL (`TEXT`, `VARCHAR(255)`, `BIGINT`, `TIMESTAMPTZ`, `JSONB`, `BYTEA`...) hay không, và ở tầng Worker (`master_ddl_generator_clickhouse.go`, `transmuter_clickhouse.go`) đang xử lý ánh xạ kiểu dữ liệu như thế nào.
3. Thiết kế phương án chuẩn hóa (S3) kèm Code Demo nguyên văn 1-1 tại HEAD, qua cổng kiểm định `DG0` và `DG1/DG2`, trình User duyệt (`Propose-Only` chờ `APPROVE`).

## 3. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L001 (Strict 1-to-1 Code Demo Parity):** Code Demo trong `09_tasks_solution` và trên chat phải trích dẫn nguyên văn từ HEAD, tuyệt đối không dùng dấu `...` rút gọn.
- **L004 (Respect Live Production Baseline & Backward Compatibility):** Bảo toàn 100% tương thích ngược cho các bảng Master cũ (Postgres, MongoDB, MySQL, và cả các rule ClickHouse cũ nếu đã lưu kiểu Postgres trong `mapping_rule_master`).
- **L009 & L018 (Frontend Call-Site Tracing & Verbatim Code Demo):** Truy vết đầy đủ từ UI (`cdc-cms-web`) -> API (`cdc-cms-service`) -> Worker (`centralized-data-service`).
- **L027 (Propose-Only Gate):** Tuyệt đối không sửa source code trước khi User `APPROVE`.
- **L071 (React Rules of Hooks & TS Build):** Mọi hook React khai báo ở top-level; đảm bảo `tsc -b` pass sạch.
