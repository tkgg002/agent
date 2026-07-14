# Hồ sơ Giải pháp: Đồng bộ hóa Chữa lành Segment B qua NATS Request-Reply & Bắn Lỗi Thiếu Rules

Hồ sơ này mô tả chi tiết phương án sửa đổi logic chữa lành Segment B (Shadow -> Master) để đồng bộ hóa quá trình transmute và sửa đổi module Transmuter để bắn lỗi khi thiếu mapping rules thay vì bỏ qua lặng lẽ.

## A. Backend (centralized-data-service)

### 1. `internal/service/master/transmuter.go`
Sửa đổi logic khi load rules cho master binding. Nếu `len(rules) == 0` (không có rule nào được approved), transmuter phải trả về lỗi thay vì đánh dấu thành công:
```go
// Tại transmuter.go, thay thế đoạn code sau:
	if len(rules) == 0 {
		err = fmt.Errorf("no approved mapping rules found for master binding ID %d (%s)", masterRow.ID, masterRow.MasterTable)
		t.markRuntimeFailure(ctx, masterRow.ID, err)
		return res, err
	}
```

### 2. `internal/handler/recon/recon_base_handler.go`
Cập nhật interface `NatsPublisher` để thêm phương thức `Request`:
```go
type NatsPublisher interface {
	Publish(subject string, data []byte) error
	Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error)
}
```

### 3. `internal/handler/recon/recon_execute_heal_handler.go`
- Định nghĩa local struct `transmuteResponse` để parse kết quả từ NATS reply của transmute worker:
```go
type transmuteResponse struct {
	Scanned    int64  `json:"scanned"`
	Inserted   int64  `json:"inserted"`
	Updated    int64  `json:"updated"`
	Skipped    int64  `json:"skipped"`
	RuleMisses int64  `json:"rule_misses"`
	TypeErrors int64  `json:"type_errors"`
	Err        string `json:"error,omitempty"`
}
```
- Sửa đổi hàm `publishTransmuteChunked`:
  - Chữ ký hàm: `publishTransmuteChunked(ctx context.Context, table string, sourceIDs []string, triggeredBy string) (int, int, error)`.
  - Sử dụng `h.natsPub.Request` thay vì `Publish` với timeout là `2 * time.Minute`.
  - Unmarshal dữ liệu trả về thành `transmuteResponse`, kiểm tra trường `Err` và trả về lỗi nếu có.
- Cập nhật hàm `executeHealSegB`:
  - Gọi `publishTransmuteChunked` một cách đồng bộ.
  - Gán `rpt.HealedMismatchedCount = inserted + updated` (hoặc `rpt.HealedMissingDestCount = inserted + updated`) nếu quá trình transmute diễn ra thành công. Nếu thất bại, ghi log lỗi và giữ nguyên giá trị `0` để report không bị nhận là đã heal ảo.

```go
if opts.HealMissingDest && len(missingGpayIDs) > 0 {
	start := time.Now()
	var ins, upd int
	var err error
	if sourceIDs, err := h.mapGpayToSourceIDs(ctx, shadowRel, missingGpayIDs); err == nil {
		ins, upd, err = h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs, "execute-heal-b")
		if err == nil {
			rpt.HealedMissingDestCount = ins + upd
			healed += ins + upd
		} else {
			h.logger.Error("[execute-heal-b] transmute failed for missing dest", zap.Error(err))
		}
	} else {
		h.logger.Error("[execute-heal-b] mapGpayToSourceIDs failed for missing dest", zap.Error(err))
	}
	rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
}
```

---

## B. Frontend (cdc-cms-web)

### 1. `cdc-cms-web/src/components/ExecuteHealModal.tsx`
Sửa bộ lọc `healedReports` để loại bỏ các phiên `partially_healed`, chỉ hiển thị các phiên hoàn thành toàn bộ (`healed_at != null` hoặc `status === 'healed'`):
```typescript
  const healedReports = (historyData?.data || []).filter(
    (r: any) => r.healed_at != null || r.status === 'healed'
  );
```
