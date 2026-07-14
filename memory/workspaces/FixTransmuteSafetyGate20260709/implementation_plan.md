# Kế hoạch triển khai của Muscle - Khắc phục transmute safety gate batchSize

Kế hoạch này được Muscle lập ra nhằm triển khai phương án sửa đổi mã nguồn đã được phê duyệt tại file `09_tasks_solution_transmute_safety_gate.md`.

## 1. Mục tiêu
- Loại bỏ kiểm tra giới hạn `onlySourceIDs` ở đầu hàm `Run` của `TransmuterModule`.
- Thực hiện chia lô (chunking) `onlySourceIDs` thành các chunk có kích thước tối đa là `batchSize` của module (mặc định là 2000).
- Thực thi vòng lặp qua các chunk, gọi xử lý shadow batch và orphan master (soft-delete) theo từng chunk, cộng dồn kết quả vào `TransmuteResult`.
- Viết test case `TestTransmuter_OrphanMasterChunking` trong `transmuter_orphan_test.go` để verify logic chia lô hoạt động đúng.
- Chạy test suite `go test -v ./internal/service/master/...` từ thư mục `centralized-data-service`.

## 2. Chi tiết các bước thực hiện

### Bước 2.1: Sửa logic trong `transmuter.go`
Sửa hàm `Run` của `TransmuterModule` tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) để:
- Loại bỏ đoạn kiểm tra check size đầu hàm:
  ```go
  if len(onlySourceIDs) > t.batchSize {
      return TransmuteResult{Master: masterName}, fmt.Errorf("transmute safety gate: len(onlySourceIDs) = %d vượt quá batchSize = %d, hãy chia lô nhỏ hơn", len(onlySourceIDs), t.batchSize)
  }
  ```
- Thêm logic chia lô (chunking) cho `onlySourceIDs`:
  ```go
  var idChunks [][]string
  if len(onlySourceIDs) > 0 {
      for i := 0; i < len(onlySourceIDs); i += t.batchSize {
          end := i + t.batchSize
          if end > len(onlySourceIDs) {
              end = len(onlySourceIDs)
          }
          idChunks = append(idChunks, onlySourceIDs[i:end])
      }
  } else {
      idChunks = [][]string{nil}
  }
  ```
- Bao bọc toàn bộ logic xử lý sync (fetchShadowBatch, orphan master check, processBatch, save checkpoint) bằng vòng lặp `for _, chunkIDs := range idChunks`. 
- Thay thế biến `onlySourceIDs` bằng `chunkIDs` trong phạm vi vòng lặp này.

### Bước 2.2: Thêm test case trong `transmuter_orphan_test.go`
- Thêm test case `TestTransmuter_OrphanMasterChunking` vào [transmuter_orphan_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_orphan_test.go).
- Test case này sẽ khởi tạo một SQLite memory DB, chèn 5 bản ghi shadow, cấu hình `batchSize = 2` bằng reflection, sau đó chạy `Run` với `onlySourceIDs` có 5 phần tử. Kết quả mong đợi là scan và insert thành công 5 bản ghi vào master DB mà không gặp lỗi safety gate.

### Bước 2.3: Thực hiện chạy thử nghiệm và kiểm chứng
- Chạy `go test -v ./internal/service/master/...` tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`.
- Kiểm chứng kết quả test, đảm bảo test case mới và các test case cũ đều pass.

### Bước 2.4: Hoàn thành tài liệu và báo cáo tiến độ
- Cập nhật nhật ký tiến độ vào [05_progress_transmute_safety_gate.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/FixTransmuteSafetyGate20260709/05_progress_transmute_safety_gate.md).
- Tạo báo cáo kết quả chi tiết gửi User.
- Không thực hiện git commit.
