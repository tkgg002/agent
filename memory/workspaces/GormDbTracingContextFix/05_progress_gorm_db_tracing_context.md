# Progress: GORM DB Tracing Context Propagation

## Root Cause Analysis
- **Vấn đề**: Mặc dù đã kích hoạt plugin OpenTelemetry cho GORM (`tracing.NewPlugin()`), traces của các câu query SQL thực tế vẫn không xuất hiện trên SigNoz/Jaeger.
- **Nguyên nhân gốc rễ**: Rất nhiều câu query GORM trong `centralized-data-service` (đặc biệt là các service phụ trách schema, bridge, activity logs, masking...) được gọi trực tiếp bằng `db.Raw`, `db.Exec`, `db.Create` mà **không thông qua `.WithContext(ctx)`**. Khi thiếu context, OTel plugin của GORM không thể liên kết các DB spans con với spans cha của request/nats message, dẫn đến trace DB bị mất hoặc cô lập.

## Audit Log
- `[2026-08-06 09:37:00] [Antigravity:Gemini 3.5 Flash] Chạy tiến trình QC phát hiện thiếu sót nghiêm trọng về việc thiếu WithContext(ctx) trên hàng loạt câu query GORM. Khởi tạo workspace GormDbTracingContextFix.`
