# BÁO CÁO TỔNG HỢP 20 WORKSPACE MỚI NHẤT (CẬP NHẬT 05/08/2026)

> **Dự án**: CDC Data Hub (`cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`, `cdc-auth-service`)  
> **Người cập nhật**: Antigravity (AI Brain Agent)  
> **Ngày lưu trữ**: 05/08/2026  

---

## 🏛️ TỔNG QUAN HỆ THỐNG PHÂN THEO 5 NHÓM CHỨC NĂNG

### 📌 NHÓM 1: CORE ENGINE SCALING & HIGH-SCALE OPTIMIZATION

#### 1. Workspace `BatchTransformScaling20260730` (30/07 – 31/07)
- **Title:** Nâng Cấp Async Batch Transform Engine Xử Lý 50M – 500M Bản Ghi
- **Tình hình hiện tại:** NATS handler chạy đồng bộ (blocking) làm đứt kết nối NATS timeout sau 15 phút khi bảng > 50M bản ghi, hard cap 100k iterations, không có theo dõi tiến độ % và nút Cancel.
- **Mong muốn đạt được:** Engine chạy async background non-blocking, xử lý đến 500M bản ghi, có UI Progress Bar real-time & Cancel action.
- **Giải pháp:** DDL `100_create_transform_jobs.sql`, GORM repo `transform_job_repo.go`, worker async goroutine với thuật toán Adaptive Chunking trong `batch_transform_handler.go`, REST API `/transform-job-status` & UI component `TransformJobStatus` trong `TableRegistry.tsx`. Sửa BUG-03 (scope 500) & BUG-04 (multi-binding).
- **Trạng thái:** **DONE (Phase 1 & Phase 2)** | **Estimate:** **16h**

---

#### 2. Workspace `TransmuteJobScaling20260731` (31/07)
- **Title:** Nâng Cấp Engine Transmute Bulk Job & Realtime Progress Tracking Bar
- **Tình hình hiện tại:** Nút "Transmute Now" chưa có live progress bar theo dõi tiến độ và rủi ro ảnh hưởng tốc độ 15ms của luồng CDC Oplog Realtime.
- **Mong muốn đạt được:** Phân tách ranh giới (Boundary Isolation): Oplog Realtime giữ nguyên 15ms (không DB tracking); Manual Transmute kích hoạt live bar & cancel action.
- **Giải pháp:** Dùng cờ `jobID != ""`, DDL `transmute_jobs`, worker `transmuter.go` phát heartbeat progress, REST API `/transmute-job-status` & UI `TransmuteJobStatus` trong `MasterRegistry.tsx` với Polling 3s, live rows counter & Cancel button.
- **Trạng thái:** **DONE (Phase 1 & Phase 2)** | **Estimate:** **14h**

---

### 📌 NHÓM 2: DATABASE PERFORMANCE & SLOW SQL OPTIMIZATION

#### 3. Workspace `FixActivityLogSlowSql20260731` (31/07)
- **Title:** Tối Ưu Truy Vấn Slow SQL Bảng Activity Log (`cdc_activity_log`)
- **Tình hình hiện tại:** Truy vấn Stats24h & Recent Errors phản hồi chậm (200ms – 380ms) do full scan partition cũ và thiếu index đa cột.
- **Mong muốn đạt được:** Giảm thời gian phản hồi toàn bộ API Activity Log xuống < 15ms.
- **Giải pháp:** Migration `012_optimize_activity_log_indexes.sql` tạo 2 composite index, refactor GORM query thêm Partition Pruning (`created_at > NOW() - 24h`) & Subquery Pagination First.
- **Trạng thái:** **DONE** | **Estimate:** **8h**

---

#### 4. Workspace `FixSystemHealthQueriesSlowSql20260731` (31/07)
- **Title:** Tối Ưu Các Câu Truy Vấn Slow SQL Trang System Health (`/system-health`)
- **Tình hình hiện tại:** Truy vấn đếm active connectors, shadow tables & status check phản hồi chậm > 300ms.
- **Mong muốn đạt được:** API System Health tải nhanh < 20ms.
- **Giải pháp:** Migration `013_optimize_system_health_indexes.sql` bổ sung index, refactor `system_health_read_repo_gorm.go` dùng `EXISTS` check và Subquery.
- **Trạng thái:** **DONE** | **Estimate:** **10h**

---

#### 5. Workspace `FixReconReadRepoSlowSql20260731` (31/07)
- **Title:** Tối Ưu Slow SQL Reconciliation Read Repository (`recon_read_repo_gorm.go`)
- **Tình hình hiện tại:** Query lấy danh sách Recon Jobs, Smoke Tests & History bị full scan khi dữ liệu phình to.
- **Mong muốn đạt được:** Thời gian tải danh sách lịch sử đối soát đạt < 30ms.
- **Giải pháp:** Bổ sung composite index `idx_recon_jobs_status_created` và refactor GORM query sang Subquery Pagination First (CTE).
- **Trạng thái:** **DONE** | **Estimate:** **10h**

---

### 📌 NHÓM 3: RECONCILIATION ENGINE, MULTI-SEGMENT & SELF-HEALING

#### 6. Workspace `FixSmokeCheckLockTime20260724` (24/07 – 27/07)
- **Title:** Tối Ưu Độ Trễ & Giảm Thời Gian DB Lock Cho Smoke Reconciliation
- **Tình hình hiện tại:** Smoke check bị lock timeout do giữ DB connection quá lâu khi quét kiểm tra dữ liệu bảng lớn.
- **Mong muốn đạt được:** Smoke check hoàn tất siêu tốc (< 50ms), giải phóng DB connection lock ngay lập tức.
- **Giải pháp:** Tối ưu SQL scan range, thu hẹp cửa sổ quét và bọc timeout 3s an toàn trong `recon_smoke_service.go`.
- **Trạng thái:** **DONE** | **Estimate:** **8h**

---

#### 7. Workspace `FixReconHistoryLabelAndManualTask20260724` (24/07)
- **Title:** Chuẩn Hóa Nhãn Recon History & Hỗ Trợ Kích Hoạt Manual Recon Task
- **Tình hình hiện tại:** Nhãn lịch sử hiển thị nhầm lẫn giữa Smoke Check và Full Recon, không có nút kích hoạt đối soát thủ công từ UI.
- **Mong muốn đạt được:** Phân biệt rõ ràng loại Recon trên UI, hỗ trợ trigger manual Recon task với tham số tùy chỉnh.
- **Giải pháp:** Phân tách type `Smoke` vs `Full` trên UI FE, cập nhật backend handler tiếp nhận payload manual run và ghi vết audit.
- **Trạng thái:** **DONE** | **Estimate:** **8h**

---

#### 8. Workspace `stale_ids_heal_modal_fix` (23/07)
- **Title:** Sửa Lỗi Interactive Heal Stale IDs Modal & Tự Phục Hồi Dữ Liệu Chênh Lệch
- **Tình hình hiện tại:** Trên UI Modal Heal, khi chọn "Heal Selected IDs" bị văng lỗi HTTP 500 do cú pháp SQL truyền mảng sai.
- **Mong muốn đạt được:** Cho phép chọn từng ID chênh lệch để trigger Sync/Heal chính xác 100% và tự động reset state UI khi thành công.
- **Giải pháp:** Sửa API `POST /api/reconciliation/heal`, truyền slice ID với cú pháp SQL `IN (?)` chuẩn GORM (thay cho `ANY(?)` gây lỗi SQL 42601).
- **Trạng thái:** **DONE** | **Estimate:** **12h**

---

#### 9. Workspace `FixReconTraceGrouping20260722` (22/07 – 23/07)
- **Title:** Chuẩn Hóa Cấu Trúc Cây OpenTelemetry Trace Spans Cho Recon Engine & Shadow-Master Flow
- **Tình hình hiện tại:** Cây Trace Spans bị phẳng (flat), thiếu phân cấp parent-child, và luồng Segment B (`shadow_master`) thiếu đối soát với Master DB.
- **Mong muốn đạt được:** Trace Spans phân cấp 4 tầng rõ ràng (`cdc.recon.chunk_stream_bucket` -> `cdc.recon.chunk_day` -> `cdc.recon.hash_window` 15-phút -> `cdc.recon.drift_drill_down`) và hỗ trợ đầy đủ Segment B (`shadow_master`).
- **Giải pháp:** Tái cấu trúc `drillSubWindows` trong `recon_stream_bucket_engine.go` truyền `ctx` phân cấp, wire `masterAgent` vào engine, đóng gói `Segment` vào NATS event.
- **Trạng thái:** **DONE** | **Estimate:** **14h**

---

#### 10. Workspace `FixReconReport24hWindow20260731` (31/07)
- **Title:** Chuẩn Hóa Múi Giờ Cửa Sổ Lọc 24h Cho Reconciliation Report
- **Tình hình hiện tại:** Lệch múi giờ giữa Go Driver và Postgres UTC làm khoảng `NOW() - 24 HOURS` bị lệch, bỏ sót báo cáo 24h gần nhất.
- **Mong muốn đạt được:** Báo cáo Reconciliation hiển thị chính xác 100% dữ liệu 24h gần nhất theo múi giờ UTC chuẩn.
- **Giải pháp:** Refactor `GetLatest24hReport` trong `recon_report_read_repo_gorm.go` ép múi giờ UTC chuẩn và thêm index `created_at DESC`.
- **Trạng thái:** **DONE** | **Estimate:** **8h**

---

### 📌 NHÓM 4: SYSTEM HARDENING, UI/UX & DATA INTEGRITY

#### 11. Workspace `FixActivityLoggingPlan20260727` (27/07)
- **Title:** Chuẩn Hóa Luồng Ghi Activity Log Toàn Hệ Thống (Backend & Sinkworker)
- **Tình hình hiện tại:** Các thao tác quan trọng (Transmute, Transform, Snapshot) bị thiếu log vào `cdc_activity_log` hoặc thiếu Trace ID.
- **Mong muốn đạt được:** 100% tác vụ được ghi vết audit log đầy đủ kèm Trace ID, module name, status và duration.
- **Giải pháp:** Xây dựng `ActivityLogMiddleware` và helper `LogActivityWithTrace` đồng bộ trong `cdc-cms-service` và `sinkworker`.
- **Trạng thái:** **DONE** | **Estimate:** **10h**

---

#### 12. Workspace `FixDataIntegrityHideOffPipelines20260724` (24/07 – 27/07)
- **Title:** Ẩn Các Pipeline Đã Tắt/Xóa Tránh Báo Động Giả Trên Data Integrity Overview
- **Tình hình hiện tại:** Trang Data Integrity (`/data-integrity`) vẫn tính toán các Pipeline/Master Table đã bị tắt (`DISABLED`) hoặc xóa, gây báo động đỏ giả.
- **Mong muốn đạt được:** Báo cáo Data Integrity chỉ tính toán và hiển thị các Pipeline đang ở trạng thái `ACTIVE`.
- **Giải pháp:** Thêm điều kiện `is_active = TRUE` trong `data_integrity_read_repo_gorm.go` và lọc `status !== 'DISABLED'` trên UI FE.
- **Trạng thái:** **DONE** | **Estimate:** **6h**

---

#### 13. Workspace `FixSourceObjectRepoSyntaxError20260731` (31/07)
- **Title:** Sửa Lỗi Cú Pháp SQL `SQLSTATE 42703` Truy Vấn Danh Sách Source Objects
- **Tình hình hiện tại:** API `source_objects` bị sập 500 do lỗi `column gpay_id does not exist`.
- **Mong muốn đạt được:** API `source_objects` hoạt động mượt mà, trả về đúng metadata.
- **Giải pháp:** Sửa query GORM trong `source_object_read_repo_gorm.go` từ `gpay_id` thành đúng tên cột DB `_gpay_id`.
- **Trạng thái:** **DONE** | **Estimate:** **6h**

---

#### 14. Workspace `FixPgScanFieldsId20260730` (30/07)
- **Title:** Fix Lỗi Quét Thiếu Cột Primary Key `id` Khi Scan Fields PostgreSQL
- **Tình hình hiện tại:** Nhấn "Scan Fields" cho bảng PostgreSQL trên UI `/shadow` bị mất cột primary key `id`.
- **Mong muốn đạt được:** Scan Fields phát hiện chính xác 100% các cột bao gồm cả primary key `id`.
- **Giải pháp:** Loại bỏ logic skip `pkColumn` trong `discovery_utils.go` và unwrap Debezium `after` payload trong `discover_handler_utils.go`.
- **Trạng thái:** **DONE** | **Estimate:** **6h**

---

#### 15. Workspace `chart1_gradient_fix` / `FixChart1CssLayout20260722` / `FixChart1GaugeRoundness20260722` / `chart1_recommendations_update` (22/07)
- **Title:** Tối Ưu Giao Diện Báo Cáo Tín Dụng `chart1.html` (Gauge 240°, CSS Layout, Dynamic Gradient & Recommendations)
- **Tình hình hiện tại:** Biểu đồ Gauge Chart bị phẳng đáy (180°), gradient màu bị gắt, CSS layout card lệch padding, và danh sách khuyến nghị thiếu icon linh hoạt.
- **Mong muốn đạt được:** Biểu đồ Gauge Arc tròn 240° mềm mại, gradient Đỏ-Cam-Vàng-Xanh mượt mà, CSS layout phẳng đẹp, tự động gán icon khuyến nghị.
- **Giải pháp:** Cập nhật tính toán SVG Gauge 240° (`strokeDasharray=284.8`), refactor CSS layout (.card-header-pill, .gauge-wrapper), thêm thuộc tính `recommendations` & dynamic icon parser trong `chart1.html`.
- **Trạng thái:** **DONE** | **Estimate:** **12h**

---

### 📌 NHÓM 5: INFRASTRUCTURE OPERATIONS & OBSERVABILITY

#### 16. Workspace `ResetDebeziumOffsets20260730` (30/07)
- **Title:** Reset Offset Debezium Connector Trực Tiếp Trên UI Khi WAL Log Postgres Bị Purge
- **Tình hình hiện tại:** Khi WAL log trên PG bị xóa do Connector dừng lâu, Debezium crash loop; phải gõ lệnh `curl` CLI phức tạp.
- **Mong muốn đạt được:** Người vận hành nhấn nút "Reset Offsets" trực tiếp trên UI System Connectors để connector tái thiết lập streaming/snapshot.
- **Giải pháp:** Backend triển khai client method `DeleteOffsets` gọi Kafka Connect REST API `DELETE /connectors/{name}/offsets`, đăng ký Destructive route; Frontend bổ sung Modal xác nhận & nút Reset Offsets trên UI.
- **Trạng thái:** **DONE** | **Estimate:** **10h**

---

#### 17. Workspace `SnapshotTraceOptimization20260728` (28/07 – 29/07)
- **Title:** Tối Ưu OpenTelemetry Trace Tree & Bổ Sung Click-to-Copy Trace ID UI
- **Tình hình hiện tại:** Trace Context bị đứt khi truyền qua NATS, nguy cơ bùng nổ 3M Spans gây tràn bộ nhớ OTel Collector, và UI không hiển thị Trace ID để debug.
- **Mong muốn đạt được:** Trace Context nối liền từ CMS Service tới Worker, Spans được khống chế ở ranh giới I/O, và UI CMS hiển thị Click-to-Copy Trace ID 32-char hex.
- **Giải pháp:** Truyền OTel headers qua NATS, di chuyển `ChildSpan` xuống `batchUpsert` tầng I/O, ghi OTel TraceID vào DB `snapshot_progress`, bổ sung cột Trace ID dạng monospace kèm nút Click-to-Copy trên Activity Log, Transmute Schedules và Master Registry UI.
- **Trạng thái:** **DONE** | **Estimate:** **12h**

---

#### 18. Workspace `RemoveTransmuteOrphanSoftDelete20260804` (04/08)
- **Title:** Chuyển Luồng Orphan Clean-up Trên Transmute Sang Hard Delete
- **Tình hình hiện tại:** Khi dòng Shadow bị xóa hoặc mảng thu hẹp, Transmute Engine chạy Soft-Delete `_deleted = true`, làm tích tụ rác vĩnh viễn trên Master DB và bị N+1 query.
- **Mong muốn đạt được:** Xóa vật lý (`DELETE FROM`) hoàn toàn dữ liệu rác trên Master DB khi Shadow bị xóa/mảng thu hẹp, và tối ưu batch 3-step xóa 1 lần.
- **Giải pháp:** Refactor `transmuter.go` thay câu `UPDATE ... SET _deleted = true` bằng `DELETE FROM ... WHERE _source_id IN (?)` và `DELETE FROM ... WHERE _gpay_id IN (?)`.
- **Trạng thái:** **DONE** | **Estimate:** **12h**

---

#### 19. Workspace `flatten_mongo_extjson_fix` (03/08 – 04/08)
- **Title:** Fix Lỗi Flatten Mongo Extended-JSON (`_id.$oid`) Trong Transmute Engine
- **Tình hình hiện tại:** Scan Service đệ quy lặp sâu vào `$oid` Mongo ExtJSON làm sinh ra path sai `_id.$oid`, và child explode mapper không unwrap `$oid` làm giá trị bị null/rỗng.
- **Mong muốn đạt được:** Trích xuất chính xác 100% ID dạng chuỗi `"6a7039c870401a4326649f7b"` từ `{"$oid": "..."}` và chỉ flatten 1 cấp mảng duy nhất.
- **Giải pháp:** Thêm `unwrapMongoExtJSON` trong `transmuter.go` và `flatten.go`, loại bỏ đệ quy sâu trên key có `$`, chỉ explode đúng 1 cấp mảng đầu tiên.
- **Trạng thái:** **DONE** | **Estimate:** **14h**

---

#### 20. Workspace `FixLogTraceConnection` (05/08)
- **Title:** Chuẩn Hóa Trace Connection ID & Logging Hạ Tầng Connector / Connection Registry
- **Tình hình hiện tại:** Một số log kết nối DB/NATS/Kafka thiếu Trace ID hoặc Connection Key khiến việc tra cứu trên Jaeger/SigNoz bị gián đoạn.
- **Mong muốn đạt được:** Đảm bảo 100% log kết nối và probe probe status mang theo `trace_id` đồng nhất.
- **Giải pháp:** Bổ sung propagation `ctx` vào connection registry resolution và gán attribute `trace_id` trong zap log context.
- **Trạng thái:** **DONE** | **Estimate:** **8h**

---

## 📊 BẢNG TỔNG HỢP 20 WORKSPACE PHÂN THEO NHÓM CHỨC NĂNG

| Nhóm Chức Năng | STT | Tên Workspace | Tóm Tắt Nhiệm Vụ | Trạng Thái | Estimate |
| :--- | :---: | :--- | :--- | :---: | :---: |
| **Nhóm 1: Core Engine Scaling** | 1 | `BatchTransformScaling20260730` | Async Engine 500M records + Progress Bar + Cancel UI | **DONE** | 16h |
| | 2 | `TransmuteJobScaling20260731` | Bulk Transmute Engine + Live Bar + Boundary Isolation | **DONE** | 14h |
| **Nhóm 2: DB Performance Tuning** | 3 | `FixActivityLogSlowSql20260731` | Composite index + Partition Pruning Activity Log | **DONE** | 8h |
| | 4 | `FixSystemHealthQueriesSlowSql20260731` | Optimize System Health queries (`< 20ms`) | **DONE** | 10h |
| | 5 | `FixReconReadRepoSlowSql20260731` | Subquery Pagination First cho Recon Jobs/History | **DONE** | 10h |
| **Nhóm 3: Recon & Healing** | 6 | `FixSmokeCheckLockTime20260724` | Smoke Check lock timeout 3s + latency optimization | **DONE** | 8h |
| | 7 | `FixReconHistoryLabelAndManualTask20260724` | Phân bóc Smoke/Full label & trigger manual Recon UI | **DONE** | 8h |
| | 8 | `stale_ids_heal_modal_fix` | Fix Interactive Heal Modal 500 error (`IN (?)` clause) | **DONE** | 12h |
| | 9 | `FixReconTraceGrouping20260722` | Phân cấp OTel Trace Spans 4 tầng & Wire Master DB | **DONE** | 14h |
| | 10 | `FixReconReport24hWindow20260731` | Fix UTC timezone drift window 24h Recon Report | **DONE** | 8h |
| **Nhóm 4: Hardening & UI/UX** | 11 | `FixActivityLoggingPlan20260727` | Standardize Activity Log middleware & Trace ID audit | **DONE** | 10h |
| | 12 | `FixDataIntegrityHideOffPipelines20260724` | Filter `is_active = TRUE` trên Data Integrity UI | **DONE** | 6h |
| | 13 | `FixSourceObjectRepoSyntaxError20260731` | Fix SQL syntax error `_gpay_id` in Source Objects | **DONE** | 6h |
| | 14 | `FixPgScanFieldsId20260730` | Scan Fields Postgres giữ lại cột Primary Key `id` | **DONE** | 6h |
| | 15 | `chart1_gradient_fix` / `FixChart1Css...` | Layout Gauge 240°, gradient & recommendations UI | **DONE** | 12h |
| **Nhóm 5: Ops & Observability** | 16 | `ResetDebeziumOffsets20260730` | REST API & UI Reset Debezium Offset khi WAL purge | **DONE** | 10h |
| | 17 | `SnapshotTraceOptimization20260728` | Nối Trace Tree NATS, chống bùng nổ Spans & Copy Trace ID UI | **DONE** | 12h |
| | 18 | `RemoveTransmuteOrphanSoftDelete20260804` | Hard delete `DELETE FROM` master_table cho Transmute orphan | **DONE** | 12h |
| | 19 | `flatten_mongo_extjson_fix` | Unwrap Mongo `$oid` & chỉ explode 1 cấp mảng | **DONE** | 14h |
| | 20 | `FixLogTraceConnection` | Trace ID propagation cho Connection Registry & Probe logs | **DONE** | 8h |
| **TỔNG CỘNG** | | **20 Workspaces** | **Toàn bộ 20/20 Workspaces đã hoàn thành 100%** | **100% DONE** | **210h** |
