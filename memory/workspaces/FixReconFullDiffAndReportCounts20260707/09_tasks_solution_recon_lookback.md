# Hồ sơ Giải pháp Kỹ thuật: Sửa đổi Cơ chế Đối soát Count (Smoke Check) dùng Khoảng Thời gian (Lookback Window) Thực tế

Hồ sơ này chỉ định cụ thể các bước chỉnh sửa mã nguồn cho vai trò **Muscle (Chief Engineer)** thực hiện.

---

## 1. Danh sách file cần chỉnh sửa
*   [MODIFY] [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)

---

## 2. Chi tiết chỉnh sửa

### Bước 1: Sửa đổi hàm `RunTotalOnlyA` (Chặng A)
Tìm đoạn code:
```go
	// Zero-tolerance: hệ thống payment — lệch 1 row bất kỳ = drift cần điều tra.
	diff := srcEst - dstActive
	statusStr := "ok"
	var diffTimeJSON []byte
	if diff != 0 {
		statusStr = "drift"
		driftTimes := rc.runLookbackCheckA(ctx, entry)
		if len(driftTimes) > 0 {
			diffTimeJSON, _ = json.Marshal(driftTimes)
		}
	}
```
Thay thế hoàn toàn bằng:
```go
	// 1. Xác định mốc giới hạn kiểm tra hi (cửa sổ an toàn, ví dụ: cách đây 10 phút)
	_, hi, ingestLagMs, srcTS, dstTS, errRange := rc.pickScanRangeWithLag(ctx, entry)
	
	srcActiveSynced := srcEst
	dstActiveSynced := dstActive
	
	if errRange == nil && srcTS != "" && dstTS != "" {
		future := time.Now().UTC().Add(24 * time.Hour)
		// Đếm nhanh số lượng bản ghi mới phát sinh trong khoảng lag
		srcNew, _ := rc.sourceAgent.CountInWindow(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, hi, future)
		dstNew, _ := rc.destAgent.CountInWindow(ctx, entry.QualifiedTarget(), dstTS, hi, future)
		
		srcActiveSynced = srcEst - srcNew
		dstActiveSynced = dstActive - dstNew
		if srcActiveSynced < 0 { srcActiveSynced = 0 }
		if dstActiveSynced < 0 { dstActiveSynced = 0 }
	}

	// Zero-tolerance: hệ thống payment — lệch 1 row bất kỳ = drift cần điều tra.
	diff := srcActiveSynced - dstActiveSynced
	statusStr := "ok"
	var diffTimeJSON []byte
	if diff != 0 {
		statusStr = "drift"
		driftTimes := rc.runLookbackCheckA(ctx, entry)
		if len(driftTimes) > 0 {
			diffTimeJSON, _ = json.Marshal(driftTimes)
		}
	}
```
Và cập nhật gán `SourceActive` và `ShadowActive` trong `SmokeResult` (dòng 279-283):
```go
		SourceTotal:  &srcEst,
		SourceActive: &srcActiveSynced,
		ShadowSchema: ptr(entry.ShadowSchema),
		ShadowTable:  ptr(entry.ShadowTable),
		ShadowTotal:  &dstTotal,
		ShadowActive: &dstActiveSynced,
```

### Bước 2: Sửa đổi hàm `RunTotalOnlyB` (Chặng B)
Tìm đoạn code:
```go
	// Zero-tolerance: payment system — lệch 1 row bất kỳ = drift cần điều tra.
	diff := shadowActive - masterActive
	statusStr := "ok"
	var diffTimeJSON []byte
	if diff != 0 {
		statusStr = "drift"
		driftTimes := rc.runLookbackCheckB(ctx, ref)
		if len(driftTimes) > 0 {
			diffTimeJSON, _ = json.Marshal(driftTimes)
		}
	}
```
Thay thế hoàn toàn bằng:
```go
	shadowRel, masterRel := ref.ShadowRel(), ref.MasterRel()
	shMax, _ := rc.destAgent.MaxWindowTs(fastCtx, shadowRel, "_source_ts")
	msMax, _ := rc.masterAgent.MaxWindowTs(fastCtx, masterRel, "_source_ts")
	transmuteLagMs := lagBetween(shMax, msMax)
	
	// Xác định mốc hi của chặng B
	hi := time.Now().Add(-rc.adaptiveFreeze(transmuteLagMs))
	if !shMax.IsZero() && shMax.Before(hi) {
		hi = shMax.Add(time.Millisecond)
	}
	if !msMax.IsZero() && msMax.Before(hi) {
		hi = msMax.Add(time.Millisecond)
	}

	future := time.Now().UTC().Add(24 * time.Hour)
	shNew, _ := rc.destAgent.CountInWindow(fastCtx, shadowRel, "_source_ts", hi, future)
	msNew, _ := rc.masterAgent.CountInWindow(fastCtx, masterRel, "_source_ts", hi, future)

	shadowActiveSynced := shadowActive - shNew
	masterActiveSynced := masterActive - msNew
	if shadowActiveSynced < 0 { shadowActiveSynced = 0 }
	if masterActiveSynced < 0 { masterActiveSynced = 0 }

	// Zero-tolerance: payment system — lệch 1 row bất kỳ = drift cần điều tra.
	diff := shadowActiveSynced - masterActiveSynced
	statusStr := "ok"
	var diffTimeJSON []byte
	if diff != 0 {
		statusStr = "drift"
		driftTimes := rc.runLookbackCheckB(ctx, ref)
		if len(driftTimes) > 0 {
			diffTimeJSON, _ = json.Marshal(driftTimes)
		}
	}
```
Và cập nhật gán `ShadowActive` và `MasterActive` trong `SmokeResult` (dòng 420-424):
```go
		ShadowTotal:  &shadowTotal,
		ShadowActive: &shadowActiveSynced,
		MasterSchema: ptr(ref.MasterSchema),
		MasterTable:  ptr(ref.MasterTable),
		MasterTotal:  &masterTotal,
		MasterActive: &masterActiveSynced,
```

---

## 3. Xác minh sau khi sửa
Muscle chạy các lệnh test sau:
```bash
go test -count=1 ./internal/service/recon/...
go test -count=1 ./internal/handler/recon/...
```
Sau khi pass, Muscle thực hiện cập nhật `05_progress_recon_cleanup.md` và bàn giao báo cáo cho Brain.
