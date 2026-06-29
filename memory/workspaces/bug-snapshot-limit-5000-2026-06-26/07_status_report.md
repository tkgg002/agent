# Status Report: Bug Snapshot Limit 5000 Records / Báo cáo Trạng thái: Lỗi Snapshot Giới hạn 5000 Records

## 1. Summary of Changes / Tóm tắt thay đổi
Sửa lỗi snapshot v2 chỉ chạy được 5000 records trên MongoDB sources có trường khóa chính `_id` dạng số (int32/int64/float64). Giải pháp được triển khai bằng cách inspect động kiểu của `_id` từ một document mẫu trước khi vào cursor loop, sau đó ép kiểu chuỗi `lastSeen` tương ứng trước khi query filter `$gt` để tránh type mismatch.

## 2. List of Changed Files & Line Counts / Danh sách các file thay đổi & Số dòng code thay đổi

| File Path | Action | Lines Changed/Added | Purpose |
| :--- | :--- | :--- | :--- |
| [snapshot_runner_utils_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_utils_test.go) | **NEW** | +83 lines | Viết unit test bao phủ toàn bộ các kiểu dữ liệu của `_id`. |
| [snapshot_runner_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_utils.go) | **MODIFY** | +36 lines | Triển khai `buildResumeFilterWithSample` hỗ trợ ép kiểu số. |
| [snapshot_runner_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go) | **MODIFY** | +18 lines | Lấy document mẫu để trích xuất `sampleID` kiểu `_id`. |
| **Total** | | **137 lines** | |

## 3. Detailed Technical Solutions / Giải pháp kỹ thuật chi tiết
1. **Trích xuất kiểu mẫu (`snapshot_runner_handler.go`)**:
   Trước khi bắt đầu vòng lặp Cursor, nếu là database MongoDB (`isMongo == true`), tiến hành query 1 bản ghi mẫu:
   ```go
   var sampleID interface{}
   if isMongo {
       sampleCtx, sampleCancel := context.WithTimeout(ctx, 5*time.Second)
       var sampleDoc bson.M
       if err := coll.FindOne(sampleCtx, bson.M{}).Decode(&sampleDoc); err == nil {
           sampleID = sampleDoc["_id"]
       }
       sampleCancel()
   }
   ```
2. **Ép kiểu resume filter (`snapshot_runner_utils.go`)**:
   Hàm `buildResumeFilterWithSample` nhận `sampleID` và ép kiểu chuỗi `lastSeen` tương ứng:
   ```go
   switch sampleID.(type) {
   case int32:
       if val, err := strconv.ParseInt(lastSeen, 10, 32); err == nil {
           return bson.M{"_id": bson.M{"$gt": int32(val)}}
       }
   // Tương tự cho int64, float64, primitive.ObjectID
   }
   ```
   Nếu `sampleID` là `nil` hoặc không phải kiểu số, hàm sẽ fallback tự động về String/ObjectID, đảm bảo tính tương thích ngược hoàn hảo.

## 4. Verification & Build Results / Kết quả Xác minh & Biên dịch
* **Unit Test**: Chạy `go test -v ./internal/handler/orchestration/...` PASS 100% với 10/10 test cases.
* **Compile Build**: Chạy `go build -v ./cmd/... ./internal/...` compile thành công 100%, không bị lỗi hồi quy.
