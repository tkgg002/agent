# Báo cáo Audit & Phân tích Parity Hệ thống Reconciliation mới

## 1. So sánh logic đối soát mới (`recon`) và cũ (`recon_bk`)

Hệ thống đối soát đã được di chuyển thành công từ cấu trúc nguyên khối `recon_bk` sang hai Handler chuyên biệt `CheckHandler` và `HealHandler` trong `internal/handler/recon`. 

### Parity Audit (Đánh giá tương đương)
- **Tách biệt logic:** Phiên bản mới đã tách biệt hoàn toàn vai trò của Check (chỉ kiểm tra sai lệch) và Heal (khắc phục sai lệch), đồng thời tái sử dụng các utils qua `ReconBase`.
- **Logic Nghiệp vụ chính:**
  - **Segment A (Source ↔ Shadow):** Giữ nguyên cơ chế so khớp hash bucket theo thời gian sử dụng `RunTier1`, `RunTier2`, `RunTier3` từ `ReconCore`.
  - **Segment B (Shadow ↔ Master):** Giữ nguyên logic so khớp dòng dữ liệu, đếm số lượng bản ghi active (loại bỏ deleted soft-deleted tombstone) và tìm kiếm stale/missing IDs.
- **Tính an toàn (Safety Thresholds):** Giữ nguyên các ngưỡng bảo vệ `healAutoMaxIDs = 1000` và `interactiveHealMaxIDs = 50000` tại `HealHandler`.
- **Dọn dẹp code:** Thư mục `internal/handler/recon_bk` đã hoàn toàn bị ngắt kết nối (không còn bất kỳ file import hay reference nào trong toàn bộ dự án `centralized-data-service`).

---

## 2. Chi tiết luồng đi (Flow) của 3 trường hợp Check

Dưới đây là chi tiết luồng xử lý từ Giao diện CMS (`cdc-cms-web`) -> API Gateway (`cdc-cms-service`) -> CDC Worker (`centralized-data-service`).

### 2.1. Lookback Mode (Hot/Cold)
- **UI Interaction:** Người dùng chọn "Lookback Mode". Lựa chọn "Hot Mode" (2 giờ gần nhất) hoặc "Cold Lookback" (7 ngày gần nhất).
- **CMS Web:** Gửi request `POST /api/reconciliation/check?tier=2` với payload:
  ```json
  {
    "table": "export_jobs",
    "lookback": "hot" | "cold",
    "segment": ""
  }
  ```
- **CMS Service:** Bọc thành `reconCmd.ReconCheckCommand` và dispatch lên NATS subject `cdc.cmd.recon-check`.
- **CDC Worker (`HandleReconCheck`):**
  1. Nhận payload. Do `start_time` và `end_time` bằng null nên không lưu Custom Range vào Context.
  2. Nếu `lookback == "cold"`, context được gán value `cold_lookback = true`.
  3. Gọi `RunTier2` (cho chặng A) hoặc `RunSegmentBFor` (cho chặng B).
  4. Hàm `pickScanRangeWithLag` gọi `effectiveLookback(ctx)`:
     - Nếu có cờ `cold_lookback`, trả về `rc.cfg.WindowLookback` (mặc định **7 ngày**).
     - Nếu không, trả về `rc.cfg.HotWindowLookback` (mặc định **2 giờ**).
  5. Tiến hành quét và đối soát theo khoảng thời gian lookback tương ứng.

### 2.2. Full Search (Full Diff)
- **UI Interaction:** Người dùng chọn "Full Search (Full Diff)", chỉ định RangePicker. FE kiểm tra nếu khoảng thời gian `> 30 ngày` thì disable nút submit và hiển thị cảnh báo lỗi.
- **CMS Web:** Gửi request `POST /api/reconciliation/check?tier=2` với payload:
  ```json
  {
    "table": "export_jobs",
    "start_time": 1782297600000,
    "end_time": 1783593600000,
    "segment": ""
  }
  ```
- **CMS Service:** Dispatch `ReconCheckCommand` chứa `StartTime` và `EndTime` dạng unix millisecond lên NATS subject `cdc.cmd.recon-check`.
- **CDC Worker (`HandleReconCheck`):**
  1. Nhận payload. Kiểm tra điều kiện: `start_time` và `end_time` bắt buộc khác null, `end_time >= start_time`, và khoảng cách không vượt quá **30 ngày** (tránh quá tải DB).
  2. Parse sang `time.Time` và lưu vào context qua `WithReconTimeRange(ctx, startT, endT)`.
  3. Trong `RunTier2` hoặc `RunSegmentB`, hàm `GetReconTimeRange(ctx)` trả về `true` và override lại khoảng thời gian quét `lo` và `hi` bằng custom range.
  4. Tiến hành so khớp hash-bucket cho khoảng thời gian tùy chọn này.

### 2.3. Deep Check
- **UI Interaction:** Người dùng chọn "Deep Check", chỉ định RangePicker (tối đa 30 ngày).
- **CMS Web:** Gửi request `POST /api/reconciliation/check?tier=2` với payload tương tự Full Search kèm `"deep": true`.
- **CMS Service:** Dispatch `ReconCheckCommand` với `"deep": true` qua NATS.
- **CDC Worker (`HandleReconCheck`):**
  1. Thực hiện các bước validate thời gian và lưu Custom Range vào Context giống hệt luồng Full Search.
  2. Phân nhánh chặng:
     - **Chặng A:** Thực hiện quét hash-bucket thông thường trên khoảng thời gian custom (do chặng A không có mapping rules nên `deep` không có tác động khác biệt).
     - **Chặng B (`segment == "shadow_master"`):** Gọi `RunSegmentB(ctx, ref, deep=true)`.
       - Sau khi tìm ra các dòng bị lệch (missing / stale IDs), do `deep = true`, Worker tiếp tục gọi `RunRowDiffB`.
       - `RunRowDiffB` lấy `_raw_data` từ shadow, chạy qua bộ mapping rules để so sánh chi tiết từng field với master table nhằm tìm ra các trường bị lệch giá trị và trả về chi tiết trong báo cáo.

---

## 3. Tính độc quyền (Mutual Exclusion) của các option trên UI

Trong file `ConfirmDestructiveModal.tsx`, các tuỳ chọn được ràng buộc chặt chẽ thông qua việc sử dụng chung một State `checkMode` kiểu `'lookback' | 'full_diff' | 'deep'`.

- Vì là các giá trị của một thẻ `<Radio.Group>`, **Lookback Mode**, **Full Search**, và **Deep Check** **không bao giờ có thể được chọn đồng thời**.
- Việc chuyển đổi giữa các `checkMode` sẽ cập nhật trực tiếp giao diện hiển thị:
  - Chọn `lookback`: Hiển thị lựa chọn nhỏ hơn (Hot Mode / Cold Lookback) và ẩn DatePicker.
  - Chọn `full_diff` hoặc `deep`: Ẩn lựa chọn Hot/Cold và hiển thị RangePicker giới hạn tối đa 30 ngày.
