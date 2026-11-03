# 01 Requirements: Audit & Fix Scan Fields PostgreSQL (`cmd-scan-fields` / `scan-fields`)

tier: T2
suffix: scan_fields_postgres
date: 2026-10-28

## 1. Yêu cầu của User
- Kiểm tra (Audit) lỗi khi chạy `scan-fields` (`cmd-scan-fields`) cho bảng PostgreSQL nguồn `core-trans-proxy-history-service.transaction_history` → `shadow_devcoretrans.transaction_history`:
  ```text
  16:30:15 28/10/2026	cmd-scan-fields	
  core-trans-proxy-history-service.transaction_history
  shadow_devcoretrans.transaction_history
  error	-	-	nats_command	-	SQL source returned 0 columns or connection failed
  16:30:15 28/10/2026	scan-fields	
  core-trans-proxy-history-service.transaction_history
  shadow_devcoretrans.transaction_history
  error	-	-	nats_command	-	SQL source returned 0 columns or connection failed
  ```
- Trả lời câu hỏi:
  1. **Hàm (`func`) xử lý `scan-fields` / `cmd-scan-fields` cho PostgreSQL nằm ở đâu (`file:line`)?** Tại sao nó trả về lỗi `"SQL source returned 0 columns or connection failed"`?
  2. **Thiết kế giải pháp theo định hướng của User:** *"cái này ko cần scan mà chỉ cần detect ddl ra là lượm đc rồi"* — đối với nguồn SQL/PostgreSQL, không cần quét dữ liệu mà chỉ cần detect/introspect DDL schema (`information_schema.columns` / catalog DDL của bảng nguồn hoặc kết nối nguồn) chuẩn xác để lấy danh sách cột + kiểu dữ liệu.
- **Yêu cầu bổ sung (Turn 2 — Master MongoDB 1-Connection-1-Database & UI/Backend Sync):**
  - *"1 . ở master mongo: khi tao master chọn mogo thi các input ko đôi label qua chuẩn cho mông là data / colection , và khi chọn database (vi dụ : master_core_trans_proxy_history_service ) thì ở mongo đang ko tự tạo. => hiện tại thì tao thấy lúc tạo Kết Nối [mongo] có chỗ nhập database, với postgress thì data-> schema->table, cong mongo thi chỉ có data -> colection. giải pháp tao đưa ra là 1 kết nối master mongo chi có 1 database. và lúc tạo table master, mày chọn nó luôn, ko cho tự nhập nếu master chọn là mongo"*
  - Cụ thể:
    1. **Quy ước Kiến trúc Master MongoDB:** Với PostgreSQL là `Database -> Schema -> Table`, còn với MongoDB là `Database -> Collection` (không có tầng Schema trung gian). **Mỗi kết nối Master MongoDB (`cdc_system.connection_registry`) chỉ gắn cố định với 1 Database (`default_database`)**.
    2. **UI `Create Master` Modal (`cdc-cms-web/src/pages/MasterRegistry.tsx`):**
       - Khi `Target Database Connection` được chọn có `engine_type === 'mongodb'`:
         - Đổi nhãn (label) sang chuẩn thuật ngữ MongoDB:
           - `Master Schema` → **`Master Database (MongoDB)`**
           - `Master Table Name` → **`Master Collection Name`**
           - Đổi tiêu đề modal (`Create Master Collection (MongoDB)`) và các placeholder/helper text để không còn hardcode `"PostgreSQL"`.
         - **Tự động gán (`form.master_schema = selectedMasterConn.default_database`) và khóa cứng (`disabled={true}`)** ô `Master Database (MongoDB)` theo `default_database` của kết nối MongoDB đã chọn; tuyệt đối không cho phép tự nhập hoặc giữ nguyên giá trị gợi ý kiểu PG schema (`master_<source_db>` như `master_core_trans_proxy_history_service`).
         - Khi đổi từ kết nối MongoDB về lại kết nối PostgreSQL, tự động mở khóa (`disabled={false}`) và khôi phục gợi ý `master_<source_db>`.
    3. **Backend (`cdc-cms-service` & `centralized-data-service`):**
       - Trong `CreateMasterHandler.Handle` (`cdc-cms-service/internal/app/commands/master/create_master.go:155-183`): Khi kết nối đích có `engine_type == 'mongodb'`, cưỡng chế lấy `default_database` của kết nối làm cả `master_database` và `master_schema` (`physical_table_fqn = <default_database>.<master_name>`) khi lưu vào `cdc_system.master_binding`.
       - Trong `UpdateMasterConnection` (`master_repo_gorm.go:750-852`) và `MasterDDLGenerator.applyMongoCollectionAndIndexes` (`centralized-data-service/internal/service/master/master_ddl_generator.go:492-580`): Đồng bộ `master_database` và `master_schema` của các bản ghi `master_binding` thuộc kết nối MongoDB khớp 100% với `default_database` thực tế mà MongoDB tạo collection (ví dụ `devtest.transaction_history` của bản ghi `id=60`).

## 2. Ràng buộc Thực thi từ Bài học (Technical Constraints — S0 Boot)
- **`L003` (`Propose-Only Discipline & Explicit Approval Gate`):** Giữ chế độ **100% Read-Only (Propose-Only)** trên toàn bộ source code `data-hub` khi đang ở bước Audit/Solution (`S1–S3`). Hiển thị trực tiếp **Code Demo** trên khung chat và dừng chờ lệnh `APPROVE` tường minh từ User trước khi sửa code.
- **`L002` & `L018` (`Explicit Runtime Call-Site & End-to-End Callchain Tracing`):** Truy vết đầy đủ chuỗi gọi hàm từ CMS (`cdc-cms-service`) publish lệnh NATS `scan-fields` → Consumer NATS tại `centralized-data-service` → hàm thực thi introspect/detect DDL PostgreSQL → lưu `field_mappings` / `shadow_schema`, và chuỗi tạo Master MongoDB từ `MasterRegistry.tsx` → `POST /api/v1/masters` (`create_master.go`) → `Approve` → `MasterDDLGenerator.applyMongoCollectionAndIndexes`.
- **`L009` (`Pre-Plan Adversarial Critique Wave DG0`):** Kiểm tra thực tế cấu trúc bảng `cdc_system.source_object_registry`, `cdc_system.shadow_binding`, `cdc_system.connection_registry`, `cdc_system.master_binding` và chạy `laya_gate.py run --stage dg0` trước khi lập `09_tasks_solution`.
- **`L020` (`Direct-Answer Summary First & 1-to-1 Finding Mapping`):** Mở đầu câu trả lời bằng khối **Direct Summary** (2–4 câu) chỉ rõ ngay `file:line` và nguyên nhân lỗi, ánh xạ 1-1 từng phát hiện `F1..Fn` sang giải pháp.
- **`L031` (`Schema Verification & ANSI SQL Type Normalization`):** Khi detect DDL từ `information_schema.columns` của PostgreSQL, đảm bảo phân giải đúng `(schema, table)` và chuẩn hóa kiểu dữ liệu ANSI SQL (`data_type`, `udt_name`).
- **`L043` (`Unconditional Top-Level Hook Execution & Early-Return Discipline`):** Trong `MasterRegistry.tsx`, mọi hook (`useMemo`, `useEffect`) tính toán `selectedMasterConn` và đồng bộ `master_schema` theo `default_database` của kết nối MongoDB bắt buộc khai báo ở top-level của component, tuyệt đối không đặt trong nhánh điều kiện JSX.
