# Implementation Plan (Refined): Flatten Only Loop 1st Element in flatten.go

## Goal
Chỉ điều chỉnh duy nhất 1 file `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmute/flatten.go` để trong bước Transmute (`flatten.go`), khi nhận mảng `elements`, nó chỉ loop đúng 1 phần tử đầu tiên (`elements[0]`), không đệ quy hay loop các phần tử tiếp theo.

## Proposed Changes

### `centralized-data-service/internal/service/master/transmute/flatten.go`
- [MODIFY] Cắt ngắn mảng `elements` chỉ lấy phần tử đầu tiên (`elements = elements[:1]`) nếu `len(elements) > 1` (hoặc ngắt vòng lặp `for idx, elem := range elements` sau `idx == 0`).

## Verification Plan
- Chạy unit tests `go test ./internal/service/master/transmute/...` kiểm tra `flatten.go` chỉ sinh ra 1 `Emit` duy nhất cho phần tử đầu tiên.
