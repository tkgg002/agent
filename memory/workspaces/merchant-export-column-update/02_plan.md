# Implementation Plan: Cập nhật template Export Merchant

## Tiếng Việt

**Mục tiêu**: Bổ sung và cập nhật lại các cột trong file Excel xuất danh sách Merchant theo yêu cầu.

### Đề xuất thay đổi (Proposed Changes)

#### [MODIFY] `domain/queries/merchant/GetMerchantExportAuxiliaryQuery.ts`
- Thêm trường `masterMerchantIds: string[]` vào class và constructor để truyền danh sách ID của Master Merchant.

#### [MODIFY] `domain/handlers/merchant/GetMerchantExportAuxiliaryHandler.ts`
- Bổ sung logic lấy thông tin `MasterMerchant`:
  - Khai báo model: `const merchantModel = this.mainProcess.models.MerchantModel;`
  - Fetch dữ liệu bằng `merchantModel.find({ _id: { $in: masterMerchantIds }, isDelete: false }).lean().exec()` và trả về trong data.

#### [MODIFY] `logics/export/merchant/merchant-export.pure.ts`
- **SubQuery Building (`buildSubQueryParams`)**:
  - Trích xuất `masterMerchantIds` từ danh sách `primaryData` (dựa trên `masterMerchantId`).
- **Data Merging (`mergeData`)**:
  - Tạo Map cho `masterMerchants`.
  - Gán `masterMerchantObj` vào từng báo cáo merchant (cả cho Mongoose object `.set()` và js object thường).
- **Columns configuration**:
  - Đổi tên cột "Số điện thoại ví" thành "Số ĐT Ví".
  - Thêm cột "Người kích hoạt" ngay sau "Ngày kích hoạt".
  - Thêm cột "MasterMerchantID" ở vị trí cuối.
  - Thêm cột "Tên Đơn vị phát lệnh" ở vị trí cuối.
- **Data Transformation (`transformRow`)**:
  - Cập nhật map cho cột "Tạo bởi" sử dụng đúng `item.createdBy || ""`.
  - Cập nhật map cho cột "Người kích hoạt" sử dụng `item.activator || ""`.
  - Cập nhật map cho cột "Số ĐT Ví" (vẫn là `item.userMobileRef`).
  - Thêm mapping cho "MasterMerchantID" (`item.masterMerchantId || ""`).
  - Thêm mapping cho "Tên Đơn vị phát lệnh" (`item.masterMerchantObj?.name || ""`).

### Verification Plan
- Chạy `yarn tsc` hoặc `npm run build` để kiểm tra lỗi cú pháp/mô hình.
- Đảm bảo các field tồn tại và được export mà không thiếu sót.
- Xác nhận logic merge data chạy độc lập và không block các query khác (fallback rỗng nếu không có masterMerchant).

---

## English

**Goal**: Add new columns and update existing columns mapping in the Merchant Export Excel file according to requirements.

### Proposed Changes

#### [MODIFY] `domain/queries/merchant/GetMerchantExportAuxiliaryQuery.ts`
- Add `masterMerchantIds: string[]` to the class and constructor to pass the Master Merchant IDs array.

#### [MODIFY] `domain/handlers/merchant/GetMerchantExportAuxiliaryHandler.ts`
- Add logic to fetch `MasterMerchant` information:
  - Add model reference: `const merchantModel = this.mainProcess.models.MerchantModel;`
  - Fetch records using `merchantModel.find({ _id: { $in: masterMerchantIds }, isDelete: false }).lean().exec()` and return them in the payload.

#### [MODIFY] `logics/export/merchant/merchant-export.pure.ts`
- **SubQuery Building (`buildSubQueryParams`)**:
  - Extract unique `masterMerchantIds` from the `primaryData` (based on `merchant.masterMerchantId`).
- **Data Merging (`mergeData`)**:
  - Map `masterMerchants` by their ID.
  - Assign `masterMerchantObj` to each merchant item in `primaryData`.
- **Columns configuration**:
  - Rename the column "Số điện thoại ví" to "Số ĐT Ví".
  - Add the "Người kích hoạt" (Activator) column right after "Ngày kích hoạt" (Activation date).
  - Add the "MasterMerchantID" and "Tên Đơn vị phát lệnh" columns at the end of the array.
- **Data Transformation (`transformRow`)**:
  - Map "Tạo bởi" to just `item.createdBy || ""`.
  - Map "Người kích hoạt" to `item.activator || ""`.
  - Map "Số ĐT Ví" (remains `item.userMobileRef`).
  - Map "MasterMerchantID" to `item.masterMerchantId || ""`.
  - Map "Tên Đơn vị phát lệnh" to `item.masterMerchantObj?.name || ""`.

### Verification Plan
- Run `yarn tsc` or `npm run build` to verify type checking.
- Code review to ensure the columns are added at precise indices and data transformation returns an array that exactly matches the defined columns schema.
