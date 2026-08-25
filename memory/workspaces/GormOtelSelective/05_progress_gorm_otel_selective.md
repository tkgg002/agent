# Progress: GORM OpenTelemetry Selective Tracing

## Root Cause Analysis
- **Vấn đề**: Kích hoạt tracing cho toàn bộ GORM sinh ra lượng spans DB khổng lồ ("spam traces"), có nguy cơ làm quá tải OTel Collector và SigNoz.
- **Nguyên nhân gốc rễ**: GORM OTel plugin mặc định sinh span cho 100% các câu query. Lọc theo Tail-based sampling ở Collector có rủi ro bỏ sót lỗi logic nghiệp vụ. Hạ sample ratio (Head-based) mù quáng làm mất dấu vết liên kết trace gốc.
- **Giải pháp**: Xây dựng Custom Tracer Sampler ở mức SDK và cấu hình quản lý trung tâm (Central Trace Controller). Chỉ cho phép sinh spans DB đối với các context được đánh dấu bằng module cụ thể nằm trong danh sách cho phép (như `recon_heal`, `discover`, v.v.). Tắt hoàn toàn metrics của GORM plugin.

## Audit Log
- `[2026-08-06 14:20:00] [Antigravity:Gemini 3.5 Flash] Thiết lập workspace GormOtelSelective. Nghiên cứu giải pháp viết Custom Tracer Sampler cho OTel SDK để bật/tắt DB trace động.`
- `[2026-08-06 14:59:00] [Antigravity:Gemini 3.5 Flash] Lập giải pháp kỹ thuật chi tiết. Uỷ quyền subagent muscle-executor thực hiện chỉnh sửa mã nguồn.`
- `[2026-08-06 15:01:00] [Antigravity:Gemini 3.5 Flash] Chạy build và unit tests verify thành công. Toàn bộ tests và builds pass 100%. Đóng task, viết walkthrough_gorm_otel_selective.md.`
- `[2026-08-06 15:09:00] [Antigravity:Gemini 3.5 Flash] Chạy tiến trình QC gắt gao rà soát toàn bộ các tệp đã modify. Không phát hiện sai sót hay thiếu sót. Toàn bộ logic khớp 100% với plan.`
- `[2026-08-06 15:24:00] [Antigravity:Gemini 3.5 Flash] Phát hiện CMS HTTP API không hiển thị GORM spans do thiếu gán module cdc. Cập nhật yêu cầu thêm module cdc vào whitelist và bọc module cdc tại http_tracer/otel_middleware.`
- `[2026-08-06 15:37:00] [Antigravity:Gemini 3.5 Flash] Chạy build và unit tests verify cho HTTP GORM spans fix thành công. Toàn bộ tests và builds pass 100%. Đóng task, viết walkthrough_gorm_otel_selective.md.`
