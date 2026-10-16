# 13 — Phân tích Audit V4: Clear Activity Log & Bảng Đối Soát (dựa trên code thật)

> Ngày: 2026-10-12 · Phương pháp: đọc `git diff` thật + trace ngược writer (centralized-data-service) → reader (cdc-cms-service) → UI (cdc-cms-web). Không dựa vào báo cáo `11_report_clear_activity_log_and_grid.md`.

## A. Phát hiện chí mạng (CRITICAL)

| # | Phát hiện | Bằng chứng (file:dòng) | Hệ quả |
|---|---|---|---|
| C1 | Activity log của Shadow **không ghi định danh pipeline**: `target_table` = tên bảng trần, `details` không có `shadow_schema`/`shadow_binding_id`. Nhánh lỗi gọi `act.Fail` → không có details. | `centralized-data-service/internal/handler/shadow/batch_buffer.go` L614 (`act.Start(opName, tableName, …)`), L639-648 (details), L619 (`act.Fail`) | Mọi filter `shadow_schema` ở CMS (List + Clear) **không cách ly được** 2 connector trùng tên bảng. |
| C2 | CMS resolve binding bằng `shadow_table = al.target_table … ORDER BY updated_at DESC LIMIT 1`; chỉ dùng `details.shadow_binding_id` nếu có (shadow log không có). | `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go` L38-48; hệ thống đã tự biết mơ hồ: `scope_ambiguous` L114, `scope_counts` L79-86 | 2 connector trùng tên → **toàn bộ log gán cho binding update gần nhất**. Connector A xem = rỗng, Clear A = xóa 0; Connector B xem/xóa = log của **cả A và B**. Claim "cách ly 100%" ở turn trước là **SAI**. |
| C3 | Route `DELETE /activity-log` đăng ký trong group `shared` (`RequireRole("admin","operator")`), **không** qua `destructiveChain` (OpsAdmin + Idempotency + Audit). Pattern chuẩn cho xóa dữ liệu kiểm toán: `DELETE /reconciliation/report/:id`. | `internal/router/router.go` L311 vs L128, L195-196 | Operator xóa được nhật ký kiểm toán, **không để lại audit trail**. Báo cáo turn trước claim "kiểm tra `perm.OpReconWrite`" — **không tồn tại trong diff**. |
| C4 | Snapshot buffer cũng bắn trigger transmute với `triggered_by:"batch-buffer"`; `HandleTransmuteShadow` hard-code `"kafka-consumer-hook"`; transmute log không có dấu snapshot. | `batch_buffer_fanout.go` L61; `transmute_handler.go` L141; `batch_buffer.go` L458/780/819/848 (không guard snapshot) | Khối "1. SNAPSHOT" phía **Master luôn = 0**; ghi Master do snapshot vẫn bị cộng vào "4. GHI VÀO DB MASTER (Từ CDC)". Ví dụ 09/10 `export_jobs` **không thể** ra Master CDC = 0 như báo cáo đã claim. |
| C5 | `ClearActivity` dùng nhánh `OR al.target_table = <master>` → bypass điều kiện schema/table; chấp nhận request chỉ có ngày (xóa **mọi pipeline**); ngày là string thô đẩy thẳng vào SQL. | `activity_log_read_repo_gorm.go` L309, L348-365, L372-398; handler cùng guard | Blast radius chéo connector khi `source_database` rỗng; 500 khi ngày sai format; lệch múi giờ (server TZ). |

## B. Phát hiện quan trọng (HIGH)

| # | Phát hiện | Bằng chứng | Hệ quả |
|---|---|---|---|
| H1 | `runDebouncedTransmute` lấy `masterBindingID`, `triggeredBy` từ `batch[0]`, trong khi debouncer key chỉ là `masterTable`. | `transmute_handler.go` L386-405 | Mẻ trộn binding/trigger → chạy sai binding + gắn nhãn sai. |
| H2 | Transmute log không chứa `master_binding_id` (có sẵn trong scope); `transmute_sub` Complete với `details=nil`. | `transmute_handler.go` L286-296, L482-492, L582 | CMS resolve master binding bằng tên LIMIT 1; số Ins/Upd của sub-batch (poison-pill split) mất khỏi bảng ngày. |
| H3 | UI chỉ fetch `operation='transmute'`, bỏ `transmute_sub`. | `ReconPipelineGrid.tsx` L490 | Khi split nhị phân, ghi Master thật nằm ở `transmute_sub` → thiếu số. |
| H4 | UI bucket ngày bằng `started_at.slice(0,10)` trên chuỗi **UTC** (`TO_CHAR … AT TIME ZONE 'UTC' … Z`). | `ReconPipelineGrid.tsx` L529, L572; repo L121 | Sự kiện 00:00–06:59 giờ VN rơi vào **ngày hôm trước**. |
| H5 | Daily stats tính trên tối đa **200 log mới nhất** mỗi loại (`LOG_FETCH_LIMIT`). | `ReconPipelineGrid.tsx` L488 | Ngày cũ/“TỔNG CỘNG” bị cắt cụt khi > 200 batch → số liệu sai. Gốc rễ cần tổng hợp phía server. |
| H6 | Fallback đoán schema `'cdc_shadow'` khi thiếu `shadow_schema`. | `ReconPipelineGrid.tsx` L891 | Vi phạm "không đoán mò"; có thể gửi scope sai. |

## C. Phát hiện nhỏ (LOW)
- L1: `reason` bị gửi cả ở query params (`params: payload`) lẫn header → rác query string. `useReconStatus.ts` diff.
- L2: `from_date/to_date` không có UI dùng → code chết (YAGNI).
- L3: Điều kiện snapshot chết: `k.operation === 'snapshot.v2'` (query đã lọc `operation='kafka-consumer'`), `details.triggered_by` (không writer nào ghi).
- L4: Khối "2. KAFKA" được suy ra từ chính log Shadow (c/u/d của batch) → không phải đo độc lập; FSM `kafka.total === shadow.total` là **tautology** (lesson 2026-10-08). Phải ghi rõ trên UI/ADR, không claim là đối soát Kafka độc lập.
- L5: Diff FE hiện tại (+968/-95) gộp nhiều phiên chưa commit từ 30/09 → con số "~225 dòng" trong báo cáo 11 không kiểm chứng được.

## D. Kết luận
Bản triển khai trước **chưa đạt Done** (Rule #14 G1/G3/G4/G6/G8 fail). Gốc rễ không nằm ở query CMS mà ở **writer không ghi định danh** và **không truyền dấu snapshot**. Plan V4 (`09_tasks_solution_audit_v4.md`) sửa tại gốc với thay đổi cộng thêm (additive), tương thích ngược.
