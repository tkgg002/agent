# Kế hoạch Triển khai Kỹ thuật - Tối ưu hóa Latency phát hiện Drift trong Smoke Check

Kế hoạch này chi tiết hóa phương án giải quyết vấn đề độ trễ 11 giây khi xảy ra drift trong quá trình Smoke Check của Recon Engine.

## Phân tích Nguyên nhân Gốc rễ
1. **Lỗi logic ở `effectiveLookback` (`recon_engine.go`):**
   - Hàm `effectiveLookback` chỉ kiểm tra `rc.cfg.WindowLookback > 0` rồi trả về ngay giá trị đó. Do hàm `applyDefaults` luôn gán giá trị mặc định là 7 ngày (`7 * 24 * time.Hour`) cho `WindowLookback`, cờ điều kiện này luôn đúng. Kết quả là hệ thống luôn trả về 7 ngày cho lookback window bất kể `RunMode` có cấu hình `"hot"` (thường trực) hay không.
2. **Hardcode 7 ngày lookback trong `runLookbackCheckB` (`recon_smoke.go`):**
   - `runLookbackCheckB` hiện tại đang hardcode cứng giá trị `upper.Add(-7 * 24 * time.Hour)`.
3. **Thực thi tuần tự các cuộc gọi `BucketCounts`:**
   - Trong cả `runLookbackCheckA` và `runLookbackCheckB`, các hàm `BucketCounts` đến Database Source (MongoDB/Postgres) và Shadow/Master DB (Postgres) đang được thực thi tuần tự, gây tăng thời gian chờ mạng (RTT).

## Giải pháp Đề xuất

### 1. Sửa đổi `effectiveLookback` (`recon_engine.go`)
Nhận diện chính xác `RunMode` từ cấu hình:
- Nếu `RunMode == "cold"`: Sử dụng `WindowLookback` (mặc định 7 ngày).
- Nếu `RunMode == "hot"` hoặc `""` (mặc định): Sử dụng `HotWindowLookback` (mặc định 2 giờ).

```go
func (rc *ReconCore) effectiveLookback(ctx context.Context) time.Duration {
	if rc.cfg.RunMode == "cold" {
		if rc.cfg.WindowLookback > 0 {
			return rc.cfg.WindowLookback
		}
		return 7 * 24 * time.Hour
	}
	if rc.cfg.HotWindowLookback > 0 {
		return rc.cfg.HotWindowLookback
	}
	return 2 * time.Hour
}
```

### 2. Sửa đổi `runLookbackCheckB` và Tối ưu hóa Song song (`recon_smoke.go`)
- **Tối ưu hóa lookback window:** Thay thế `-7 * 24 * time.Hour` bằng `-rc.effectiveLookback(ctx)`.
- **Tối ưu hóa cột timestamp:** Phân giải cột timestamp động `tsCol` từ Table Registry tương tự như Segment B chính để tránh hardcode `_source_ts` nếu bảng sử dụng custom timestamp.
- **Thực thi song song (Concurrency):** Sử dụng `sync.WaitGroup` để gọi `BucketCounts` của Shadow và Master đồng thời (cho B) và Source và Shadow đồng thời (cho A).

#### Demo Code `runLookbackCheckA` mới:
```go
func (rc *ReconCore) runLookbackCheckA(ctx context.Context, entry source.TableRegistry) []string {
	lo, hi, _, srcTS, dstTS, err := rc.pickScanRangeWithLag(ctx, entry)
	if err != nil {
		rc.logger.Warn("smoke lookback check A: pick scan range failed", zap.Error(err))
		return nil
	}

	var srcBuckets map[int64]int64
	var dstBuckets map[int64]BucketStat
	var errSrc, errDst error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		srcBuckets, err = rc.sourceAgent.BucketCounts(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
		errSrc = err
	}()
	go func() {
		defer wg.Done()
		dstBuckets, err = rc.destAgent.BucketCounts(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)
		errDst = err
	}()
	wg.Wait()

	if errSrc != nil || errDst != nil {
		rc.logger.Warn("smoke lookback check A: bucket counts failed", zap.Error(errSrc), zap.NamedError("dst_err", errDst))
		return nil
	}
    // ... logic so sánh bucket giữ nguyên ...
}
```

#### Demo Code `runLookbackCheckB` mới:
```go
func (rc *ReconCore) runLookbackCheckB(ctx context.Context, ref MasterBindingRef) []string {
	upper := time.Now().UTC()
	lower := upper.Add(-rc.effectiveLookback(ctx))

	shadowRel := ref.ShadowRel()
	masterRel := ref.MasterRel()

	tsCol := "_source_ts"
	if entry, err := rc.registryRepo.GetByTargetTable(ctx, ref.MasterTable); err == nil && entry != nil {
		entry.ShadowSchema = ref.ShadowSchema
		_, dstTS, errTS := rc.resolveSourceAndDestTSFields(ctx, *entry)
		if errTS == nil && dstTS != "" {
			tsCol = dstTS
		}
	}

	var shBuckets, msBuckets map[int64]BucketStat
	var errS, errM error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		shBuckets, errS = rc.destAgent.BucketCounts(ctx, shadowRel, segBPKColumn, tsCol, lower, upper)
	}()
	go func() {
		defer wg.Done()
		msBuckets, errM = rc.masterAgent.BucketCounts(ctx, masterRel, segBPKColumn, tsCol, lower, upper)
	}()
	wg.Wait()

	if errS != nil || errM != nil {
		rc.logger.Warn("smoke lookback check B: bucket counts failed", zap.Error(errS), zap.NamedError("master_err", errM))
		return nil
	}
    // ... logic so sánh bucket giữ nguyên ...
}
```

## Kế hoạch Xác minh (Verification Plan)
1. **Automated Tests:** Chạy toàn bộ test suite `internal/service/recon/...` để đảm bảo code biên dịch thành công và các test hiện có vượt qua (không gây regression).
2. **Manual verification:** Kiểm tra xem linter quy trình (`verify_governance.py`) có thông báo lỗi tuân thủ nào không.
