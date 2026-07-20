# Hồ sơ giải pháp: Cập nhật biến Total/Active sạch và mốc CheckedAt trong Smoke Recon Segment A

## 1. File sửa đổi
- `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go`

## 2. Giải pháp chi tiết

### Định nghĩa `dstTotalClean`
Thêm định nghĩa `dstTotalClean` sau khi tính toán `dstActiveClean` tại dòng 315:
```go
	dstTotalClean := dstTotal
	if val := dstTotal - dstRecentTotal; val >= 0 {
		dstTotalClean = val
	} else {
		dstTotalClean = 0
	}
```

### Cập nhật struct trả về `recon.SmokeResult`
Thay thế:
```go
	result := &recon.SmokeResult{
		RunID:        entry.RunID,
		Segment:      "source_shadow",
		SourceType:   ptr(entry.SourceType),
		SourceHost:   ptr(extractHost(entry.SourceURL)), // ĐÃ CHE CREDENTIALS
		SourceDB:     ptr(entry.SourceDB),
		SourceTable:  ptr(entry.SourceTable),
		SourceTotal:  &srcEstClean, //&srcEst,
		SourceActive: &srcEstClean, //&srcEst,
		ShadowSchema: ptr(entry.ShadowSchema),
		ShadowTable:  ptr(entry.TargetTable),
		ShadowTotal:  &dstTotalClean, //&dstTotal,
		ShadowActive: &dstActiveClean, //&dstActive,
		Diff:         diff,
		DiffTime:     json.RawMessage(diffTimeJSON),
		Status:       statusStr,
		DurationMs:   &dur,
		CheckedAt:    fromTime,
	}
```
Trong đó `CheckedAt` sử dụng `fromTime` thay cho `time.Now().UTC()`.
