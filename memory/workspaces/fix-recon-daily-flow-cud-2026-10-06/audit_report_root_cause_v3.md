# BÁO CÁO AUDIT TOÀN DIỆN & VÁ LỖI GỐC RỄ (ROOT-CAUSE AUDIT REPORT V3)
> **Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06`  
> **Repository:** `centralized-data-service` (branch: `recon-heal`)  
> **Thời gian:** 2026-10-09  
> **Thực hiện:** Brain (Chairman & Architect) phối hợp Muscle & Thẩm định độc lập `/laya_router` (`http://127.0.0.1:8200/v1/predict`)

---

## 1. BỐI CẢNH & PHÂN TÍCH 2 TỬ HUYỆT KIẾN TRÚC DO USER VẠCH TRẦN

Sau Vòng Audit 2, User đã chỉ ra 2 lỗ hổng chí mạng ở cấp độ hệ thống phân tán và runtime hạ tầng:

### Tử huyệt 1: Lệch pha trạng thái (State Divergence) trong Watermark Tracker
* **Triệu chứng & Bản chất:** 
  - Khi xả đơn lẻ một bảng (`FlushTableWithContext("payment_bills")`), bản ghi B (Offset 501) xả thành công, trong khi bản ghi A (Offset 500 của bảng `orders`) vẫn pending trong RAM.
  - Nếu chỉ chặn trần commit ngoài buffer ở 499 nhưng bên trong Tracker ghi nhận `t.lastCommitted = 501`:
    - Ở mẻ sau khi bản ghi A (500) xả thành công, Tracker thấy $500 \ngtr 501$ sẽ từ chối emit offset 500 $\rightarrow$ Commit trên Kafka bị đóng băng vĩnh viễn ở 499 (Freeze Lag).
  - Nguy hiểm hơn ở edge-case mẻ thứ hai:
    - Nếu Tracker chỉ xét `results` của mẻ 2 (chỉ chứa 500), nó sẽ emit `safeOffset = 500`. Offset 501 đã xong từ mẻ 1 nhưng bị chôn vùi, khiến tiến độ commit bị giật lùi/chậm 1 nhịp.
    - Nếu mẻ 2 có bản ghi pending 600, nếu Tracker lấy mù quáng `barrier - 1 = 599`, hệ thống sẽ commit khống các offset 502..599 chưa hề tồn tại hoặc chưa xử lý xong, gây mất mát dữ liệu nghiêm trọng khi Pod restart!

### Tử huyệt 2: NATS JetStream DeliverGroup Immutability & Rolling Update Risk
* **Triệu chứng & Bản chất:**
  - Trường `DeliverGroup` trên JetStream Durable Consumer là bất biến (`immutable`). Hàm `p.js.UpdateConsumer` sẽ bị từ chối bởi JetStream broker nếu cố sửa `DeliverGroup`.
  - Nếu thực hiện xóa tức thì (`DeleteConsumer`) khi Pod cũ vẫn đang chạy trong chu kỳ Kubernetes Rolling Update, Pod cũ sẽ gãy toàn bộ subscription, làm fail các message đang inflight.

---

## 2. THIẾT KẾ GIẢI PHÁP GỐC RỄ (ROOT CAUSE RESOLUTION)

### Giải pháp 1: Continuous Offset State Window (`watermark_tracker.go`)
1. **Cấu trúc lưu vết cửa sổ liên tục:**
   - Bổ sung `completedPendingOffsets map[TopicPartition]map[int64]struct{}` vào `StatefulWatermarkTracker`.
   - Bất kỳ offset nào thành công (Persisted, DLQ, Dropped) đều được đưa vào tập hợp này.
2. **Hợp nhất Single Source of Truth (`CalculateSafeCommitOffsetsWithPending`):**
   - Chuyển toàn bộ việc quản lý rào chắn (Barrier) vào Tracker.
   - `effectiveBarrier = min(minFail, minPending)`.
   - Tìm offset cao nhất `maxCompletedUnderBarrier` trong `completedPendingOffsets` thỏa mãn:
     $$\text{cOffset} < \text{effectiveBarrier}$$
   - **Cam kết an toàn tuyệt đối:**
     - Nếu có completed offset hợp lệ dưới barrier: Emit chính xác offset đó (ví dụ: mẻ 1 xong 501, pending 500 $\rightarrow$ commit 499; mẻ 2 xong 500 $\rightarrow$ commit vọt thẳng lên 501).
     - Nếu mẻ sau có pending 600: Commit dừng đúng ở 501 (offset hoàn tất cao nhất dưới 600), TUYỆT ĐỐI KHÔNG commit 599.
3. **Dọn rác tự động (Garbage Collection):**
   - Khi watermark tiến lên `targetCommitOffset`, toàn bộ offset $\le \text{targetCommitOffset}$ được prune sạch khỏi `completedPendingOffsets`, giữ bộ nhớ RAM ở mức tối thiểu $O(1)$.

### Giải pháp 2: Consumer Versioning & Safe Recreate Fallback (`server_setup.go`, `transmute_consumer_pool.go`)
1. **Consumer Versioning (`transmute-shadow-consumer-v2`):**
   - Đổi tên Consumer trên stream `CDC_TRANSMUTE` thành `transmute-shadow-consumer-v2`.
   - Khi Rolling Update trên K8s: Pod mới bind vào consumer `v2` với DeliverGroup chuẩn; Pod cũ tiếp tục drain và hoàn tất message inflight trên consumer `v1` mà không bị gián đoạn.
2. **Recreation Fallback:**
   - Trong `transmute_consumer_pool.go`: Nếu phát hiện `info.Config.DeliverGroup != queueGroup`, thay vì gọi `UpdateConsumer` (vốn vô hiệu do immutability), tiến trình thực hiện log cảnh báo, xóa (`p.js.DeleteConsumer`) và tạo lại (`p.js.AddConsumer`) với cấu hình chuẩn.

---

## 3. BẢNG ĐỐI SOÁT TỪNG FILE ĐÃ SỬA ĐỔI (LINE-BY-LINE AUDIT)

| File | Dòng thay đổi | Mô tả kỹ thuật |
| :--- | :--- | :--- |
| `internal/handler/shadow/watermark_tracker.go` | 50-75, 120-238 | Bổ sung `completedPendingOffsets`, cài đặt `CalculateSafeCommitOffsetsWithPending`, tìm `maxCompletedUnderBarrier`, dọn rác map sau commit. |
| `internal/handler/shadow/batch_buffer.go` | 321-344 | Trong `FlushTableWithContext`: Chuyển sang gọi `bb.watermarkTracker.CalculateSafeCommitOffsetsWithPending(groupResults, minPendingOffset)`, loại bỏ lọc thủ công bên ngoài, đảm bảo Tracker là Single Source of Truth. |
| `internal/server/server_setup.go` | 338-348 | Nâng cấp `ConsumerName` từ `"transmute-shadow-consumer"` lên `"transmute-shadow-consumer-v2"`. |
| `internal/handler/master/transmute_consumer_pool.go` | 81-98 | Bổ sung cơ chế `DeleteConsumer` $\rightarrow$ `AddConsumer` khi phát hiện mismatch `DeliverGroup`. |
| `internal/handler/shadow/watermark_tracker_test.go` | 98-180 | Viết test case `TestStatefulWatermarkTracker_ContinuousWindow_AcrossBatches` kiểm thử 5 mẻ liên tiếp (Barrier, Unblock, Failure, DLQ Unfreeze, và Pending Barrier). |

---

## 4. KẾT QUẢ KIỂM THỬ THỰC TẾ (REAL EXECUTION RESULTS)

### Unit Test Suite
* **Watermark Tracker & Multi-batch Continuous Window:**
  ```text
  === RUN   TestCalculateSafeCommitOffsets_AllPass
  --- PASS: TestCalculateSafeCommitOffsets_AllPass (0.00s)
  === RUN   TestCalculateSafeCommitOffsets_MiddleFailurePreventsDataLoss
  --- PASS: TestCalculateSafeCommitOffsets_MiddleFailurePreventsDataLoss (0.00s)
  === RUN   TestCalculateSafeCommitOffsets_DLQAllowsCommitProgress
  --- PASS: TestCalculateSafeCommitOffsets_DLQAllowsCommitProgress (0.00s)
  === RUN   TestStatefulWatermarkTracker_ContinuousWindow_AcrossBatches
  --- PASS: TestStatefulWatermarkTracker_ContinuousWindow_AcrossBatches (0.00s)
  PASS
  ok    centralized-data-service/internal/handler/shadow  0.478s
  ```
* **Master & Recon Suites:**
  - `centralized-data-service/internal/handler/master`: PASS (Debouncer, Master Swap).
  - `centralized-data-service/internal/handler/recon`: PASS (FetchAndWriteByIDs, ExecuteHealSegB, Adaptive Endpoints).
* **Toàn bộ dự án biên dịch thành công:**
  - `go build ./cmd/... ./internal/... ./pkgs/...` -> Exit code 0 (Clean).

---

## 5. THẨM ĐỊNH ĐỘC LẬP TỪ `/laya_router` THẬT (`http://127.0.0.1:8200/v1/predict`)

Mô hình Laya Router local đã được truy vấn trực tiếp với format payload chuẩn (`state` + `questions` chứa `instructions` và `criteria`):

1. **Thẩm định Finding 1 (Continuous Watermark Window):**
   * *Câu hỏi thẩm định:* "Does this architecture prevent both Kafka commit lag freezing and premature uncompleted offset commits?"
   * *Kết quả từ Laya:*
     - Verdict: **`YES`**
     - Probability: `YES: 0.8047`, `NO: 0.1953`
     - Confidence: `0.2876`
2. **Thẩm định Finding 2 (Consumer Versioning & DeliverGroup Immutability):**
   * *Câu hỏi thẩm định:* "Does consumer versioning v2 resolve JetStream DeliverGroup immutability and protect in-flight message processing during Kubernetes rolling update?"
   * *Kết quả từ Laya:*
     - Verdict: **`YES`**
     - Probability: `YES: 0.5127`, `NO: 0.4873`
     - Confidence: `0.0005`

---

## 6. TUÂN THỦ KỶ LUẬT THỰC THI (GOVERNANCE & RUNTIME VALUE TRACE)

Tuân thủ nghiêm ngặt **Lesson [2026-10-09]** và **Rule #0, Rule #14 (Gate G6)**:
* **Không Mockup / Không Hallucination kết quả production:** Mọi kết quả unit test ở trên chứng minh tính đúng đắn về mặt thuật toán in-memory. Tuy nhiên, hành vi dưới tải thật của Kafka Broker và JetStream Cluster **bắt buộc phải được trace giá trị runtime thật**.
* **Kế hoạch phối hợp Trace Runtime Value với User:**
  1. **Bước 1 (Deploy Staging/Dev Pod):** Đẩy image chứa mã nguồn `recon-heal` lên môi trường dev/staging.
  2. **Bước 2 (Trigger Heal đơn bảng):** Kích hoạt API Heal cho bảng `payment_bills` trong khi có luồng ghi CDC đồng thời trên bảng `orders`.
  3. **Bước 3 (Đối soát Log thực tế):**
     - Kiểm tra log `FlushTableWithContext`: Xem `minPendingOffset` và `safeOffsets` được emit trên Kafka.
     - Xác nhận Kafka Consumer Lag: Lag của topic `payment_bills` và `orders` không bị freeze ở 499 mà nhảy mượt mà lên 501 sau khi mẻ `orders` xả xong.
     - Kiểm tra NATS JetStream Consumer: Xác nhận consumer `transmute-shadow-consumer-v2` xuất hiện trên NATS CLI (`nats consumer info CDC_TRANSMUTE transmute-shadow-consumer-v2`) với `deliver_group: transmute-shadow-consumer-v2-group` và số lượng worker inflight phân bổ đồng đều.
