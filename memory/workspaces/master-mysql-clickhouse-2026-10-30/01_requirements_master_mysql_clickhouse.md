---
tier: T2
suffix: master_mysql_clickhouse
workspace: master-mysql-clickhouse-2026-10-30
created_at: 2026-10-30T15:32:10+07:00
---

# 01. Requirements — Hỗ trợ Master MySQL & Master ClickHouse (`master_mysql_clickhouse`)

## 1. Yêu cầu của User
- Mở rộng tính năng **Master Destination** hỗ trợ đồng thời cả **MySQL (InnoDB / MariaDB)** và **ClickHouse (ReplacingMergeTree)** bên cạnh PostgreSQL và MongoDB hiện có.
- **Đối chiếu trực tiếp với thiết kế của tính năng Master MongoDB vừa triển khai** (`master_ddl_generator_mongo.go`, `transmuter_mongo.go`, `mongo_type_converter.go`, `connection_manager.go`, `recon` và `cdc-cms-service` / `cdc-cms-web`).
- **Kỷ luật Kiến trúc & Single-Responsibility (Bắt buộc):**
  - Mọi thay đổi bắt buộc phải đặt đúng layer theo Design Pattern gốc của từng repository (`cdc-cms-service` Clean/DDD `domain` -> `app` -> `infra` -> `api`; `centralized-data-service` Service/Repository/Database layer; `cdc-cms-web` Feature/Component/Type layer).
  - Tách file/module theo đúng trách nhiệm đơn (Single-Responsibility Principle), **tuyệt đối không nhồi nhét thêm hàm lớn vào các file lớn (God-files)**.
- Lập kế hoạch giải pháp (`S3`) kèm **Code Demo** chi tiết và dừng chờ `APPROVE` của User trước khi thực thi (`S4`).

## 2. Ràng buộc Thực thi (Technical Constraints trích xuất từ Lessons `INDEX.md`)
- **`L004` & `L020` (Explicit Runtime Call-Site & End-to-End Callchain Tracing):** Mọi hàm tích hợp mới trong `09_tasks_solution` bắt buộc phải kèm bảng ánh xạ **Chạy ở đâu (`caller file:line`) + Chạy khi nào (`Lifecycle Stage / Event Trigger`)** xuyên suốt từ Frontend (`cdc-cms-web`) -> CMS (`cdc-cms-service`) -> NATS/Worker (`centralized-data-service`: ConnectionManager, MasterDDLGenerator, TransmuterModule, Recon).
- **`L005` (Propose-Only Discipline & Explicit Approval Gate):** Giữ nguyên chế độ Read-Only đối với source code, trình bày giải pháp duy nhất kèm Code Demo trực tiếp trên khung chat và dừng chờ `APPROVE`.
- **`L008` & `L039` (Single-Responsibility Module Split & Line-Budget Guard):** Tuyệt đối không nhồi thêm hàm lớn vào các file đã vượt ngưỡng (`transmuter.go`, `master_ddl_generator.go`, `connection_manager.go`, `recon_smoke.go`). Tách các file chuyên biệt cho MySQL và ClickHouse theo đúng khuôn mẫu `*_mongo.go` (`master_ddl_generator_mysql.go`, `master_ddl_generator_clickhouse.go`, `transmuter_mysql.go`, `transmuter_clickhouse.go`, `mysql_type_converter.go`, `clickhouse_type_converter.go`, `connection_manager_mysql_ch.go`, v.v.).
- **`L028` (Batch I/O Boundary Tracing):** Chỉ đặt OpenTelemetry `ChildSpan` tại ranh giới I/O theo lô (`bulkUpsertMySQLMaster`, `bulkUpsertClickHouseMaster`), kế thừa `ctx` từ cha.
- **`L033` (Zero Runtime DDL & No Full-Table Scans):** Không chạy DDL trong constructor hay vòng lặp worker; chỉ chạy DDL trong `MasterDDLGenerator.Apply` khi rule được approve.
- **`L034` & `L045` (Multi-Tenant Identity Tuple & Spec-Driven PK/Timestamp Resolution):** Định tuyến kết nối động theo `master_connection_key`; giữ nguyên cơ chế phân giải `pk` và `timestamp_field` từ `transform_spec` của bảng Master (cấm auto-detect ngầm).
- **`L037` & `L038` (Immutable Raw Data, UTC Timezone & OCC `_source_ts`):** Bảo toàn `_raw_data`, chuẩn hóa thời gian về UTC, và kiểm soát đồng quy lạc quan (OCC) theo `_source_ts` (kèm dung sai lệch đồng hồ `2000ms`) + khử trùng lặp theo khóa nguồn trong cùng lô trước khi ghi xuống MySQL và ClickHouse.
- **`L046` (Recon Window Alignment & Prefetch Freeze-Cutoff):** Hỗ trợ đếm dòng và đối soát cửa sổ lùi (`cutoffHi`) cho cả đích Master MySQL và Master ClickHouse mà không quy chụp lỗi timeout thành `drift`.
- **`L048` & `L051` (React Rules of Hooks & Cascading Selectors):** Trên `cdc-cms-web`, khai báo mọi React Hooks ở Top-Level và mở rộng bộ chọn/hiển thị kiểu dữ liệu Master cho `mysql` và `clickhouse`.
