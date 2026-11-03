---
tier: T2
workspace: audit-duplicate-source-ts-upsert-2026-10-29
suffix: 2026_10_29
mode: standard
created_at: 2026-10-29T17:43:30+07:00
---

# 09. Technical Solution: Khắc phục Triệt để Lỗi `column "_source_ts" specified more than once (SQLSTATE 42701)` & Ô nhiễm Lỗi Chéo Bảng Khi Chạy Snapshot/Flush Shadow

## 1. Bảng Ánh xạ 1-1 Giữa Phát hiện Audit (`F1..F4`) và Giải pháp Kỹ thuật (`S-F1..S-F4`)

| Mã Phát hiện | Nguyên nhân Gốc rễ (`VERIFIED` `file:line` & DB) | Giải pháp Kỹ thuật Tối ưu Duy nhất (`S-F1..S-F4`) | File Dự kiến Sửa đổi / Tạo mới |
| :--- | :--- | :--- | :--- |
| **`F1`** | **Rò rỉ cột hệ thống (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`) vào `mapping_rule_v2` khi Scan Fields SQL và bị `UpdateRegistry` auto-approve hàng loạt:**<br>- Bảng nguồn `public.cdc_transaction_history_payload` (`source_object_id = 219`, `shadow_binding_id = 224`) có sẵn 4 cột metadata `_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`.<br>- `processDiscoveryRows` (`discover_handler_utils.go:28-30`) chỉ lọc 3 cột (`_raw_data`, `_synced_at`, `_source`), còn `scanFieldsSQLSource` (`discover_handler_sql.go:34-47`) và `inferPGCols` (`discovery_utils_sql.go:36-63`) không lọc cột hệ thống, làm sinh ra 4 rule `id = 922..925` (`pending`).<br>- Khi bật active bảng lúc `17:21:49`, `sourceRepoGorm.UpdateRegistry` (`source_repo_gorm.go:383-393`) auto-approve toàn bộ rule `status != 'approved'` mà không loại trừ `dbutil.IsSystemColumn`, khiến 4 rule `922..925` chuyển sang `approved, is_active = true`. | **`S-F1`:**<br>1. Tạo helper đơn trách nhiệm `IsReservedShadowSystemColumn(col string) bool` tại `centralized-data-service/internal/service/shadow/system_columns.go` (chứa đủ 11 cột hệ thống CDC/Shadow) và chặn ngay tại `processDiscoveryRows`, `scanFieldsSQLSource`, `inferPGCols`, `inferMySQLCols`.<br>2. Trong `cdc-cms-service`, bổ sung điều kiện `AND LOWER(source_field) NOT IN (...) AND LOWER(target_column) NOT IN (...)` vào `sourceRepoGorm.UpdateRegistry` (`source_repo_gorm.go:384-389`) và `syncRulesFromAllShadowsTx` (`master_multi_shadow_repo_gorm.go:155-159`). | - `centralized-data-service/internal/service/shadow/system_columns.go` *(new)*<br>- `centralized-data-service/internal/handler/source/discover_handler_utils.go`<br>- `centralized-data-service/internal/handler/source/discover_handler_sql.go`<br>- `centralized-data-service/internal/handler/source/discovery_utils_sql.go`<br>- `cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go`<br>- `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go` |
| **`F2`** | **Ghép trùng cột `"_source_ts"` trong `allCols` của `SchemaAdapter` (`BuildBatchUpsertSQLsInSchema` & `BuildUpsertSQLInSchema`) gây `SQLSTATE 42701` và làm hỏng cả fallback single-row (`fallback persisted 0 rows`):**<br>- `BuildBatchUpsertSQLsInSchema` (`schema_adapter.go:570-582`) và `BuildUpsertSQLInSchema` (`schema_adapter.go:352-374`) chỉ khử trùng ở nhánh `ON CONFLICT DO UPDATE` (`seenUpdateCols`), nhưng **không khử trùng `allCols` (`INSERT INTO`)** giữa `dataCols` (từ `r.MappedData`) và `metaCols := getMetadataInsertCols(schema, pkField)` (`schema_adapter.go:674-698`).<br>- Khi rule `923 (_source_ts)` active, `"_source_ts"` xuất hiện 2 lần trong `INSERT INTO`, gây lỗi `SQLSTATE 42701` ở cả batch lẫn single-row fallback (`batch_buffer.go:801`).<br>- `BatchTransformHandler` (`batch_transform_handler.go:198-242`) cũng chưa lọc cột hệ thống trước khi gán `_updated_at = NOW()`. | **`S-F2`:**<br>1. Trong `DynamicMapper.MapData` (`dynamic_mapper.go:86`), bỏ qua mọi mapping rule có `IsReservedShadowSystemColumn(rule.TargetColumn)`.<br>2. Trong `SchemaAdapter.BuildUpsertSQLInSchema` (`schema_adapter.go:352-374`) và `BuildBatchUpsertSQLsInSchema` (`schema_adapter.go:570-582`), lọc `dataCols` qua helper `FilterBusinessDataColumns` (`system_columns.go`) để loại bỏ `pkField` và mọi cột thuộc `getMetadataInsertCols` / cột hệ thống trước khi dựng `allCols` và `values`.<br>3. Trong `BatchTransformHandler` (`batch_transform_handler.go:198`), bỏ qua `shadow.IsReservedShadowSystemColumn(col)`. | - `centralized-data-service/internal/service/shadow/system_columns.go`<br>- `centralized-data-service/internal/service/shadow/dynamic_mapper.go`<br>- `centralized-data-service/internal/service/shadow/schema_adapter.go`<br>- `centralized-data-service/internal/handler/shadow/batch_transform_handler.go` |
| **`F3`** | **Ô nhiễm lỗi chéo bảng (Cross-Table Error Contamination) qua `snapshotBatchBuffer` dùng chung khiến `transaction_history` chết oan ở mốc `15,000` dòng:**<br>- `shadow_binding_id = 222` (`transaction_history`) hoàn toàn không có rule `_source_ts` và thực tế đã ghi thành công `20,012` dòng vào `shadow_testcoretrans.transaction_history`.<br>- Tuy nhiên `SnapshotRunner.runSnapshot` (`snapshot_runner_handler.go:855, 919`) gọi `r.eventHandler.FlushBatchBuffer(ctx)` trên `snapshotBatchBuffer` dùng chung (`server_setup.go:166`), làm xả luôn các bản ghi đang chờ của `cdc_transaction_history_payload` và trả lỗi `SQLSTATE 42701` của bảng kia về cho job `transaction_history` ở cuối batch thứ 4 (trước khi cập nhật mốc `20,000`, nên dừng ở `15,000`). | **`S-F3`:**<br>- Tại `SnapshotRunner.runSnapshot` (`snapshot_runner_handler.go:855` và `919`), đổi `r.eventHandler.FlushBatchBuffer(ctx)` thành `r.eventHandler.FlushTableBatchBuffer(ctx, p.TargetTable)` (sử dụng `FlushTableWithContext` đã có tại `event_handler.go:106` & `batch_buffer.go:271`) để cô lập phạm vi flush theo đúng bảng đích của từng job Snapshot. | - `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go` |
| **`F4`** | **Nuốt lỗi ngầm trong `BatchBuffer.timerLoop` (`_, _ = bb.Flush()`) kết hợp `maxSize = 1000` khiến `cdc_transaction_history_payload` báo ảo `30,000` dòng (`11.98%`) dù `0` dòng được ghi vào DB:**<br>- Mỗi batch của `SnapshotRunner` đọc `5,000` dòng, nhưng `snapshotBatchBuffer` có `maxSize = 1000` (`server_setup.go:162-166`).<br>- Ở dòng thứ 5,000 của 6 batch đầu tiên (`30,000` dòng), `AddAll` (`batch_buffer.go:204-209`) kích hoạt `bb.flushCh` khiến `timerLoop()` (`batch_buffer.go:259-260`) rút sạch buffer ra xả, nổ `SQLSTATE 42701`, đẩy vào `failed_sync_logs` rồi **bỏ qua lỗi (`_, _ = bb.Flush()`)**.<br>- Khi `runSnapshot` gọi `FlushBatchBuffer(ctx)` ở dòng `855`, buffer đã rỗng (`len(bb.records) == 0`) nên trả về `0, nil`, làm cộng dồn tiến độ giả lên `30,000` dòng. | **`S-F4`:**<br>- Trong `BatchBuffer` (`batch_buffer.go`), bổ sung `asyncFlushErrs map[string]error` ghi nhận lỗi từ các lần flush nền (`timerLoop`) theo từng bảng (`targetTable` / `groupKey`).<br>- Khi `FlushTableWithContext` hoặc `FlushWithContext` được gọi đồng bộ bởi `SnapshotRunner`, kiểm tra và trả về ngay lỗi async tồn đọng của bảng tương ứng (kèm xóa cờ lỗi đã đọc) để `SnapshotRunner` trip circuit breaker ngay từ Batch 1 thay vì báo tiến độ ảo. | - `centralized-data-service/internal/handler/shadow/batch_buffer.go` |

---

## 2. Ràng buộc Thực thi (Technical Constraints)
- **Single-Responsibility (`L008`):** Không nhồi thêm logic kiểm tra tên cột hệ thống rải rác vào `schema_adapter.go` (đang có 1,158 dòng). Tách toàn bộ hằng số và hàm lọc cột hệ thống (`IsReservedShadowSystemColumn`, `FilterBusinessDataColumns`) sang file độc lập `centralized-data-service/internal/service/shadow/system_columns.go` (~60 dòng).
- **Zero Runtime DDL & No Cheat DB (`L031`):** Không chạy DDL trong worker; việc dọn 4 rule rác `id IN (922, 923, 924, 925)` trong `cdc_system.mapping_rule_v2` chỉ thực hiện sau khi User `APPROVE` thông qua câu lệnh chuẩn hóa trạng thái rule và re-run snapshot.
- **Backward Compatibility:** Giữ nguyên hành vi cho trường hợp `pkField` là cột nghiệp vụ hoặc `_source_id`; đảm bảo số lượng cột trong `allCols` luôn khớp 1-1 với số lượng placeholder `($1, $2, ...)` và mảng `values` trong cả `BuildUpsertSQLInSchema` lẫn `BuildBatchUpsertSQLsInSchema`.

---

## 3. Code Demo Chi tiết (Propose-Only — Chờ User `APPROVE`)

### 3.1. `[NEW]` `centralized-data-service/internal/service/shadow/system_columns.go` (Giải quyết `F1` & `F2`)
```go
package shadow

import "strings"

// reservedShadowSystemColumns liệt kê đầy đủ các cột metadata hệ thống của bảng Shadow/CDC.
// Các cột này do SchemaAdapter và hệ thống tự động quản lý, tuyệt đối không được phép
// đi vào luồng business mapping (dataCols) để tránh lỗi SQLSTATE 42701 (duplicate column).
var reservedShadowSystemColumns = map[string]struct{}{
	"_gpay_id":    {},
	"_source_id":  {},
	"_raw_data":   {},
	"_source":     {},
	"_source_ts":  {},
	"_synced_at":  {},
	"_version":    {},
	"_hash":       {},
	"_deleted":    {},
	"_created_at": {},
	"_updated_at": {},
}

// IsReservedShadowSystemColumn kiểm tra một tên cột có phải là cột metadata hệ thống hay không.
func IsReservedShadowSystemColumn(col string) bool {
	clean := strings.ToLower(strings.Trim(strings.TrimSpace(col), `"`))
	_, exists := reservedShadowSystemColumns[clean]
	return exists
}

// ShouldSkipDataColumn trả về true nếu cột trong MappedData trùng với pkField
// hoặc là cột metadata hệ thống sẽ được getMetadataInsertCols quản lý
// (ngoại trừ "_deleted" vốn được EventHandler gán tường minh mappedColumns["_deleted"] = isDelete).
func ShouldSkipDataColumn(col string, pkField string) bool {
	clean := strings.ToLower(strings.Trim(strings.TrimSpace(col), `"`))
	if clean == "" {
		return true
	}
	if strings.EqualFold(clean, strings.TrimSpace(pkField)) {
		return true
	}
	if clean == "_deleted" {
		return false
	}
	return IsReservedShadowSystemColumn(clean)
}
```

### 3.2. `centralized-data-service/internal/service/shadow/schema_adapter.go` & `dynamic_mapper.go` (Giải quyết `F2`)
```go
// File: centralized-data-service/internal/service/shadow/dynamic_mapper.go (lines 85-90)
	for _, rule := range rules {
		if !rule.IsActive || IsReservedShadowSystemColumn(rule.TargetColumn) {
			continue
		}

// File: centralized-data-service/internal/service/shadow/schema_adapter.go
// 1) Trong BuildUpsertSQLInSchema (lines 352-372):
	var dataCols []string
	for col := range record.MappedData {
		if ShouldSkipDataColumn(col, pkField) {
			continue
		}
		if _, exists := schema.Columns[col]; exists {
			dataCols = append(dataCols, col)
		}
	}
	sort.Strings(dataCols)

// 2) Trong BuildBatchUpsertSQLsInSchema (lines 558-574):
	colsMap := make(map[string]bool)
	for _, r := range records {
		for col := range r.MappedData {
			if ShouldSkipDataColumn(col, pkField) {
				continue
			}
			if _, exists := schema.Columns[col]; exists {
				colsMap[col] = true
			}
		}
	}

	var dataCols []string
	for col := range colsMap {
		dataCols = append(dataCols, col)
	}
	sort.Strings(dataCols)
```

### 3.3. `centralized-data-service/internal/handler/source/discover_handler_utils.go`, `discover_handler_sql.go`, `discovery_utils_sql.go` & `cdc-cms-service` (Giải quyết `F1`)
```go
// File: centralized-data-service/internal/handler/source/discover_handler_utils.go (lines 27-31)
		for k, v := range targetMap {
			if shadow.IsReservedShadowSystemColumn(k) {
				continue
			}

// File: centralized-data-service/internal/handler/source/discovery_utils_sql.go (lines 52-57)
		if shadow.IsReservedShadowSystemColumn(colName) {
			continue
		}

// File: cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go (lines 383-393)
				res := tx.Table("cdc_system.mapping_rule_v2").
					Where(`source_object_id = ? AND shadow_binding_id = ? AND status != ?
						AND LOWER(source_field) NOT IN ('_gpay_id','_source_id','_raw_data','_source','_source_ts','_synced_at','_version','_hash','_deleted','_created_at','_updated_at')
						AND LOWER(target_column) NOT IN ('_gpay_id','_source_id','_raw_data','_source','_source_ts','_synced_at','_version','_hash','_deleted','_created_at','_updated_at')`,
						sourceObjectID, shadowBindingID, "approved").
					Updates(map[string]interface{}{
						"status":    "approved",
						"is_active": true,
					})
```

### 3.4. `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go` & `internal/handler/shadow/batch_buffer.go` (Giải quyết `F3` & `F4`)
```go
// File: centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go
// Tại cuối mỗi batch (line 855) và pha drain cuối (line 919), cô lập flush theo đúng bảng đích p.TargetTable:
		flushDLQ()

		_, flushErr := r.eventHandler.FlushTableBatchBuffer(ctx, p.TargetTable)
		if flushErr != nil {
			return tripBreaker(fmt.Sprintf("flush failed: %v", flushErr))
		}

// File: centralized-data-service/internal/handler/shadow/batch_buffer.go
// Ghi nhận lỗi khi timerLoop() xả nền thất bại để FlushTableWithContext / FlushWithContext không báo thành công giả (0, nil):
func (bb *BatchBuffer) recordAsyncFlushError(byTable map[string][]*shadow.UpsertRecord, err error) {
	if err == nil {
		return
	}
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if bb.asyncFlushErrs == nil {
		bb.asyncFlushErrs = make(map[string]error)
	}
	for k, recs := range byTable {
		bb.asyncFlushErrs[k] = err
		for _, r := range recs {
			if r != nil && r.TableName != "" {
				bb.asyncFlushErrs[strings.ToLower(strings.TrimSpace(r.TableName))] = err
			}
		}
	}
}
```

---

## 4. Vòng Lặp Tự Chữa Lành (`S5 -> S3 -> S4 -> S5` Self-Healing Update cho Phát hiện Phản biện `F5..F8`)

Khi kiểm toán phản biện từng dòng `git diff`, đo `wc -l` trước/sau và rà soát phân tầng kiến trúc (`L008`, `L021`, `L032`), phát hiện 4 điểm cần tự chữa lành triệt để (`F5..F8`):

| Mã Phát hiện | Nội dung Phản biện (`file:line`) | Giải pháp Tự chữa lành (`S-F5..S-F8`) |
| :--- | :--- | :--- |
| **`F5`** | Trong `SchemaAdapter.BuildUpsertSQLInSchema` (`schema_adapter.go:378`), vòng lặp dựng `updateSets` (`ON CONFLICT DO UPDATE`) vẫn dùng `if col == pkField` thay vì `ShouldSkipDataColumn(col, pkField)`. Nếu `mappedData` chứa `"_updated_at"` hoặc `"_source_ts"`, cột này đã bị loại khỏi `INSERT INTO` (dòng 357) nhưng lại lọt vào `updateSets` thành `"_updated_at" = EXCLUDED."_updated_at"` và ghi đè `seenUpdateCols["_updated_at"] = true`, làm mất `"_updated_at" = NOW()` của `buildMetadataUpdateSets`. | Tạo helper `ShouldIncludeDataColumn(schema *TableSchema, col, pkField string) bool` và `collectBatchDataColumns` trong `internal/service/shadow/system_columns.go` và áp dụng đồng nhất tại cả `BuildUpsertSQLInSchema` (dòng `357` và `378`) lẫn `BuildBatchUpsertSQLsInSchema` (dòng `558`). |
| **`F6`** | Trong `BatchBuffer` (`batch_buffer.go:343-362`), hàm `matchTargetTable` dùng `strings.Trim(rFQN, "\"")` không xóa được dấu `"` ở giữa chuỗi `"\"shadow_testcoretrans\".\"transaction_history\""` (do `qualifiedShadowTable` sinh ra tại `event_handler.go:581`) và chưa kiểm tra `r.SchemaName`. Đồng thời `flushSnapshotTables` (`snapshot_runner_utils.go:203-236`) truyền tên bảng trần `targetTable` khi `scopedBinding != nil`, có nguy cơ xả nhầm bảng trùng tên ở schema khác (`L032`). | Chuẩn hóa `normalizeTableRef` & `matchTargetTable` (chuyển sang `batch_buffer_async_err.go`) dùng `strings.ReplaceAll(..., `"`, "")` và so khớp đầy đủ `(schema, table)` từ `r.SchemaName`, `r.TableName`, `r.PhysicalTableFQN`. Trong `flushSnapshotTables`, định danh bảng đích theo `scopedBinding.ShadowSchema + "." + scopedBinding.ShadowTable`. |
| **`F7`** | **Vi phạm `L008` (Line-Budget Guard `NOW <= BEFORE` trên 5 file lớn `> 500` dòng & Đặt lệch tầng Domain/Model):**<br>- 5 file `> 500` dòng bị tăng số dòng so với `HEAD`: `schema_adapter.go` (`1158 -> 1161`), `batch_transform_handler.go` (`642 -> 643`), `scan_service.go` (`790 -> 792`), `snapshot_runner_handler.go` (`1022 -> 1024`), `batch_buffer.go` (`968 -> 992`).<br>- Danh sách 11 cột hệ thống chưa đặt ở tầng Domain/Model gốc mà đặt ở `service/shadow` và bị lặp lại mảng literal trong `scan_service.go`, `source_repo_gorm.go`, `master_multi_shadow_repo_gorm.go` (thiếu `LOWER(TRIM(...))` ở `master_multi_shadow_repo_gorm.go:157`). | 1. Đặt danh mục 11 cột hệ thống tại đúng tầng Domain/Model: `[NEW] centralized-data-service/internal/model/shadow/system_columns.go` và `[NEW] cdc-cms-service/internal/domain/shadow/system_columns.go`.<br>2. Tách `[NEW] centralized-data-service/internal/service/source/scan_field_filter.go` và `[NEW] centralized-data-service/internal/handler/shadow/batch_transform_helpers.go`, chuyển `matchTargetTable` sang `batch_buffer_async_err.go` và `resolveSnapshotRouteTargetTable` sang `snapshot_runner_utils.go` để cả 5 file lớn đều giảm dòng (`NOW <= BEFORE`).<br>3. Trong `source_repo_gorm.go` và `master_multi_shadow_repo_gorm.go`, dùng `LOWER(TRIM(...)) NOT IN (?)` với `domainshadow.ReservedShadowSystemColumns()`. |
| **`F8`** | **Báo cáo lệch chi tiết cài đặt trong `audit_report_2026_10_29.md` (`L021`):** Mục `(b)` ghi `asyncFlushErrors map[string]error` và `bb.asyncErrMu`, trong khi code thật dùng `asyncFlushErrs []asyncTableFlushError` khóa bởi `bb.mu`; mục `(a)` ghi `LOWER(TRIM(target_column))` trong khi code thật thiếu `TRIM`. | Đồng bộ code dùng `LOWER(TRIM(...))` và cập nhật lại `audit_report_2026_10_29.md` khớp 100% từng biến, từng hàm và số dòng `wc -l` thực tế trên đĩa. |


---

## 5. Round-3 Re-Audit (2026-10-30) — Phương án Đề xuất Duy nhất (PROPOSE-ONLY, chờ `APPROVE`)

> Trạng thái: **CHƯA sửa source code**. Round 3 chỉ đọc code, chạy test/build và kiểm chứng sinh SQL; code fix bên dưới chờ User `APPROVE`.

| Mã | Phát hiện (VERIFIED theo code, tần suất runtime = HYPOTHESIS) | Phương án tối ưu duy nhất |
| :--- | :--- | :--- |
| **`C1`** | `snapshotEventHandler` (buffer `snapshotBatchBuffer`) được dùng chung bởi `SnapshotRunner` (`server_setup.go:428-430`) và `ExecuteHealHandler` (`server_setup.go:476-480`). `popAsyncFlushErrorLocked` xoá lỗi khi đã lấy ra (`batch_buffer_async_err.go:45-62`) ⇒ một lần heal flush trên cùng bảng có thể "lấy mất" lỗi nền của job snapshot (F4 tái xuất hiện trong race này). | Tách buffer riêng cho heal: thêm `healBatchBuffer` + `healEventHandler` bằng helper mới `server_setup_buffers.go` (`newIsolatedSnapshotBuffer(...)`) và dùng helper này cho cả snapshot lẫn heal để `server_setup.go` (596 dòng) đạt `NOW <= BEFORE`. |
| **`C2`** | Các nhánh `return` sớm trong `runSnapshot` (`snapshot_runner_handler.go:686, 694, 734, 741, 811, 847`) thoát trước `flushSnapshotTables` (`:855`, `:919`). Bản ghi còn lại bị `timerLoop` xả nền sau đó; nếu lỗi thì lỗi nằm trong `asyncFlushErrs` cho tới khi job **kế tiếp** của cùng bảng lấy ra ⇒ job mới khoẻ có thể bị dừng oan ở lần flush đầu. | Thêm `DiscardAsyncFlushErrors(targetTable)` trong `batch_buffer_async_err.go` + hàm chuyển tiếp ở `event_handler_flush.go` (file mới, không làm phình `event_handler.go` 671 dòng) + gọi một lần lúc bắt đầu job qua `r.resetSnapshotTableErrors(...)` trong `snapshot_runner_utils.go`; `snapshot_runner_handler.go` giữ `NOW <= BEFORE` bằng cách gộp lời gọi vào helper khởi tạo có sẵn. |
| **`C3`** | `DynamicMapper.MapData` (`dynamic_mapper.go:86`) và `shouldSkipTransformRule` (`batch_transform_helpers.go:10-12`) chỉ kiểm tra `TargetColumn`. | **Không sửa code** (map `_source_ts` của nguồn sang cột nghiệp vụ là hợp lệ; lỗi 42701 chỉ xảy ra theo cột đích). Chỉ đính chính báo cáo. |
| **`C4`** | `TestFlushTableWithContext_IsolatesAsyncFlushErrorsPerTable` dựng buffer với `nil` DB và tự gọi `recordAsyncFlushError` ⇒ không đi qua `timerLoop → flushFromBackground`. | **Không viết integration test có ghi DB** (User cấm can thiệp DB). Kiểm chứng F4 bằng trace runtime phối hợp với User (mục 5.2). |
| **`C5`** | `master_multi_shadow_repo_gorm.go:58-59` vẫn hardcode 11 cột (file chưa track, thuộc task multi-shadow khác). | Thay bằng `NOT IN (?)` + `domainShadow.ReservedShadowSystemColumns()` (cùng file đã dùng ở `:148`). Cần User xác nhận vì file thuộc task khác. |

### 5.1 Code Demo (chưa áp dụng)
```go
// internal/handler/shadow/batch_buffer_async_err.go  (+10 dòng, file SRP)
// DiscardAsyncFlushErrors xoá lỗi nền cũ của targetTable khi một job mới bắt đầu (C2).
func (bb *BatchBuffer) DiscardAsyncFlushErrors(targetTable string) {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	_ = bb.popAsyncFlushErrorLocked(targetTable) // chỉ thao tác slice trong RAM, không I/O khi giữ lock (L024)
}

// internal/handler/orchestration/snapshot_runner_utils.go
func (r *SnapshotRunner) resetSnapshotTableErrors(targetTable string, sb *shadow.ShadowBinding) {
	if d, ok := r.eventHandler.(interface{ DiscardTableAsyncErrors(string) }); ok {
		d.DiscardTableAsyncErrors(qualifyShadowBindingTable(sb, targetTable))
	}
}

// internal/server/server_setup_buffers.go  (file mới, C1)
func newIsolatedSnapshotBuffer(size int, db *gorm.DB, sa *shadow.SchemaAdapter, cm *connection.Manager,
	masking *governance.MaskingService, nc *nats.Conn, js nats.JetStreamContext,
	fsl *repository.FailedSyncLogRepo, logger *zap.Logger) *handlershadow.BatchBuffer {
	bb := handlershadow.NewBatchBuffer(size, 5*time.Second, db, sa, logger)
	bb.SetConnectionManager(cm); bb.SetMaskingService(masking)
	bb.SetNatsConn(nc); bb.SetJetStream(js); bb.SetFailedSyncLogRepo(fsl)
	bb.SetSnapshotMode(true)
	return bb
}
// server_setup.go: snapshotBatchBuffer := newIsolatedSnapshotBuffer(...)
//                  healBatchBuffer     := newIsolatedSnapshotBuffer(...)
//                  healEventHandler := handlershadow.NewEventHandler(..., healBatchBuffer, logger)
//                  executeHealHandler...WithEventHandler(healEventHandler)
```
> Kiểu dữ liệu tham số của `newIsolatedSnapshotBuffer` sẽ khớp đúng các type đang dùng tại `server_setup.go:152-174` khi thực thi S4.

### 5.2 Kiểm chứng
- Unit: test `DiscardAsyncFlushErrors` (in-memory, chỉ chứng minh logic slice — **không** suy ra runtime).
- Runtime (phối hợp User, không can thiệp DB): sau khi User restart worker + xoá/scan lại, đối chiếu kết quả vận hành thực tế.

---

## 6. S2–S3 Giải Pháp cho Lỗi `"Cặp (source DB + source table + target table) đã được đăng ký rồi"` Khi Xóa Tạo Lại (`F9`–`F11`, Propose-Only Chờ `APPROVE`)

### 6.1 Bảng Ánh xạ 1-1 Nguyên nhân Gốc rễ (`F9`–`F11`) $\leftrightarrow$ Giải pháp Duy nhất (`S-F9`–`S-F11`)

| Mã | Nguyên nhân gốc rễ (`100% VERIFIED` qua Static Code) | Giải pháp tối ưu duy nhất (Propose-Only, chờ `APPROVE`) |
| :--- | :--- | :--- |
| **`F9`** | **Câu lệnh tự dọn V1 `cdc_table_registry` mồ côi trong `Register` (`cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go:260-268`) — vốn được viết từ commit `1106319` (`20/07/2026`) để fix lỗi xóa tạo lại — bị hỏng trong môi trường Multi-Connector (sau Migration `054/055/056`):**<br>Hiện tại `Register` chạy:<br>`DELETE FROM cdc_system.cdc_table_registry WHERE target_table = ? AND NOT EXISTS (SELECT 1 FROM cdc_system.shadow_binding WHERE shadow_table = ?)`<br>- Subquery `NOT EXISTS` kiểm tra `shadow_table` trần trên **toàn cục mọi connector** mà không lọc theo `source_object_registry` / `source_connection_id`. Chỉ cần một connector khác (ví dụ `devcoretrans` song song với `testcoretrans`) đang có `shadow_binding` trùng tên bảng `transaction_history`, mệnh đề `NOT EXISTS` lập tức trả về `FALSE` -> bản ghi `cdc_table_registry` mồ côi của `testcoretrans` **không bao giờ được dọn**, khiến `tx.Create(entry)` ở dòng `273` nổ lỗi `23505 (cdc_table_registry_conn_source_db_table_target_key)` (`"Cặp (source DB + source table + target table) đã được đăng ký rồi."`)!<br>- Ngược lại, nếu không có `shadow_binding` nào trùng tên, `WHERE target_table = ?` lại xóa lan sang `cdc_table_registry` của các connector khác vì thiếu điều kiện `(source_connection_id, source_db, source_table)`. | **`S-F9` (`source_repo_gorm.go:260-272`):** Cô lập câu lệnh dọn `cdc_table_registry` mồ côi theo đúng bộ khóa 4 thành phần `(source_connection_id, source_db, source_table, target_table)` của `entry`, và trong `NOT EXISTS` phải `JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id` khớp đúng `(source_connection_id, source_db, source_table, target_table)`. |
| **`F10`** | **`DeleteShadowBindingHandler` (`cdc-cms-service/internal/app/commands/shadow/delete_shadow_binding.go:164-173`) chỉ xóa `cdc_table_registry` khi FE truyền `cmd.RegistryID > 0` (`WHERE id = ?`) mà không tự xóa theo bộ khóa định danh của chính `shadow_binding` đang xóa:**<br>- Nếu FE không truyền `registry_id` (`cmd.RegistryID == 0`, ví dụ dòng đang ở trạng thái `v2_shadow_only`), bước 3 bị bỏ qua hoàn toàn в khi bước 4 & 5 vẫn xóa `shadow_binding` và `source_object_registry` -> trực tiếp tạo ra bản ghi `cdc_table_registry` mồ côi.<br>- Nếu `cmd.RegistryID` từ FE bị ghép chéo từ connector khác (do `F11`), `WHERE id = ?` sẽ xóa nhầm `cdc_table_registry` của connector khác và **để sót lại bản ghi `cdc_table_registry` của chính connector đang xóa**!<br>- Ngoài ra, khi `DeleteShadowBindingHandler` bị chặn lúc xóa lần đầu do còn Master đang active (`ErrActiveMasterBindingsExist` ở dòng `107-109`), giao diện cần hiển thị rõ tên Connector + Schema trên Modal Xóa để tránh xóa nhầm sang bảng cùng tên của connector khác (`devcoretrans` vs `testcoretrans`). | **`S-F10` (`delete_shadow_binding.go:164-175`):** Trong transaction xóa `shadow_binding`, xóa `cdc_system.cdc_table_registry` bằng câu lệnh `DELETE FROM cdc_system.cdc_table_registry tr USING cdc_system.source_object_registry so WHERE so.id = ? AND ...` đối chiếu trực tiếp `(so.source_connection_id, COALESCE(NULLIF(so.source_schema, ''), so.source_database), so.source_object_name, info.ShadowTable)` của chính `shadow_binding` đang xóa (kết hợp `cmd.RegistryID` có kiểm tra `tr.source_connection_id = so.source_connection_id`). Đảm bảo xóa `shadow_binding` nào là xóa sạch 100% `cdc_table_registry` của đúng connector đó, không phụ thuộc query param từ FE và không bao giờ xóa nhầm connector khác. |
| **`F11`** | **5 câu truy vấn JOIN / Lookup giữa `source_object_registry` và `cdc_table_registry` bỏ sót điều kiện `source_connection_id` (`L033`):**<br>Trong khi `ListEnriched` (`source_object_read_repo_gorm.go:68-71`) đã có `AND (tr.source_connection_id = so.source_connection_id OR tr.source_connection_id IS NULL)`, thì 5 vị trí sau bị bỏ sót:<br>1. `ListShadowBindings` (`source_object_read_repo_gorm.go:416-419`): `LEFT JOIN cdc_system.cdc_table_registry tr` chỉ nối theo `(source_db, source_table, target_table)` làm nhân bản dòng (Cartesian Join) và **gắn chéo `registry_id` của `devcoretrans` sang `testcoretrans`** trên tab `Shadow Bindings`!<br>2. `GetMappingContextByRegistryID` (`source_object_read_repo_gorm.go:261-263`): thiếu `so.source_connection_id = tr.source_connection_id`.<br>3. `GetStats` (`source_object_read_repo_gorm.go:324-330`): thiếu `tr.source_connection_id = so.source_connection_id`.<br>4. `GetByRegistryID` (`source_repo_gorm.go:90`): thiếu `so.source_connection_id = tr.source_connection_id`.<br>5. `UpdateRegistry` (`source_repo_gorm.go:357-360`): khi tìm `sourceObjectID` để auto-approve mapping rules, thiếu `source_connection_id = entry.SourceConnectionID`, làm lấy nhầm `source_object_id` của connector khác. | **`S-F11` (`source_object_read_repo_gorm.go`, `[NEW] source_object_read_repo_bindings.go`, `source_repo_gorm.go`):**<br>- Bổ sung điều kiện `(tr.source_connection_id = so.source_connection_id OR tr.source_connection_id IS NULL)` (và dùng `LEFT JOIN LATERAL ... LIMIT 1` ưu tiên khớp `source_connection_id` trước `NULL`) vào cả 5 vị trí.<br>- Tách `GetStats` và `ListShadowBindings` từ `source_object_read_repo_gorm.go` (`517` dòng `> 500`) sang file đơn trách nhiệm `[NEW] source_object_read_repo_bindings.go` để `source_object_read_repo_gorm.go` giảm xuống `~305` dòng (`NOW <= BEFORE` theo `L008`). |

### 6.2 Code Demo (`S-F9` – `S-F11`, Chưa Áp dụng — Chờ User `APPROVE`)

#### (1) `cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go` (`S-F9` & `S-F11`)
```go
// GetByRegistryID (line 86-93): bổ sung đối chiếu source_connection_id
	err := r.db.WithContext(ctx).Raw(`
		SELECT so.id, so.object_code, so.source_engine_type, so.source_database, so.source_object_name,
		       so.primary_key_field, so.primary_key_type, so.timestamp_field, so.is_active, so.provisioning_state, so.notes
		FROM cdc_system.source_object_registry so
		JOIN cdc_system.cdc_table_registry tr
		  ON COALESCE(NULLIF(so.source_schema, ''), so.source_database) = tr.source_db
		 AND tr.source_table = so.source_object_name
		 AND (tr.source_connection_id = so.source_connection_id OR tr.source_connection_id IS NULL)
		WHERE tr.id = ?
		ORDER BY (tr.source_connection_id IS NULL) ASC, so.id DESC
		LIMIT 1
	`, rid).Scan(&row).Error

// Register (line 258-284): cô lập dọn bản ghi V1 mồ côi theo đúng bộ 4 khóa (source_connection_id, source_db, source_table, target_table)
func (r *sourceRepoGorm) Register(ctx context.Context, entry *sourcemodel.TableRegistry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var connID int64
		if entry.SourceConnectionID != nil {
			connID = *entry.SourceConnectionID
		}
		// Tự động dọn dẹp bản ghi V1 mồ côi của ĐÚNG cặp (source_connection_id, source_db, source_table, target_table)
		// khi không còn shadow_binding tương ứng thuộc đúng source_object_registry của connector đó.
		err := tx.Exec(`
			DELETE FROM cdc_system.cdc_table_registry tr
			WHERE tr.source_db = ?
			  AND tr.source_table = ?
			  AND tr.target_table = ?
			  AND (
			        (?::bigint > 0 AND tr.source_connection_id = ?::bigint)
			     OR (?::bigint <= 0 AND tr.source_connection_id IS NULL)
			  )
			  AND NOT EXISTS (
				  SELECT 1
				  FROM cdc_system.shadow_binding sb
				  JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id
				  WHERE sb.shadow_table = tr.target_table
				    AND COALESCE(NULLIF(so.source_schema, ''), so.source_database) = tr.source_db
				    AND so.source_object_name = tr.source_table
				    AND (
				          (tr.source_connection_id IS NOT NULL AND so.source_connection_id = tr.source_connection_id)
				       OR (tr.source_connection_id IS NULL)
				    )
			  )
		`, entry.SourceDB, entry.SourceTable, entry.TargetTable, connID, connID, connID).Error
		if err != nil {
			return err
		}

		if err := tx.Create(entry).Error; err != nil {
			return err
		}
```

#### (2) `cdc-cms-service/internal/app/commands/shadow/delete_shadow_binding.go` (`S-F10`)
```go
	// 3. Xoá legacy registry (cdc_table_registry) record thuộc ĐÚNG source_object_id + shadow_table
	// (không phụ thuộc mù quáng vào cmd.RegistryID từ FE và không bao giờ xóa nhầm sang connector khác)
	if err := tx.Exec(`
		DELETE FROM cdc_system.cdc_table_registry tr
		USING cdc_system.source_object_registry so
		WHERE so.id = ?
		  AND (tr.source_connection_id = so.source_connection_id OR tr.source_connection_id IS NULL)
		  AND (
		        (tr.target_table = ?
		         AND COALESCE(NULLIF(so.source_schema, ''), so.source_database) = tr.source_db
		         AND tr.source_table = so.source_object_name)
		     OR (?::bigint > 0 AND tr.id = ?::bigint)
		  )
	`, sourceObjectID, info.ShadowTable, cmd.RegistryID, cmd.RegistryID).Error; err != nil {
		tx.Rollback()
		h.logger.Error("delete legacy registry failed",
			zap.Int64("source_object_id", sourceObjectID),
			zap.String("shadow_table", info.ShadowTable),
			zap.Int64("registry_id", cmd.RegistryID),
			zap.Error(err))
		return nil, err
	}
```

#### (3) `cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go` & `[NEW] source_object_read_repo_bindings.go` (`S-F11` & `L008`)
```go
// Trong ListShadowBindings (tách sang source_object_read_repo_bindings.go để giảm dòng file gốc < 500):
		LEFT JOIN LATERAL (
			SELECT tr.id
			FROM cdc_system.cdc_table_registry tr
			WHERE COALESCE(NULLIF(so.source_schema, ''), so.source_database) = tr.source_db
			  AND tr.source_table = so.source_object_name
			  AND tr.target_table = sb.shadow_table
			  AND (
			        tr.source_connection_id = so.source_connection_id
			     OR tr.source_connection_id IS NULL
			  )
			ORDER BY (tr.source_connection_id IS NULL) ASC, tr.id ASC
			LIMIT 1
		) tr ON TRUE
```

### 6.3 Bảng Ánh xạ Điểm Gọi Runtime (`L004`)
| Hàm được sửa | Điểm gọi (`Caller file:line`) | Thời điểm kích hoạt (`Lifecycle Trigger`) |
| :--- | :--- | :--- |
| `sourceRepoGorm.Register` | `internal/app/commands/source/register_registry.go:119` (`RegisterRegistryHandler.Handle` -> step `register-db`) | Khi bấm **Register Sync Table** (`POST /api/v1/source-objects/register`). |
| `DeleteShadowBindingHandler.Handle` | `internal/api/shadow/shadow_binding_actions_handler.go:113` (`DELETE /api/v1/shadow-bindings/:id`) | Khi bấm **Delete / Xoá** trên trang Shadow (`TableRegistry.tsx:1828`). |
| `sourceObjectReadRepoGorm.ListShadowBindings` & `GetMappingContextByRegistryID` | `internal/app/queries/source/source_object_reader.go` (`GET /api/v1/shadow-bindings`, `GET /api/v1/source-objects/registry/:id`) | Khi tải danh sách tab `Shadow Bindings` và màn hình `Mapping`. |

---

## 7. Giải pháp Bổ sung (`TASK-05`): Khắc phục Lỗi Scan Bảng `cdc_transaction_history_payload` Vẫn Hiện `_deleted`, `_gpay_id`, `_source_ts`, `_updated_at` (`pending | in shadow`) (`F12`–`F14`)

### 7.1 Bảng Ánh xạ 1-1 Giữa Phát hiện (`F12`–`F14`) và Giải pháp (`S-F12`–`S-F14`)

| Mã Phát hiện | Hiện tượng & Nguyên nhân gốc rễ (`file:line` & Runtime PID) | Mã Giải pháp | Giải pháp Triệt để (Single Best Solution) |
| :--- | :--- | :--- | :--- |
| **`F12`** | **Tiến trình `centralized-data-service` (`PID 30933`, cổng `:8082`) chưa từng được restart từ hôm qua (`STARTED = Thu Oct 29 17:22:57 2026`, binary `/var/folders/cv/.../go-build3497938306/b001/exe/main`):** Trong khi `cdc-cms-service` (`PID 18959`, cổng `:8083`) đã được restart lúc `Fri Oct 30 10:17:54 2026`, thì `centralized-data-service` (`PID 30933`) vẫn chạy bản binary trên RAM biên dịch từ `29/10/2026 17:22:57` (trước khi có bộ lọc `IsReservedSystemColumn` tại `discover_handler_sql.go:79` & `discovery_utils_sql.go:54`). Khi anh bấm **Scan** trên `cdc_transaction_history_payload`, `cdc-cms-service` bắn lệnh `cdc.cmd.scan-fields` qua NATS sang `PID 30933`; binary cũ của `PID 30933` quét bảng nguồn `core-trans-proxy-history-service.public.cdc_transaction_history_payload` (vốn có 4 cột vật lý `_deleted`, `_gpay_id`, `_source_ts`, `_updated_at`) và chèn cả 13 dòng vào `mapping_rule_v2` với `status = 'pending'`. Sau đó khi anh bật Active, `cdc-cms-service` (`PID 18959` chạy code mới `source_repo_gorm.go:384-393`) chặn duyệt 4 cột hệ thống nên chỉ 9 cột nghiệp vụ chuyển sang `approved`, còn 4 cột `_deleted`, `_gpay_id`, `_source_ts`, `_updated_at` giữ nguyên `pending`! | **`S-F12`** | Sau khi `APPROVE`, nạp lại (restart) cả tiến trình `centralized-data-service` (`PID 30933`) lẫn `cdc-cms-service` (`PID 18959`) và kiểm chứng trực tiếp `ps -ww -o pid,lstart,command` (`L021`). |
| **`F13`** | **`mappingRuleRepoGorm.buildFilter` (`cdc-cms-service/internal/infra/persistence/master/mapping_rule_repo_gorm.go:152-191`) chưa lọc bỏ `ReservedShadowSystemColumns()` khi trả kết quả `GET /api/mapping-rules`, và `scanFieldsSQLSource` / `processDiscoveryRows` chưa tự dọn các dòng cột hệ thống tồn dư khi Scan lại:** Nếu trong `mapping_rule_v2` đã lỡ có 4 dòng `_deleted`, `_gpay_id`, `_source_ts`, `_updated_at` (do binary cũ `PID 30933` chèn vào), `GET /api/mapping-rules` vẫn trả thẳng 4 dòng này lên UI `MappingFieldsPage.tsx`, và khi Scan lại worker chỉ `continue` bỏ qua chứ không tự xoá các dòng hệ thống tồn dư của `(source_object_id, shadow_binding_id)`. | **`S-F13`** | **(1)** Tại `mapping_rule_repo_gorm.go:152-191` (`buildFilter`), bổ sung `AND LOWER(TRIM(mr.source_field)) NOT IN (?) AND LOWER(TRIM(mr.target_column)) NOT IN (?)` với `domainshadow.ReservedShadowSystemColumns()` để `GET /api/mapping-rules` không bao giờ trả cột hệ thống lên UI.<br>**(2)** Tại `centralized-data-service/internal/repository/master/mapping_rule_v2_repo.go`, bổ sung `DeleteReservedSystemRulesBySourceObjectAndBinding(ctx, sourceObjectID, shadowBindingID)` và gọi ngay đầu `scanFieldsSQLSource` (`discover_handler_sql.go:56`) & `processDiscoveryRows` (`discover_handler_utils.go:46`) để mỗi lần Scan tự động dọn sạch các dòng cột hệ thống tồn dư mà không cần can thiệp DB thủ công. |
| **`F14`** | **`SyncRulesFromLegacyTx` (`cdc-cms-service/internal/infra/persistence/source/source_object_v2_sync.go:267-308`) và `BridgeMappingRulesToV2` (`centralized-data-service/internal/service/metadata/mapping_utils.go:217-250`) đồng bộ từ bảng V1 `cdc_mapping_rules` sang `mapping_rule_v2` mà không lọc `ReservedShadowSystemColumns()`:** Nếu bảng V1 `cdc_mapping_rules` còn lưu dòng cũ của `cdc_transaction_history_payload`, hàm bridge sẽ chép ngược các cột hệ thống vào `mapping_rule_v2`. | **`S-F14`** | Bổ sung `AND LOWER(TRIM(source_field)) NOT IN (?) AND LOWER(TRIM(target_column)) NOT IN (?)` trong `SyncRulesFromLegacyTx` (`source_object_v2_sync.go:297`) và thêm chốt `if modelshadow.IsReservedSystemColumn(r.SourceField) || modelshadow.IsReservedSystemColumn(r.TargetColumn) { continue }` trong `BridgeMappingRulesToV2` (`mapping_utils.go:218`). |

### 7.2 Code Demo Trực tiếp (`S-F13` & `S-F14`)

#### (1) `cdc-cms-service/internal/infra/persistence/master/mapping_rule_repo_gorm.go` (`S-F13` — Chặn cột hệ thống ngay tại `buildFilter` của `GET /api/mapping-rules`)
**Trước (`mapping_rule_repo_gorm.go:186-191`):**
```go
	if f.IsActive != nil {
		q += ` AND mr.is_active = ?`
		args = append(args, *f.IsActive)
	}
	return q, args
}
```
**Sau:**
```go
	if f.IsActive != nil {
		q += ` AND mr.is_active = ?`
		args = append(args, *f.IsActive)
	}
	reservedCols := domainshadow.ReservedShadowSystemColumns()
	q += ` AND LOWER(TRIM(mr.source_field)) NOT IN (?) AND LOWER(TRIM(mr.target_column)) NOT IN (?)`
	args = append(args, reservedCols, reservedCols)
	return q, args
}
```

#### (2) `centralized-data-service/internal/repository/master/mapping_rule_v2_repo.go` & `discover_handler_sql.go` / `discover_handler_utils.go` (`S-F13` — Tự động xoá rule cột hệ thống tồn dư khi chạy Scan)
**Thêm vào `mapping_rule_v2_repo.go:93`:**
```go
func (r *MappingRuleV2Repo) DeleteReservedSystemRulesBySourceObjectAndBinding(ctx context.Context, sourceObjectID int64, shadowBindingID int64) error {
	reservedCols := shadow.ReservedSystemColumns()
	query := r.db.WithContext(ctx).
		Where("source_object_id = ? AND (LOWER(TRIM(source_field)) IN ? OR LOWER(TRIM(target_column)) IN ?)",
			sourceObjectID, reservedCols, reservedCols)
	if shadowBindingID > 0 {
		query = query.Where("shadow_binding_id = ?", shadowBindingID)
	}
	return query.Delete(&mastermodel.MappingRuleV2{}).Error
}
```
**Gọi tại `discover_handler_sql.go:56` và `discover_handler_utils.go:46`:**
```go
	_ = h.mappingV2Repo.DeleteReservedSystemRulesBySourceObjectAndBinding(ctx, registry.ID, shadowBindingID)
	existingRules, err := h.mappingV2Repo.ListBySourceObjectAndBinding(ctx, registry.ID, shadowBindingID)
```

#### (3) `cdc-cms-service/internal/infra/persistence/source/source_object_v2_sync.go` & `centralized-data-service/internal/service/metadata/mapping_utils.go` (`S-F14` — Chặn cột hệ thống ở cầu đồng bộ V1 -> V2)
**Trước (`source_object_v2_sync.go:296-298`):**
```go
		FROM cdc_system.cdc_mapping_rules
		WHERE source_table = ?
		ON CONFLICT (source_object_id, COALESCE(shadow_binding_id, 0), COALESCE(master_binding_id, 0), target_column) DO UPDATE SET
```
**Sau:**
```go
		FROM cdc_system.cdc_mapping_rules
		WHERE source_table = ?
		  AND LOWER(TRIM(source_field)) NOT IN (?)
		  AND LOWER(TRIM(target_column)) NOT IN (?)
		ON CONFLICT (source_object_id, COALESCE(shadow_binding_id, 0), COALESCE(master_binding_id, 0), target_column) DO UPDATE SET
```
**Trước (`mapping_utils.go:217-220`):**
```go
	for _, r := range v1Rules {
		if !r.IsActive {
			continue
		}
```
**Sau:**
```go
	for _, r := range v1Rules {
		if !r.IsActive || modelshadow.IsReservedSystemColumn(r.SourceField) || modelshadow.IsReservedSystemColumn(r.TargetColumn) {
			continue
		}
```

### 7.3 Bảng Ánh xạ Điểm Gọi Runtime (`L004`)
| Hàm được sửa | Điểm gọi (`Caller file:line`) | Thời điểm kích hoạt (`Lifecycle Trigger`) |
| :--- | :--- | :--- |
| `mappingRuleRepoGorm.buildFilter` | `internal/infra/persistence/master/mapping_rule_repo_gorm.go:196, 220` (`List` & `ListPaginated` <- `GET /api/mapping-rules`) | Khi trang `MappingFieldsPage.tsx:195` tải danh sách Mapping Rules. |
| `MappingRuleV2Repo.DeleteReservedSystemRulesBySourceObjectAndBinding` | `internal/handler/source/discover_handler_sql.go:56` & `discover_handler_utils.go:46` (`ScanFieldsDebezium` <- `cdc.cmd.scan-fields`) | Khi bấm **Scan Fields** trên trang Source Objects / Shadow. |
| `SourceObjectV2SyncService.SyncRulesFromLegacyTx` & `BridgeMappingRulesToV2` | `internal/infra/persistence/source/source_object_v2_sync.go:255` & `internal/handler/source/discover_handler.go:183` | Khi đăng ký/cập nhật bảng nguồn hoặc chạy discover provisioning. |
