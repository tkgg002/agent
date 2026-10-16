# 01_requirements_recon_multiconn_isolation.md
## Yêu cầu Kỹ thuật: Khắc phục Lỗi Hiển thị Nhật ký Đối soát (Auto / Manual Recon) và Cô lập Đa Kết nối cho trans_his

---

### 1. Bối cảnh & Vấn đề Phát sinh
1. **Lỗi 1 (Tab Auto recon mất chặng Source -> Shadow):**
   - Tại màn hình Data Integrity / Recon Pipeline, khi mở Drawer "Nhật ký đối soát (30 phiên gần nhất)", tab "Auto recon" không hiển thị bất kỳ dòng nào của chặng `Source → Shadow` (Segment A), chỉ hiển thị chặng `Shadow → Master` (Segment B).
2. **Lỗi 2 (Tab Manual recon không hiển thị kết quả):**
   - Khi người dùng bấm "Bắt đầu đối soát" thủ công cho chặng Ingest hoặc Transmute, kết quả chạy xong không xuất hiện trong tab "Manual recon".
3. **Lỗi 3 (Toàn bộ API của trans_his không phân biệt 2 source connector):**
   - Hệ thống có 2 source connector cùng trỏ tới collection/table `trans_his` (`traitestctphs` với schema `shadow_traitestctphs` và `traitestmongodevct` với schema `shadow_traitestmongodevct`).
   - Các API load báo cáo (`GET /api/reconciliation/report`, `GET /api/reconciliation/report/:table`, `GET /api/reconciliation/jobs/active`) hoặc gom nhóm bảng theo tên trần `target_table`, hoặc không phân lập theo `shadow_schema` và `source_connection_code`, dẫn đến tình trạng 2 pipeline đè lên nhau, hiển thị sai lệch thông tin và mất khả năng giám sát độc lập.

---

### 2. Yêu cầu Chi tiết (Specs & Definition of Done)

#### Req-1: Sửa triệt để mệnh đề WHERE trong `GetTableHistory` (Backend CMS)
- Khi query bảng `cdc_system.cdc_recon_smoke_result` (Auto recon) và `cdc_system.cdc_reconciliation_report` (Manual recon):
  - Segment A (`source_shadow`): Record có `shadow_schema` và `shadow_table`, `master_table` là NULL/rỗng.
  - Segment B (`shadow_master`): Record có `shadow_schema`, `shadow_table`, và `master_table`.
- **Yêu cầu:** Mệnh đề WHERE phải tách biệt hoàn toàn 2 nhánh:
  - Nhánh Segment A: Khớp `shadow_schema` và `shadow_table` (với `segment != 'shadow_master'`). CẤM áp điều kiện `AND master_table = ?` lên nhánh này.
  - Nhánh Segment B: Khớp `shadow_schema` và `master_table` (với `segment = 'shadow_master'`).
- Đảm bảo thứ tự và số lượng placeholders `?` trong SQL khớp 100% với mảng `args`.

#### Req-2: Loại bỏ rào cản Master Binding đối với Segment A Smoke Check (CDS Worker Engine)
- Hàm `validatePipelineConnections` trong `recon_engine_run.go` hiện tại bắt buộc phải có `master_binding` với `schema_status = 'approved'`.
- **Yêu cầu:** Segment A là chặng Ingest độc lập (Source $\rightarrow$ Shadow). `validatePipelineConnections` chỉ được kiểm tra `source_conn_id > 0` và `shadow_conn_id > 0`. Không được chặn Segment A khi Master Binding chưa có hoặc chưa approved.

#### Req-3: Chuẩn hóa Định danh Qualified Target Table cho `recon_jobs` (CDS Worker Handler)
- Khi khởi tạo job thủ công trong `recon_check_handler.go`:
- **Yêu cầu:** Ghi `TargetTable` vào `cdc_system.recon_jobs` dưới dạng Qualified FQN (`<shadow_schema>.<shadow_table>`) nếu có `ShadowSchema`.
- Giúp backend CMS `GetActiveReconJobs` lọc chính xác job của từng connector, không bị lẫn lộn giữa các connector có cùng tên bảng.

#### Req-4: Phân lập Đa Kết Nối Xuyên Suốt Toàn Bộ API và UI cho `trans_his`
- **API `GET /api/reconciliation/report`:**
  - Chuyển `LEFT JOIN cdc_table_registry reg` sang `LEFT JOIN LATERAL (...) reg ON TRUE` có `LIMIT 1` để triệt tiêu nguy cơ duplicate dòng khi 2 connector có cùng tên bảng `trans_his`.
  - Cung cấp chính xác `source_connection_code` từ `active_bindings sb` theo đúng `shadow_schema`.
- **Frontend `ReconPipelineGrid.tsx` & `DataIntegrity.tsx`:**
  - Khóa gom nhóm `uniqueMap` trong `buildPipelines` phải bao gồm `source_connection_code` và `shadow_schema`.
  - Không dùng map key tên bảng trần `target_table` đơn lẻ khi tìm kiếm report dòng hành động.
