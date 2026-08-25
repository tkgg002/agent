# Solution Specification: Fix OOM Transient & Poison Pill

## Technical Solution Details

### File 1: `centralized-data-service/internal/handler/master/transmute_handler.go`

1. Cập nhật hàm `isTransientError`:
```go
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection pool") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "sqlstate 08") ||
		strings.Contains(msg, "out of memory") ||
		strings.Contains(msg, "sqlstate 53200")
}
```

2. Cập nhật hàm `processSubBatch`:
```go
	res, err := h.svc.Run(runCtx, masterTable, ids, "")
	resp := TransmuteResponse{TransmuteResult: res}

	if err != nil {
		if logEntry != nil {
			h.activity.Fail(logEntry, err.Error())
		}
		// Nếu lỗi hạ tầng/mạng tạm thời xảy ra trong lúc chia tách, dừng split ngay lập tức
		if isTransientError(err) {
			for _, t := range subBatch {
				h.replyErr(ctx, t.Msg, t.Req.CorrelationID, "transient_db_error: "+err.Error())
			}
			return
		}
		// Đệ quy tiếp tục chia đôi nửa này
		h.binarySearchSplit(ctx, masterTable, subBatch)
	} else {
```
