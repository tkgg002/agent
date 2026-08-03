# Kế hoạch Sửa Lỗi Cú Pháp SQLSTATE 42601 (source_object_read_repo_gorm.go)

## Mô Tả Lỗi
API `GET /api/v1/source-objects` bị lỗi **HTTP 500 Internal Server Error**:
- File: `source_object_read_repo_gorm.go:127`
- Message: `ERROR: syntax error at or near "LEFT" (SQLSTATE 42601)`

---

## Nguyên Nhân & Giải Pháp Tối Ưu (Single Best Approach)

### Nguyên nhân:
Hằng số `listBaseFromWhere` trong `source_object_read_repo_gorm.go` bị thiếu dòng kết thúc subquery LATERAL `rr`:
```sql
ORDER BY rr.checked_at DESC
-- THIẾU: LIMIT 1 \n ) rr ON TRUE
LEFT JOIN LATERAL (
        SELECT status, rows_affected, job_id, error_message
        ...
) tj ON TRUE
```

### Giải pháp:
Bổ sung `LIMIT 1 \n ) rr ON TRUE` cho subquery `rr`, đồng thời thêm `AND rr.checked_at >= NOW() - INTERVAL '7 days'` để tối ưu tốc độ scan.

---

## User Review Required

> [!IMPORTANT]
> - Vui lòng phản hồi **`APPROVE`** để tiến hành sửa file `source_object_read_repo_gorm.go`.

---

## Proposed Changes

### cdc-cms-service (Go Backend)

#### [MODIFY] `internal/infra/persistence/source/source_object_read_repo_gorm.go`
- Sửa hằng số `listBaseFromWhere`: Bổ sung `LIMIT 1 \n ) rr ON TRUE` ở cuối subquery `rr`.

---

## Verification Plan

### Automated Tests
- Chạy biên dịch kiểm tra syntax Go: `go build ./cmd/server` tại repo `cdc-cms-service`.

### Manual Verification
- Xác nhận câu SQL biên dịch hợp lệ và API `/api/v1/source-objects` trả về lời gọi 200 OK.
