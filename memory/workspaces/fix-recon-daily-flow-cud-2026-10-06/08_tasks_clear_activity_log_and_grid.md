# Danh sách Tác vụ (Task Checklist): Tính năng Clear Activity Log Động & Tái cấu trúc Bảng Đối soát

## Phase 1: Backend CMS Service (`cdc-cms-service`) — Simplicity First & Minimal Impact
- [ ] **Task 1.1: Persistence Layer (`internal/infra/persistence/system/activity_log_read_repo_gorm.go`)**
  - Bổ sung struct `ClearActivityParams` hỗ trợ: `TargetTable`, `MasterTable`, `BeforeDate`, `FromDate`, `ToDate`.
  - Bổ sung method `ClearActivity(ctx context.Context, p ClearActivityParams) (int64, error)`.
  - Cài đặt **Guard Clause**: Bắt buộc có ít nhất 1 điều kiện lọc (`TargetTable`, `MasterTable`, `BeforeDate`, `FromDate`) để triệt tiêu vĩnh viễn nguy cơ GORM missing where clause hoặc xóa sạch toàn bộ hệ thống.
  - Cài đặt **Partition Pruning**: Bắt buộc filter `created_at < ? AND started_at < ?` để PostgreSQL chỉ quét đúng child partition của partitioned table `cdc_system.cdc_activity_log`.
  - Cài đặt **Exact Match IN**: Dùng `WHERE target_table IN (?)` cho cả Shadow và Master table, TUYỆT ĐỐI KHÔNG dùng `LIKE %.table` để loại bỏ rủi ro xóa nhầm log của pipeline khác (Zero Cross-Tenant Blast Radius).
- [ ] **Task 1.2: HTTP Handler (`internal/api/system/activity_log_handler.go`)**
  - Bổ sung method `Clear(c *fiber.Ctx) error` xử lý request `DELETE /api/activity-log`.
  - Parse query params: `target_table`, `master_table`, `before_date`, `from_date`, `to_date`.
- [ ] **Task 1.3: Router Binding (`internal/router/router.go`)**
  - Đăng ký endpoint: `dual("DELETE", shared, "/activity-log", h.System.ActivityLog.Clear)`.

---

## Phase 2: Frontend API & Hook (`cdc-cms-web`)
- [ ] **Task 2.1: React Query Mutation (`src/hooks/useReconStatus.ts`)**
  - Thêm `useClearActivityLogMutation()` gọi `DELETE /api/activity-log` kèm headers audit `X-Action-Reason`.
  - Invalidate cache `['pipeline-activity-log']`, `['table-history']`, `['recon-report']` sau khi xóa thành công.

---

## Phase 3: UI Dọn dẹp & Tái cấu trúc Bảng Đối soát (`ReconPipelineGrid.tsx`)
- [ ] **Task 3.1: Modal Dọn dẹp Nhật ký với DatePicker Động**
  - Đặt nút "Dọn dẹp nhật ký" tại Header của Modal Daily Flow Stats và trong Drawer Chi tiết Pipeline.
  - Tích hợp Ant Design `DatePicker` cho phép người vận hành chọn ngày mốc linh hoạt trên bất kỳ môi trường nào (Prod, Staging, Dev) — Loại bỏ hoàn toàn hardcode ngày cố định.
  - Hỗ trợ 2 chế độ:
    + "Xóa các bản ghi trước ngày [DatePicker]" (Mặc định chọn 7 ngày trước).
    + "Xóa toàn bộ nhật ký của pipeline này" (Reset sạch cho bảng hiện tại).
  - Tự động gửi cả `target_table` (Shadow) và `master_table` (Master) để xóa đồng thời cả 2 trạm.
- [ ] **Task 3.2: Tách bạch Dứt khoát Khối Snapshot và Kafka CDC Stream**
  - Cập nhật struct `DailyFlowRecord`: Khối `snapshot` riêng (`shadowWritten`, `masterWritten`), Khối `kafka` riêng, Khối `shadow` riêng, Khối `master` riêng.
  - Trong vòng lặp reducer:
    + Bản ghi có `is_snapshot === true` hoặc `buffer_type === 'snapshot'` hoặc `triggered_by === 'snapshot'` hoặc `operation === 'snapshot.v2'` -> Đẩy vào `row.snapshot.shadowWritten`.
    + Bản ghi Transmute do snapshot kích hoạt (hoặc cùng `trace_id`) -> Đẩy vào `row.snapshot.masterWritten`.
    + Bản ghi stream Debezium -> Đẩy vào `row.kafka` và `row.shadow` CDC.
    + Bản ghi Master transmute từ stream CDC -> Đẩy vào `row.master`.
- [ ] **Task 3.3: Cấu trúc Cột 6 Khối Chuẩn Hóa**
  - Cột cố định trái: `Ngày` (`fixed: 'left'`).
  - `1. SNAPSHOT / RE-SYNC` (Shadow Nạp, Master Sync).
  - `2. KAFKA CDC BẮN VỀ` (Tổng Ops, c, u, d).
  - `3. GHI VÀO DB SHADOW (Từ CDC Stream)` (Tổng Ops & Dòng DB, c, u, d).
  - `4. GHI VÀO DB MASTER (Từ CDC Stream)` (Tổng Ops & Ghi DB, Ins, Upd, Skip).
  - `5. ĐỐI SOÁT & DLQ (Lỗi Luồng CDC)` (DLQ, Thiếu, Stale, Heal).
  - Cột cố định phải: `ĐÁNH GIÁ (FSM Luồng CDC)` (`fixed: 'right'`).
- [ ] **Task 3.4: Bỏ Phân trang & Chuẩn hóa Footer**
  - Dùng `pagination={false}` và `scroll={{ y: 450, x: 1950 }}` để xem toàn bộ lịch sử dạng bảng cuộn.
  - Dòng `Table.Summary` (Footer TỔNG CỘNG) đồng bộ số liệu toàn bảng.
