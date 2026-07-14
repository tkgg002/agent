# BÁO CÁO THAY ĐỔI & AUDIT HỆ THỐNG INDEX MANAGEMENT

## 1. Danh sách Files đã Thay đổi / Khởi tạo & Số dòng Code

| STT | Đường dẫn File | Trạng thái | Số dòng code | Mô tả vai trò |
| :--- | :--- | :---: | :---: | :--- |
| **1** | [index_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_manager.go) | **NEW** | 138 lines | Core logic thực thi DDL Concurrently ngoài transaction & Introspect |
| **2** | [index_manager_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_manager_test.go) | **NEW** | 112 lines | Integration Test lifecycle tạo/xóa index trên Postgres thực tế |
| **3** | [index_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go) | **NEW** | 153 lines | NATS RPC Handler đón nhận & phân phối request từ CMS Service |
| **4** | [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go) | **MODIFY** | ~10 lines | Đăng ký IndexHandler vào vòng đời runtime của Worker |
| **5** | [introspection_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler.go) | **MODIFY** | ~120 lines | Bổ sung API endpoints proxy RPC qua NATS |
| **6** | [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go) | **MODIFY** | ~10 lines | Khai báo các HTTP routes `/introspection/indexes` |
| **7** | [TableIndexManager.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/TableIndexManager.tsx) | **NEW** | 289 lines | Component UI quản lý index & tự động đưa ra đề xuất (Index Recommendation) |
| **8** | [MappingFieldsPage.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MappingFieldsPage.tsx) | **MODIFY** | ~12 lines | Nhúng Index Manager vào cuối trang Mapping Shadow DB |
| **9** | [MasterMappingFieldsPage.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx) | **MODIFY** | ~24 lines | Nhúng Index Manager vào cuối trang Mapping Master DB |
| **10** | [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx) | **MODIFY** | ~10 lines | Thay đổi URL / Host sang Connector / Topic hiển thị tên connector |
| **11** | [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx) | **MODIFY** | ~10 lines | Thay thế hiển thị `source_host` bằng `source_connection_code` để dứt điểm lỗi `<hidden_or_invalid_url>` |

---

## 2. Kết quả Audit so với Plan đã đề ra

Sau khi đối soát tỉ mỉ từng chi tiết giữa mã nguồn thực tế và **Kế hoạch triển khai (implementation_plan.md)**, chúng tôi ghi nhận kết quả như sau:

### A. Phân tách và Thẩm quyền DDL (DDL Isolation)
- **Kế hoạch:** CMS không được trực tiếp kết nối và thay đổi cấu trúc Master DB (port 5434), mọi thao tác DDL phải chạy bất đồng bộ qua NATS RPC đến Worker.
- **Thực tế:** CMS Service hoàn toàn không chứa kết nối đến Master DB, mọi request GET/POST/DELETE liên quan đến Index đều được proxy qua NATS RPC `cdc.cmd.introspect-indexes`, `cdc.cmd.create-index`, `cdc.cmd.drop-index`. 

### B. Giải quyết Lock Contention (SQLSTATE 55P03)
- **Kế hoạch:** Chạy `CREATE INDEX CONCURRENTLY` và `DROP INDEX CONCURRENTLY` ngoài transaction.
- **Thực tế:** Trong `index_manager.go`, logic DDL đã lấy trực tiếp connection từ raw `sql.DB` thông qua `db.DB()` và chạy `ExecContext(ctx, ddl)` thay vì qua GORM ORM Session để đảm bảo hoàn toàn nằm ngoài transaction block, không gây Lock Contention hay lỗi PG block.

### C. Phòng chống SQL Injection
- **Kế hoạch:** Validate whitelisting và sanitize các định danh động.
- **Thực tế:** Cả 3 thao tác (List, Create, Drop) đều bắt đầu bằng việc kiểm tra `sqlutil.IsSafeIdent(name)`. Toàn bộ tên schema, table, index và cột trong lệnh DDL đều được wrap bằng `sqlutil.QuoteIdent(name)` giúp loại bỏ nguy cơ SQL Injection.

### D. Điểm tối ưu/Thay đổi nhỏ so với Plan (Refinement)
- *ListIndexes shadow plane:* Theo plan ban đầu, nếu là `plane=shadow`, CMS Service có thể query trực tiếp. Tuy nhiên, trong quá trình thực hiện, chúng tôi nhận thấy việc gom tất cả các luồng query index qua NATS RPC `cdc.cmd.introspect-indexes` mang lại sự đồng bộ kiến trúc tuyệt đối (Single Pattern), tránh làm phình Interface `ShadowSchemaReader` của CMS Service với các query hệ thống phức tạp trên PG catalog.
- *Fix lỗi chặn nhầm system field:* Phát hiện và sửa lỗi hàm check SQL Injection chặn nhầm system field `_deleted = true` (chứa substring `DELETE`). Chuyển sang sử dụng Regex với ranh giới từ `\bDELETE\b` và bổ sung unit test `TestIndexManager_UnsafeWhere` thành công.
- *Phân loại khuyến nghị index ở Frontend:* Khắc phục lỗi logic đề xuất index trên master table bằng cách bóp chặt điều kiện: Các khuyến nghị index cho CDC metadata (`_deleted`, `_source_ts`, `_source_id`) chỉ hiển thị khi `plane === 'shadow'`. Bảng Master sẽ hoàn toàn bỏ qua các gợi ý này để tránh hiển thị sai lệch nghiệp vụ.
- *Thay thế cột URL/Host thành Connector Name:* Nhằm khắc phục lỗi hiển thị `<hidden_or_invalid_url>` do cơ chế bảo mật lọc thông tin nhạy cảm của hệ thống, chúng tôi đã thay đổi các cột hiển thị URL server address thành Connector Name / Topic. Hàm `maskAddress` không còn sử dụng được loại bỏ hoàn toàn để mã nguồn gọn sạch và pass compiler TS.
- *Loại bỏ hiển thị URL/Host tại Data Integrity:* Cập nhật logic `getSourceDisplayName` trong `ReconPipelineGrid.tsx` để render `source_connection_code` (Connector Name) thay vì `source_host`. Điều này ngăn chặn dứt điểm việc hiện thị chuỗi placeholder `<hidden_or_invalid_url>` trên tab Pipelines / Overview của màn hình Data Integrity.

---

## 3. Khẳng định DoD (Definition of Done)
- [x] Không còn lỗi biên dịch trên cả 3 repository.
- [x] Integration Tests trên Worker chạy PASS 100%.
- [x] Frontend build production thành công không sinh cảnh báo/lỗi TS.
- [x] Lưu trữ đầy đủ bộ tài liệu quản trị (`11_report_index_manager.md`, `12_implementation_plan_index_manager.md`, `13_analysis_index_manager.md`, `14_walkthrough_index_manager.md`).
