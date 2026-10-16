# 08_tasks: PHÂN TÁCH PIPELINE SPAN THEO NGUỒN ĐA KẾT NỐI & CHUẨN HÓA ĐỊNH DANH TRỰC QUAN

## Phase 1: Mở Rộng Dữ Liệu Tuyến Đường Nguồn trong Recon
- [x] Task 1.1: Mở rộng `MasterBindingRef` trong `internal/service/recon/recon_engine_segment_b.go`:
  - Thêm các trường: `SourceEngine`, `SourceConnectionKey`, `SourceDB`, `SourceTable`.
  - SELECT thêm các trường này từ `sor` và `cr_src` trong `ListActiveMasterBindings(ctx)`.

## Phase 2: Thay Thế Cơ Chế Gom Nhóm Span Trong Recon Smoke
- [x] Task 2.1: Trong `internal/service/recon/recon_smoke.go`:
  - Tạo struct `PipelineFlowInfo` với hàm `FormatSpanName() string` tạo định dạng:
    `pipeline: [<SourceEngine>:<SourceConn>.<SourceDB>.<SourceTable>] -> [Shadow:<ShadowSchema>.<ShadowTable>] -> [Master:<MasterConn>:<MasterSchema>.<MasterTable>]`.
  - Xóa bỏ việc dùng `TargetTable` / `ShadowTable` làm map key cho span context (`tableCtxs[e.TargetTable]`).
  - Thay bằng `pipelineCtxs[pipelineKey]` với `pipelineKey` là khóa duy nhất: `e.QualifiedTarget()` (hoặc `r.ShadowRel()` = `<ShadowSchema>.<ShadowTable>`).
  - Khi có 2 pipeline cùng có tên bảng `trans_his` nhưng thuộc 2 schema/kết nối khác nhau, chúng sẽ có 2 `pipelineKey` khác nhau và sinh ra **2 span độc lập** thay vì bị gộp chung.
  - Cập nhật `ScanTarget.PipelineKey`: gán `PipelineKey` thay vì `TableKey`.
  - Cập nhật các vị trí chạy prefetch và `executeGo` (Segment A và Segment B) để lấy context từ `pipelineCtxs[pipelineKey]`.

## Phase 3: Kiểm Thử & Nghiệm Thu
- [x] Task 3.1: Viết Unit Test `test/internal/service/recon_pipeline_span_test.go`:
  - Test 2 pipeline trùng tên bảng `trans_his` nhưng khác `ShadowSchema` / `SourceConnection`: Xác nhận sinh ra 2 span riêng biệt, không bị trùng đè.
  - Test định dạng chuỗi: Đủ 3 trạm (Mongo $\rightarrow$ Shadow $\rightarrow$ Master), Segment A only, Segment B only.
- [x] Task 3.2: Static Verification & Compiler Check (DoD G3):
  - Viết suite test hoàn chỉnh tại `test/internal/service/recon_pipeline_span_test.go`.
  - Thực hiện Adversarial Line-by-Line Static Review đối soát từng kiểu dữ liệu, scope biến và chữ ký hàm.

