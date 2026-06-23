# Context: Refactor BatchTransformHandler for Performance and Reliability

## Goal
Tối ưu hóa và tái cấu trúc `BatchTransformHandler` trong `internal/handler/master/batch_transform_handler.go` để giải quyết các rủi ro về hiệu năng (DB crash, missing index khi quét shadow table) và tăng độ bền bỉ cho luồng ánh xạ dữ liệu cdc.

## Active Files
- [batch_transform_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/batch_transform_handler.go)

## Current Status
- Chưa phân tích chi tiết mã nguồn hiện tại của file `batch_transform_handler.go`. Cần thực hiện research cấu trúc, thuật toán chunked update và các câu lệnh SQL để xác định chính xác các điểm cần refactor.
