# Yêu cầu chi tiết: Tối ưu hoá Trace Tree cho luồng Snapshot V2

## Bối cảnh
Người dùng phát hiện Trace Tree bị cụt khi gọi API Snapshot ở `cdc-cms-service`, và thắc mắc liệu việc chạy snapshot 3 triệu record có gây ra crash hoặc tràn bộ nhớ trace hay không. 

## Yêu cầu
1. **Nối liền Trace Tree**: Luồng Trace từ `cdc-cms-service` (qua NATS) phải được kế thừa đúng đắn xuống `cdc-worker` (`centralized-data-service`).
2. **Ngăn chặn Trace Leak**: Đối với các tiến trình chạy theo batch kéo dài (như quét 3 triệu dòng), tuyệt đối không để 1 Trace Span mở liên tục trong nhiều giờ hoặc sinh ra hàng triệu span con, gây tràn giới hạn của OpenTelemetry.
3. **Cập nhật ID**: Ghi nhận đúng TraceID của OTel vào database `cdc_system.snapshot_progress` thay vì uuid ngẫu nhiên.
