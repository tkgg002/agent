# 13_analysis_recon_multiconn_isolation.md
## Phân tích Chuyên sâu: Nguyên nhân Gốc rễ Lỗi Mất Chặng Source->Shadow và Gộp Nhầm Multi-Connector

---

### 1. Phân tích Hiện tượng 1 & 2: Mất kết quả Source -> Shadow ở cả Auto recon và Manual recon
1. **Truy vết mã nguồn:**
   - Tại `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`:
     ```go
     where := "(shadow_table = ? OR master_table = ?)"
     args := []interface{}{table, table}
     if shadowSchema != "" {
         where = "((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND master_table = ?))"
         args = []interface{}{shadowSchema, table, table}
         if masterTable != "" {
             where += " AND master_table = ?"
             args = append(args, masterTable)
         }
     }
     ```
2. **Cơ chế gây lỗi:**
   - Frontend gửi `shadow_schema` và `master_table` khi xem drill-down của một pipeline có master binding.
   - Chuỗi nối chuỗi khiến câu query SQL thành:
     `WHERE ((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND master_table = ?)) AND master_table = ?`
   - Bản ghi Segment A (Source $\rightarrow$ Shadow) trong CẢ HAI bảng (`cdc_recon_smoke_result` và `cdc_reconciliation_report`) đều có `segment = 'source_shadow'` và `master_table = NULL`.
   - Điều kiện `AND master_table = ?` áp lên toàn bộ biểu thức khiến 100% bản ghi Segment A bị loại bỏ, dẫn đến tab "Auto recon" và tab "Manual recon" hoàn toàn trống trơn chặng Source $\rightarrow$ Shadow.

---

### 2. Phân tích Hiện tượng 3: Không phân biệt 2 Source Connectors cho `trans_his`
1. **Lỗi gộp chùm tại tầng Frontend:**
   - Trong `ReconPipelineGrid.tsx:buildPipelines`: Khóa dedup `uniqueMap` trước đây chỉ dựa vào `${schema}::${table}::${segment}`. Nếu `schema` bị thiếu hoặc 2 connector cùng map tới schema tương đương, chúng sẽ đè lên nhau.
   - Thêm vào đó, trong `DataIntegrity.tsx`, map `reportByTarget` chỉ dùng key là bare name `row.target_table` ("trans_his"). Khi có 2 connector cùng tên bảng, connector phía sau ghi đè connector phía trước.
2. **Lỗi duplicate hoặc cross-join tại tầng Backend:**
   - Trong `listLatestPrimary`, việc JOIN `cdc_table_registry reg` theo `target_table = s.shadow_table` (không có schema) khiến cho 1 bản ghi smoke bị nhân bản thành nhiều dòng nếu bảng `cdc_table_registry` có nhiều cấu hình trùng tên bảng đích.
   - Cần dùng `LEFT JOIN LATERAL (...) reg ON TRUE` với `LIMIT 1` để đảm bảo 1-1 mapping.
