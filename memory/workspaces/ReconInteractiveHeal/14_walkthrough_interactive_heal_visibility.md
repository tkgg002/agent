# Walkthrough - Khắc phục hiển thị dữ liệu chưa Heal và Nâng cấp modal Chữa lành đối soát

## Những thay đổi đã thực hiện

### 1. Frontend (`cdc-cms-web`)

#### [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- Cập nhật hàm `openHeal` để thiết lập `executeHealTarget` thay vì `modalPlan`. Khi click nút "Chữa lành" (MedicineBoxOutlined), giao diện sẽ mở trực tiếp modal `ExecuteHealModal`.

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- **Tiêu đề:** Thay đổi tiêu đề hiển thị từ `"Chữa lành drift — "` sang `"Chữa lành đối soát cho "`.
- **Cột Loại kiểm tra:** Thêm cột hiển thị thông tin loại kiểm tra (`check_type`) dưới dạng các nhãn Tag thân thiện: Smoke Check (7 ngày), Hash Window, Full Search, Deep Check.
- **Cột ID lệch:** 
  - Thêm cột hiển thị danh sách ID bị lệch.
  - Khi số lượng ID lớn hơn 2, hiển thị 2 ID đầu kèm icon 👁️ (Popover). Khi người dùng click vào icon sẽ hiển thị Popover chứa toàn bộ danh sách ID lệch dạng Tag và cung cấp nút copy nhanh toàn bộ ID.
- **Unique IDs:** Lọc trùng danh sách `reportIds` ở Frontend trước khi gọi mutation gửi về Backend.
- **Kích thước Modal:** Tăng độ rộng của modal `width` lên `960` để hiển thị các cột cân đối, premium.

### 2. Backend (`centralized-data-service`)

#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- **Unique Report IDs:** Sử dụng hàm helper `uniqueUint64s` để lọc trùng danh sách `opts.ReportIDs` nhận từ API trước khi thực hiện các bước tiếp theo.
- **Unique Record IDs:** 
  - Trong `executeHealSegA`: lọc trùng các mảng `staleA.Mismatched`, `staleA.MissingFromSrc`, và `missingIDs` trước khi heal.
  - Trong `executeHealSegB`: lọc trùng các mảng `staleB.StaleIDs`, `staleB.OrphanInMaster`, và `missingGpayIDs` trước khi heal.
  - Nhờ đó, loại bỏ hoàn toàn việc xử lý trùng lặp đối với cùng một record ID trong các chặng Ingest (A) và Transmute (B).
- **Hàm helper:** Bổ sung `uniqueStrings` và `uniqueUint64s` ở cuối tệp.

## Kết quả kiểm thử & Xác minh

### Kiểm tra tĩnh & Biên dịch (Static Verification)
- **Frontend (`cdc-cms-web`):**
  ```bash
  npx tsc --noEmit
  ```
  Kết quả: Thành công 100%, không phát sinh lỗi biên dịch.
- **Backend (`centralized-data-service`):**
  ```bash
  go build ./cmd/... ./internal/...
  ```
  Kết quả: Thành công 100%, không có lỗi biên dịch.
- **Unit Tests:**
  ```bash
  go test -v ./internal/handler/recon/...
  go test -v ./internal/service/recon/...
  ```
  Kết quả: Toàn bộ test suites đều **PASS**.

### Kiểm tra tuân thủ quy trình (Governance Audit)
- Chạy script kiểm tra quy trình:
  ```bash
  python3 verify_governance.py
  ```
  Kết quả: **PASS 100%** (Workspace: `ReconInteractiveHeal`).
