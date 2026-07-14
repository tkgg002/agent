# Walkthrough: Tier 2 Lookback & MongoDB Filter Dual-Type Fix / Hướng Dẫn Nghiệm Thu: Tùy Chọn Lookback Tier 2 & Sửa Lỗi Bộ Lọc Lệch Kiểu MongoDB

This walkthrough details the changes made to both cdc-cms-web (Frontend) and centralized-data-service (Backend) to address lookback options for both Check and Heal actions, hide datetimes dynamically on UI, resolve MongoDB date-type mapping issues, and fix the "upper bound range clamp" bug for manual scans.

---

## English Version

### Summary of Changes

#### 1. Frontend Modifications (cdc-cms-web)
* **`src/hooks/useReconStatus.ts`**:
  * Updated `useCheckTableMutation` payload schema to accept `lookback?: string` and post it in the payload.
  * Updated `useHealMutation` payload schema to include `lookback?: string` and post it in the payload.
* **`src/components/ConfirmDestructiveModal.tsx`**:
  * Added `isCheckTier2?: boolean` prop control.
  * Dynamically hide the `startTime` and `endTime` datetime-local pickers when `mode === 'window'` or when checking Tier 2 (`isCheckTier2 === true`).
  * Render radio buttons for lookback selection (`Hot Mode` - 2h lookback, `Cold Lookback` - 7d lookback) when `mode === 'window'` or when checking Tier 2 (`isCheckTier2 === true`).
  * Conditionally render datetime pickers only in `full_diff` mode.
  * Updated `onConfirm` callback signature to propagate `lookback` value back to the parent page.
* **`src/pages/DataIntegrity.tsx`**:
  * Configured `openCheckTable` action to pass `isCheckTier2: String(record.tier) === '2'`.
  * Caught `lookback` callback from modal submit and routed it to `checkTable.mutateAsync` (for check action) and `heal.mutateAsync` (for heal action).
  * Rendered the modal with `isCheckTier2` prop mapped from modal state.

#### 2. Backend Modifications (centralized-data-service)
* **`internal/handler/recon/recon_handler_run.go`**:
  * Extended NATS unmarshal payload struct with `Lookback` field for both `HandleReconCheck` and `HandleReconHeal`.
  * For Tier 2 check:
    * Injected `"manual_lookback" = true` into the context.
    * If `payload.Lookback == "cold"`, wraps context containing `"cold_lookback" = true` (7-day lookback).
    * Otherwise, routes to a default context (2-hour lookback).
  * Propagated `payload.Lookback` to `healSegmentA` for heal actions.
* **`internal/handler/recon/recon_heal_v4.go`**:
  * Refactored `healSegmentA` signature to accept `lookback`.
  * Injected `"manual_lookback" = true` into context for both Hot and Cold modes.
  * Integrated lookback routing:
    * `lookback == "hot"` routes to context (2-hour scan lookback).
    * Otherwise, routes to context containing `"cold_lookback" = true` (7-day scan lookback).
* **`internal/service/recon/recon_tier_a.go`**:
  * In `pickScanRangeWithLag`, checks if context contains `"manual_lookback" = true`.
  * If the cờ exists, **bypasses** clamping `upper` to the historical `srcMax` and `dstMax`, keeping `upper = nowFreeze` (realtime UTC now minus lag margin).
  * This guarantees that manual lookbacks (Hot/Cold) scan actual recent windows (e.g. last 7 days/2 hours) rather than getting stuck in the past when source DB is static.
* **`internal/service/recon/recon_stream.go`**:
  * Fixed MongoDB filter inside `StreamIDsInTimeRange` using `$or` operator:
    * Supports both Date format (`time.Time`) and Epoch Milliseconds (`int64` / `primitive.DateTime`), resolving the 0-doc empty stream bug.

---

### Verification and Test Results

#### 1. Frontend Build Pass
* Ran `npm run build` inside `cdc-cms-web`.
* Compilation was 100% successful with typecheck passed.

#### 2. Backend Unit Test Pass
* Ran unit tests inside `internal/handler/recon/...` and `internal/service/recon/...`.
* All tests passed without regress.

---

## Tiếng Việt

### Tóm Tắt Các Thay Đổi

#### 1. Sửa Đổi Frontend (cdc-cms-web)
* **`src/hooks/useReconStatus.ts`**:
  * Cập nhật payload `useCheckTableMutation` thêm tham số tùy chọn `lookback?: string`.
  * Cập nhật payload `useHealMutation` thêm tham số tùy chọn `lookback?: string`.
  * Gửi kèm giá trị `lookback` vào body request POST tới API tương ứng.
* **`src/components/ConfirmDestructiveModal.tsx`**:
  * Bổ sung prop `isCheckTier2?: boolean` để điều khiển hiển thị.
  * Ẩn hoàn toàn DatePicker `startTime`/`endTime` khi chọn `mode === 'window'` hoặc khi thực hiện kiểm tra Tier 2 (`isCheckTier2 === true`).
  * Hiển thị radio button chọn cửa sổ lookback: Hot Mode (quét 2 giờ gần nhất) / Cold Lookback (quét 7 ngày gần nhất) khi ở chế độ window hoặc check Tier 2.
  * Chỉ hiển thị DatePicker chọn thời gian khi ở chế độ `full_diff` của Heal.
  * Cập nhật callback signature của `onConfirm` truyền tiếp `lookback` lên component cha.
* **`src/pages/DataIntegrity.tsx`**:
  * Cấu trúc `openCheckTable` truyền thêm cờ `isCheckTier2: String(record.tier) === '2'`.
  * Nhận `lookback` từ modal callback và gửi sang `checkTable.mutateAsync` (đối với Check) và `heal.mutateAsync` (đối với Heal).
  * Truyền prop `isCheckTier2` cho `ConfirmDestructiveModal` dựa trên trạng thái modal hiện tại.

#### 2. Sửa Đổi Backend (centralized-data-service)
* **`internal/handler/recon/recon_handler_run.go`**:
  * Cấu trúc thêm trường `Lookback` trong NATS payload unmarshal struct cho cả 2 hàm `HandleReconCheck` và `HandleReconHeal`.
  * Đối với kiểm tra Tier 2:
    * Luôn truyền cờ `"manual_lookback" = true` vào context.
    * Nếu `payload.Lookback == "cold"`, wrap context chứa key `"cold_lookback" = true` (quét lookback 7 ngày).
    * Ngược lại, dùng context gốc (mặc định quét lookback 2 giờ).
  * Chuyển tiếp `payload.Lookback` vào hàm `healSegmentA`.
* **`internal/handler/recon/recon_heal_v4.go`**:
  * Thay đổi chữ ký hàm `healSegmentA` để nhận thêm tham số `lookback`.
  * Truyền cờ `"manual_lookback" = true` vào context cho cả Hot và Cold modes.
  * Phân nhánh context:
    * `lookback == "hot"` dùng context (mặc định quét lookback 2 giờ).
    * Ngược lại dùng context chứa key `"cold_lookback" = true` (quét lookback 7 ngày).
* **`internal/service/recon/recon_tier_a.go`**:
  * Trong hàm `pickScanRangeWithLag`, kiểm tra nếu context có cờ `"manual_lookback" = true`.
  * Nếu phát hiện cờ, **bỏ qua việc kẹp lùi `upper`** về `srcMax` và `dstMax` trong quá khứ, giữ `upper = nowFreeze` (thời điểm thực tế hiện tại trừ lag margin).
  * Điều này đảm bảo rằng các lượt quét thủ công (Hot/Cold) sẽ quét thực tế trên 7 ngày/2 giờ gần nhất thay vì bị mắc kẹt tại mốc dữ liệu cũ khi database nguồn tĩnh.
* **`internal/service/recon/recon_stream.go`**:
  * Sửa lỗi bộ lọc MongoDB trong `StreamIDsInTimeRange` sử dụng `$or`:
    * Hỗ trợ song song cả kiểu Date (`time.Time`) và Epoch Ms (`int64` / `primitive.DateTime`), khắc phục triệt để lỗi trả về 0 docs do lệch kiểu dữ liệu.

#### 3. Sửa Đổi API Gateway (cdc-cms-service)
* **`internal/app/commands/recon/recon_check.go`**: Mở rộng `ReconCheckCommand` struct nhận thêm `Lookback` field.
* **`internal/app/commands/recon/recon_async.go`**: Mở rộng `ReconHealCommand` struct nhận thêm `Mode`, `StartTime`, `EndTime`, và `Lookback`.
* **`internal/api/recon/reconciliation_handler_commands.go`**:
  * Trong cả `TriggerCheck` và `TriggerCheckAll` (định tuyến khi UI gửi request không chứa table param): Sử dụng struct gộp để unmarshal `lookback` từ JSON request body và gán chính xác vào `ReconCheckCommand` gửi đi NATS.
* **`internal/api/recon/reconciliation_handler_heal.go`**:
  * Trong `TriggerHeal`: Unmarshal đầy đủ các tham số `mode`, `start_time`, `end_time` và `lookback` từ HTTP body để gán và dispatch sang `ReconHealCommand`.

---

### Kết Quả Xác Minh & Kiểm Thử

#### 1. Frontend Build Thành Công
* Chạy build thành công `npm run build` trong `cdc-cms-web`. Không phát sinh bất kỳ lỗi compile nào.

#### 2. Backend Unit Test Pass 100%
* Chạy unit tests trong `internal/handler/recon/...` và `internal/service/recon/...` của `centralized-data-service` đều PASS sạch sẽ.
* Chạy unit tests trong `internal/...` của `cdc-cms-service` đều PASS sạch sẽ.

