# 13_analysis_shadow_trigger_master_index.md
# BÁO CÁO PHÂN TÍCH GỐC RỄ VÀ TƯ DUY PHẢN BIỆN KỸ THUẬT (ROOT CAUSE & ADVERSARIAL AUDIT)

---

## I. BỐI CẢNH & NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE ANALYSIS)

### 1. Vấn đề 1: Tại sao Snapshot ở `shadow_traitestmongodevct.trans_his` lại bị `_gpay_id = NULL` trong khi `shadow_traitestces.export_jobs` vẫn có?

- **Truy vết mã nguồn**:
  1. Khi một bảng Shadow được khởi tạo qua flow tự động của CMS (`ShadowAutomator.EnsureShadowTableV2`):
     - Hàm `attachSonyflakeTrigger` được gọi (`shadow_automator.go:120`).
     - Trigger `trg_<table_name>_sonyflake_fallback` được tạo trên bảng shadow.
     - Khi ingest dữ liệu (snapshot hoặc cdc), nếu `_gpay_id` không được cung cấp, Trigger PostgreSQL tự động sinh ID qua `gen_sonyflake_id()`.
  2. Bảng `export_jobs` được tạo qua flow chuẩn này nên sequence và trigger vẫn tồn tại -> `_gpay_id` có giá trị.
  3. Bảng `trans_his` đã bị User **DROP TABLE**, sau đó User bấm nút `Create` trên cột Shadow Actions tại `TableRegistry.tsx`:
     - Frontend gọi API `POST /api/v1/source-objects/:id/create-default-columns`.
     - NATS bắn lệnh `cdc.cmd.create-default-columns`.
     - Worker CDS xử lý qua `HandleCreateDefaultColumns` (`schema_ddl_handler.go:176-240`):
       * Bước 1: `CreateEmptyTable` -> Tạo schema và table rỗng.
       * Bước 2: `EnsureCDCColumnsInSchema` -> Thêm các cột metadata `_gpay_id`, `_source_id`, `_raw_data`, `_source_ts`, `_updated_at`, `_deleted` và các index.
       * Bước 3: `AddPrimaryKeyColumn` & `AddPrimaryKeyConstraint`.
       * **KHIẾM KHUYẾT CHÍ MẠNG**: Toàn bộ quy trình `HandleCreateDefaultColumns` **HOÀN TOÀN KHÔNG CÓ BƯỚC GẮN TRIGGER SONYFLAKE**.
     - Hậu quả: Bảng `trans_his` được tạo lại trơ trụi không có trigger fallback. Khi Debezium snapshot đổ 4,683 dòng vào PostgreSQL, PostgreSQL không có trigger để sinh `_gpay_id` -> Cả 4,683 dòng đều có `_gpay_id = NULL`.
     - Khi Transmute chạy với điều kiện `WHERE _gpay_id > 0`, câu query scan qua nhưng bỏ qua toàn bộ các dòng có `_gpay_id IS NULL`, dẫn tới chỉ quét được 11 dòng (các dòng từ lần test trước).

---

### 2. Vấn đề 2: Tại sao Master Table không có Index / Index đề xuất biến mất?

- **Truy vết mã nguồn**:
  1. **Frontend Chặn Cứng (`TableIndexManager.tsx:127`)**:
     - Trong `TableIndexManager.tsx`, khối code tính toán đề xuất index bị bọc trong:
       `if (plane === 'shadow') { ... }`
     - Khi component này được nhúng vào `MasterMappingFieldsPage.tsx:930` với prop `plane="master"`, điều kiện này trả về `false`.
     - Toàn bộ các đề xuất cho `_deleted`, `_source_ts`, và cả `backendRecommendations` đều bị bỏ qua -> Trả về mảng rỗng `[]`.
  2. **Worker Hardcode Default Connection (`index_handler.go:58, 116, 181`)**:
     - Khi `TableIndexManager` gọi API `/api/introspection/indexes/:table`, NATS bắn `cdc.cmd.introspect-indexes`.
     - Handler `HandleIntrospectIndexes` trong Worker lấy database qua:
       `targetDB, err = h.connMgr.GetMasterDB(ctx, "default")`
     - Khi bảng Master được cấu hình nằm trên kết nối riêng (ví dụ `postgres-develop` hoặc `gpay-postgres-master-2`), Worker kết nối nhầm vào DB default (`goopay_dest` trên port 5434).
     - Bảng không tồn tại trên DB default -> `indexManager.ListIndexes` trả về 0 index!
  3. **Backend Recommendations Bị Lệch Cột (`index_manager.go:171`)**:
     - `index_manager.go` tự động đề xuất tạo index trên `_source_id`.
     - Bảng Master **không hề có cột `_source_id`** (Master được thiết kế tinh gọn: `_gpay_id` PK, `_source_ts`, `_deleted`, `_updated_at` và business columns).
     - Đề xuất tạo index trên cột không tồn tại sẽ gây lỗi runtime khi người dùng bấm tạo.

---

## II. TƯ DUY PHẢN BIỆN: AUDIT CÁC RỦI RO & THIẾU SÓT (ADVERSARIAL AUDIT)

Sau khi rà soát kỹ lưỡng toàn bộ bản kế hoạch và các đoạn demo code, chúng tôi phát hiện và khắc phục 5 điểm rủi ro:

### 1. Rủi ro Nuốt Lỗi (Silent Failure) trong `EnsureCDCColumnsInSchema`
- **Phát hiện**: Trong bản nháp ban đầu, dòng code `_ = sa.EnsureSonyflakeTrigger(...)` đã bỏ qua lỗi trả về.
- **Phản biện**: Nếu việc tạo sequence hoặc trigger gặp lỗi (ví dụ do lock timeout hoặc permission), việc bỏ qua lỗi sẽ khiến bảng tiếp tục ở trạng thái không có trigger mà không ai hay biết. Lỗi cũ sẽ tái diễn âm thầm!
- **Khắc phục**: BẮT BUỘC kiểm tra lỗi:
  ```go
  if err := sa.EnsureSonyflakeTrigger(ctx, schemaName, tableName); err != nil {
      return fmt.Errorf("ensure sonyflake trigger failed: %w", err)
  }
  ```

### 2. Rủi ro Multi-Tenant / Schema Isolation với Sequence `fencing_token_seq`
- **Phát hiện**: Hàm `EnsureSonyflakeTrigger` tạo sequence `fencing_token_seq` bên trong chính schema của shadow (`<schemaName>.fencing_token_seq`).
- **Phản biện**: Liệu nhiều bảng trong cùng 1 shadow schema dùng chung 1 sequence có gây xung đột không?
- **Đánh giá**: HOÀN TOÀN KHÔNG XUNG ĐỘT. Đây chính là kiến trúc A3 Hybrid đã được chuẩn hóa trong `shadow_automator.go:111`: "One function and sequence per shadow schema is reused by every table inside that schema". Sonyflake sử dụng `nextval` để lấy chuỗi tuần tự cho 6 bits cuối của ID. Nhiều bảng trong cùng schema gọi `nextval` chỉ giúp đảm bảo các ID không bao giờ trùng nhau trong cùng 1 millisecond.

### 3. Rủi ro Đề xuất Index Sai Cột trên Master Table
- **Phát hiện**: Worker `index_manager.go` đề xuất `_source_id` một cách vô điều kiện.
- **Phản biện**: Master table dùng `_gpay_id` làm Primary Key (unique), không lưu `_source_id`. Nếu đề xuất `_source_id`, người dùng bấm tạo index sẽ nhận lỗi PostgreSQL: `column "_source_id" does not exist`.
- **Khắc phục**: Truyền `plane` vào `GetRecommendations`. Nếu `plane == "master"`, chỉ đề xuất `_updated_at`, `_source_ts`, `_deleted` và timestamp field nghiệp vụ. Tuyệt đối không đề xuất `_source_id` cho Master.

### 4. Rủi ro Fallback Connection khi Client Không Truyền `connection_key`
- **Phát hiện**: Nếu phiên bản frontend cũ hoặc client API gọi thẳng mà không truyền `connection_key`, Worker sẽ làm gì?
- **Phản biện**: Nếu rỗng mà fallback về `"default"` thì Master Table trên custom connection vẫn bị lỗi 0 index.
- **Khắc phục**: Trong `resolveDB`, nếu `connKey == ""` hoặc `"default"`, Worker tự động tra cứu bảng `cdc_system.master_binding` theo `table` / `schema.table` để lấy đúng `connection_code` từ `connection_registry`. Nhờ đó, tính tương thích ngược (backward compatibility) đạt 100%.

### 5. Rủi ro SQL Injection qua Tên Schema/Table trong DDL Trigger
- **Phát hiện**: Các câu lệnh `CREATE OR REPLACE FUNCTION` và `CREATE TRIGGER` sử dụng nội suy chuỗi SQL.
- **Phản biện**: Nếu tên table hoặc schema chứa ký tự đặc biệt có thể dẫn tới lỗi cú pháp hoặc SQL injection.
- **Khắc phục**: Sử dụng hàm `sqlutil.QuoteIdent` bao bọc tất cả identifier (`schemaName`, `tableName`, `triggerName`). Riêng đối với literal string trong `nextval('schema.fencing_token_seq')`, chuỗi được chuẩn hóa an toàn.

---

## III. KẾT LUẬN & ĐÁNH GIÁ ARCHITECTURE
- Giải pháp hoàn toàn tuân thủ **Simplicity First** và **Minimal Impact**:
  - Không thêm thư viện ngoài.
  - Không thay đổi cấu trúc bảng hay luồng dữ liệu CDC.
  - Sửa đúng vào gốc rễ: gắn trigger tự động tại adapter schema, và thông suốt luồng `connection_key` cho Index Manager.
