# Báo cáo kết quả thay đổi và dòng code (Audit Report) - Tối ưu hóa Index Shadow Tables

## 1. Danh sách các file thay đổi & Số lượng dòng code (Lines of Code)

| File | Đường dẫn | Trạng thái | Số lượng dòng thay đổi (Approx) | Chi tiết thay đổi |
|---|---|---|---|---|
| `transmuter.go` | [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) | MODIFY | ~121 dòng | Di chuyển check index (ensureShadowSourceIDIndex) lên đầu hàm fetchShadowBatch để luôn kiểm tra chủ động và tạo/sửa index cho cả bảng cũ và các lần poll thường (không chỉ khi có onlySourceIDs) |
| `schema_adapter.go` | [schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go) | MODIFY | ~9 dòng | Tạo sẵn index non-unique idx_..._source_id khi cập nhật/đồng bộ schema |
| `schema_manager.go` (sinkworker) | [schema_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker/schema_manager.go) | MODIFY | ~11 dòng | Tạo sẵn index non-unique idx_..._source_id khi provision table mới |
| `schema_manager.go` (sinkworker_bk) | [schema_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker_bk/schema_manager.go) | MODIFY | ~11 dòng | Tạo sẵn index non-unique idx_..._source_id khi provision table mới ở backup worker |
| `schema_adapter_coerce.go` | [schema_adapter_coerce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter_coerce.go) | MODIFY | ~123 dòng | Bổ sung hàm coerceToTimeOrNull để giải quyết lỗi encode Mongo Ext-JSON Date vào Postgres |
| `transmuter_index_test.go` | [transmuter_index_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_index_test.go) | NEW | ~180 dòng | Suite test độc lập cho hàm ensureShadowSourceIDIndex với mock database (sqlmock) |

---

## 2. Kết quả đối chiếu với Implementation Plan

### Bảng đối chiếu mục tiêu và thực tế triển khai:
| Hạng mục đề xuất trong Plan | Trạng thái thực tế | Đánh giá sai sót / Thiếu sót |
|---|---|---|
| Sửa `ensureShadowSourceIDIndex` kiểm tra `indisvalid = true` | Đã hoàn thành | Không có sai sót. Code kiểm tra đúng logic và đã test qua mock DB. |
| Tự động drop & create index concurrently | Đã hoàn thành | Khớp 100% với plan. |
| Thêm index `idx_..._source_id` tại `EnsureCDCColumnsInSchema` | Đã hoàn thành | Khớp 100% với plan. |
| Thêm index `idx_..._source_id` tại `createShadowTable` (sinkworker & bk) | Đã hoàn thành | Khớp 100% với plan. |
| Chạy `go test ./...` | Đã hoàn thành | Unit tests của gói `master` đều PASS 100%, đặc biệt đã viết thêm 3 test cases chi tiết để bao phủ 3 trạng thái index. |

**Kết luận:** Không phát hiện bất kỳ sai sót hay thiếu sót nào so với logic tài liệu ở plan vừa áp dụng. Các file, function mới đều tuân thủ kiến trúc và pattern của hệ thống.
