# BÁO CÁO TỔNG KẾT TOÀN DIỆN: CẢI TIẾN RECON HEAL, ĐIỀU PHỐI NATS VÀ STATEFUL WATERMARK TRACKER
> **Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06`  
> **Repository:** `centralized-data-service` (branch: `recon-heal`)  
> **Thời gian:** 2026-10-09  
> **Tác giả:** Brain (Chairman & Architect) & Muscle (Chief Engineer)  
> **Thẩm định độc lập:** `/laya_router` (`http://127.0.0.1:8200/v1/predict`)

---

## 1. TỔNG QUAN CÁC HẠNG MỤC CẦN LÀM (OBJECTIVES & SCOPE)

1. **Hạng mục 1: Khắc phục Ô nhiễm Trace Context & Tách biệt Đệm Snapshot/CDC trong luồng Heal (Trace & Buffer Isolation):**
   - Triệt tiêu hiện tượng luồng `execute-heal` của bảng `payment_bills` bị ô nhiễm span `order_updates`.
   - Thay thế lệnh xả toàn bộ buffer dùng chung (`FlushBatchBuffer`) bằng cơ chế xả cô lập đúng bảng mục tiêu (`FlushTableWithContext(targetTable)`).
   - Khởi tạo 2 thể hiện BatchBuffer độc lập: `cdcBatchBuffer` (cho dòng CDC thời gian thực) và `snapshotBatchBuffer` (cho Snapshot / Heal / Scan).

2. **Hạng mục 2: Tối ưu hóa NATS JetStream Transmute Trigger (Deduplication, True Concurrency & Chống Deadlock):**
   - Khắc phục rủi ro trùng lặp khi xả Shadow: Sinh NATS Message ID tất định (`cdc-transmute-{table}-{firstPK}-{lastPK}-{hash}-{ts}`) để broker deduplicate trong 2 phút.
   - Xây dựng Consumer Pool trên JetStream với `DeliverGroup` để đạt True Concurrency trên nhiều worker subscription, loại bỏ nguy cơ xử lý trùng lặp hoặc nghẽn cổ chai.
   - Chống deadlock goroutine trong `TableDebouncer` khi Kubernetes thực hiện Graceful Shutdown (SIGTERM).

3. **Hạng mục 3: Quản lý Watermark Kafka & Cứu hộ Offset (Kafka Offset Rescue & Watermark Tracker):**
   - Bổ sung cờ điều hướng luồng Watermark (`IsMarker`, `IsDLQMarker`, `IsNoOpMarker`) để bypass SQL Builder mà vẫn điều hướng commit offset an toàn.
   - Cứu hộ Kafka Offset cho các bản ghi CDC vô tình đi cùng mẻ xả theo bảng của luồng Heal.
   - Ngăn chặn nguy cơ Commit Vượt Mặt khi còn bản ghi có offset nhỏ hơn của bảng khác kẹt lại trong RAM.

4. **Hạng mục 4: Các vòng Audit phản biện chuyên sâu (Adversarial Audit Rounds) & Vá lỗi gốc rễ:**
   - Vạch trần và khắc phục 7 lỗ hổng tinh vi phát sinh trong quá trình thiết kế và thực thi, bao gồm lệch pha trạng thái Watermark Tracker và tính bất biến của NATS JetStream DeliverGroup.

---

## 2. TỔNG HỢP CÁC GIẢI PHÁP ĐÃ TRIỂN KHAI

1. **Kiến trúc Đệm Kép (Dual-Buffer Architecture):**
   - Tách rời hoàn toàn luồng ghi thời gian thực (`cdcBatchBuffer`) và luồng nạp hàng loạt (`snapshotBatchBuffer`). Buffer Snapshot được cấu hình `isSnapshotBuffer = true` để không bao giờ can thiệp vào commit offset của Kafka.
2. **Cơ chế Xả Đệm Định Hướng Bảng (Table-Scoped Flush):**
   - Hàm `FlushTableWithContext(parentCtx, targetTable)` chỉ lọc các bản ghi thuộc đúng bảng chỉ định để xả xuống DB, giữ nguyên các bản ghi của bảng khác trong RAM.
   - Chuẩn hóa tên bảng qua `matchTargetTable` để loại bỏ sai lệch do tiền tố schema `public.` hay dấu ngoặc kép.
3. **Cửa Sổ Trạng Thái Offset Liên Tục (Continuous Offset State Window):**
   - `StatefulWatermarkTracker` quản lý `completedPendingOffsets` (Set các offset đã hoàn tất) và `uncommittedFailures` (Set các offset lỗi).
   - Tích hợp `minPendingOffset` vào tính toán rào chắn: $\text{effectiveBarrier} = \min(\text{minFail}, \text{minPending})$.
   - Tìm offset hoàn tất cao nhất dưới rào chắn (`maxCompletedUnderBarrier`) để cho phép Watermark nhảy vọt giải phóng lag mà không bao giờ commit vượt mặt hoặc commit khống message chưa xử lý.
4. **NATS JetStream Push Consumer Pool & Versioning:**
   - Triển khai `TransmuteConsumerPool` với `DeliverGroup` trên `ConsumerConfig` để JetStream phân phối tải đều cho $N$ worker subscriptions.
   - Áp dụng Consumer Versioning (`transmute-shadow-consumer-v2`) để rolling update an toàn trên K8s mà không phá vỡ subscription của Pod cũ.
   - Tự động xóa và tạo lại (`DeleteConsumer` $\rightarrow$ `AddConsumer`) khi phát hiện lệch cấu hình `DeliverGroup`.
5. **Non-blocking Graceful Shutdown trong Debouncer:**
   - Khi nhận tín hiệu `ctx.Done()` từ Pod shutdown, `TableDebouncer` xả trực tiếp các tác vụ còn lại vào một goroutine nền với `context.Background()`, loại bỏ nguy cơ treo tiến trình do ghi vào unbuffered channel.

---

## 3. CHI TIẾT CÁC FILE VÀ DÒNG CODE ĐÃ THỰC HIỆN

| STT | File | Vị trí Line Code | Chi tiết thay đổi |
| :---: | :--- | :--- | :--- |
| 1 | `internal/model/shadow/cdc_event.go` | Lines 25–27 | Bổ sung 3 trường Marker: `IsMarker`, `IsDLQMarker`, `IsNoOpMarker` (gorm/json ignored). |
| 2 | `internal/handler/shadow/watermark_tracker.go` | Lines 1–259 (File mới) | Xây dựng Tracker trạng thái với `completedPendingOffsets`, `CalculateSafeCommitOffsetsWithPending`, `maxCompletedUnderBarrier`, dọn rác tự động. |
| 3 | `internal/handler/shadow/batch_buffer.go` | Lines 268–347 | Thêm `FlushTableWithContext`, `matchTargetTable`, tích hợp `CalculateSafeCommitOffsetsWithPending`. |
| 4 | `internal/handler/shadow/batch_buffer.go` | Lines 391–429 | `FlushWithContext`: Phân loại Marker vs Data, commit offset với timeout 5s độc lập. |
| 5 | `internal/handler/shadow/batch_buffer.go` | Lines 470–474 | `publishTransmuteTrigger`: Tạo deterministic NatsMsgID chống trùng lặp. |
| 6 | `internal/handler/shadow/batch_buffer.go` | Lines 522–571 | `evacuateFailedChunkToDLQ`: Sơ tán lỗi sang DLQ với timeout 10s độc lập. |
| 7 | `internal/handler/shadow/batch_buffer_fanout.go` | Lines 16–35 | Bổ sung `AddAll` gom khóa Mutex 1 lần, bảo vệ tính nguyên tử của batch fan-out. |
| 8 | `internal/handler/shadow/event_handler.go` | Lines 451–497 | Staging các record sau mapping và nạp nguyên tử qua `batchBuffer.AddAll`. |
| 9 | `internal/server/server_setup.go` | Lines 185–235 | Khởi tạo tách biệt `cdcBatchBuffer` và `snapshotBatchBuffer`. |
| 10 | `internal/server/server_setup.go` | Lines 338–362 | Khởi tạo `TransmuteConsumerPool` với consumer `v2`; loại bỏ double dispatch Core NATS. |
| 11 | `internal/server/server_setup.go` | Lines 476–480 | Nối `snapshotEventHandler` vào `recon` để cô lập hoàn toàn trace với CDC. |
| 12 | `internal/handler/recon/recon_heal_fetch.go` | Lines 110–120 | Đổi lệnh xả từ `FlushBatchBuffer` sang `FlushTableBatchBuffer` định hướng bảng. |
| 13 | `internal/handler/master/transmute_consumer_pool.go` | Lines 1–167 (File mới) | Push consumer pool với `DeliverGroup`, xử lý poison pill (`MaxDeliver = 5`), recreate fallback. |
| 14 | `internal/handler/master/debounce.go` | Lines 128–159 | Sửa `triggerFlushLocked` và `workerLoop` xả nền non-blocking khi `ctx.Done()`. |
| 15 | `internal/handler/shadow/watermark_tracker_test.go` | Lines 1–180 (File mới) | Suite kiểm thử: Middle Failure, DLQ Unfreeze, Continuous Window qua 5 mẻ liên tiếp. |
| 16 | `internal/handler/shadow/snapshot_cdc_isolation_test.go` | Lines 1–125 (File mới) | Suite kiểm thử cô lập Snapshot/CDC và rào chắn `minPendingOffset`. |
| 17 | `internal/handler/master/debounce_test.go` | Lines 1–70 (File mới) | Suite kiểm thử xả batch debouncer và shutdown không deadlock. |

---

## 4. CÁC ISSUES GẶP PHẢI TRONG QUÁ TRÌNH LÀM & CÁCH ĐÃ VÁ

Trong quá trình thực thi và qua 3 vòng Audit phản biện chuyên sâu (Adversarial Audit), hệ thống đã chủ động truy vết và bít kín **7 tử huyệt kiến trúc**:

### 🛑 Issue 1: Double Dispatching trên NATS Subject `cdc.cmd.transmute-shadow`
* **Vấn đề:** Ban đầu, cả `TransmuteConsumerPool` (JetStream) và Core NATS `natsClient.Conn.QueueSubscribe` đều đăng ký lắng nghe subject `cdc.cmd.transmute-shadow`. Hậu quả: mỗi mẻ xả Shadow bị trigger và biến đổi 2 lần ở Master.
* **File & Line vá:** `internal/server/server_setup.go` (Lines 360–362).
* **Nội dung vá:** Bọc điều kiện `if natsClient.JS == nil`, chỉ kích hoạt Core NATS subscriber khi không có JetStream.

### 🛑 Issue 2: Table Name Mismatch gây trượt xả đệm trong `FlushTableWithContext`
* **Vấn đề:** Khi luồng Heal gọi xả bảng `payment_bills`, các bản ghi trong đệm có thể chứa `public.payment_bills` hoặc `"public"."payment_bills"`. Phép so sánh chuỗi đơn giản `r.TableName == targetTable` bị sai lệch khiến 0 bản ghi nào được xả.
* **File & Line vá:** `internal/handler/shadow/batch_buffer.go` (Lines 349–368).
* **Nội dung vá:** Xây dựng hàm `matchTargetTable` chuẩn hóa tên bảng: loại bỏ schema prefix `public.`, chuyển về chữ thường và xóa sạch dấu ngoặc kép bọc quanh.

### 🛑 Issue 3: Fake Concurrency trên NATS JetStream Push Consumer
* **Vấn đề:** Khởi tạo $N$ worker subscriptions vào JetStream mà không có trường `DeliverGroup` trong `nats.ConsumerConfig`. JetStream broker sẽ phân phối trùng lặp tin nhắn cho toàn bộ worker hoặc không load-balancing.
* **File & Line vá:** `internal/handler/master/transmute_consumer_pool.go` (Lines 61–73).
* **Nội dung vá:** Khai báo tường minh `DeliverGroup: queueGroup` trong `nats.ConsumerConfig` ngay tại bước `AddConsumer`.

### 🛑 Issue 4: Nguy cơ Commit Vượt Mặt (Advance Commit Hazard)
* **Vấn đề:** Trong `FlushTableWithContext("payment_bills")`, nếu mẻ xả thành công offset 501, nhưng trong RAM vẫn còn bản ghi offset 500 của bảng `orders` chưa được xả. Nếu commit 501 lên Kafka, bản ghi 500 sẽ bị mất vĩnh viễn nếu Pod gặp sự cố trước khi xả bảng `orders`.
* **File & Line vá:** `internal/handler/shadow/batch_buffer.go` (Lines 312–322).
* **Nội dung vá:** Quét tìm `minPendingOffset` của tất cả các bản ghi còn kẹt lại trong `bb.records` theo từng `TopicPartition` để thiết lập rào chắn an toàn.

### 🛑 Issue 5: Lệch pha trạng thái (State Divergence) trong Watermark Tracker
* **Vấn đề (User chỉ ra ở Audit Vòng 2):** Khi `batch_buffer.go` chặn trần commit ngoài buffer ở 499, nhưng Tracker bên trong đã lưu `t.lastCommitted = 501`. Ở mẻ sau, khi offset 500 xả thành công, Tracker thấy $500 \ngtr 501$ sẽ từ chối emit offset 500 $\rightarrow$ Commit lag trên Kafka bị đóng băng vĩnh viễn ở 499.
* **File & Line vá:**
  - `internal/handler/shadow/watermark_tracker.go` (Lines 50–75, 120–238).
  - `internal/handler/shadow/batch_buffer.go` (Lines 324–344).
* **Nội dung vá:** Bổ sung `completedPendingOffsets` vào Tracker. Chuyển `minPendingOffset` vào hàm `CalculateSafeCommitOffsetsWithPending`. Loại bỏ lọc ngoài buffer để Tracker là nguồn chân lý duy nhất.

### 🛑 Issue 6: Nguy cơ Commit Khống Offset chưa xử lý ở mẻ sau (Edge-case Barrier Jump)
* **Vấn đề:** Khi offset 500 hoàn tất ở mẻ 2 nhưng trong RAM lại xuất hiện offset 600 pending. Nếu Tracker chỉ lấy `barrier - 1 = 599`, hệ thống sẽ commit khống các offset 502..599 chưa hề tồn tại hoặc chưa xử lý xong.
* **File & Line vá:** `internal/handler/shadow/watermark_tracker.go` (Lines 197–217).
* **Nội dung vá:** Quét tìm `maxCompletedUnderBarrier` trong `completedPendingOffsets` thỏa mãn $\text{offset} < \text{effectiveBarrier}$. Commit dừng chính xác ở 501, cấm tuyệt đối việc nhảy lên 599.

### 🛑 Issue 7: JetStream DeliverGroup Immutability & Rủi ro Kubernetes Rolling Update
* **Vấn đề:** Thuộc tính `DeliverGroup` trên JetStream Durable Consumer là bất biến (`immutable`). Lệnh `UpdateConsumer` bị broker từ chối. Nếu xóa nóng consumer cũ khi Pod cũ đang chạy, Pod cũ sẽ gãy subscription và lỗi message inflight.
* **File & Line vá:**
  - `internal/server/server_setup.go` (Line 342).
  - `internal/handler/master/transmute_consumer_pool.go` (Lines 83–98).
* **Nội dung vá:** Nâng cấp phiên bản consumer thành `transmute-shadow-consumer-v2` để Pod mới bind vào `v2`, Pod cũ drain `v1`. Bổ sung fallback `DeleteConsumer` $\rightarrow$ `AddConsumer` nếu cấu hình bị lệch.

---

## 5. TỔNG HỢP KẾT QUẢ NGHIỆM THU & THẨM ĐỊNH

1. **Biên dịch & Unit Tests:**
   - Biên dịch: `go build ./cmd/... ./internal/... ./pkgs/...` $\rightarrow$ PASS (Exit 0).
   - Test suites `internal/handler/shadow/...`: PASS 100% (Bao gồm test 5 mẻ continuous window).
   - Test suites `internal/handler/master/...`: PASS 100%.
   - Test suites `internal/handler/recon/...`: PASS 100%.
2. **Thẩm định độc lập từ mô hình `/laya_router` (`http://127.0.0.1:8200/v1/predict`):**
   - Vòng 1 (Unified Architecture v8.0): 10/10 claims được đồng thuận cao (Double dispatch 85.1%, Staged AddAll 83.5%, Match table 71.4%).
   - Vòng 2 (Adversarial Audit): Xác nhận lỗ hổng State Divergence Tracker (65.5% YES).
   - Vòng 3 (Root-Cause Fix): Đồng thuận giải pháp Continuous Window (80.47% YES) và Consumer Versioning v2 (51.27% YES).
3. **Kế hoạch phối hợp Trace Runtime Value thực tế:**
   - Sẵn sàng phối hợp cùng User deploy lên cụm test/staging, trigger đồng thời luồng Heal và luồng CDC để đối soát runtime logs của `minPendingOffset`, Kafka Lag trên broker và NATS CLI.
