# 01_requirements: PHÂN TÁCH PIPELINE SPAN THEO NGUỒN ĐA KẾT NỐI & CHUẨN HÓA ĐỊNH DANH TRỰC QUAN

## 1. Vấn đề gốc rễ (Root Cause)
- Khi hệ thống hỗ trợ đa kết nối nguồn (Multi-connection), các nguồn khác nhau có thể chứa cùng một tên bảng / collection nguồn (ví dụ: `trans_his`).
- Trong module Reconcile Tracing (`centralized-data-service/internal/service/recon/recon_smoke.go`):
  * Code đang dùng tên bảng trần (`entry.TargetTable` và `ref.ShadowTable` = `"trans_his"`) làm map key để gom nhóm trace:
    ```go
    key := e.TargetTable
    tableCtxs[key] = tCtx // gom toàn bộ pipeline cùng tên bảng vào 1 span!
    ```
  * Hậu quả: Toàn bộ các pipeline khác nhau có cùng bảng `trans_his` bị **GOM CHÙM VÀO MỘT SPAN DUY NHẤT: `pipeline:trans_his`**.
  * Trên SigNoz / APM: Không thể phân biệt được `trans_his` này thuộc về kết nối nào, database nào, và chuyển tới master nào.

## 2. Yêu cầu chi tiết
1. **Phân tách triệt để các Pipeline (Disaggregate Pipeline Spans):**
   - Xóa bỏ hoàn toàn việc dùng `TargetTable` / `ShadowTable` ngắn làm key gom trace.
   - Mỗi pipeline phải có một định danh duy nhất dựa trên khóa tuyến đường: `<ShadowSchema>.<ShadowTable>` (ví dụ: `shadow_traitestmongodevct.trans_his` tách biệt hoàn toàn với `shadow_goopay.trans_his` hoặc các connection khác).
   - Mỗi pipeline được cấp một ChildSpan riêng biệt, không còn tình trạng đè span của nhau.

2. **Chuẩn hóa Tên Span Trực quan Đầy đủ Tuyến Đường (Full Dataflow Path):**
   - Định dạng tên span theo đúng yêu cầu:
     `pipeline: [<SourceEngine>:<SourceConn>.<SourceDB>.<SourceTable>] -> [Shadow:<ShadowSchema>.<ShadowTable>] -> [Master:<MasterConn>:<MasterSchema>.<MasterTable>]`
   - Ví dụ với bảng `trans_his` của kết nối `traitestmongodevct`:
     `pipeline: [Mongo:traitestmongodevct.gpay_core.trans_his] -> [Shadow:shadow_traitestmongodevct.trans_his] -> [Master:postgres-develop:public.cdc_transaction_history]`
   - Nếu pipeline chỉ có Segment A:
     `pipeline: [Mongo:traitestmongodevct.gpay_core.trans_his] -> [Shadow:shadow_traitestmongodevct.trans_his]`
   - Gắn đầy đủ OTel attributes cho từng span để lọc trên APM:
     * `pipeline.source_engine`, `pipeline.source_conn`, `pipeline.source_db`, `pipeline.source_table`
     * `pipeline.shadow_schema`, `pipeline.shadow_table`
     * `pipeline.master_conn`, `pipeline.master_schema`, `pipeline.master_table`

## 3. Tiêu chí hoàn thành (Definition of Done)
- [ ] G1: Không còn tình trạng gom gộp các pipeline trùng tên bảng vào 1 span `pipeline:trans_his`. Mỗi pipeline sinh 1 span độc lập.
- [ ] G2: Tên span hiển thị đầy đủ 3 trạm `pipeline: [Mongo:connect.data.collection] -> [Shadow:schema.table] -> [Master:conn:schema.table]`.
- [ ] G3: Các bước prefetch, count, RunTotalOnlyA, RunTotalOnlyB chạy dưới đúng span context của từng pipeline tương ứng.
- [ ] G4: Unit test xác thực chính xác việc sinh span name và phân tách pipeline không bị va chạm.
- [ ] G5: Biên dịch `go build ./cmd/worker` pass 100%.
