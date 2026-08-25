# 06_test_cases.md — Kế hoạch kiểm thử

## 1. Test Cases Backend (`cdc-cms-service`)
- **TC-BE-01: Read Model trả về `snapshot_max_rps`**
  - Input: Gọi `GET /api/v1/source-objects`.
  - Kỳ vọng: JSON response chứa trường `snapshot_max_rps` với kiểu integer hoặc null.
- **TC-BE-02: Cập nhật `snapshot_max_rps` hợp lệ**
  - Input: Gọi `PATCH /api/v1/source-objects/:id` với payload `{"snapshot_max_rps": 1500}`.
  - Kỳ vọng: HTTP 200, database được cập nhật `snapshot_max_rps = 1500`.
- **TC-BE-03: Xóa giới hạn (Clear về NULL)**
  - Input: Gọi `PATCH /api/v1/source-objects/:id` với payload `{"snapshot_max_rps": 0}`.
  - Kỳ vọng: HTTP 200, database được cập nhật `snapshot_max_rps = NULL`.
- **TC-BE-04: Validate giá trị không hợp lệ**
  - Input: Gọi `PATCH /api/v1/source-objects/:id` với payload `{"snapshot_max_rps": 5}` hoặc `{"snapshot_max_rps": 200000}`.
  - Kỳ vọng: HTTP 400 Bad Request (`invalid_snapshot_max_rps`).

## 2. Test Cases Frontend (`cdc-cms-web`)
- **TC-FE-01: Hiển thị trường Snapshot Max RPS trong Modal**
  - Thao tác: Bấm nút "Sửa Source" tại một Source Object bất kỳ.
  - Kỳ vọng: Modal hiển thị trường "Snapshot Max RPS (snapshot.v2)" với InputNumber và placeholder đúng.
- **TC-FE-02: Lưu giá trị và load lại**
  - Thao tác: Nhập `1500` -> Bấm OK -> Mở lại Modal.
  - Kỳ vọng: Giá trị `1500` hiển thị chính xác trong Form.
- **TC-FE-03: Xóa giá trị**
  - Thao tác: Xóa trắng ô InputNumber -> Bấm OK -> Mở lại Modal.
  - Kỳ vọng: Trường hiển thị placeholder "Để trống = không giới hạn tốc độ".
