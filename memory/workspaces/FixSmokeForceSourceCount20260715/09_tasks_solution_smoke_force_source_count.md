# 09_tasks_solution: Sửa lỗi cú pháp và ép số lượng source về shadow

## 1. Vấn đề thực tế
Trong file `internal/service/recon/recon_smoke.go`, việc gán `SourceTotal` và `SourceActive` sử dụng toán tử ba ngôi `? :` vốn không được hỗ trợ trong Go:
```go
SourceTotal:  diff==0?&dstActiveClean:&srcEstClean,
SourceActive: diff==0?&dstActiveClean:&srcEstClean,
```
Điều này dẫn đến lỗi biên dịch (syntax error).

Đồng thời, nghiệp vụ mong muốn là:
Khi `diff == 0` (đối soát Smoke Segment A xác định không lệch), ép số lượng của Source (cả Total và Active) về bằng số lượng Shadow Active (`dstActiveClean`). Lý do là ta tin tưởng hoàn toàn vào EstCount + HashWindow.

## 2. Giải pháp kỹ thuật
Thay vì dùng toán tử ba ngôi trực tiếp khi khởi tạo struct:
- Khai báo 2 biến con trỏ `srcTotalPtr` và `srcActivePtr` kiểu `*int64`.
- Sử dụng cấu trúc rẽ nhánh `if diff == 0` chuẩn của Go để gán địa chỉ:
  - Nếu `diff == 0`: trỏ cả 2 về `&dstActiveClean`.
  - Ngược lại: trỏ cả 2 về `&srcEstClean`.
- Gán `SourceTotal: srcTotalPtr` và `SourceActive: srcActivePtr` vào struct `recon.SmokeResult`.

## 3. Chi tiết mã nguồn thay đổi

### File: `internal/service/recon/recon_smoke.go`

```go
	var srcTotalPtr *int64
	var srcActivePtr *int64
	if diff == 0 {
		srcTotalPtr = &dstActiveClean
		srcActivePtr = &dstActiveClean
	} else {
		srcTotalPtr = &srcEstClean
		srcActivePtr = &srcEstClean
	}

	var diffTimeJSON []byte
	result := &recon.SmokeResult{
		RunID:        entry.RunID,
		Segment:      "source_shadow",
		SourceType:   ptr(entry.SourceType),
		SourceHost:   ptr(extractHost(entry.SourceURL)), // ĐÃ CHE CREDENTIALS
		SourceDB:     ptr(entry.SourceDB),
		SourceTable:  ptr(entry.SourceTable),
		SourceTotal:  srcTotalPtr,
		SourceActive: srcActivePtr,
		ShadowSchema: ptr(entry.ShadowSchema),
		ShadowTable:  ptr(entry.TargetTable),
		ShadowTotal:  &dstTotalClean,  //&dstTotal,
		ShadowActive: &dstActiveClean, //&dstActive,
		Diff:         diff,
		DiffTime:     json.RawMessage(diffTimeJSON),
		Status:       statusStr,
		DurationMs:   &dur,
		CheckedAt:    fromTime,
	}
```
