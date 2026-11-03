---
tier: T2
mode: standard
suffix: master_row_count_mismatch_oct30_20261030
---

# 01. REQUIREMENTS — `master_row_count_mismatch_oct30_20261030`

## 1. Yêu cầu của User
- Dòng `2026-10-30` trên bảng **Thống kê đối soát 3 trạm & DLQ theo ngày** của pipeline `shadow_testcoretrans.transaction_history_1 -> devtest.transaction_history_1`:
  - `1. SNAPSHOT / RE-SYNC`: `Shadow Nạp = 250,137` | `Master Sync = 250,240`
  - `2. KAFKA CDC BẮN VỀ`: `735 Ops` (`317 Insert(c)`, `418 Update(u)`, `0 Delete(d)`)
  - `3. GHI VÀO DB SHADOW (Từ CDC Stream)`: `735 Ops (326 dòng DB)` (`317 Insert(c)`, `418 Update(u)`)
  - `4. GHI VÀO DB MASTER (Từ CDC Stream)`: `809 Ops (386 dòng DB)` (`347 Inserted`, `462 Updated`)
  - `5. ĐỐI SOÁT & DLQ`: `DLQ / Lỗi = 25`, `ĐÁNH GIÁ = CÓ LỆCH`
- Câu hỏi của anh:
  1. **"sao có 250,137 + 317 (318) ở shadow mà ở master lại là 250,240 + 347 (348). mới phát đầu đã lỗi."**
  2. **"có cần a truncate và chạy lại ko"**

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- `L005` (Strict Propose-Only Gate): Chế độ Propose-Only, không sửa source code khi chưa có `APPROVE`.
- `L009` (Single-Responsibility Module Split): Tách các file vượt ngưỡng dòng (`transmute_handler.go`, `batch_buffer.go`, `mongo_bson_restorer_test.go`, `ReconPipelineGrid.tsx`) theo đúng trách nhiệm đơn và Layer kiến trúc.
- `L011` (Pre-Plan Adversarial Critique — DG0): Đối chiếu số dòng thực tế trong DB (`shadow_testcoretrans.transaction_history_1`, `shadow_testcoretrans.cdc_transaction_history_payload_1`, MongoDB `devtest.transaction_history_1`, và `cdc_system.cdc_activity_log`) trước khi kết luận.
- `L031` & `L032` (Zero Direct DB CLI): Tuyệt đối không dùng `psql` hay `docker exec`.
- `L039` (Multi-Shadow 1:1 Binding): Kiểm tra cơ chế `multi_shadow` (`transaction_history_1` + `cdc_transaction_history_payload_1` cùng fanout về 1 bảng Master `devtest.transaction_history_1`).

## 3. Yêu cầu Bổ sung Sau Audit (`2026-10-30T17:04:13+07:00`)
- Lập giải pháp và xử lý 4 Issues còn tồn đọng ở Mục I:
  1. **Issue #1 (Build & Dialector):** Xác nhận/đảm bảo `m.reg.OpenGormWithEngine` trong `pkgs/database/multi_dialector.go` và `connection_manager.go:405` biên dịch & chạy test sạch 100% trên toàn bộ `centralized-data-service`.
  2. **Issue #2 (Logic `recon_daily` & `Status` trong `GetDailyFlow`):** Sửa CTE `recon_daily` trong `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:473-485` lấy `missing_count` / `stale_count` của **lần đối soát mới nhất** trong ngày (thay vì `SUM` cộng dồn mọi lần quét cũ đã heal), đồng thời kết hợp bảng `cdc_system.reconciliation_report` (`stage = 'shadow_master'`) để khi trạng thái đối soát mới nhất đã cân bằng (`missing == 0 && stale == 0`) mà trong ngày có `DLQ > 0` lịch sử đã heal thì trả về `WARNING` (`CẢNH BÁO`) hoặc `BALANCED` thay vì khóa chết ở `DRIFT` (`CÓ LỆCH`).
  3. **Issue #3 (Khử trùng số đếm `transmute_sub` bị retry lặp trong `act_daily`):** Trong `GetDailyFlow` (`activity_log_read_repo_gorm.go`), loại bỏ các dòng log `transmute_sub` trùng lặp sinh ra từ các vòng NATS redelivery của cùng 1 mẻ (`split_retry = true` trùng `correlation_id` + `rows_affected` + `inserted` + `updated`) để số liệu cột `4. GHI VÀO DB MASTER` phản ánh đúng số bản ghi thực tế không bị phình bởi 5 vòng retry lúc 14h chiều.
  4. **Issue #4 (Tách 4 God-files theo đúng Single-Responsibility Principle `SRP`):**
     - Tách cụm hàm cô lập Poison Pill (`binarySearchSplit`, `splitSingleTaskPayload`, `runSingleTaskSubSlice`) khỏi `centralized-data-service/internal/handler/master/transmute_handler.go` (`1,074` dòng) sang `transmute_poison_pill_split.go`.
     - Tách cụm hàm retry & sơ tán DLQ (`batchUpsertWithRetry`, `evacuateFailedChunkToDLQ`, `splitAndRetryBatch`) khỏi `centralized-data-service/internal/handler/shadow/batch_buffer.go` (`967` dòng) sang `batch_buffer_dlq.go`.
     - Tách cụm test tích hợp ExtendedJSON & UUID Live Mongo (`dòng 685–832`) khỏi `centralized-data-service/internal/service/master/mongo_bson_restorer_test.go` (`832` dòng) sang `mongo_bson_uuid_live_test.go`.
     - Tách component Drawer bảng thống kê 3 trạm theo ngày khỏi `cdc-cms-web/src/components/ReconPipelineGrid.tsx` (`1,961` dòng) sang `cdc-cms-web/src/components/ReconDailyFlowModal.tsx`.
