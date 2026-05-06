# Report: Bổ sung dữ liệu Export cho giao dịch REFUND_CASHIN

**Ngày thực hiện:** 04/05/2026
**Workspace:** `feature-internal-transfer-export-refund-cashin`
**Dự án:** `centralized-export-service`

## 1. Yêu cầu (Context)
Lấy thêm dữ liệu cho loại giao dịch “Hoàn tiền nạp tiền” (REFUND_CASHIN) khi thực hiện xuất file dữ liệu, cụ thể đối với 7 cột sau:
- Mã giao dịch tham chiếu
- Connector
- Lý do thất bại
- Mã giao dịch chuyển khoản gốc
- Ngân hàng gửi
- Số tài khoản gửi
- Tên tài khoản gửi

## 2. Giải pháp triển khai
Cập nhật file `logics/export/trans-his/internal-transfer-export.pure.ts` (Hàm `transformRow`).
Logic đã được cập nhật ĐÚNG VỚI cấu trúc object từ hàm getList do user cung cấp:

Cụ thể mapping bổ sung cho `REFUND_CASHIN`:
1. **Mã giao dịch tham chiếu:** Lấy từ `history.transIdRef`.
2. **Mã giao dịch chuyển khoản gốc:** Ưu tiên lấy từ `history.originalTransHisId` và dự phòng lấy từ `history.info.refundFor`.
3. **Connector:** Lấy từ `history.info.bankConnector`.
4. **Lý do thất bại:** Fallback đọc từ `history.error.message` và `history.info.bankTransferResponse.failedReason`.
5. **Ngân hàng gửi:** Ưu tiên từ `history.info.dstBankAccount.bankCode` (fallback `history.sender.bankCode`).
6. **Số tài khoản gửi:** Ưu tiên từ `history.info.dstBankAccount.bankAccount` (fallback `history.sender.bankAccount`).
7. **Tên tài khoản gửi:** Ưu tiên từ `history.info.dstBankAccount.bankAccountName` (fallback `history.sender.bankAccountName`).

## 3. Test & Verification
- Compile test qua lệnh `npm run build`: **PASS** (Không có lỗi type checking).
- Logic đảm bảo khớp 100% về cấu trúc truy xuất với API GetList hiện hành.

## 4. Kỹ năng đã dùng
- System Workspace Management (Context, Progress, Plan tracking theo Governance).
- Source Code Modification (`replace_file_content`).
- TypeScript Compile Verification.
