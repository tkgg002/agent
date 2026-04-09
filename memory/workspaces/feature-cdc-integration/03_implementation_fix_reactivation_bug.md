# Implementation Plan: Fix Reactivation Sync Bug (Registry -> Airbyte)

Người dùng báo cáo: Khi chuyển trạng thái từ `inactive` sang `active`, Airbyte không được cập nhật. 

## 1. Gốc rễ vấn đề (Root Cause)
Trong hàm `syncRegistryStateToAirbyte`:
- Khi người dùng `inactive` bảng cuối cùng, hệ thống tự động đặt lại `Selected = true` (dòng 486) để thỏa mãn yêu cầu của Airbyte API ("Phải có ít nhất 1 stream được chọn").
- Đồng thời trạng thái Connection được đặt thành `inactive`.
- Khi người dùng `active` lại bảng đó:
    - `shouldSync` là `true`.
    - Nhưng `originalSelected` đã là `true` (do bị force ở bước trên).
    - `newStatus` là `active`.
    - Nếu Connection ID đó đã ở trạng thái `active` (do các stream khác) HOẶC logic so sánh `hasChanges` bị sai lệch, hệ thống sẽ bỏ qua lệnh `UpdateConnection` (dòng 497-499).

## 2. Giải pháp đề xuất

### [MODIFY] [registry_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/registry_handler.go)
- Cải thiện logic `hasChanges`: Đảm bảo rằng nếu trạng thái mong muốn (`shouldSync`) khác với trạng thái thực tế trong Catalog của Airbyte dành cho stream đó, chúng ta PHẢI thực hiện update.
- Loại bỏ việc "revert" `Selected = true` nếu chúng ta sắp đưa Connection về trạng thái `inactive`. Airbyte chấp nhận `Status: inactive` ngay cả khi Catalog có vấn đề (hoặc ít nhất chúng ta nên thử để giảm side-effect). Nếu API vẫn yêu cầu, chúng ta sẽ giữ 1 stream bất kỳ nhưng logic so sánh phải tách biệt.
- **Quan trọng**: Bổ sung log chi tiết hơn để debug quá trình so sánh catalog.

## 3. Các bước thực hiện
1. Sửa hàm `syncRegistryStateToAirbyte`:
   - So sánh trực tiếp `shouldSync` với `s.Config.Selected` của stream mục tiêu.
   - Luôn đặt `hasChanges = true` nếu có sự sai lệch giữa `IsActive` (CMS) và `Selected` (Airbyte).
2. Kiểm tra lại logic `create_cdc_table` để đảm bảo không bị lỗi khi bảng đã tồn tại.

## 4. Kế hoạch xác minh (Verification)
- Scenario A: Active -> Inactive (bảng duy nhất). Kiểm tra Airbyte Connection status.
- Scenario B: Inactive -> Active (bảng duy nhất). Kiểm tra Airbyte Connection status có về `active` và stream có được `selected` không.
- Scenario C: Update nhiều bảng cùng lúc.
