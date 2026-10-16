# 09_tasks_solution_smoke_trace_and_trans_his_drift.md
## Hồ Sơ Giải Pháp Kỹ Thuật: Chuẩn Hóa Cây Span Traces Recon Smoke & Xử Lý Lệch trans_his

### 1. Phân Tích Gốc Rễ Chuyên Sâu (Root Cause Analysis)

#### Vấn đề 1: SigNoz Tracing Collision / Hierarchy Collapse
- **Mã nguồn bị lỗi:** `centralized-data-service/internal/service/recon/recon_smoke.go` (dòng 910-1115).
- **Cơ chế gây lỗi:**
  1. `pipelineFlows` được nhóm theo `key := r.ShadowRel()`. Khi một bảng Shadow (`shadow_traitestces.export_jobs`) có 3 Master Bindings (`export_jobs`, `export_jobs_1`, `export_jobs_2`), vòng lặp `validRefs` ghi đè `flow.MasterTable`, chỉ giữ lại duy nhất binding cuối cùng (`export_jobs_2`).
  2. Toàn bộ các Span cha của Pipeline (`pCtx, pSpan := observability.ChildSpan(...)`) được khởi tạo trước đồng loạt tại cùng 1 microsecond và kết thúc đồng loạt ở cuối hàm (`defer endAllPipelines()`).
  3. Các target prefetch và worker goroutine chạy bất đồng bộ. Dưới góc nhìn của OpenTelemetry Tracer và SigNoz UI, các span con chia sẻ lifecycle mập mờ, dẫn đến việc SigNoz lồng toàn bộ các span `prefetch.master` (cả 3 cái) và `shadow_master` (cả 3 cái) vào Pipeline Span đầu tiên được render (`export-jobs`).
  4. Cơ chế Advisory Lock `rc.withTableLock(ctx, entry.TargetTable)` sử dụng tên bảng trần (`"trans_his"`). Khi 2 connector của `trans_his` (`traitestctphs` và `traitestmongodevct`) chạy song song, connector thứ 2 bị `acquired = false` và âm thầm trả về `nil`, làm rơi rụng span và kết quả đối soát.

#### Vấn đề 2: Lệch Số Liệu trans-his (Source: 235,036 | Shadow: 234,756 | Master: 235,214)
- **Số liệu thực tế:**
  * Source (MongoDB): `235,036`
  * Shadow (PostgreSQL): `234,756` (Ingest: `-280`)
  * Master (PostgreSQL): `235,214` (Transmute: `+458`)
- **Giải mã toán học:**
  * `234,756 + 458 = 235,214`!
  * 458 bản ghi đã bị soft-deleted ở Shadow (`_deleted = true`).
  * Tầng Shadow tính active = `total - del = 235,214 - 458 = 234,756`.
  * Tuy nhiên, Transmuter (`transmuter.go:749`) quét Shadow theo con trỏ tăng dần `WHERE _gpay_id > cursor`. Các bản ghi bị xóa mềm này có `_gpay_id` cũ từ trước nên Transmuter không bao giờ quét qua lại chúng. Vì vậy lệnh hard-delete trên Master (`transmuter.go:1027`) không được kích hoạt cho 458 bản ghi này!
  * Master vẫn giữ nguyên 235,214 bản ghi (`_deleted = false`), tạo ra độ lệch chính xác `+458`.
  * Về phía Source: MongoDB `EstimatedDocumentCount()` đọc metadata WiredTiger có độ trôi dạt (WiredTiger internal metadata drift) lệch 280 bản ghi so với thực tế live document count.

---

### 2. Giải Pháp Kỹ Thuật Tối Ưu (The Single Best Approach)

#### Giải Pháp 1: Tái Cấu Trúc Per-Pipeline Execution & Trace Lifecycle trong `recon_smoke.go`
Thay vì mở hàng loạt Pipeline Spans ở đầu hàm rồi gom scanTargets lộn xộn:
1. **Pipeline Key Phân Lập Tuyệt Đối:**
   - Xây dựng danh sách Pipeline Items phân lập theo cặp:
     `PipeID = fmt.Sprintf("%s::%s", shadowRel, masterRel)` (hoặc `shadowRel` nếu không có master).
   - Đảm bảo mỗi master binding (ví dụ `export_jobs`, `export_jobs_1`, `export_jobs_2`) là một pipeline riêng biệt với span name trực quan riêng.
2. **Per-Pipeline Trace Isolation:**
   - Với mỗi Pipeline Item:
     * Khởi tạo span riêng: `pipeCtx, pipeSpan := observability.ChildSpan(ctx, pipelineName, ...)`
     * Chạy Segment A (nếu chưa chạy hoặc theo pipeline):
       Lấy kết quả Shadow Count (dùng cache relation-level tránh query lặp).
       Chạy `RunTotalOnlyA` trực tiếp dưới `pipeCtx`.
     * Chạy Segment B:
       Lấy kết quả Master Count (dùng cache relation-level).
       Chạy `RunTotalOnlyB` trực tiếp dưới `pipeCtx`.
     * Đóng `pipeSpan` ngay khi pipeline đó hoàn tất (`observability.EndSpan(pipeSpan, nil)`).
   - **Kết quả trên SigNoz:** Mỗi Pipeline Span là một cây độc lập 100%, chứa chính xác `source_shadow`, `prefetch`, `shadow_master` của chính nó. Không còn hiện tượng 1 pipeline ôm hết con và các pipeline khác bị rỗng!

#### Giải Pháp 2: Khắc Phục Advisory Lock Cho Multi-Connector
1. Trong `RunTotalOnlyA`:
   - Đổi `rc.withTableLock(ctx, entry.TargetTable)` thành `rc.withTableLock(ctx, entry.QualifiedTarget())`.
2. Trong `RunTotalOnlyB`:
   - Đổi `rc.withTableLock(ctx, "reconB_"+ref.runName())` thành `rc.withTableLock(ctx, "reconB_"+ref.ShadowRel()+"_"+ref.MasterRel())`.

#### Giải Pháp 3: Khắc Phục Lệch Tombstone Trên Master (+458) & Phân Biệt Estimated Count
1. **Đồng bộ Tombstone Soft-Delete sang Master:**
   - Bổ sung hàm Sync Tombstones trong Transmuter hoặc Recon Heal:
     Truy vấn các `_gpay_id` có `_deleted = true` trong Shadow:
     `SELECT _gpay_id FROM <shadow_table> WHERE _deleted = true`
     Và thực thi hard-delete trên Master:
     `DELETE FROM <master_table> WHERE _gpay_id IN (...)`
   - Ngay khi 458 dòng này được xóa khỏi Master, số lượng active trên Master sẽ trở về chính xác `234,756`, khớp 100% với Shadow!
2. **Bổ Sung Chú Thích & Exact Verify Cho Source:**
   - Ghi chú rõ trên UI và Span attributes: `source.count_method = "mongo_estimated_metadata"`.
   - Cung cấp tùy chọn Exact Count (`CountDocuments`) khi người dùng kích hoạt đối soát chuyên sâu (Tier-A / Manual Recon).
