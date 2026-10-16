# 12_implementation_plan: KHẮC PHỤC TRIỆT ĐỂ LỖI ĐỊNH DANH ROUTE, OVERRIDE SEGMENT B & KIỂM THỬ THỰC TẾ TRÊN DATABASE THẬT CDC_DW

## 1. Bối cảnh & Nhận thức Sai sót (Forensic Reality Check)
- **Vấn đề User chấn chỉnh:** *"test bằng mockup và suy diễn kết quả à."*
- **Thực tế:** File test `recon_pipeline_span_test.go` trước đó hoàn toàn là **Unit Test Mock in-memory**: tự tạo struct `PipelineFlowInfo` trong RAM rồi assert format chuỗi và length của map. Nó **KHÔNG hề chạy qua `CheckAllUnified` thật**, **KHÔNG kết nối vào database `cdc_dw` thật**, và **KHÔNG trích xuất spans thực tế từ OpenTelemetry Tracer**.
- **Hậu quả nghiêm trọng:** Test mock xanh lè (PASS 5/5) đã che giấu 3 lỗi runtime thực tế:
  1. `recon_smoke.go:848`: Segment A gọi `rc.metadata.ResolveTargetRoute(e.TargetTable)` với tên bảng trần `"trans_his"`, khiến 2 pipeline cùng có bảng `trans_his` sẽ bốc nhầm route của pipeline nạp sau.
  2. `recon_smoke.go:881`: Segment B guard `if flow.SourceEngine == ""` khiến dữ liệu nguồn chính xác từ query SQL JOIN của `validRefs` bị bỏ qua nếu Segment A đã nạp `SourceEngine`.
  3. `metadata_registry_service.go:403`: `ListTableConfigs()` duyệt qua `targetCache` (chứa cả tên ngắn và qualified) khiến danh sách bảng V2 bị nhân đôi, làm Segment A chạy 2 lần cho cùng 1 bảng.

---

## 2. Kế hoạch Sửa đổi Mã nguồn Runtime

### A. Tệp tin `centralized-data-service/internal/service/recon/recon_smoke.go`
1. **Sửa Route Lookup Segment A (Dòng 848):**
   Ưu tiên phân giải theo khóa định danh duy nhất `e.QualifiedTarget()` trước, nếu không có mới fallback về `e.TargetTable`:
   ```go
   if rc.metadata != nil {
       route := rc.metadata.ResolveTargetRoute(e.QualifiedTarget())
       if route == nil {
           route = rc.metadata.ResolveTargetRoute(e.TargetTable)
       }
       if route != nil {
           srcConn = route.SourceConnectionKey
       }
   }
   ```
2. **Sửa Bù đắp Dữ liệu Nguồn từ Segment B (Dòng 881–887):**
   Xóa bỏ điều kiện chặn `flow.SourceEngine == ""`. Cập nhật độc lập từng trường từ `r` (vốn là dữ liệu JOIN SQL chính xác 100% từ `cdc_system.master_binding`):
   ```go
   if r.SourceConnectionKey != "" {
       flow.SourceConnectionKey = r.SourceConnectionKey
   }
   if r.SourceDB != "" {
       flow.SourceDB = r.SourceDB
   }
   if r.SourceTable != "" {
       flow.SourceTable = r.SourceTable
   }
   if r.SourceEngine != "" {
       flow.SourceEngine = r.SourceEngine
   }
   ```
3. **Phòng chống Rò rỉ Traces (Dòng 889–910):**
   Thêm `defer` gọi `pipelineEndFns` ngay sau khi tạo spans để bảo đảm spans luôn được đóng kể cả khi luồng chính bị panic.
4. **Deduplication Guard cho `validEntries` (Dòng 803):**
   Thêm map kiểm tra `seen[e.QualifiedTarget()]` để ngăn chặn việc chạy duplicate checks trong Segment A.

### B. Tệp tin `centralized-data-service/internal/service/source/metadata_registry_service.go`
1. **Sửa `ListTableConfigs()` (Dòng 403):**
   Duyệt qua `rs.idCache` (vốn định danh duy nhất theo `src.ID`) thay vì `rs.targetCache` (vốn chứa duplicate keys cho bảng V2):
   ```go
   func (rs *MetadataRegistryService) ListTableConfigs() []source.TableRegistry {
       rs.mu.RLock()
       defer rs.mu.RUnlock()
       out := make([]source.TableRegistry, 0, len(rs.idCache))
       for _, item := range rs.idCache {
           if item == nil { continue }
           out = append(out, *item)
       }
       return out
   }
   ```

---

## 3. Kế hoạch Kiểm thử Thực tế trên Database Thật `cdc_dw` (Real Integration Test)

Thay vì test mock in-memory, viết kịch bản Integration Test kết nối trực tiếp vào container PostgreSQL `gpay-postgres-cdc` (`postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw`):
1. **Khởi tạo kết nối DB thật:** Dùng GORM mở kết nối tới database `cdc_dw` trên port 5433 (nơi chứa sẵn dữ liệu thật của 2 binding trùng bảng `trans_his`: Binding #55 và Binding #45).
2. **Khởi tạo `MetadataRegistryService` & `ReconCore` thật:** Tải dữ liệu metadata thật từ database.
3. **Cài đặt OpenTelemetry In-Memory Tracer:** Đăng ký `tracetest.NewInMemoryExporter()` vào OTel TracerProvider để hứng toàn bộ spans sinh ra.
4. **Kích hoạt Chu trình thật:** Gọi `rc.CheckAllUnified(ctx)`.
5. **Đối soát Spans thật:**
   Trích xuất spans từ exporter và kiểm tra:
   - Span của Binding #55: Phải mang đúng `SourceConnectionKey = traitestmongodevct`, Master `postgres-develop:public.cdc_transaction_history`.
   - Span của Binding #45: Phải mang đúng `SourceConnectionKey = traitestctphs`, Master `default_master:master_core_trans_proxy_history_service.trans_his`.
   - Hai spans sinh ra hoàn toàn tách biệt, không bị gộp chung và không bị ghi đè thông tin kết nối nguồn của nhau.
