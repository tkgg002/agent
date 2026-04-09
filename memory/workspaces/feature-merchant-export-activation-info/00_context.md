# Workspace: Merchant Export Activation Info

## 00_context.md - Tổng quan bối cảnh

### Mục tiêu
Bổ sung các trường thông tin: **Merchant Code**, **Ngày kích hoạt**, **Người kích hoạt** vào file export Merchant.

### Logic đặc biệt (Activation Date)
- **TH1 (Tạo mới)**: Lấy `createdAt`.
- **TH2 (Tái kích hoạt)**: Lấy ngày kích hoạt cuối cùng trong **Merchant History** (Khi trạng thái chuyển từ Inactive sang Active).

### Tài liệu liên quan
- **File đích**: `/Users/trainguyen/Documents/work/centralized-export-service/logics/export/merchant/merchant-export.pure.ts`
- **Nguồn dữ liệu**: Merchant Collection & Merchant History Collection.

### Cập nhật 2026-03-12
- **Yêu cầu**: Thêm filter Merchant Name, Merchant Code. Bỏ filter SĐT ví. Tìm gần đúng Name, Email, MST.
- **Scope**: Update `merchant-export.params.ts` và `merchant-export.pure.ts`.

### Metadata
- **Agent**: Brain (antigravity), Muscle (CC CLI)
- **Status**: 🟡 Active
