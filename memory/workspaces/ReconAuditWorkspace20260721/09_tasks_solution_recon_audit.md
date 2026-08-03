# 09 — Hồ Sơ Giải Pháp Kỹ Thuật Audit (Technical Audit Solutions)

> **Workspace:** `ReconAuditWorkspace20260721`  

---

## I. GIẢI PHÁP 1: CHUẨN HÓA KHÁCH THỂ TRÍCH XUẤT MONGO ID RAW (`extractMongoIDFromRaw`)

### 1.1 Nguyên nhân gốc rễ
Bảng Shadow PostgreSQL (`shadow_testpbs.payment_bills`) có cấu trúc:
- `_gpay_id`: kiểu `bigint` (chuỗi ID Sonyflake nội bộ hệ thống).
- `_id`: kiểu `bigint` (lưu ID từ bảng gốc MongoDB, ví dụ `504`).
- `_source_id`: kiểu `text` (lưu chuỗi đại diện ID gốc, ví dụ `"504"`).

Tại MongoDB nguồn (`payment-bills`), trường `_id` có kiểu dữ liệu là BSON `Int64` (hoặc `Int32`).
Trong hàm `extractMongoIDFromRaw` (`recon_hash.go`):
```go
func extractMongoIDFromRaw(v bson.RawValue) string {
	if oid, ok := v.ObjectIDOK(); ok {
		return oid.Hex()
	}
	if s, ok := v.StringValueOK(); ok {
		return s
	}
	// Fallback: bson.RawValue.String()
	return v.String()
}
```
Do thiếu kiểm tra kiểu `Int64OK()`, `Int32OK()`, `DoubleOK()`, hàm fallback `v.String()` trên `bson.RawValue` của BSON Int64 đã trả về chuỗi JSON bọc:
`{"$numberLong":"504"}`

Dẫn đến:
- Mongo hash fingerprint bằng chuỗi: `{"$numberLong":"504"}|1719216000000`
- Postgres hash fingerprint bằng chuỗi: `504|1719216000000`

Mặc dù `Count` (số lượng bản ghi) khớp 100%, nhưng `XorHash` của 2 bên bị sai khác trên mọi sub-window, tạo ra drift giả `recon.chunk_drift_count` (40/216 sub-windows).

### 1.2 Giải pháp mã nguồn (`recon_hash.go`)
Cập nhật `extractMongoIDFromRaw` trích xuất đầy đủ các kiểu số nguyên/thực BSON:

```go
func extractMongoIDFromRaw(v bson.RawValue) string {
	if oid, ok := v.ObjectIDOK(); ok {
		return oid.Hex()
	}
	if s, ok := v.StringValueOK(); ok {
		return s
	}
	if i, ok := v.Int64OK(); ok {
		return strconv.FormatInt(i, 10)
	}
	if i, ok := v.Int32OK(); ok {
		return strconv.Itoa(int(i))
	}
	if d, ok := v.DoubleOK(); ok {
		return strconv.FormatInt(int64(d), 10)
	}
	return v.String()
}
```

Kết quả: Mongo trích xuất đúng chuỗi `"504"`, trùng khớp 100% với chuỗi `"504"` trên Postgres!

---

## II. GIẢI PHÁP 2: CHUẨN HÓA BÁO CÁO BẢN GHI TRONG RECONCILIATION REPORT (`cdc_reconciliation_report`)

### 2.1 Nguyên nhân gốc rễ
Trong `recon_job_worker.go`:
- Biến `totalDiff` được gán = `int64(len(drifts))` (số lượng 15-min sub-windows bị drift, ví dụ 216 sub-windows).
- Mã nguồn gán `report.Diff = 216`, `report.MissingCount = 216`, trong khi bỏ trống `SourceCount` (DB lưu `NULL`) và `DestCount` (DB lưu `0`).
- Dẫn đến DB ghi nhận dòng ID 65: `source_count = NULL`, `dest_count = 0`, `diff = 216`, `missing_count = 216` — ghi sai bản chất dữ liệu đối soát.

### 2.2 Giải pháp mã nguồn (`recon_stream_bucket_engine.go` & `recon_job_worker.go`)
Cấu trúc lại kết quả trả về từ `ChunkStreamBucketEngine.Execute`:

```go
type ChunkEngineResult struct {
	Drifts        []DriftWindow
	TotalSrcCount int64
	TotalDstCount int64
	RecordDiff    int64
	MissingCount  int64
}
```

Trong `recon_job_worker.go`:
```go
report := &modelrecon.ReconciliationReport{
	ShadowSchema: entry.ShadowSchema,
	ShadowTable:  entry.TargetTable,
	RunID:        event.JobID,
	SourceDB:    entry.SourceDB,
	SourceTable: entry.SourceTable,
	SourceType:  entry.SourceType,
	SourceHost:  extractHost(entry.SourceURL),
	SourceCount:  &res.TotalSrcCount,
	DestCount:    res.TotalDstCount,
	Diff:         res.RecordDiff,
	MissingCount: int(res.MissingCount),
	Segment:      "source_shadow",
	CheckType:    "chunk_stream_bucket",
	Status:       status,
	DurationMs:   &durationMs,
	CheckedAt:    time.Now().UTC(),
	ReconStartTime: &reconStart,
	ReconEndTime:   &reconEnd,
}
```

---

## III. GIẢI PHÁP 3: BỔ SUNG `total_record_diff_count`, `source_count`, `dest_count` VÀ `stale_ids` VÀO `recon_jobs` & `cdc_reconciliation_report`

### 3.1 Yêu cầu kỹ thuật
1. **`total_record_diff_count`**: Tổng số record bị chênh lệch/lệch dữ liệu thực tế giữa Source và Dest trong toàn bộ khoảng thời gian đối soát.
2. **`total_diff_count`**: Giữ nguyên làm tổng số sub-windows có drift (`len(drifts)`).
3. **`source_count` & `dest_count`**: Tổng số record bên Nguồn (Source) và Đích (Shadow) tích lũy qua tất cả các ngày/chunk đối soát.
4. **`stale_ids`**: Trường JSONB định dạng chính xác:
   ```json
   {
     "mismatched": null,
     "missing_from_master": ["70767058790383625", "70767209823076362"],
     "missing_from_shadow": null
   }
   ```
   - **`mismatched`**: ID tồn tại ở cả 2 bên nhưng bị sai lệch hash/nội dung.
   - **`missing_from_master`**: ID tồn tại ở Shadow Postgres nhưng thiếu ở Master/Source (MongoDB).
   - **`missing_from_shadow`**: ID tồn tại ở Source (MongoDB) nhưng thiếu ở Shadow Postgres.
   - Mặc định khi danh mục rỗng sẽ serialize thành `null`.

### 3.2 Thiết kế struct & DB Schema
- **`StaleIDsPayload` struct (`recon_stream_bucket_engine.go`)**:
  ```go
  type StaleIDsPayload struct {
  	Mismatched        []string `json:"mismatched"`
  	MissingFromMaster []string `json:"missing_from_master"`
  	MissingFromShadow []string `json:"missing_from_shadow"`
  }
  ```
- **Bổ sung cột vào `cdc_system.recon_jobs` DB**:
  - `total_record_diff_count` `BIGINT DEFAULT 0`
  - `source_count` `BIGINT DEFAULT 0`
  - `dest_count` `BIGINT DEFAULT 0`
- **Ghi nhận vào `result_summary` JSONB**:
  Chứa `drifts`, `stale_ids`, `source_count`, `dest_count`, `total_record_diff_count`, `total_diff_count`.
```
