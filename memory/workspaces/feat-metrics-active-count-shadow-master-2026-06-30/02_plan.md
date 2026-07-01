# Implementation Plan - Bổ sung bắn Metrics Active Count cho Shadow & Master

## Bối cảnh và Mục tiêu
Hiện tại, metrics đếm tổng số dòng (`MasterTableRowCount`) và số dòng hoạt động (`MasterActiveRowCount`) cho master table chỉ được cập nhật trong `recon_tier_b.go` khi dữ liệu khớp hoàn toàn (Tier-0 check pass). Khi xảy ra lệch dữ liệu (drift), các metrics này không được bắn/cập nhật, dẫn đến việc giám sát trên dashboard hiển thị thông tin không chính xác hoặc stale. 
Đồng thời, tại Tier B (đối soát Shadow ↔ Master), metrics của shadow table (`ShadowTableRowCount`, `ShadowActiveRowCount`) chưa được cập nhật cùng thời điểm với master table.

Mục tiêu:
- Cập nhật metrics `MasterTableRowCount` và `MasterActiveRowCount` bất kể kết quả đối soát khớp hay lệch (drift).
- Cập nhật metrics `ShadowTableRowCount` và `ShadowActiveRowCount` cho shadow table trong Tier B để đồng bộ hóa dữ liệu hiển thị trên dashboard.

---

## Các thay đổi đề xuất

### Component: Centralized Data Service (data-hub/centralized-data-service)

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
- Di chuyển logic bắn metrics cho shadow và master ra ngoài block `if` kiểm tra khớp (Tier-0).
- Đưa logic này lên ngay sau khi đã tính toán thành công `shadowActive` và `masterActive` (sau dòng 89).
- Thêm logic bắn metrics cho shadow table:
  ```go
  if errSF == nil {
      metrics.ShadowTableRowCount.WithLabelValues(shadowRel).Set(float64(shadowFull))
      metrics.ShadowActiveRowCount.WithLabelValues(shadowRel).Set(float64(shadowActive))
  }
  ```
- Cập nhật logic bắn metrics cho master table:
  ```go
  if errMF == nil {
      metrics.MasterTableRowCount.WithLabelValues(masterFQN).Set(float64(masterFull))
      metrics.MasterActiveRowCount.WithLabelValues(masterFQN).Set(float64(masterActive))
  }
  ```
- Xoá bỏ đoạn code bắn metrics dư thừa bên trong block `if` khớp ở dòng 104-107.

---

## Chi tiết mã nguồn sửa đổi dự kiến (Draft Diff)

```diff
diff --git a/internal/service/recon/recon_tier_b.go b/internal/service/recon/recon_tier_b.go
index xxxxxxx..xxxxxxx 100644
--- a/internal/service/recon/recon_tier_b.go
+++ b/internal/service/recon/recon_tier_b.go
@@ -89,6 +89,16 @@ func (rc *ReconCore) RunSegmentB(ctx context.Context, ref MasterBindingRef, deep
 		}
 	}
 
+	// Dashboard node metrics: Shadow & Master row counts/active counts for SigNoz panels.
+	if errSF == nil {
+		metrics.ShadowTableRowCount.WithLabelValues(shadowRel).Set(float64(shadowFull))
+		metrics.ShadowActiveRowCount.WithLabelValues(shadowRel).Set(float64(shadowActive))
+	}
+	if errMF == nil {
+		metrics.MasterTableRowCount.WithLabelValues(masterFQN).Set(float64(masterFull))
+		metrics.MasterActiveRowCount.WithLabelValues(masterFQN).Set(float64(masterActive))
+	}
+
 	if errSF == nil && errMF == nil && shadowActive == masterActive && transmuteLagMs == 0 {
 		// KHỚP cả count lẫn watermark → DỪNG. Không bucket, không drill-down.
 		duration := int(time.Since(handle.started).Milliseconds())
@@ -101,10 +111,6 @@ func (rc *ReconCore) RunSegmentB(ctx context.Context, ref MasterBindingRef, deep
 		rc.stampB(report, ref)
 		rc.finishRun(ctx, handle, "success", "")
 		metrics.ReconDrift.WithLabelValues(masterFQN, "4").Set(0)
-		if errMF == nil {
-			metrics.MasterTableRowCount.WithLabelValues(masterFQN).Set(float64(masterFull))
-			metrics.MasterActiveRowCount.WithLabelValues(masterFQN).Set(float64(masterActive))
-		}
 		observability.Ctx(ctx, rc.logger).Info("segment B tier0 ok",
 			zap.String("master", masterRel),
 			zap.Int64("total", masterFull), zap.Int64("active", masterActive))
```

---

## Kế hoạch kiểm thử & Xác minh (Verification Plan)

### Automated Tests
- Chạy toàn bộ các test cases của recon service để đảm bảo không làm hỏng logic hiện có:
  `cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go test -v ./internal/service/recon/...`

### Manual Verification
- Kiểm tra tính đúng đắn của code bằng cách biên dịch dự án centralized-data-service:
  `cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./...`
