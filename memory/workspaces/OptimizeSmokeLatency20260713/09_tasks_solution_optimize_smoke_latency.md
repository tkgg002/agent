# Hồ sơ giải pháp kỹ thuật - Tối ưu hóa Latency trong Smoke Check

Hồ sơ này cung cấp chi tiết vị trí và nội dung mã nguồn cần sửa đổi.

## 1. Sửa đổi `internal/service/recon/recon_engine.go`

Thay đổi phương thức `effectiveLookback` để nhận diện chính xác `RunMode`:

```go
// Line 226:
func (rc *ReconCore) effectiveLookback(ctx context.Context) time.Duration {
	if rc.cfg.RunMode == "cold" {
		if rc.cfg.WindowLookback > 0 {
			return rc.cfg.WindowLookback
		}
		return 7 * 24 * time.Hour
	}
	// Mặc định hoặc khi RunMode là "hot"
	if rc.cfg.HotWindowLookback > 0 {
		return rc.cfg.HotWindowLookback
	}
	return 2 * time.Hour
}
```

## 2. Sửa đổi `internal/service/recon/recon_smoke.go`

### Thay đổi 2.1: Song song hóa `runLookbackCheckA`
Thực thi các cuộc gọi `BucketCounts` đồng thời bằng `sync.WaitGroup`:

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
		srcBuckets, errSrc = rc.sourceAgent.BucketCounts(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
	}()
	go func() {
		defer wg.Done()
		dstBuckets, errDst = rc.destAgent.BucketCounts(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)
	}()
	wg.Wait()

	if errSrc != nil || errDst != nil {
		rc.logger.Warn("smoke lookback check A: bucket counts failed", zap.Error(errSrc), zap.NamedError("dst_err", errDst))
		return nil
	}
```

### Thay đổi 2.2: Thay lookback động, tìm timestamp, song song hóa `runLookbackCheckB`
Cập nhật `runLookbackCheckB` để lấy khoảng thời gian lookback động qua `effectiveLookback(ctx)`, tự động phân giải trường timestamp nghiệp vụ thay vì hardcode `_source_ts`, và song song hóa `BucketCounts` cho Shadow và Master DB:

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
```
