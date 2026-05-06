# Context

Yêu cầu: Backend lấy thêm dữ liệu cho loại giao dịch “Hoàn tiền nạp tiền” (REFUND_CASHIN) khi thực hiện xuất file dữ liệu `InternalTransferExport`, cụ thể đối với các cột dữ liệu sau:
- Mã giao dịch tham chiếu
- Connector
- Lý do thất bại
- Mã giao dịch chuyển khoản gốc
- Ngân hàng gửi
- Số tài khoản gửi
- Tên tài khoản gửi

File thực thi: `/Users/trainguyen/Documents/work/centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts`
