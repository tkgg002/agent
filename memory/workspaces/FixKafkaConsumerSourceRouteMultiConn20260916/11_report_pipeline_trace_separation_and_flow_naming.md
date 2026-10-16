# 11_report: TỔNG HỢP THAY ĐỔI MÃ NGUỒN PHÂN TÁCH PIPELINE TRACE & ĐỊNH DANH 3 TRẠM

## 1. Danh sách các tệp tin đã thay đổi và số lượng dòng code

| STT | Tệp tin | Trạng thái | Số dòng thay đổi (Approx) | Mục đích thay đổi |
| :--- | :--- | :--- | :--- | :--- |
| 1 | `centralized-data-service/internal/service/recon/recon_engine_segment_b.go` | MODIFY | +15 dòng | Mở rộng `MasterBindingRef` và SELECT 4 trường nguồn |
| 2 | `centralized-data-service/internal/service/recon/recon_smoke.go` | MODIFY | +120 dòng | Struct `PipelineFlowInfo`, format 3 trạm, đổi `ScanTarget.PipelineKey`, phân lập span theo `QualifiedTarget()` |
| 3 | `centralized-data-service/test/internal/service/recon_pipeline_span_test.go` | NEW | 135 dòng | Bộ unit test kiểm tra format 3 trạm và test cô lập chống gộp chùm |

---

## 2. Tổng quan chi tiết cách thức thay đổi

### 2.1. `internal/service/recon/recon_engine_segment_b.go`
- Bổ sung 4 trường vào struct `MasterBindingRef`: `SourceEngine`, `SourceConnectionKey`, `SourceDB`, `SourceTable`.
- Cập nhật câu query trong `ListActiveMasterBindings`: SELECT thêm thông tin nguồn từ `sor` (`source_object_registry`) và `cr_src` (`connection_registry`).

### 2.2. `internal/service/recon/recon_smoke.go`
- Định nghĩa struct `PipelineFlowInfo` với method `FormatSpanName() string` tự động ghép 3 trạm:
  `pipeline: [<SourceEngine>:<SourceConn>.<SourceDB>.<SourceTable>] -> [Shadow:<ShadowSchema>.<ShadowTable>] -> [Master:<MasterConn>:<MasterSchema>.<MasterTable>]`
  (ví dụ: `pipeline: [Mongo:traitestmongodevct.gpay_core.trans_his] -> [Shadow:shadow_traitestmongodevct.trans_his] -> [Master:postgres-develop:public.cdc_transaction_history]`).
- Đổi trường `ScanTarget.TableKey` thành `ScanTarget.PipelineKey`.
- Trong `CheckAllUnified`:
  * Xóa bỏ hoàn toàn việc dùng tên bảng trần `TargetTable` / `ShadowTable` làm map key.
  * Sử dụng khóa phân lập duy nhất của Pipeline: `e.QualifiedTarget()` và `r.ShadowRel()` (dạng `shadow_schema.shadow_table`).
  * Khởi tạo span độc lập cho từng pipeline với tên sinh từ `flow.FormatSpanName()`.
  * Gắn 11 OTel attributes cho từng span để tra cứu dễ dàng trên SigNoz.
  * Các bước prefetch, count và execution của Segment A & Segment B đều trỏ đúng context của pipeline đó kèm defensive fallback.

### 2.3. `test/internal/service/recon_pipeline_span_test.go`
- 5 test cases độc lập kiểm thử tính chính xác của format và tính năng phân lập 2 pipeline trùng tên bảng `trans_his` nhưng khác connection / shadow schema.
