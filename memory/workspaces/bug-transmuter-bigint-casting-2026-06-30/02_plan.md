# Plan: Sửa lỗi transmuter cast bigint và làm giàu log lỗi (Cập nhật theo yêu cầu User)

## Kế hoạch chi tiết

1. **Bổ sung hàm helper tìm field lỗi từ err ở `internal/service/master/transmuter_utils.go`**:
   - Không tự động ép kiểu (no coercion) cho bigint/integer nữa.
   - Viết hàm `findFailedFieldFromErr(err error, records []map[string]any) string` để trích xuất giá trị lỗi trong nháy kép từ error message và tìm ra cột tương ứng trong record.

2. **Làm giàu thông tin lỗi tại `internal/service/master/transmuter.go`**:
   - Trong hàm `bulkUpsertMaster`, nếu câu lệnh Raw SQL Scan trả về error, gọi `findFailedFieldFromErr`.
   - Nếu tìm thấy cột bị lỗi, đóng gói thêm thông tin này vào error trả về: `fmt.Errorf("%w (potential failed field: %q)", err, failedField)`.

3. **Viết unit test kiểm nghiệm**:
   - Viết trực tiếp unit test trong `internal/service/master/transmuter_test.go` hoặc file test phù hợp để kiểm nghiệm:
     - Logic bóc tách field lỗi từ error message khi có lỗi casting xảy ra.

4. **Chạy kiểm thử và Audit**:
   - Chạy `go test` trên package master để đảm bảo tất cả test cases đều pass và không có regression.
   - Chạy build `go build ./cmd/worker` để verify code compile tốt.
