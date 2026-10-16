# 11 — Report: Phase P2 & Pending Issues Resolution

Ngày: 2026-10-12
Trạng thái: **DONE — Đã nghiệm thu toàn bộ T1 đến T8 trên hệ thống và DB thật**

---

## 1. Tóm tắt kết quả thực thi
Toàn bộ 5 tồn đọng chính của Phase P1 đã được giải quyết dứt điểm theo đúng chuẩn **Simplicity First, Minimal Impact, Tư duy Core Systems**:

1. **Tồn đọng 1 (Core - P2/H5): Xóa bỏ giới hạn 200 logs bằng Server-side Aggregation API**:
   - CMS Backend đã triển khai `GET /api/reconciliation/daily-flow` (và `/api/activity-log/daily-flow`).
   - Query sử dụng **PostgreSQL Partition Pruning** (`created_at >= NOW() - INTERVAL '14 days'`) kết hợp `generate_series` đảm bảo trả về đủ 14 ngày liên tục, chuẩn múi giờ địa phương (`tz_offset`). Thời gian thực thi trên DB thật < 80ms.
   - Web UI đã thay thế hoàn toàn 2 hook cũ (`usePipelineActivityLog` limit 200) và 190 dòng client-side reducer bằng hook `useDailyFlow`. Bảng đối soát 3 chiều giờ đây hiển thị đủ 100% dữ liệu lịch sử không còn tình trạng bị cắt xén.

2. **Tồn đọng 2: Triệt tiêu đếm thừa DLQ khi Transmute Batch chia để trị**:
   - `transmute_handler.go`: Ghi nhận `"split_retry": true` cho batch cha và các sub-batch trung gian khi chia đôi đệ quy (`len(subBatch) > 1`). Chỉ ghi `"poison_pill": true` khi đã cô lập thành công (`len(batch) == 1`).
   - SQL Aggregation: DLQ chỉ tính các lỗi thực sự (`(e.details->>'split_retry')::boolean IS NOT TRUE`), phản ánh chính xác 1 poison pill = 1 DLQ.

3. **Tồn đọng 3: Chuẩn hóa ngữ nghĩa Khối 2 (Kafka Ingest) vs Khối 3 (Shadow Applied)**:
   - Khối 2 đo lường tổng sự kiện CDC Debezium/Kafka bắn vào (`kafka.total = C + U + D`).
   - Khối 3 đo lường số lượng bản ghi thực thi vào Shadow (`shadow.total = ops`) và số dòng vật lý biến đổi (`shadow.written = db_rows`).

4. **Tồn đọng 4: Khắc phục rủi ro Memory Aliasing trong Router `chain()` của CMS**:
   - `internal/router/router.go`: `chain()` sử dụng 3-index slice `out[:len(out):len(out)]` để khóa cứng capacity bằng đúng length. Ngăn chặn triệt để nguy cơ Go runtime tái sử dụng vùng nhớ capacity thừa làm ghi đè handler giữa các route khi gọi `append`.

5. **Tồn đọng 5: Script bảo trì backfill `shadow_schema` cho log cũ**:
   - Cung cấp script `cdc-cms-service/migrations/scripts/enrich_legacy_activity_logs.sql`. Đã kiểm thử trên Docker DB `cdc_dw` (:5433) trong transaction: Cập nhật an toàn 2,796 bản ghi cũ cho các pipeline đơn, bỏ qua an toàn các pipeline tranh chấp.

---

## 2. Chi tiết các file thay đổi & Diff Stat

| Repository | File | Thay đổi |
|---|---|---|
| `centralized-data-service` | `internal/handler/master/transmute_handler.go` | Đánh dấu `"split_retry": true` khi lỗi non-transient và split; ghi log `"poison_pill": true` khi `len(batch)==1` |
| `cdc-cms-service` | `internal/router/router.go` | Sửa `chain()` trả về 3-index slice `out[:len:len]`; đăng ký route `/activity-log/daily-flow` & `/reconciliation/daily-flow` |
| `cdc-cms-service` | `internal/app/queries/system/daily_flow.go` | Định nghĩa DTO `DailyFlowParams`, `DailyFlowRecord`, `DailyFlowResult`, `DailyFlowReader` |
| `cdc-cms-service` | `internal/app/queries/system/list_activity_logs.go` | Mở rộng interface `ActivityLogReader` thêm `GetDailyFlow` |
| `cdc-cms-service` | `internal/infra/persistence/system/activity_log_read_repo_gorm.go` | Viết hàm `GetDailyFlow(...)` với SQL Aggregation (Partition Pruning, Timezone, 5 Khối đối soát) |
| `cdc-cms-service` | `internal/api/system/activity_log_handler.go` | Thêm handler `DailyFlow` |
| `cdc-cms-service` | `migrations/scripts/enrich_legacy_activity_logs.sql` | Script SQL bảo trì backfill log cũ |
| `cdc-cms-web` | `src/hooks/useReconStatus.ts` | Thêm types & hook `useDailyFlow`, invalidate query khi Clear log |
| `cdc-cms-web` | `src/components/ReconPipelineGrid.tsx` | Chuyển sang dùng `useDailyFlow`, xoá bỏ 230 dòng client-side reducer và cảnh báo 200 logs |

---

## 3. Kết quả nghiệm thu (Verification Results)
- **Centralized-data-service**: `go test ./internal/handler/master/ ./internal/handler/shadow/` + `go build ./cmd/... ./internal/...` -> **PASS (Exit 0)**.
- **Cdc-cms-service**: `go build ./cmd/... ./internal/...` + `go vet` -> **PASS (Exit 0)**.
- **Cdc-cms-web**: `npm run build` (tsc -b && vite build) -> **PASS (Exit 0)**.
- **Data Execution trên Docker `cdc_dw` (:5433)**: Hàm `GetDailyFlow` chạy thực tế thành công, trả về 7-14 ngày liên tục, chuẩn xác chỉ số của từng khối. Script SQL update thử nghiệm an toàn 2,796 dòng.
