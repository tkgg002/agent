# 07_status_report_heal_fix: Báo Cáo Hiện Trạng & Kiểm Toán Sửa Lỗi Đối Soát

Báo cáo này tài liệu hóa quá trình sửa đổi và kiểm toán hệ thống đối soát dữ liệu (Reconciliation) và cơ chế tự động chữa lành (Heal Segment A & B).

## 1. Lý Do Thay Đổi (Root Cause)
1. **Lệch Cột Timestamp**: Đối soát nguồn (MongoDB) dùng trường thời gian tùy chỉnh `lastUpdatedAt` nhưng Postgres Shadow không có cột này (chỉ có cột mặc định hoặc metadata `_source_ts`). GORM nảy sinh lỗi khi filter theo cột không tồn tại, hoặc đối soát báo lệch khống (false positive) do lệch hệ quy chiếu thời gian.
2. **Lỗi Heal Segment B**: Khi kích hoạt heal cho Segment B, hệ thống truy vấn báo cáo đối soát mới nhất bằng table name thô (`payment_bills`) thay vì Master FQN (`shadow_test1111.payment_bills`), dẫn đến không tìm thấy báo cáo và bỏ qua quá trình tự động chữa lành.
3. **Trích xuất Timestamp Tĩnh**: Một số thành phần như `backfill_source_ts.go`, `helpers.go`, `dlq_worker.go` fix cứng trường thời gian `"updated_at"` hoặc trích xuất không đồng bộ với cấu hình `TimestampField` trong Shadow Object Registry.

---

## 2. Danh Sách Các File Đã Sửa Đổi
Dưới đây là danh sách chi tiết các file mã nguồn thực tế đã được chỉnh sửa trong phiên làm việc này:

| # | Đường dẫn File | Mô tả thay đổi | Số dòng thay đổi (tính toán thực tế) |
|---|---|---|---|
| 1 | `internal/service/governance/helpers.go` | Nhận `tsField *string` để trích xuất timestamp động từ MongoDB document | ~15 lines |
| 2 | `internal/service/governance/backfill_source_ts.go` | Bổ sung `TimestampField` vào MongoDB projection & truyền tham số trích xuất | ~25 lines |
| 3 | `internal/service/recon/recon_heal_utils.go` | Nhận `tsField *string` để trích xuất timestamp động trong utils heal | ~15 lines |
| 4 | `internal/service/recon/recon_heal.go` | Truyền `entry.TimestampField` cấu hình vào `extractSourceTsFromDoc` | ~3 lines |
| 5 | `internal/service/recon/dlq_worker.go` | Xử lý động trường `registry.TimestampField` thay vì fix cứng `updated_at` | ~10 lines |
| 6 | `internal/service/recon/recon_engine_segment_b.go` | Xuất bản (Export) phương thức `ListActiveMasterBindings` (viết hoa chữ cái đầu) | ~5 lines |
| 7 | `internal/service/recon/recon_dest_query.go` | Bổ sung hàm `ColumnExists` kiểm tra sự tồn tại của cột trong Postgres Shadow | ~15 lines |
| 8 | `internal/service/recon/recon_tier_a.go` | Cập nhật logic `resolveSourceTSField` sử dụng `ColumnExists` và ưu tiên cột cấu hình | ~121 lines |
| 9 | `internal/handler/recon/recon_heal_v4.go` | Giải quyết FQN trong `healSegmentB` bằng `ListActiveMasterBindings`, cập nhật check stale | ~71 lines |
| 10 | `internal/handler/recon/recon_handler_run.go` | Truyền FQN `entry.QualifiedTarget()` vào `GetLatestMissingReport` | ~2 lines |
| 11 | `internal/service/recon/recon_smoke.go` | Đồng bộ sử dụng phương thức `ListActiveMasterBindings` mới | ~62 lines |

**Tổng số file thay đổi**: 11 files
**Tổng số dòng thay đổi (thực tế)**: ~344 lines

---

## 3. Kết Quả Kiểm Thử & Xác Minh (Verification Evidence)
1. **Kiểm thử tự động**:
   - Chạy lệnh `go test -v ./internal/service/recon/...` và `go test -v ./internal/service/governance/...` thành công.
   - Các kịch bản kiểm thử bao gồm kiểm tra lọc cột không tồn tại, kiểm tra trích xuất động trường thời gian và kiểm tra stale report fallback đều PASS.
2. **Kiểm thử thủ công**:
   - Restart worker thành công.
   - Phát tín hiệu heal qua NATS. Nhật ký hệ thống ghi nhận đối soát Tier 2 thành công và gửi tín hiệu Debezium Signal với filter ID chuẩn xác cho 68 bản ghi lệch dữ liệu.

---

## 4. Báo Cáo Kiểm Toán Quá Trình (Governance & Process Audit)
* **Quy tắc Workspace-First (Rule #9 / L-001)**: Bị vi phạm ở đầu phiên làm việc (đã được ghi nhận vào `05_progress.md` của workspace). Đã được khắc phục bằng cách thiết lập workspace trước khi sửa mã nguồn.
* **Quy tắc Prefix Tài liệu (Rule #7)**: Tất cả tài liệu phục vụ cho task đều được tạo dưới dạng file vật lý trong workspace (`00_context.md`, `01_todo.md`, `02_plan.md`, `05_progress.md`, và báo cáo hiện tại `07_status_report_heal_fix.md`).
* **Tính toàn vẹn của tệp Memory (Rule #11)**: Không sử dụng `Overwrite: true` trên các tệp memory như `lessons.md`, `active_plans.md`. Các cập nhật đều sử dụng cơ chế `replace_file_content` nối đuôi (append).
* **Kiểm toán mã nguồn (Rule #12 / Rule #6)**: Các thay đổi chỉ tập trung vào việc giải quyết lỗi gốc rễ ở tầng hệ thống core, không cheat DB, không sửa cấu hình môi trường, blast radius được khoanh vùng tối thiểu (minimal impact).
