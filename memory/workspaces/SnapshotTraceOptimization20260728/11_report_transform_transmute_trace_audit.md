# Báo Cáo Audit Thay Đổi Toàn Diện: Chuẩn Hóa Tracing & Click-To-Copy Trace ID UI Cho Transform & Transmute (Backend & CMS FE)

## 1. Danh Sách Các File Đã Thay Đổi (Modified Files)

| Component | File | Số Dòng Thay Đổi | Nội Dung Thay Đổi |
| :--- | :--- | :--- | :--- |
| **CMS FE** | `src/pages/ActivityLog.tsx` | +21 / -0 lines | Bổ sung cột **Trace ID** vào bảng Activity Log với nút **Click-to-Copy** chuẩn 32 ký tự Hex (dạng monospace `a0b1c2...e4f5`). |
| **CMS FE** | `src/pages/TransmuteSchedules.tsx` | +20 / -1 lines | Bổ sung Toast Notification chứa **Click-to-Copy Trace ID 32-char hex** khi người dùng nhấn nút **Run-now / Sync ngay**. |
| **CMS FE** | `src/pages/MasterRegistry.tsx` | +23 / -2 lines | Bổ sung Toast Notification chứa **Click-to-Copy Trace ID 32-char hex** khi người dùng nhấn nút **Sync ngay** từ danh sách Master tables. |
| **CMS BE** | `internal/app/commands/scheduler/run_now.go` | +7 / -0 lines | Trích xuất `trace_id` từ `ctx` và đính kèm vào response JSON body của lệnh `run-now`. |
| **CMS BE** | `internal/api/scheduler/transmute_schedule_handler.go` | +1 / -0 lines | Phản hồi `trace_id` trong HTTP response 202 JSON của endpoint `/api/v1/schedules/:id/run-now`. |
| **Worker BE** | `internal/handler/shadow/batch_transform_handler.go` | +18 / -4 lines | Bổ sung `ChildSpan` `shadow.batch_transform: <target_table>` cho từng mẻ CTE SQL UPDATE chunk (1.000 dòng/lần) trong `HandleBatchTransform()`. |
| **Worker BE** | `internal/handler/master/transmute_handler.go` | +42 / -12 lines | 1. Gom `oteltrace.Link` và khởi tạo `ChildSpanWithLinks` (`cdc.transmute.debounced_batch: <master_table>`). <br> 2. Trích xuất **Trace ID chuẩn 32 ký tự Hex** truyền vào `TransmuteResponse` và `ActivityLog`. |
| **Worker BE** | `internal/service/master/transmuter.go` | +16 / -2 lines | Thêm `ChildSpan` `master.bulk_upsert: <schema>.<table_name>` tại ranh giới I/O Master DB (`bulkUpsertMaster`). |

---

## 2. Kết Quả Verification (End-to-End Build & Test)

1. **Frontend Web (`cdc-cms-web`):** `npm run build` -> **PASS (Exit Code 0, 3690 modules transformed)**.
2. **CMS Backend (`cdc-cms-service`):** `go build ./cmd/server/main.go` -> **PASS (Exit Code 0)**.
3. **Worker Backend (`centralized-data-service`):** `go build ./cmd/worker/main.go` -> **PASS (Exit Code 0)**.
4. **Unit Tests:** `go test ./internal/handler/master/... ./internal/handler/shadow/... ./internal/service/master/...` -> **PASS (100% OK)**.

---

## 3. Hướng Dẫn Xem & Click-to-Copy Trace ID Trên CMS FE (Exact UI Navigation)

1. **Xem & Copy Trace ID của tất cả tác vụ Transform / Transmute / Snapshot / Sync:**
   - Mở CMS FE -> Truy cập menu **CDC Worker Activity Log** ([/activity-log](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityLog.tsx)).
   - Tại bảng Activity Log, cột **Trace ID** mới bổ sung hiển thị mã Hex 32 ký tự dạng `a0b1c2...e4f5`.
   - **Thao tác Click-to-Copy:** Rê chuột hoặc click vào icon Copy trên dòng tác vụ -> Chuỗi Trace ID 32 ký tự chuẩn SigNoz sẽ được lưu vào Clipboard và thông báo "Copied!".
2. **Xem & Copy Trace ID khi bấm "Sync ngay / Run Now":**
   - Truy cập menu **Transmute Schedules** ([/schedules](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TransmuteSchedules.tsx)) hoặc **Master Registry** ([/masters](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx)).
   - Click nút **"Sync ngay"** hoặc **"Run Now"**.
   - Màn hình góc trên bên phải xuất hiện Toast Notification thông báo Dispatched thành công kèm dòng **Trace ID** 32 ký tự Hex có icon Copy để bấm copy dán thẳng vào SigNoz tra cứu tức thì!
