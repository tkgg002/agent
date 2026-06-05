# Báo cáo kỹ thuật: Khắc phục Data Hazard & Tối ưu OCC trong CDC (2026-05-26)

## 1. Vấn đề (Root Cause Analysis)
1. **Data Hazard trên Batch Upsert**: Mặc dù Batch Upsert giúp tăng tốc, nhưng do các bản ghi trong cùng một lô (chunk) có thể có cấu trúc schema (keys) khác nhau (spec drift). Việc dùng chung một schema cố định cho 500 bản ghi sẽ dẫn đến việc các field không tồn tại trong một số bản ghi bị ép chèn giá trị `NULL` (hoặc zero-value) đè lên dữ liệu cũ đang hợp lệ.
2. **Redundant Immutable Update**: Trong `schema_adapter.go`, cột `_gpay_source_id` (được dùng làm conflict target hoặc anchor key) lại được gán `"_gpay_source_id" = EXCLUDED."_gpay_source_id"` trong nhánh `UPDATE SET`. Vì đây là cột không thể thay đổi, hành động này vừa vô nghĩa, vừa vi phạm best practice của PostgreSQL, gây phình WAL log (Write-Ahead Logging).
3. **OCC Drift trên Test**: Khi áp dụng luật LWW (Last Writer Wins) bằng cách thay thế `<= EXCLUDED._source_ts` thành `< EXCLUDED._source_ts`, các file test case tương ứng trong hệ thống chưa được update theo, dẫn đến test fail.

## 2. Giải pháp đã thực thi (Execution)
- **Triển khai "Signature Grouping"**: 
  - Tại `internal/handler/batch_buffer.go`, trước khi tiến hành Upsert, batch (500 records) được băm nhỏ thành các `subChunk` theo `Signature` (tổ hợp các schema keys của record).
  - Kết quả: Batch Upsert vẫn giữ nguyên hiệu năng siêu tốc, nhưng mỗi câu truy vấn gộp giờ đây hoàn toàn đồng nhất về cấu trúc Schema, ngăn chặn 100% rủi ro overwrite `NULL` do thiếu field.
- **Dọn dẹp Redundant Update (`_gpay_source_id`)**:
  - Gỡ bỏ hoàn toàn logic ép update cột `_gpay_source_id` trong 2 hàm `BuildUpsertSQLInSchema` và `BuildBatchUpsertSQLInSchema` của `internal/service/schema_adapter.go`.
  - Cập nhật lại `internal/service/schema_adapter_test.go` loại bỏ `_gpay_source_id` khỏi check-list `UPDATE SET`.
- **Cập nhật Test Coverage (OCC LWW)**:
  - Cập nhật lại 3 test trong `internal/service/recon_heal_test.go` để expect toán tử `<` theo chuẩn LWW mới nhất.
  - Sửa lại test `TestRegistryNamespaces_ThreeEnginesDebezium` trong `registry_service_test.go` do ảnh hưởng bởi lỗi regex "airbyte -> debezium" trước đó.

## 3. Xác minh (Verification)
- Đã chạy toàn bộ test suite (`go test ./... -v`). Mọi package, đặc biệt là `internal/service` và `internal/sinkworker` đều PASS 100%.
- Không còn bất cứ cảnh báo nào liên quan đến SQL syntax hoặc drift data.

## 4. Skills Đã Sử Dụng
- `view_file` / `grep_search`: Phân tích chuyên sâu `schema_adapter.go` và toàn bộ flow OCC/testing.
- `multi_replace_file_content`: Sửa chữa hàng loạt test case với độ chính xác tuyệt đối.
- `run_command`: Xác minh Unit Test & truy cập hệ thống file.
