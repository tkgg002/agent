# Kế hoạch Triển khai: Sửa đổi Cơ chế Đối soát Count (Smoke Check) dùng Khoảng Thời gian (Lookback Window) Thực tế

Chúng tôi đề xuất cải tiến cơ chế đối soát nhanh (Smoke Check / Count Check) chặng A và chặng B. Thay vì so sánh tổng số lượng bản ghi tuyệt đối ở thời điểm `NOW()` (luôn sai lệch do lag đồng bộ), hệ thống sẽ đếm và so sánh số lượng bản ghi thực tế **nằm trong khoảng thời gian đã đồng bộ (cũ hơn mốc lag đồng bộ)**.

## 1. Bối cảnh & Yêu cầu
*   **Vấn đề**: Hiện tại, hàm đối soát nhanh `RunTotalOnlyA` (chặng A) và `RunTotalOnlyB` (chặng B) trong `recon_smoke.go` so sánh tổng số lượng bản ghi của toàn bộ bảng tính đến thời điểm hiện tại (`EstimatedCount` của MongoDB so với count của Postgres).
*   Với các bảng có dữ liệu được thêm liên tục (high-throughput writes), do có độ trễ đồng bộ (replication lag / Kafka delay), tổng số lượng bản ghi giữa nguồn và đích luôn luôn lệch nhau ở thời điểm `NOW()`.
*   Điều này dẫn đến việc Smoke Check luôn báo trạng thái `drift` (mismatch) mặc dù dữ liệu cũ đã được đồng bộ hoàn toàn chính xác.
*   **Giải pháp**:
    1. Xác định mốc thời gian giới hạn an toàn `upper = NOW() - freeze margin` (ví dụ: cách đây 10 phút, loại bỏ hoàn toàn khoảng lag ghi dữ liệu mới).
    2. Đếm số lượng bản ghi mới phát sinh trong khoảng lag `(upper, Future]` (công việc này cực kỳ nhanh vì chỉ quét chỉ mục cho vài bản ghi mới).
       * Gọi số lượng bản ghi mới của Mongo là `srcNew`.
       * Gọi số lượng bản ghi mới của Postgres là `dstNew`.
    3. Tính toán số lượng bản ghi thực tế nằm trong cửa sổ thời gian đã đồng bộ (`[1970-01-01, upper]`):
       * `srcActiveSynced = srcEst (Tổng Mongo) - srcNew (Mongo mới)`
       * `dstActiveSynced = dstActive (Tổng Postgres) - dstNew (Postgres mới)`
    4. So sánh số lượng đã đồng bộ này: `diff = srcActiveSynced - dstActiveSynced`.
       * Nếu `diff == 0`: Trạng thái đối soát là **`ok`**. Số lượng hiển thị sẽ là `98 / 98` (số lượng thực tế cũ hơn 10 phút trước) và `diff = 0`. Hoàn toàn đúng logic toán học và phản ánh trung thực dữ liệu DB!
       * Nếu `diff != 0`: Trạng thái đối soát là **`drift`** thực tế.

---

## 2. Thiết kế Kỹ thuật & Đề xuất Thay đổi

### [centralized-data-service]

#### [MODIFY] [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
*   **Hàm `RunTotalOnlyA`**:
    *   Cập nhật block so sánh `diff` và tính toán số lượng synced:
        ```go
        	_, hi, ingestLagMs, srcTS, dstTS, errRange := rc.pickScanRangeWithLag(ctx, entry)
        	
        	srcActiveSynced := srcEst
        	dstActiveSynced := dstActive
        	
        	if errRange == nil && srcTS != "" && dstTS != "" {
        		future := time.Now().UTC().Add(24 * time.Hour)
        		srcNew, _ := rc.sourceAgent.CountInWindow(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, hi, future)
        		dstNew, _ := rc.destAgent.CountInWindow(ctx, entry.QualifiedTarget(), dstTS, hi, future)
        		
        		srcActiveSynced = srcEst - srcNew
        		dstActiveSynced = dstActive - dstNew
        		if srcActiveSynced < 0 { srcActiveSynced = 0 }
        		if dstActiveSynced < 0 { dstActiveSynced = 0 }
        	}

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
*   **Hàm `RunTotalOnlyB`**:
    *   Cập nhật block so sánh `diff` tương tự cho chặng B:
        ```go
        	shadowRel, masterRel := ref.ShadowRel(), ref.MasterRel()
        	fastCtx, cancel := context.WithTimeout(ctx, timeout)
        	defer cancel()

        	shMax, _ := rc.destAgent.MaxWindowTs(fastCtx, shadowRel, "_source_ts")
        	msMax, _ := rc.masterAgent.MaxWindowTs(fastCtx, masterRel, "_source_ts")
        	transmuteLagMs := lagBetween(shMax, msMax)
        	
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

---

## 3. Kịch bản Kiểm thử & Xác minh (Verification Plan)

### Kiểm thử Tự động (Automated Tests)
Chạy bộ kiểm thử đối soát để đảm bảo không bị lỗi biên dịch hoặc panic logic:
```bash
go test -count=1 ./internal/service/recon/...
go test -count=1 ./internal/handler/recon/...
```

### Kiểm thử Thủ công (Manual Verification)
1. Restart worker và CMS server.
2. Theo dõi log chạy recon tự động cho các bảng có dữ liệu ghi liên tục để xác nhận các chu kỳ đối soát trả về kết quả `ok` (hoặc `ok_empty`) thay vì báo còi `drift` liên tục.
