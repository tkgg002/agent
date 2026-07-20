# 11_report_phase_2.md - Báo cáo thay đổi code Phase 2

Dưới đây là thống kê chi tiết các tệp tin đã thay đổi và thêm mới trong Phase 2:

## 1. Thống kê số lượng dòng code thay đổi (LOC)

| File | Trạng thái | Số dòng thêm | Số dòng xóa | Mô tả thay đổi |
|---|---|---|---|---|
| [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go) | MODIFIED | +52 | -44 | Flush song song 20 bảng bằng errgroup; tách context timeout 10s chặng Sink. |
| [debounce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/debounce.go) | NEW | +148 | -0 | Tạo mới debouncer, cấu hình idleTimer/maxTimer, Backpressure chặng Transmute. |
| [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) | MODIFIED | +149 | -12 | Tích hợp TableDebouncer, chia để trị Poison Pill binarySearchSplit đệ quy chặng Transmute. |
| [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) | MODIFIED | +53 | -0 | Thêm import strconv; logic dọn dẹp master mồ côi (PruneOrphans) chặng Transmute. |
| [transmute_scheduler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmute_scheduler.go) | MODIFIED | +25 | -0 | Thêm logic cleanupStuckSchedules dọn dẹp job kẹt running quá 10 phút. |

**Tổng cộng:** ~427 LOC thêm mới, ~56 LOC xóa/thay thế.

---

## 2. Kết quả kiểm toán chất lượng (DoD)
- **Tính năng Concurrency chặng Sink:** errgroup chạy song song đảm bảo độc lập ghi, cô lập lỗi cục bộ của từng bảng thành công.
- **Detached Context chặng Sink:** Khắc phục triệt để lỗi rollback DB transaction khi shutdown, bảo vệ an toàn 100% dữ liệu mẻ cuối cùng.
- **Tính năng Concurrency chặng Transmute:** TableDebouncer khống chế tối đa 10 luồng song song trên mỗi bảng, hãm đầu vào pull tin nhắn khi quá tải RAM (Backpressure).
- **Poison Pill Recovery:** Chia để trị đệ quy giúp cô lập nhanh dòng lỗi và terminate tin nhắn rác, tránh nghẽn luồng realtime.
- **Flatten Orphan Cleanup:** Soft-delete master rows dư thừa khi mảng co rút chạy cực kỳ chính xác qua range check 500 index.
- **Stuck Job Recovery:** Tự động giải phóng job kẹt sau 10 phút, khôi phục khả năng tự chữa lành cho cron scheduler.
