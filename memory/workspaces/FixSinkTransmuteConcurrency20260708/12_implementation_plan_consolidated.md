# Kế hoạch Triển khai Tổng thể Kỹ thuật: Concurrency, Batching & FE Visualizations Optimization

Tài liệu này đặc tả chi tiết thiết kế mã nguồn, cấu trúc dữ liệu và logic xử lý tổng thể của toàn bộ phiên tối ưu hóa, tích hợp giải pháp triệt để cho **12 lỗ hổng phân tán ban đầu** và **19 lỗ hổng thiết kế nghiêm trọng phát hiện sau các vòng rà soát chéo (Hardenings Phase 5 - Update 1)** tại tầng ghi Shadow DB (Sink Worker), Master DB (Transmute Worker), và các cải tiến trực quan biểu đồ đối soát trên Frontend.

---

## 1. Phân tích 19 Vấn đề Nghiêm trọng & Giải pháp Gia cố (Hardenings)

### 1.1. Vấn đề 1: Trả về số lượng `written` "nửa vời" khi Sink Flush bị lỗi
*   **Vấn đề:** Trong hàm `Flush()`, `written` cộng dồn số lượng bản ghi của tất cả các bảng bao gồm cả bảng lỗi. Khi người gọi nhận được `err != nil` nhưng `written > 0`, họ dễ hiểu nhầm toàn bộ mẻ thất bại và thực hiện retry trùng lặp dữ liệu.
*   **Giải pháp:** Chỉ cộng dồn `groupWritten` vào biến trả về `written` khi lô ghi của bảng đó thành công (`gerr == nil`). Nếu bảng lỗi, không tính số dòng của bảng đó vào `written` để người gọi xác định chính xác số lượng dữ liệu đã lưu thành công.

### 1.2. Vấn đề 2: Poison Pill Binary Search có thể Term() oan các ID hợp lệ
*   **Vấn đề:** Khi chia nhỏ mẻ ghi về kích thước 1 task (`len(batch) == 1`), task này có thể chứa danh sách nhiều `SourceIDs` do được gom lại trước đó. Nếu chỉ có 1 ID lỗi, việc `Term()` toàn bộ message sẽ hủy oan các ID hợp lệ khác.
*   **Giải pháp:** Khi mẻ chia nhỏ chỉ còn 1 task, nếu `len(task.Req.SourceIDs) > 1`, phân rã và chạy thử nghiệm `runTransmute` đơn lẻ cho **từng ID** trong danh sách. Chỉ ghi DLQ và hủy các ID thực sự lỗi. Các ID thành công sẽ được bỏ qua (vì đã lưu DB thành công). Sau đó mới gọi `Term()` tin nhắn gốc để giải phóng NATS.

### 1.3. Vấn đề 3: Lỗi tạm thời (Transient Error) trong lúc đệ quy chia nhỏ bị biến thành Poison Pill
*   **Vấn đề:** Trong `processSubBatch`, nếu gặp lỗi kết nối DB / sập mạng (transient error), code không kiểm tra mà vẫn tiếp tục chia đôi `binarySearchSplit`. Kết quả là toàn bộ các tin nhắn bình thường đều bị chia nhỏ đến tận cùng và bị `Term()` oan do DB không phản hồi.
*   **Giải pháp:** Tại `processSubBatch` và `processBatch`, trước khi bắt đầu `binarySearchSplit`, kiểm tra `isTransientError(err)`. Nếu đúng là lỗi transient, lập tức gọi `Nak()` cho toàn bộ sub-batch / batch để NATS gửi lại sau, tuyệt đối không tiếp tục chia nhỏ.

### 1.4. Vấn đề 4: Graceful Shutdown chưa đợi các goroutine `processBatch` đang chạy
*   **Vấn đề:** Khi nhận tín hiệu dừng (`td.ctx.Done()`), `workerLoop` return thoát ngay lập tức, bỏ mặc các goroutine chạy ngầm `processBatch` đang thực thi `runTransmute` bị hủy giữa chừng hoặc trôi ACK/Nak.
*   **Giải pháp:** Tích hợp quản lý vòng đời chặt chẽ bằng `sync.WaitGroup`. Gọi `td.wg.Add(1)` khi khởi chạy goroutine xử lý mẻ ghi, và `defer td.wg.Done()` khi kết thúc. Cung cấp phương thức `Close()` để cancel context và block chờ `td.wg.Wait()` cho đến khi toàn bộ worker ngầm xử lý xong.

### 1.5. Vấn đề 5: Thiếu giới hạn toàn cục (Global Semaphore) về số goroutine ghi Master DB
*   **Vấn đề:** Semaphore 10 của từng `TableDebouncer` chỉ giới hạn trên từng bảng. Khi có tải lớn trên 200 bảng, hệ thống có thể mở tới 2000 goroutine ghi DB đồng thời, làm sập connection pool của Postgres.
*   **Giải pháp:** Triển khai **Global Semaphore** (chênh lệch kích thước khoảng `30 - 50` slots) được khởi tạo tập trung tại `TransmuteHandler` và truyền vào từng `TableDebouncer`. Các worker ghi của bất kỳ bảng nào đều phải lấy slot từ Global Semaphore trước khi ghi DB, bên cạnh Semaphore cục bộ của bảng đó (ví dụ tối đa 3 concurrent/bảng để chống Row Lock Contention).

### 1.6. Vấn đề 6: Deadlock trong `triggerFlushLocked` khi channel đầy
*   **Vấn đề:** Thao tác đẩy dữ liệu vào `td.flushCh <- td.tasks` diễn ra bên trong phạm vi đang giữ Mutex Lock (`td.mu.Lock()`). Nếu channel bị đầy do worker bị block, luồng đẩy tin sẽ bị block vô hạn trong khi đang giữ Lock, dẫn đến toàn bộ luồng `Add()` khác bị deadlock theo.
*   **Giải pháp:** Thiết kế lại cơ chế khóa: Tách biệt hoàn toàn thao tác trích xuất dữ liệu và đẩy dữ liệu. Trích xuất batch dưới lock và trả về ngoài lock, sau đó mới đẩy vào channel để tránh hoàn toàn việc chiếm giữ lock khi channel bị đầy (Xem chi tiết Vấn đề 18).

### 1.7. Vấn đề 7: Đánh rơi dữ liệu (Orphaned Batch) khi tắt máy (Graceful Shutdown)
*   **Vấn đề:** Khi context Done, workerLoop bốc mẻ tasks cuối cùng và đẩy vào `flushCh` thông qua `triggerFlushLocked()`, nhưng ngay sau đó goroutine workerLoop thực hiện return thoát luôn. Do không còn worker đọc từ `flushCh`, mẻ tasks cuối cùng này bị kẹt vô hạn trên RAM và mất tích.
*   **Giải pháp:** Khi tắt máy, bốc trực tiếp mẻ cuối ra khỏi đệm và gọi `processBatch` đồng bộ để hoàn tất xử lý trước khi thoát.

### 1.8. Vấn đề 8: Ảo giác `InProgress` khi chạy Binary Search Poison Pill
*   **Vấn đề:** Tiến trình chia nhị phân và chạy bulk/single ghi DB cho mẻ lớn có thể kéo dài vài phút nếu DB chậm. Việc chỉ gọi `t.Msg.InProgress()` một lần ở đầu hàm không đủ duy trì trạng thái, khiến NATS Server hết hạn AckWait và giao tin nhắn trùng lặp cho node khác.
*   **Giải pháp:** Khởi tạo một goroutine Heartbeat định kỳ (mỗi 5s) tự động gọi `t.Msg.InProgress()` cho toàn bộ các tin nhắn trong mẻ chừng nào tiến trình `processBatch` của mẻ đó vẫn đang chạy. Đảm bảo tắt Heartbeat thông qua cơ chế đóng kênh `doneCh` khi tiến trình xử lý kết thúc.

### 1.9. Vấn đề 9: Nghẽn cổ chai tài nguyên (Lock Inversion / Starvation) giữa Global và Local Semaphore
*   **Vấn đề:** Việc lấy `globalSem` trước rồi mới đến `localSem` có thể gây tình trạng "starvation". Ví dụ mẻ tin nhắn của bảng A lấy hết 30 slot của `globalSem` nhưng sau đó bị nghẽn ở `localSem` (chỉ cho phép 3 slot). 27 slots của `globalSem` còn lại sẽ bị kẹt vô ích bởi bảng A, làm tê liệt các bảng B, C, D khác.
*   **Giải pháp:** **Luôn lấy khóa cục bộ (Local Sem) trước, khóa toàn cục (Global Sem) sau**. Khi goroutine lọt qua được vòng hạn chế cục bộ của bảng, mới đi xếp hàng xin cấp phát tài nguyên Postgres toàn cục.

### 1.10. Vấn đề 10: Mất dữ liệu (Data Loss) do hủy context khi tắt Pod (SIGTERM)
*   **Vấn đề:** Khi Kubernetes gửi SIGTERM, context hủy (`td.ctx.Done()`). Câu lệnh DB transmute dở dang bị ngắt và trả về lỗi `context.Canceled` hoặc `context.DeadlineExceeded`. Do hàm `isTransientError(err)` không bắt lỗi context, hệ thống hiểu nhầm đây là lỗi dữ liệu cứng (Poison Pill), lập tức chia nhị phân và kết quả là hủy (`Term()`) toàn bộ lô tin nhắn cuối.
*   **Giải pháp:** Bổ sung các lỗi `context.Canceled`, `context.DeadlineExceeded` và các chuỗi text tương ứng vào hàm lọc lỗi `isTransientError(err)` để Nak() và lưu giữ toàn bộ dữ liệu an toàn.

### 1.11. Vấn đề 11: Block vĩnh viễn ở `triggerFlushLocked` khi Shutdown
*   **Vấn đề:** Dù đã nhả Mutex trước khi đẩy vào channel `td.flushCh <- batch`, nhưng nếu channel đầy và workerLoop đã thoát do Shutdown, goroutine đẩy tin sẽ bị treo vĩnh viễn tại dòng này, gây rò rỉ goroutine.
*   **Giải pháp:** Sử dụng `select` kết hợp với `ctx.Done()`. Nếu context đã Done khi đang đợi đẩy channel, chuyển sang gọi `Nak()` toàn bộ lô tin nhắn để NATS phân phối lại ở lần khởi động kế tiếp.

### 1.12. Vấn đề 12: Tác dụng phụ của Heartbeat InProgress lên tin nhắn đã giải quyết (Ack/Term)
*   **Vấn đề:** Heartbeat định kỳ lặp qua danh sách `batch` và gửi `InProgress()`. Khi một số tin nhắn con đã được `Ack()` hoặc `Term()` trong đệ quy, việc tiếp tục gọi `InProgress()` sẽ sinh ra log lỗi cảnh báo rác từ thư viện NATS.
*   **Giải pháp:** Chủ động bỏ qua lỗi bằng cách gán về blank identifier `_ = t.Msg.InProgress()`.

### 1.13. Vấn đề 13: Heartbeat gửi `InProgress()` trên message đã hoàn tất gây panic/lỗi runtime
*   **Vấn đề:** Khi đệ quy nhị phân `binarySearchSplit` chia tách và lần lượt `Ack()`/`Nak()` các sub-batches con, luồng Heartbeat chạy ngầm ở `processBatch` vẫn quét mảng gốc và cố đấm ăn xôi gọi `t.Msg.InProgress()` trên các tin nhắn đã hoàn tất. Điều này sẽ kích hoạt panic hoặc lỗi runtime trong thư viện NATS.
*   **Giải pháp:** Đưa trạng thái `resolved` (kiểu con trỏ `*int32` bảo vệ bằng atomic CAS) vào trực tiếp trong `NatsMsgTask`. Khi gọi `Ack()`, `Nak()`, hoặc `Term()`, thiết lập trạng thái này thành `1`. Heartbeat chỉ gửi `InProgress()` nếu `resolved` bằng `0`.

### 1.14. Vấn đề 14: Bỏ sót lỗi tuần tự hóa (Serialization Failure / Deadlock) của Postgres
*   **Vấn đề:** Dưới tải cao Upsert song song, Postgres thường ném ra lỗi `serialization failure` (SQLSTATE 40001) hoặc `deadlock detected` (SQLSTATE 40P01). Đây là lỗi tạm thời (Transient) chỉ cần retry là ghi được. If không đưa vào `isTransientError`, hệ thống sẽ coi là Poison Pill và gọi chia nhị phân rồi Term() oan dữ liệu.
*   **Giải pháp:** Bổ sung các chuỗi định danh `"serialization failure"`, `"deadlock detected"`, `"40001"`, `"40p01"` vào hàm `isTransientError(err)`.

### 1.15. Vấn đề 15: Đảo thứ tự Semaphore cục bộ/toàn cục trong phần lý thuyết đặc tả
*   **Vấn đề:** Phần giải thích và mã nguồn cần đảm bảo đồng bộ 100% trong việc lấy Local Semaphore trước, Global Semaphore sau để tối ưu hóa việc phân chia tài nguyên và ngăn nghẽn cổ chai (Starvation).
*   **Giải pháp:** Đảm bảo toàn bộ tài liệu đặc tả thiết kế và mã nguồn Go sử dụng chính xác trình tự Local Sem trước $\rightarrow$ Global Sem sau.

### 1.16. Vấn đề 16: Lỗi Double Unlock Mutex gây Panic khi timer kích hoạt
*   **Vấn đề:** Việc thiết kế hàm `triggerFlushLocked` tự ý Unlock rồi Lock lại Mutex bên trong, kết hợp với khối lệnh callback timer `mu.Lock() -> triggerFlushLocked() -> mu.Unlock()` vô tình dẫn đến mất cân bằng hoặc double unlock nếu có luồng xen ngang hoặc thay đổi trạng thái, dẫn tới panic sập chương trình.
*   **Giải pháp:** Loại bỏ hoàn toàn logic tự ý thay đổi lock bên trong các hàm con. Thay vào đó, trích xuất dữ liệu batch ra ngoài dưới lock qua hàm `extractBatchLocked()`, sau đó xử lý đẩy channel hoàn toàn ngoài lock (Clean Lock Pattern).

### 1.17. Vấn đề 17: Lặp lại trùng lặp dữ liệu (Over-duplication) khi `isolatePoisonPillTask` gọi Nak()
*   **Vấn đề:** Khi tách riêng từng ID trong task bị lỗi dữ liệu, nếu một ID gặp lỗi transient (như DB rớt mạng tạm thời) ở giữa chặng, code gọi `task.Nak()` và dừng lại. NATS sẽ phân phối lại toàn bộ mẻ tin nhắn gốc chứa tất cả ID ban đầu (bao gồm cả các ID đã được ghi thành công vào DB ở chặng trước đó), gây lãng phí tài nguyên và rủi ro ghi đè dữ liệu cũ.
*   **Giải pháp:** Tích hợp bộ thử lại cục bộ với khoảng trễ tăng dần (Local Exponential Backoff Retry - tối đa 3 lần) cho các lỗi transient ngay bên trong hàm `isolatePoisonPillTask`. Điều này giúp giải quyết phần lớn lỗi DB lock/serialization tạm thời mà không cần Nak() toàn bộ tin nhắn gốc về NATS.

### 1.18. Vấn đề 18: Race Condition và thiết kế Anti-Pattern của `triggerFlushLocked`
*   **Vấn đề:** Tự ý thay đổi trạng thái lock trong hàm mang hậu tố `Locked` là phản kiến trúc, gây khó kiểm soát luồng. Ngoài ra, việc nhả lock và đứng chờ tại channel push mở ra khe hở lớn để các goroutine NATS Consumer ghi đè `td.tasks` hoặc timers tạo nên race condition.
*   **Giải pháp:** Refactor triệt để. Tạo hàm `extractBatchLocked()` chỉ làm nhiệm vụ lấy đệm tasks và dừng timer dưới lock. Trả về batch tasks ra ngoài và gọi hàm `pushToFlushCh(batch)` để đẩy channel hoàn toàn bên ngoài lock.

### 1.19. Vấn đề 19 (MỚI): Mất dữ liệu từ các batch đã nằm trong `flushCh` khi Graceful Shutdown
*   **Vấn đề:** Khi context bị hủy (SIGTERM), `workerLoop` lấy batch cuối từ `td.tasks` rồi thoát ngay lập tức. Tuy nhiên, các batch được đẩy vào `flushCh` trước đó bởi timers hoặc các luồng `Add()` khác nhưng chưa kịp được `workerLoop` tiêu thụ sẽ bị bỏ lại trên RAM mà không có phản hồi Ack/Nak/Term. Điều này dẫn tới trôi AckWait, redeliver chậm trễ hoặc mất tin nhắn nếu quá số lần thử lại tối đa.
*   **Giải pháp:** Khi shutdown, sau khi giải phóng và xử lý đồng bộ mẻ cuối từ buffer nội bộ, `workerLoop` tiến hành quét sạch (drain) các batch còn sót lại trong `flushCh` bằng cơ chế đọc non-blocking và gọi `Nak()` lập tức cho toàn bộ các tin nhắn đó để NATS phân phối lại sau khi hệ thống khởi động lại.

---

## 2. Thiết kế Mã nguồn Backend (Gia cố Toàn diện)

### 2.1. Shadow Sink Layer (`batch_buffer.go`)
*   **Đường dẫn file:** [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)

```go
package shadow

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func (bb *BatchBuffer) Flush() (written int, err error) {
	ctx := bb.ctx

	bb.mu.Lock()
	if len(bb.records) == 0 {
		bb.mu.Unlock()
		return 0, nil
	}
	batch := bb.records
	bb.records = make([]*shadow.UpsertRecord, 0, bb.maxSize)
	bb.lastFlush = time.Now()
	bb.mu.Unlock()

	// Phân nhóm bản ghi theo bảng
	byTable := make(map[string][]*shadow.UpsertRecord)
	for _, r := range batch {
		byTable[bb.groupKey(r)] = append(byTable[bb.groupKey(r)], r)
	}

	var mu sync.Mutex
	var g errgroup.Group // Sử dụng errgroup.Group thuần (không WithContext)
	g.SetLimit(20)       // Khống chế số lượng bảng ghi song song đồng thời

	for groupKey, records := range byTable {
		gk := groupKey
		recs := records
		g.Go(func() error {
			// Sử dụng parent ctx, không dùng cancelCtx bị liên đới lỗi của bảng khác
			groupWritten, gerr := bb.batchUpsert(ctx, recs)
			
			mu.Lock()
			defer mu.Unlock()

			if gerr != nil {
				// Ghi nhận lỗi cục bộ của bảng đó
				bb.logger.Error("batch upsert failed for table", 
					zap.Error(gerr), 
					zap.String("group", gk),
				)
				if err == nil {
					err = gerr // Lưu lỗi đầu tiên gặp phải
				}
				// Không cộng dồn groupWritten vào written khi lỗi (Fix Vấn đề 1)
			} else {
				metrics.BatchesFlushed.WithLabelValues("postgres", recs[0].TableName).Inc()
				if bb.natsConn != nil {
					ids := make([]string, 0, len(recs))
					for _, r := range recs {
						if r.PrimaryKeyValue != "" {
							ids = append(ids, r.PrimaryKeyValue)
						}
					}
					bb.publishTransmuteTrigger(ctx, recs[0].SchemaName, recs[0].TableName, ids)
				}
				written += groupWritten // Chỉ cộng số lượng thành công thực tế
			}
			return nil // Luôn trả về nil để bảo vệ các goroutine khác khỏi bị cancel
		})
	}

	_ = g.Wait()
	return written, err
}
```

### 2.2. Master Transmute Layer (`debounce.go`)
*   **Đường dẫn file:** [debounce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/debounce.go)

```go
package master

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
)

// NatsMsgTask bọc tin nhắn NATS và trạng thái resolved thớt an toàn (Fix Vấn đề 13)
type NatsMsgTask struct {
	Msg      *nats.Msg
	Req      TransmuteRequest
	resolved *int32 // Con trỏ chia sẻ trạng thái: 0 = active, 1 = resolved
}

func (t *NatsMsgTask) Ack() {
	if atomic.CompareAndSwapInt32(t.resolved, 0, 1) {
		_ = t.Msg.Ack()
	}
}

func (t *NatsMsgTask) Nak() {
	if atomic.CompareAndSwapInt32(t.resolved, 0, 1) {
		_ = t.Msg.Nak()
	}
}

func (t *NatsMsgTask) Term() {
	if atomic.CompareAndSwapInt32(t.resolved, 0, 1) {
		_ = t.Msg.Term()
	}
}

func (t *NatsMsgTask) InProgress() {
	if atomic.LoadInt32(t.resolved) == 0 {
		_ = t.Msg.InProgress()
	}
}

type TableDebouncer struct {
	mu           sync.Mutex
	tableName    string
	tasks        []NatsMsgTask
	maxSize      int
	idleTimeout  time.Duration
	maxTimeout   time.Duration
	idleTimer    *time.Timer
	maxTimer     *time.Timer
	flushCh      chan []NatsMsgTask
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	localSem     chan struct{} // Khống chế concurrency cục bộ của bảng (vd: 3)
	globalSem    chan struct{} // Semaphore toàn cục dùng chung cho tất cả các bảng (Fix Vấn đề 5)
	runTransmute func(ctx context.Context, table string, ids []string) error
	failedLogger func(ctx context.Context, table string, id string, err error)
}

func NewTableDebouncer(
	ctx context.Context,
	tableName string,
	maxSize int,
	idleTimeout time.Duration,
	maxTimeout time.Duration,
	localLimit int,
	globalSem chan struct{}, // Nhận global semaphore dùng chung (Fix Vấn đề 5)
	runTransmute func(ctx context.Context, table string, ids []string) error,
	failedLogger func(ctx context.Context, table string, id string, err error),
) *TableDebouncer {
	subCtx, cancel := context.WithCancel(ctx)
	td := &TableDebouncer{
		tableName:    tableName,
		maxSize:      maxSize,
		idleTimeout:  idleTimeout,
		maxTimeout:   maxTimeout,
		flushCh:      make(chan []NatsMsgTask, 50),
		ctx:          subCtx,
		cancel: cancel,
		localSem:     make(chan struct{}, localLimit),
		globalSem:    globalSem,
		runTransmute: runTransmute,
		failedLogger: failedLogger,
	}

	td.wg.Add(1)
	go td.workerLoop()

	return td
}

// Add tích hợp Backpressure và Clean Lock Pattern (Fix Vấn đề 18)
func (td *TableDebouncer) Add(msg *nats.Msg, req TransmuteRequest) {
	for {
		td.mu.Lock()
		
		// Backpressure: Hãm phanh nếu RAM phình to
		if len(td.tasks) >= td.maxSize*2 {
			td.mu.Unlock()
			select {
			case <-td.ctx.Done():
				_ = msg.Nak()
				return
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		resolved := int32(0)
		td.tasks = append(td.tasks, NatsMsgTask{
			Msg:      msg,
			Req:      req,
			resolved: &resolved,
		})

		var batch []NatsMsgTask
		if len(td.tasks) >= td.maxSize {
			batch = td.extractBatchLocked()
		} else {
			td.setupTimersLocked()
		}
		td.mu.Unlock()

		// Đẩy vào channel hoàn toàn bên ngoài Lock
		if len(batch) > 0 {
			td.pushToFlushCh(batch)
		}
		break
	}
}

func (td *TableDebouncer) setupTimersLocked() {
	if td.idleTimer != nil {
		td.idleTimer.Stop()
	}
	td.idleTimer = time.AfterFunc(td.idleTimeout, func() {
		td.mu.Lock()
		batch := td.extractBatchLocked()
		td.mu.Unlock()

		if len(batch) > 0 {
			td.pushToFlushCh(batch)
		}
	})

	if td.maxTimer == nil {
		td.maxTimer = time.AfterFunc(td.maxTimeout, func() {
			td.mu.Lock()
			batch := td.extractBatchLocked()
			td.mu.Unlock()

			if len(batch) > 0 {
				td.pushToFlushCh(batch)
			}
		})
	}
}

// extractBatchLocked trích xuất batch ra ngoài dưới Lock để tránh Double Unlock và Race Condition (Fix Vấn đề 16, 18)
func (td *TableDebouncer) extractBatchLocked() []NatsMsgTask {
	if len(td.tasks) == 0 {
		return nil
	}

	if td.idleTimer != nil {
		td.idleTimer.Stop()
		td.idleTimer = nil
	}
	if td.maxTimer != nil {
		td.maxTimer.Stop()
		td.maxTimer = nil
	}

	batch := td.tasks
	td.tasks = make([]NatsMsgTask, 0, td.maxSize)
	return batch
}

// pushToFlushCh đẩy batch vào channel một cách an toàn bên ngoài Mutex Lock (Fix Vấn đề 11, 18)
func (td *TableDebouncer) pushToFlushCh(batch []NatsMsgTask) {
	select {
	case td.flushCh <- batch:
	case <-td.ctx.Done():
		// Đang tắt máy, Nak lô này để NATS gửi lại sau (Fix Vấn đề 11)
		for _, t := range batch {
			t.Nak()
		}
	}
}

// workerLoop xử lý mẻ cuối đồng bộ và quét sạch kênh flushCh khi Graceful Shutdown (Fix Vấn đề 7, 19)
func (td *TableDebouncer) workerLoop() {
	defer td.wg.Done()

	for {
		select {
		case <-td.ctx.Done():
			// 1. Bốc mẻ cuối từ buffer nội bộ trực tiếp xử lý đồng bộ (Fix Vấn đề 7)
			td.mu.Lock()
			batch := td.extractBatchLocked()
			td.mu.Unlock()

			if len(batch) > 0 {
				td.wg.Add(1)
				td.processBatch(batch)
			}

			// 2. Dọn sạch (drain) toàn bộ các batch còn tồn đọng trong flushCh và Nak() (Fix Vấn đề 19)
			for {
				select {
				case pendingBatch := <-td.flushCh:
					for _, t := range pendingBatch {
						t.Nak()
					}
				default:
					return
				}
			}
		case batch := <-td.flushCh:
			td.wg.Add(1) // Tăng WaitGroup để track worker xử lý ngầm (Fix Vấn đề 4)
			go td.processBatch(batch)
		}
	}
}

func (td *TableDebouncer) processBatch(batch []NatsMsgTask) {
	defer td.wg.Done() // Hoàn tất xử lý ngầm (Fix Vấn đề 4)

	// 1. LẤY LOCAL SEMAPHORE TRƯỚC (Fix Vấn đề 9, 15)
	select {
	case td.localSem <- struct{}{}:
		defer func() { <-td.localSem }()
	case <-td.ctx.Done():
		for _, t := range batch {
			t.Nak()
		}
		return
	}

	// 2. LẤY GLOBAL SEMAPHORE SAU (Fix Vấn đề 9, 15)
	select {
	case td.globalSem <- struct{}{}:
		defer func() { <-td.globalSem }()
	case <-td.ctx.Done():
		for _, t := range batch {
			t.Nak()
		}
		return
	}

	// Kích hoạt Heartbeat liên tục gia hạn InProgress cho NATS (Fix Vấn đề 8, 12, 13)
	doneCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Bắn định kỳ mỗi 5s
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, t := range batch {
					t.InProgress() // An toàn, tự bỏ qua nếu đã Resolved (Fix Vấn đề 13)
				}
			case <-doneCh:
				return
			}
		}
	}()
	defer close(doneCh) // Đảm bảo đóng Heartbeat khi xử lý xong mẻ tin nhắn

	idSet := make(map[string]struct{})
	for _, t := range batch {
		for _, id := range t.Req.SourceIDs {
			if id != "" {
				idSet[id] = struct{}{}
			}
		}
	}

	uniqueIDs := make([]string, 0, len(idSet))
	for id := range idSet {
		uniqueIDs = append(uniqueIDs, id)
	}

	if len(uniqueIDs) == 0 {
		for _, t := range batch {
			t.Ack()
		}
		return
	}

	// Thực hiện Bulk Transmute
	err := td.runTransmute(td.ctx, td.tableName, uniqueIDs)
	if err == nil {
		for _, t := range batch {
			t.Ack()
		}
		return
	}

	// Lỗi transient -> Nak toàn bộ để gửi lại
	if isTransientError(err) {
		for _, t := range batch {
			t.Nak()
		}
		return
	}

	// Lỗi Poison Pill -> Chia đôi để trị
	td.binarySearchSplit(batch)
}

func (td *TableDebouncer) binarySearchSplit(batch []NatsMsgTask) {
	if len(batch) == 1 {
		// Cô lập Poison Pill Task và xử lý chi tiết từng ID (Fix Vấn đề 2)
		td.isolatePoisonPillTask(batch[0])
		return
	}

	mid := len(batch) / 2
	left := batch[:mid]
	right := batch[mid:]

	td.processSubBatch(left)
	td.processSubBatch(right)
}

func (td *TableDebouncer) processSubBatch(subBatch []NatsMsgTask) {
	idSet := make(map[string]struct{})
	for _, t := range subBatch {
		for _, id := range t.Req.SourceIDs {
			idSet[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	err := td.runTransmute(td.ctx, td.tableName, ids)
	if err == nil {
		for _, t := range subBatch {
			t.Ack()
		}
		return
	}

	// Kiểm tra lỗi transient trước khi đệ quy chia nhỏ để tránh Nak oan và Term oan (Fix Vấn đề 3 & Vấn đề 10)
	if isTransientError(err) {
		td.logger.Warn("Transient DB error during binary search. NAKing sub-batch.", zap.Error(err))
		for _, t := range subBatch {
			t.Nak()
		}
		return
	}

	td.binarySearchSplit(subBatch)
}

// Xử lý Poison Pill ở mức độ chi tiết ID để không huỷ oan dữ liệu hợp lệ (Fix Vấn đề 2, 17)
func (td *TableDebouncer) isolatePoisonPillTask(task NatsMsgTask) {
	if len(task.Req.SourceIDs) <= 1 {
		id := ""
		if len(task.Req.SourceIDs) == 1 {
			id = task.Req.SourceIDs[0]
		}
		td.logger.Error("Poison pill isolated successfully!", 
			zap.String("table", td.tableName), 
			zap.String("id", id),
		)
		td.failedLogger(td.ctx, td.tableName, id, fmt.Errorf("poison pill isolated"))
		task.Term() // Hủy message gốc trên NATS
		return
	}

	// Task chứa nhiều ID, tiến hành chạy đơn lẻ từng ID để tìm ID lỗi thực sự
	td.logger.Info("Multiple IDs in single poison task. Isolating individually...", 
		zap.Int("count", len(task.Req.SourceIDs)),
	)
	
	for _, id := range task.Req.SourceIDs {
		// Thử lại cục bộ với exponential backoff cho transient error để tránh Nak và lặp việc (Fix Vấn đề 17)
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			err = td.runTransmute(td.ctx, td.tableName, []string{id})
			if err == nil {
				break
			}
			if !isTransientError(err) {
				break // Lỗi Poison Pill thực sự, không cần retry
			}
			// Sleep tăng dần: 100ms, 200ms, 300ms
			time.Sleep(time.Duration(100*(attempt+1)) * time.Millisecond)
		}

		if err == nil {
			// Ghi đơn thành công -> ID này hợp lệ, bỏ qua
			continue
		}

		if isTransientError(err) {
			// Lỗi DB tạm thời kể cả sau khi retry cục bộ -> Nak cả mẻ tin nhắn
			td.logger.Warn("Transient error after local retries while isolating individual ID. NAKing message.", zap.Error(err))
			task.Nak()
			return
		}

		// Thực sự là ID lỗi dữ liệu -> DLQ
		td.logger.Error("Individual poison pill isolated successfully!", 
			zap.String("table", td.tableName), 
			zap.String("id", id),
			zap.Error(err),
		)
		td.failedLogger(td.ctx, td.tableName, id, err)
	}

	// Đã tách lọc xong, hủy message gốc
	task.Term()
}

// Close thực hiện Graceful Shutdown hoàn chỉnh (Fix Vấn đề 4)
func (td *TableDebouncer) Close() {
	td.cancel()
	td.wg.Wait() // Đợi toàn bộ goroutine xử lý ngầm hoàn tất
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true // Bắt chuẩn lỗi context (Fix Vấn đề 10)
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection pool") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "context canceled") || // Bắt chuỗi context cancel (Fix Vấn đề 10)
		strings.Contains(msg, "deadline exceeded") || // Bắt chuỗi context timeout (Fix Vấn đề 10)
		strings.Contains(msg, "serialization failure") || // Lỗi tuần tự hoá Postgres 40001 (Fix Vấn đề 14)
		strings.Contains(msg, "deadlock detected") ||      // Lỗi deadlock Postgres 40P01 (Fix Vấn đề 14)
		strings.Contains(msg, "40001") ||                 // SQLSTATE 40001
		strings.Contains(msg, "40p01") ||                 // SQLSTATE 40P01
		strings.Contains(msg, "sqlstate 08")
}
```

---

## 3. Thiết kế Cơ cơ chế Quan sát & Giám sát (Observability - OpenTelemetry / SigNoz)

Để truy vết vòng đời của một ID cụ thể bị từ chối từ lúc chui vào `BatchBuffer` (chặng Sink), truyền tải qua NATS, đi vào `TableDebouncer` (chặng Transmute), chia nhị phân, cho tới khi lưu DLQ, ta thiết kế mô hình Tracing tích hợp sau:

### 3.1. Lan truyền Trace Context (Propagator)
1.  **Chặng Sink:**
    *   Khi Kafka Consumer đọc message, khởi tạo hoặc trích xuất trace context hiện tại: `ctx, span := tracer.Start(ctx, "Sink HandleMessage")`.
    *   Mỗi bản ghi được lưu đệm trong `BatchBuffer` kèm metadata Trace ID.
2.  **Liên kết NATS JetStream:**
    *   Khi Sink Worker gọi `Flush()` và bắn transmute trigger lên NATS, ta thực hiện **Inject Trace Context** vào NATS Header:
        ```go
        header := nats.Header{}
        otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
        // Gửi trigger kèm header chứa traceparent
        ```

### 3.2. Chặng Transmute & Phân rã Nhị phân
1.  **Trích xuất Context (Extract):**
    *   Khi `TransmuteHandler` kéo tin nhắn từ NATS, ta trích xuất trace parent:
        ```go
        parentCtx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.HeaderCarrier(msg.Header))
        ctx, span := tracer.Start(parentCtx, "Transmute TableDebouncer ProcessBatch")
        ```
2.  **Child Spans cho các bước Binary Search:**
    *   Mỗi khi thuật toán chia nhị phân `binarySearchSplit` được kích hoạt, ta tạo một Child Span tương ứng với phạm vi phân tích:
        ```go
        ctx, childSpan := tracer.Start(ctx, fmt.Sprintf("Binary Split Range - Size %d", len(subBatch)))
        childSpan.SetAttributes(
            attribute.String("db.table", td.tableName),
            attribute.Int("batch.size", len(subBatch)),
        )
        // Khi kết thúc bước chia, gọi childSpan.End()
        ```
3.  **Cô lập Poison Pill (Isolate):**
    *   Tại thời điểm cô lập được ID lỗi cứng trong `isolatePoisonPillTask`, tạo span con cuối cùng: `Isolate Poison Pill - ID: <id>`.
    *   Lưu thông tin lỗi bằng `span.RecordError(err)` và đánh dấu trạng thái `codes.Error`.

### 3.3. Liên kết với Bảng DLQ (`failed_sync_logs`)
*   Khi ghi nhận bản ghi lỗi vào bảng `failed_sync_logs`, ta lưu thêm hai trường thông tin metadata: `trace_id` và `span_id` (trích xuất từ `span.SpanContext().TraceID().String()`).
*   **Lợi ích:** Khi kiểm tra lỗi trong bảng DLQ trên Admin Portal, ta có thể nhấp trực tiếp vào liên kết (Deep Link) dẫn tới SigNoz / Jaeger dựa trên `trace_id`, hiển thị trực quan toàn bộ sơ đồ hình cây của quá trình chia nhị phân cô lập lỗi.

---

## 4. Kế hoạch Kiểm thử & Xác minh (Verification Plan)

### 4.1. Unit Tests bổ sung
1.  `TestBatchBufferPartialSuccessFlush`: Ghi song song 2 bảng, 1 bảng lỗi và 1 bảng thành công. Xác minh `written` trả về khớp đúng số lượng dòng của bảng thành công, và `err != nil`.
2.  `TestPoisonPillIndividualIsolation`: Giả lập task gom 5 IDs (`ID1` đến `ID5`), trong đó `ID3` là Poison Pill. Xác minh hệ thống cô lập chạy riêng lẻ, lưu DLQ riêng cho `ID3`, không làm hỏng dữ liệu của các ID khác, và cuối cùng gọi `Term()`.
3.  `TestTransientErrorRetryInBinarySearch`: Giả lập lỗi transient xuất hiện giữa chừng khi chia nhị phân. Xác minh hệ thống gọi `Nak()` kịp thời cho mảng con đó thay vì chia nhỏ đến cùng và `Term()` oan.
4.  `TestGracefulShutdownRemainingTasks`: Giả lập shutdown khi còn tasks trong hàng chờ, xác minh tasks được xử lý trực tiếp thay vì bị mất.
5.  `TestHeartbeatDuringLongProcessing`: Giả lập DB bị chậm và xử lý nhị phân kéo dài. Xác minh heartbeat gửi tín hiệu InProgress định kỳ thành công.
6.  `TestLockOrderingLocalGlobalSem`: Kiểm thử việc chạy song song nhiều bảng độc lập, đảm bảo rằng một bảng bị nghẽn (local semaphore) không chiếm dụng slots và chặn đứng các bảng khác xử lý (global semaphore).
7.  `TestCleanLockPatternNoDoubleUnlock`: Kiểm thử việc flush và timers kích hoạt liên tục, xác minh không xảy ra deadlock hoặc panic double unlock.
8.  `TestIsolatePoisonPillLocalRetry`: Giả lập lỗi transient xuất hiện khi đang chạy đơn lẻ ID trong task lỗi. Xác minh hệ thống chạy retry cục bộ thành công mà không Nak toàn bộ tin nhắn.
9.  `TestGracefulShutdownDrainFlushCh`: Giả lập shutdown khi có 3 batch đã được xếp hàng sẵn trong `flushCh` nhưng chưa xử lý. Xác minh toàn bộ các message trong 3 batch đó đều được gọi `Nak()` thành công để giải phóng NATS trước khi thoát.

### 4.2. Quy trình bàn giao
*   Chạy linter quy trình: `python3 agent/tooling/verify_governance.py` trước khi hoàn tất task.
