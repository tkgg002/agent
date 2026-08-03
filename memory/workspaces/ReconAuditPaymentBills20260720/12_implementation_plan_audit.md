# 12 — Implementation Plan: Audit & Fix Timezone Drift in Recon Dest Agent

> Cập nhật: 2026-07-20T13:45:00+07:00 | Agent: Gemini-3.5-Flash
> Loại task: Hotfix/Refactor

---

## 1. Kết quả Audit Quá trình vừa thực hiện

### A. Những gì đã hoàn thành (Phase 1):
1. **Sửa đổi `recon_dest_hash.go` (Hàm `HashWindow`)**:
   - Đã adjust `tLo` và `tHi` sang múi giờ của DB (`da.getDBLocation()`) trước khi truyền vào câu lệnh SQL query cho nhánh domain timestamp (`lastUpdatedAt`).
   - Đã giúp XOR hash của các record trong window khớp chính xác giữa Source MongoDB và Destination Postgres (khi dùng domain timestamp).
2. **Sửa đổi `recon_dest_agent_test.go`**:
   - Mock thêm câu lệnh `SHOW TIMEZONE` trả về `UTC` khi khởi tạo Mocked Agent để tránh lỗi test.
   - Sửa đổi tham số `tLo` và `tHi` sang UTC trong `TestDestAgent_HashWindow_DomainTS` để tránh lệch múi giờ của máy chạy test.

### B. Thiếu sót nghiêm trọng phát hiện được (So với logic hệ thống đồng bộ):
Dù `HashWindow` đã được sửa, nhưng các phương thức query dữ liệu theo domain timestamp khác trong `ReconDestAgent` (`internal/service/recon/recon_dest_query.go`) **chưa được sửa đổi tương tự**. Chúng vẫn gửi trực tiếp `tLo` và `tHi` dưới dạng UTC naive vào Postgres, bao gồm:
1. `CountInWindow` (dòng 196)
2. `CountRecentDeletedRows` (dòng 257)
3. `BucketCounts` (dòng 348)
4. `ListIDTsInWindow` (dòng 454)

> [!WARNING]
> **Hậu quả nếu không sửa đồng bộ:**
> - `HashWindow` trả về hash khớp (không drift), nhưng `CountInWindow` vẫn trả về count lệch. Hệ thống recon sẽ báo lỗi "count mismatch in window" dù hash có thể khớp.
> - Khi đi vào drill-down, `ListIDTsInWindow` lấy sai tập ID/TS, dẫn đến so sánh lệch và tiếp tục trigger drift hoặc báo cáo sai lệch.
> - `BucketCounts` (dùng để recon tổng quát ở Tier B) sẽ tính sai count/xor theo từng bucket.

---

## 2. Kế hoạch đề xuất sửa đổi (Proposed Changes)

Chúng ta cần sửa đổi đồng bộ cả 4 hàm trong [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go) tương tự như `HashWindow`.

### [centralized-data-service](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/)

#### [MODIFY] [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go)

##### 1. Hàm `CountInWindow` (Domain TS Branch):
```go
	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		var count int64
		sql := fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE %s >= ? AND %s < ?`,
			quoteRelation(tableName), quoteIdent(tsCol), quoteIdent(tsCol),
		)
		dbLoc := da.getDBLocation()
		loForDB := tLo.In(dbLoc)
		hiForDB := tHi.In(dbLoc)
		if err := tx.Raw(sql, loForDB, hiForDB).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
```

##### 2. Hàm `CountRecentDeletedRows` (Domain TS Branch):
```go
	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		var count int64
		sql := fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE %s >= ? AND %s < ? AND "_deleted" = true`,
			quoteRelation(tableName), quoteIdent(tsCol), quoteIdent(tsCol),
		)
		dbLoc := da.getDBLocation()
		loForDB := tLo.In(dbLoc)
		hiForDB := tHi.In(dbLoc)
		if err := tx.Raw(sql, loForDB, hiForDB).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
```

##### 3. Hàm `BucketCounts` (Domain TS Branch):
```go
	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		dbLoc := da.getDBLocation()
		loForDB := tLo.In(dbLoc)
		hiForDB := tHi.In(dbLoc)
		rows, err := tx.Raw(sql, loForDB, hiForDB).Rows()
```

##### 4. Hàm `ListIDTsInWindow` (Domain TS Branch):
```go
	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()

		dbLoc := da.getDBLocation()
		loForDB := tLo.In(dbLoc)
		hiForDB := tHi.In(dbLoc)
		rows, err := tx.Raw(sql, loForDB, hiForDB).Rows()
```

#### [MODIFY] [recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go)
Cập nhật các test case tương ứng để sử dụng múi giờ UTC hoặc đồng nhất múi giờ nhằm đảm bảo các assertions của SQLMock khớp chính xác:
- `TestDestAgent_CountInWindow_DomainTS`
- `TestDestAgent_BucketCounts_DomainTS`
- `TestDestAgent_ListIDTsInWindow_DomainTS`
- `TestDestAgent_CountRecentDeletedRows_DomainTS`

---

## 3. Verification Plan

### Automated Tests
- Chạy unit tests cho gói recon:
  `go test ./internal/service/recon/... -v -run TestDestAgent`
- Đảm bảo tất cả 9/9 tests pass và không có panic hay SQLMock mismatch.

### Manual Verification
- Chạy thử local server và kiểm tra log.
