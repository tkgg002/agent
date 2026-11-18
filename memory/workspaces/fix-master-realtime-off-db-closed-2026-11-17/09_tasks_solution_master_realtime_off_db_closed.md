# 09 Tasks Solution — Khắc Phục Triệt Để Lỗi Vẫn Chạy `kafka-consumer-hook` (`transmute` / `transmute_sub` / `transmute_poison_pill` — `sql: database is closed`) Sau Khi Tắt Sync Realtime Ở Master

- **Workspace:** `fix-master-realtime-off-db-closed-2026-11-17`
- **Suffix:** `master_realtime_off_db_closed`
- **Tier:** `T2` | **Mode:** `standard` (Brain Propose-Only — Chờ `APPROVE`)

---

## 1. Bức Tranh Tổng Quan Vật Lý & Bằng Chứng Điều Tra (`S2` + `DG0` — 100% `VERIFIED`)

Khi anh tắt **Sync Realtime (`mode = 'post_ingest'`)** ở Master, luồng dữ liệu đi qua 4 chặng:
`MasterRegistry UI (cdc-cms-web)` $\rightarrow$ `POST/PATCH/DELETE /api/v1/schedules (cdc-cms-service)` $\rightarrow$ `cdc_system.transmute_schedule` $\rightarrow$ `BatchBuffer / SinkWorker (Producer)` $\rightarrow$ `NATS JetStream CDC_TRANSMUTE (cdc.cmd.transmute-shadow)` $\rightarrow$ `TransmuteHandler + TableDebouncer (Consumer)`.

Đối chiếu trực tiếp mã nguồn trên đĩa và nhật ký lỗi lúc `15:40:28` – `15:40:35 17/11/2026`, có **4 nguyên nhân gốc rễ xác thực (`F1..F4`)** giải thích trọn vẹn cả 2 vế ("vì sao tắt Realtime vẫn chạy" và "vì sao vẫn nổ `sql: database is closed` + `transmute_poison_pill`"):

### `F1` (`VERIFIED`) — Chiều Consumer `TransmuteHandler` & `MasterBindingRepo` BỎ QUA HOÀN TOÀN cờ `post_ingest` (`transmute_schedule.is_enabled = true`)
- **REF:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go:80-112`, `199-290`, `630-713` và `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/master/master_binding_repo.go:78-139`.
- Ở chiều phát (`BatchBuffer.resolvePostIngestSchedule`, `batch_buffer_fanout.go:212-235`), cổng `post_ingest` chỉ đếm gộp ở mức **bảng Shadow** (`SELECT COUNT(*) ... WHERE ts.mode = 'post_ingest' AND ts.is_enabled = true AND sb.shadow_table = ?`).
- Nhưng ở chiều nhận (`cdc.cmd.transmute-shadow` $\rightarrow$ `handleTransmuteShadowWithOutcome` $\rightarrow$ `resolveMasterTargets` $\rightarrow$ `ListMasterTargetsByShadowIdentity` / `ListMasterTargetsByShadowTable`), câu SQL chỉ lọc `WHERE mb.is_active = true AND mb.schema_status = 'approved'`, **hoàn toàn không kiểm tra `cdc_system.transmute_schedule` (`mode = 'post_ingest' AND is_enabled = true`)**!
- **Hệ quả thực tế trên hệ thống của anh:**
  1. **Rò rỉ chéo `1 Shadow -> M Masters`:** Cả 2 cụm trong log đều là quan hệ `1 Shadow -> 2 Masters` (`shadow_payment_bill_service.payment_bills` $\rightarrow$ `public.cdc_payment_bill` + `public.cdc_payment_bill_payload`; `shadow_payment_service.payments` $\rightarrow$ `public.cdc_payment` + `public.cdc_payment_payload`). Hễ 1 bảng Master còn bật `post_ingest`, `BatchBuffer` vẫn bắn `cdc.cmd.transmute-shadow`, và `TransmuteHandler` fan-out ép chạy `kafka-consumer-hook` cho **cả bảng Master đã tắt Realtime**!
  2. **Bão phát lại từ NATS JetStream `CDC_TRANSMUTE` & `TableDebouncer`:** Kể cả khi đã tắt Realtime trên toàn bộ các bảng Master, những bản tin `cdc.cmd.transmute-shadow` đã nằm sẵn trong stream `CDC_TRANSMUTE` (`AckWait = 30s`, `MaxDeliver = 5`, `server_setup.go:311-317`) hoặc đang gom trong `TableDebouncer` khi gặp lỗi DB bị `NakWithDelay` phát lại liên tục (`15:40:28` $\rightarrow$ `15:40:29` $\rightarrow$ `15:40:35`) vẫn tiếp tục bị `TransmuteHandler` đưa thẳng vào `execRunEvents` thay vì `Ack` bỏ qua ngay!

### `F2` (`VERIFIED`) — Tắt/Bật/Xóa Schedule bên `cdc-cms-service` KHÔNG phát tín hiệu NATS xóa cache `piCache` (`30s` TTL) trên `BatchBuffer` & `SinkWorker`
- **REF:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/scheduler/create_schedule.go:50-72`, `toggle_schedule.go:43-70`, `delete_schedule.go:36-55`, `internal/app/commands/master/toggle_master_active.go:44-65`, và `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_lifecycle.go:47-53`, `batch_buffer_fanout_cache.go:6-33`, `internal/sinkworker/worker.go:89-104,428-443`.
- Khi anh gạt nút **Realtime -> TẮT** trên UI, `cdc-cms-service` cập nhật DB thành công nhưng **không phát bất kỳ sự kiện NATS nào** xuống Worker để xóa `piCache`.
- Bên Worker, `FlushSchemaCache()` (`batch_buffer_lifecycle.go:47-53`) chỉ xóa `SchemaAdapter` chứ không xóa `bb.piCache` (`postIngestCacheTTL = 30s`). Ngoài ra, nếu truy vấn làm mới gặp lỗi DB tạm thời, `postIngestEntryOnLookupError` (`batch_buffer_fanout_cache.go:20-23`) còn gán lại `stale.at = now` gia hạn `enabled = true` thêm nguyên chu kỳ 30s nữa; riêng `SinkWorker` (`worker.go:433-436`) còn thiếu nhánh join `multi_shadow` (`transform_spec->'shadow_sources'`).

### `F3` (`VERIFIED`) — Regression ở tầng CMS/UI: Lọc Schedule theo chuỗi `master_table`/`master_schema` thay vì `master_binding_id` & chặn nhầm `master_schema` chứa dấu `-`
- **REF:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx:879-896`, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx:1625`, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go:34-38`, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/scheduler/transmute_schedule_handler.go:94-99`.
- Sau đợt nâng cấp đa Engine Master (`MongoDB`/`MySQL`/`ClickHouse`), `ListEnriched` (`master_read_repo_gorm.go:34-38`) trả `master_schema = mc.default_database` (ví dụ `payment-service-v2`), trong khi `GET /api/v1/schedules` (`transmute_schedule_read_repo_gorm.go:36`) trả `mb.master_schema` gốc.
- Tại `MasterRegistry.tsx:883-886`, khi mở Modal Sync để nạp trạng thái nút gạt `Realtime`, code lọc theo `s.master_table === syncRow.master_name && (!syncRow.master_schema || s.master_schema === syncRow.master_schema)` mà **bỏ quên `s.master_binding_id === syncRow.id`** (tương tự `ReconPipelineGrid.tsx:1625`). Hậu quả là khi `mb.master_schema` rỗng `''` còn `syncRow.master_schema` là `'public'` hoặc tên DB, nút gạt hiển thị sai trạng thái thực tế hoặc đọc nhầm binding khác cùng tên bảng!
- Đồng thời tại `transmute_schedule_handler.go:94-96`, `schedNameRe` (`^[a-z_][a-z0-9_]{0,62}$`) kiểm tra cứng `masterSchema` ngay cả khi `req.MasterBindingID > 0` (trong khi dòng `97` đã miễn kiểm tra `masterTable` khi `req.MasterBindingID > 0`), làm các DB có dấu `-` như `payment-service-v2` bị chặn `400 invalid_master_schema`.

### `F4` (`VERIFIED`) — Tiến trình Worker trên Testing lúc `15:40:35` chưa nạp binary `e82ed6e` + Bổ sung dọn `bb.adapters` giữ con trỏ `*gorm.DB` cũ
- **REF:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_poison_pill_split.go:54,265-280`, `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_utils.go:52-77`, `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_config_reload.go:38-55`.
- **Bằng chứng toán học về binary đang chạy (`L065`):** Trong toàn bộ codebase `centralized-data-service`, chỉ duy nhất `transmute_poison_pill_split.go:54` sinh ra `transmute_poison_pill` (nằm sau chốt chặn `if isTransientError(err) { return }` tại `transmute_handler.go:723` và `transmute_poison_pill_split.go:201`). Ở commit `e82ed6e` (`15:03:08` hôm nay), `isTransientError` đã bắt cả `"database is closed"` lẫn `"failed to verify master table existence"`. Việc log lúc `15:40:28–15:40:35` vẫn lọt vào `transmute_sub` và `transmute_poison_pill` chứng minh 100% rằng pod/tiến trình `centralized-data-service` trên môi trường Testing tại thời điểm đó **chưa được rollout lên bản `e82ed6e`** (hoặc vẫn còn replica cũ đang chạy song song trong NATS queue group `"transmute-workers"`, ôm con trỏ `*gorm.DB` đã đóng của `postgrestestingpaymentv2` từ buổi trưa; tương tự trên máy local hiện tại `cdc-cms-service` PID `79300` cũng đang chạy từ `10:56:18` sáng).
- **Khóa kín lỗ hổng cache cuối cùng trong code (`batch_buffer_utils.go:52-77`):** Khi `record.ConnectionRole == "master"`, `bb.adapters` lưu `*SchemaAdapter` bọc con trỏ `*gorm.DB` dưới khóa `"master:" + key`, nhưng `"connection.config.changed"` mới chỉ gọi `FlushSchemaCache()` (xóa map cột) mà chưa xóa entry `"master:" + connKey` khỏi `bb.adapters`.

---

## 2. Thiết Kế Giải Pháp Duy Nhất Tối Ưu Nhất (Ánh Xạ 1-1 `F1..F4`)

Tuân thủ tuyệt đối **Clean Layered Architecture** (`centralized-data-service`), **Hexagonal + CQRS** (`cdc-cms-service`), **Feature/Hook Architecture** (`cdc-cms-web`), bảo toàn **100% Backward Compatibility (`INV-1..INV-5`)**, **0 `if/else` mới** trong `forbidden_ifelse_files`, và đảm bảo **`NOW <= HEAD`** trên toàn bộ file lớn (`>600` dòng):

| Phát hiện | Tệp sửa / tách mới | Thiết kế chi tiết & Cam kết Line Budget (`NOW <= HEAD`) |
| :--- | :--- | :--- |
| **`F1`** | 1. `centralized-data-service/internal/repository/master/master_binding_repo.go`<br/>2. **Tạo mới** `centralized-data-service/internal/handler/master/transmute_realtime_gate.go`<br/>3. `centralized-data-service/internal/handler/master/transmute_handler.go` | - Giữ nguyên 100% hành vi của `ListMasterTargetsByShadowIdentity` và `ListMasterTargetsByShadowTable` cho các luồng `bridge-oplog`, `heal`, `orphan_prune`, `batch-transform` và test cũ.<br/>- Bổ sung `FilterPostIngestEnabledIDs(ctx, bindingIDs)` và `IsPostIngestExplicitlyDisabled(ctx, bindingID)` vào `MasterBindingRepo`.<br/>- Tách `resolveMasterTargets` từ `transmute_handler.go` sang `transmute_realtime_gate.go` (**giúp `transmute_handler.go` giảm từ `793` xuống `~769` dòng, đạt `NOW <= HEAD`!**) kèm 2 cổng bảo vệ:<br/>  1) Tại `handleTransmuteShadowWithOutcome`: khi `isRealtimePostIngestTrigger(req.TriggeredBy)` (`"batch-buffer"`, `"sinkworker"`, `"snapshot"`, `"kafka-consumer-hook"`, `"snapshot-hook"`), chỉ giữ lại các `masterTargets` có `post_ingest` đang `is_enabled = true`. Nếu rỗng $\rightarrow$ trả `nil` để JetStream `Ack()` ngay lập tức, dập tắt 100% bão redelivery.<br/>  2) Tại `runDebouncedGroup`: trước khi mở log `transmute` hay chạm `GetMasterDB`, nếu `triggeredBy` là `"kafka-consumer-hook"`/`"snapshot-hook"` và `IsPostIngestExplicitlyDisabled` trả `true` $\rightarrow$ hoàn tất êm (`t.complete(nil)`), `0` lỗi DB!<br/>  3) Giữ nguyên nhãn `"batch-transform"` trong `switch req.TriggeredBy` (`CP-4`). |
| **`F2`** | 1. `cdc-cms-service/internal/app/commands/scheduler/{create_schedule,toggle_schedule,delete_schedule}.go` & `internal/app/commands/master/toggle_master_active.go`<br/>2. `cdc-cms-service/internal/server/server.go`<br/>3. `centralized-data-service/internal/handler/shadow/batch_buffer_lifecycle.go` & `batch_buffer_fanout_cache.go`<br/>4. `centralized-data-service/internal/server/server_setup_buffers.go` & `server_config_reload.go`<br/>5. `centralized-data-service/internal/sinkworker/worker.go` | - Bổ sung builder tương thích ngược `.WithPublisher(natsPublisher)` cho 4 command handlers (`Create/Toggle/DeleteTransmuteScheduleHandler` & `ToggleMasterActiveHandler`) để bắn tín hiệu NATS Core `"transmute.schedule.changed"` ngay sau khi lưu DB.<br/>- Bổ sung `FlushPostIngestCache()` trên `BatchBuffer` (và gọi luôn trong `FlushSchemaCache()`), đăng ký subscriber `"transmute.schedule.changed"` trong `registerConfigReloadSubscribers` và trong `sinkworker.New` (kèm đồng bộ điều kiện `multi_shadow` cho `SinkWorker`).<br/>- Giới hạn thời gian sống của stale cache khi query lỗi trong `postIngestEntryOnLookupError` ở mức `postIngestFailOpenTTL` (`5s`) thay vì gia hạn vô tận `30s`. |
| **`F3`** | 1. `cdc-cms-service/internal/api/scheduler/transmute_schedule_handler.go`<br/>2. **Tạo mới** `cdc-cms-web/src/components/master/masterScheduleUtils.ts`<br/>3. `cdc-cms-web/src/pages/MasterRegistry.tsx` & `src/components/ReconPipelineGrid.tsx` | - Tại `transmute_schedule_handler.go:94`: chỉ kiểm tra regex `schedNameRe` cho `masterSchema` khi `req.MasterBindingID <= 0` (đồng bộ với `masterTable` ở dòng `97`), cho phép tạo/tắt schedule bình thường với DB có dấu `-` như `payment-service-v2`.<br/>- Tạo helper thuần `matchScheduleForMasterBinding` tại `src/components/master/masterScheduleUtils.ts` ưu tiên khớp tuyệt đối `s.master_binding_id === bindingId`, fallback chuẩn hóa `(schema || 'public') + table` khi `master_binding_id` rỗng; áp dụng tại `MasterRegistry.tsx` (`0` `if/else` mới, giảm dòng `1775 -> ~1771`) và `ReconPipelineGrid.tsx` (giữ `<= 1730` dòng). |
| **`F4`** | `centralized-data-service/internal/handler/shadow/batch_buffer_lifecycle.go`, `internal/server/server_setup_buffers.go`, `internal/server/server_config_reload.go` | - Bổ sung `InvalidateConnectionAdapter(connKey)` xóa `"master:" + connKey` và `"shadow:" + connKey` khỏi `bb.adapters`, gọi ngay trong subscriber `"connection.config.changed"` để không bao giờ giữ lại `*SchemaAdapter` bọc `*gorm.DB` đã đóng.<br/>- Hướng dẫn vận hành rõ ràng: rollout/restart binary mới cho `centralized-data-service` trên Testing và `cdc-cms-service` (PID `79300` trên local). |

---

## 3. Ràng Buộc Thực Thi (`Technical Constraints` từ Lessons)

- **`L065` & `L100`:** Khóa rõ trạng thái tiến trình live vs code trên đĩa; sau khi được `APPROVE`, biên dịch và kiểm tra tiến trình local + chạy kiểm thử thực tế trên PostgreSQL 15 & NATS JetStream thật.
- **`L015` & `L049`:** Đối chiếu nguyên bản từng dòng file trên đĩa; mọi file chạm vào đang `>600` dòng (`transmute_handler.go`, `MasterRegistry.tsx`, `ReconPipelineGrid.tsx`) bắt buộc đạt `NOW <= HEAD`.
- **`L074` & Rule #20:** Không ghi bất kỳ secret/password/URI nhạy cảm nào vào code, test hay tài liệu workspace.
- **`L087` & `L088`:** Đảm bảo khi `post_ingest` tắt, bản tin JetStream `cdc.cmd.transmute-shadow` của luồng realtime được `Ack()` sạch sẽ, không gây kẹt stream hay cô lập nhầm `transmute_poison_pill`.

---

## 4. Code Demo Chi Tiết

### 4.1. `centralized-data-service` — Cổng chắn Realtime Consumer & Debouncer + Xóa Cache tức thì (`F1`, `F2`, `F4`)

```go
// [MỚI] internal/handler/master/transmute_realtime_gate.go
package master

import (
	"context"
	"strings"
	"time"

	repomaster "centralized-data-service/internal/repository/master"
	"centralized-data-service/pkgs/observability"

	"go.uber.org/zap"
)

func isRealtimePostIngestTrigger(triggeredBy string) bool {
	switch strings.TrimSpace(triggeredBy) {
	case "batch-buffer", "sinkworker", "snapshot", "kafka-consumer-hook", "snapshot-hook":
		return true
	default:
		return false
	}
}

func (h *TransmuteHandler) resolveMasterTargets(
	ctx context.Context,
	shadowTable, shadowSchema, shadowConnectionKey, shadowBindingCode string,
) ([]repomaster.MasterTargetIdentity, error) {
	if h.masterBindingRepo == nil {
		return nil, nil
	}
	if shadowBindingCode != "" || shadowSchema != "" || shadowConnectionKey != "" {
		targets, err := h.masterBindingRepo.ListMasterTargetsByShadowIdentity(
			ctx, shadowTable, shadowSchema, shadowConnectionKey, shadowBindingCode,
		)
		if err != nil {
			observability.Ctx(ctx, h.logger).Warn("transmute-shadow: identity-aware master lookup failed",
				zap.String("shadow", shadowTable),
				zap.String("shadow_schema", shadowSchema),
				zap.String("shadow_connection_key", shadowConnectionKey),
				zap.Error(err))
			return nil, err
		}
		return targets, nil
	}
	targets, err := h.masterBindingRepo.ListMasterTargetsByShadowTable(ctx, shadowTable)
	if err != nil {
		observability.Ctx(ctx, h.logger).Warn("transmute-shadow: master lookup failed",
			zap.String("shadow", shadowTable), zap.Error(err))
		return nil, err
	}
	return targets, nil
}

func (h *TransmuteHandler) filterRealtimeMasterTargets(
	ctx context.Context,
	targets []repomaster.MasterTargetIdentity,
) ([]repomaster.MasterTargetIdentity, error) {
	if len(targets) == 0 || h.masterBindingRepo == nil {
		return targets, nil
	}
	ids := make([]int64, 0, len(targets))
	for _, t := range targets {
		if t.ID > 0 {
			ids = append(ids, t.ID)
		}
	}
	if len(ids) == 0 {
		return targets, nil
	}
	enabledMap, err := h.masterBindingRepo.FilterPostIngestEnabledIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	filtered := make([]repomaster.MasterTargetIdentity, 0, len(targets))
	for _, t := range targets {
		if enabledMap[t.ID] {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

func (h *TransmuteHandler) shouldSkipDisabledRealtimeBatch(ctx context.Context, masterBindingID int64, triggeredBy string) bool {
	if masterBindingID <= 0 || h.masterBindingRepo == nil {
		return false
	}
	if triggeredBy != "kafka-consumer-hook" && triggeredBy != "snapshot-hook" {
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	disabled, err := h.masterBindingRepo.IsPostIngestExplicitlyDisabled(checkCtx, masterBindingID)
	return err == nil && disabled
}
```

```go
// internal/repository/master/master_binding_repo.go (bổ sung 2 hàm, giữ nguyên 100% hàm cũ)
func (r *MasterBindingRepo) FilterPostIngestEnabledIDs(ctx context.Context, bindingIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(bindingIDs))
	if len(bindingIDs) == 0 || r == nil || r.db == nil {
		return out, nil
	}
	var enabledIDs []int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT ts.master_binding_id
		  FROM cdc_system.transmute_schedule ts
		  JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
		 WHERE ts.master_binding_id IN ?
		   AND ts.mode = 'post_ingest'
		   AND ts.is_enabled = true
		   AND mb.is_active = true
		   AND mb.schema_status = 'approved'
	`, bindingIDs).Scan(&enabledIDs).Error
	if err != nil {
		return nil, err
	}
	for _, id := range enabledIDs {
		out[id] = true
	}
	return out, nil
}

func (r *MasterBindingRepo) IsPostIngestExplicitlyDisabled(ctx context.Context, bindingID int64) (bool, error) {
	if bindingID <= 0 || r == nil || r.db == nil {
		return false, nil
	}
	var disabled bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			  FROM cdc_system.transmute_schedule ts
			 WHERE ts.master_binding_id = ?
			   AND ts.mode = 'post_ingest'
			   AND ts.is_enabled = false
		)
	`, bindingID).Scan(&disabled).Error
	return disabled, err
}
```

```go
// internal/handler/master/transmute_handler.go (tại handleTransmuteShadowWithOutcome & runDebouncedGroup)
	if isRealtimePostIngestTrigger(req.TriggeredBy) {
		masterTargets, err = h.filterRealtimeMasterTargets(lookupCtx, masterTargets)
		if err != nil {
			return fmt.Errorf("transmute-shadow post_ingest filter failed for %s: %w", req.ShadowTable, err)
		}
	}
	if len(masterTargets) == 0 {
		return nil // TransmuteConsumerPool gọi msg.Ack() ngay, dừng hẳn redelivery!
	}
	hookTrigger := "kafka-consumer-hook"
	switch req.TriggeredBy {
	case "snapshot", "snapshot-hook":
		hookTrigger = "snapshot-hook"
	case "bridge-oplog":
		hookTrigger = "bridge-oplog"
	case "batch-transform":
		hookTrigger = "batch-transform"
	case "heal", "recon-heal", "execute-heal":
		hookTrigger = req.TriggeredBy
	}

// ... và đầu hàm runDebouncedGroup trước khi mở logEntry / chạm masterDB:
	if h.shouldSkipDisabledRealtimeBatch(ctx, masterBindingID, triggeredBy) {
		for _, t := range batch {
			h.reply(t.Msg, TransmuteResponse{CorrelationID: t.Req.CorrelationID})
			t.complete(nil)
		}
		return
	}
```

```go
// internal/handler/shadow/batch_buffer_lifecycle.go & batch_buffer_fanout_cache.go (F2 + F4)
func (bb *BatchBuffer) FlushPostIngestCache() {
	bb.piCache.Range(func(k, _ any) bool {
		bb.piCache.Delete(k)
		return true
	})
}

func (bb *BatchBuffer) InvalidateConnectionAdapter(connKey string) {
	connKey = strings.TrimSpace(connKey)
	if connKey == "" {
		return
	}
	for _, prefix := range []string{"master:", "shadow:"} {
		if v, ok := bb.adapters.LoadAndDelete(prefix + connKey); ok {
			if adapter, ok := v.(*serviceshadow.SchemaAdapter); ok && adapter != nil {
				adapter.FlushAll()
			}
		}
	}
}

func (bb *BatchBuffer) FlushSchemaCache() {
	bb.adapters.Range(func(_, v any) bool {
		v.(*serviceshadow.SchemaAdapter).FlushAll()
		return true
	})
	bb.schemaAdapter.FlushAll()
	bb.FlushPostIngestCache()
}

// Trong postIngestEntryOnLookupError (batch_buffer_fanout_cache.go):
	if hasStale {
		stale.at = now.Add(postIngestFailOpenTTL - postIngestCacheTTL)
		bb.piCache.Store(key, stale)
		return stale
	}
```

```go
// internal/server/server_config_reload.go (bổ sung lắng nghe transmute.schedule.changed & dọn adapter master)
	natsConn.Subscribe("transmute.schedule.changed", func(_ *nats.Msg) {
		flushBatchBufferPostIngestCaches(cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)
	})

	natsConn.Subscribe("connection.config.changed", func(msg *nats.Msg) {
		connKey := strings.TrimSpace(string(msg.Data))
		if connKey == "" {
			return
		}
		connectionManager.InvalidatePool(connKey)
		if reconCore != nil {
			reconCore.InvalidateMasterAgent(connKey)
		}
		if chunkEngine != nil {
			chunkEngine.InvalidateMasterAgent(connKey)
		}
		invalidateBatchBufferConnectionAdapters(connKey, cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)
		flushBatchBufferSchemaCaches(cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)
		if logger != nil {
			logger.Info("connection pool invalidated via NATS signal",
				zap.String("connection_code", connKey))
		}
	})
```

### 4.2. `cdc-cms-service` — Phát NATS `"transmute.schedule.changed"` & Mở khóa `master_schema` khi có `master_binding_id` (`F2`, `F3`)

```go
// internal/app/commands/scheduler/{create_schedule,toggle_schedule,delete_schedule}.go & master/toggle_master_active.go
// Giữ nguyên hàm khởi tạo New*Handler(repo) để tương thích ngược 100%, bổ sung builder WithPublisher:
func (h *CreateTransmuteScheduleHandler) WithPublisher(pub ports.Publisher) *CreateTransmuteScheduleHandler {
	h.pub = pub
	return h
}
// Sau khi lưu/toggle/xóa schedule thành công:
if h.pub != nil {
	_ = h.pub.Publish(ctx, "transmute.schedule.changed", body)
}
```

```go
// internal/api/scheduler/transmute_schedule_handler.go:94-99
	if req.MasterBindingID <= 0 && masterSchema != "" && !schedNameRe.MatchString(masterSchema) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_master_schema"})
	}
	if req.MasterBindingID <= 0 && !schedNameRe.MatchString(masterTable) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_master_table"})
	}
```

### 4.3. `cdc-cms-web` — Khớp Schedule chuẩn xác theo `master_binding_id` (`F3`)

```ts
// [MỚI] src/components/master/masterScheduleUtils.ts
export interface MasterScheduleMatchCandidate {
  id?: number;
  mode: string;
  master_binding_id?: number | null;
  master_table?: string | null;
  master_schema?: string | null;
  is_enabled?: boolean;
  cron_expr?: string | null;
}

export function matchScheduleForMasterBinding<T extends MasterScheduleMatchCandidate>(
  schedules: T[] | undefined | null,
  bindingId: number | undefined | null,
  masterName: string,
  masterSchema?: string | null,
): T[] {
  const list = schedules || [];
  if (bindingId && bindingId > 0) {
    const byId = list.filter((s) => s.master_binding_id === bindingId);
    if (byId.length > 0 || list.some((s) => !!s.master_binding_id)) {
      return byId;
    }
  }
  const normSchema = (masterSchema || '').trim() || 'public';
  return list.filter((s) => {
    if (s.master_table !== masterName) return false;
    const sSchema = (s.master_schema || '').trim() || 'public';
    return !masterSchema || sSchema === normSchema;
  });
}
```

```tsx
// src/pages/MasterRegistry.tsx (trong syncMut & useEffect mở Sync Modal — 0 if/else mới!)
const rows = matchScheduleForMasterBinding(list.data.data, syncRow.id, syncRow.master_name, syncRow.master_schema);
```
