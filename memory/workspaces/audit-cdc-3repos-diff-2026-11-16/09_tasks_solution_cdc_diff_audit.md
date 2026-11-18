# 09. TASKS SOLUTION — Sửa dứt điểm 4 Critical Issues trong Git Diff 3 Repo CDC

- **workspace:** `audit-cdc-3repos-diff-2026-11-16`
- **suffix:** `cdc_diff_audit`
- **date:** `2026-11-16`

## 1. Bảng ánh xạ 1-1 Phát hiện $\leftrightarrow$ Giải pháp duy nhất

| Mã | Phát hiện (`file:line`) | Giải pháp duy nhất |
| :--- | :--- | :--- |
| **F1 (CRITICAL-1)** | `centralized-data-service/internal/service/recon/recon_multi_shadow_b.go:180-184`: `shouldSkipMongoRowDiffRule` trả về `true` cho mọi cột bị thiếu/`nil` trên MongoDB Master nếu document có bất kỳ nested `map[string]any` nào. | Thay vòng lặp mù bằng hàm `isDenormalizedInMongoNestedSubDoc(mrow, col)`: chỉ skip khi `col` thực sự là cột phẳng (denormalized) khớp tên trường con hoặc `<parent>_<child>` bên trong nested sub-document (`map[string]any` / `bson.M`) của `mrow`. Các cột nghiệp vụ top-level (`value`, `trans_id`, `bill_code`...) bị thiếu/`nil` trên Master sẽ trả về `false` để `RunRowDiffB` bắt đúng lệch. |
| **F2 (CRITICAL-2)** | `centralized-data-service/internal/service/master/transmuter_subset_array_utils.go:274-286`, `mongo_extjson_unwrapper.go:712-746`, `mongo_bson_restorer.go:521-531`: `isValueSubsetCarrierMap` nhận nhầm mọi sub-document có `_id` là cột metadata `@value_subset` và xóa mất sub-document; `promoteMongoDocumentID` duyệt map ngẫu nhiên. | (1) Trong `extractValueSubsetMetadata`, nếu `len(colTypes) > 0` và `colTypes[k]` đã khai báo nhưng không gắn cờ `@value_subset` (và `k` không phải `source_field_values`/`sourceFieldValues`) thì không coi là carrier. (2) Trong `isValueSubsetCarrierMap`, kiểm tra nếu `doc["_id"] != nil` thì `m["_id"]` bắt buộc phải khớp `doc["_id"]` (sub-document như `sender._id` có ID riêng khác `doc["_id"]` sẽ không bị xóa nhầm). (3) Trong `promoteMongoDocumentID`, sắp xếp danh sách key ứng viên deterministic và loại trừ các prefix khóa ngoại (`original_`, `sender_`, `receiver_`, `parent_`). |
| **F3 (CRITICAL-3)** | `centralized-data-service/internal/handler/orchestration/snapshot_runner_lifecycle.go:225-308`, `snapshot_runner_handler.go:166-175,933-937`, `snapshot_runner_state.go:43-50`: `recoverOrphanedSnapshots` không giới hạn số lần auto-resume (gây vòng lặp OOM vô tận), thiếu `FOR UPDATE SKIP LOCKED`, ghi sai `status='success'`/`'error'` vào `cdc_activity_log` khi pause/skip, và thiếu guard `r.stopping.Load()` ở `final flush`. | (1) Bổ sung kiểm soát `[auto-resume #N]` (tối đa `MaxOrphanAutoResumeAttempts = 3`) cho `snapshot_progress` tương tự `transmute_jobs`/`transform_jobs`; quá 3 lần tự động chuyển sang `status = 'error'`. (2) Dùng CTE `FOR UPDATE SKIP LOCKED` khi claim orphan trong `recoverOrphanedSnapshots`. (3) Cập nhật `runSnapshot` để không ghi `status = 'success'` giả khi `paused`/`skipped`, và kiểm tra `r.stopping.Load()` tại `final flush`. Đồng bộ `snapshot_progress_read_repo_gorm.go` ở `cdc-cms-service` bảo toàn tag `[auto-resume #N]`. |
| **F4 (CRITICAL-4)** | `cdc-cms-service/internal/app/commands/master/update_column_metadata.go:68-71,94-97` & `migrations/schema/worker/025_create_daily_source_counts.sql:20-51`: BE tự xóa `transform_fn` khi đổi kiểu sang `String` trên ClickHouse; migration `025` chạy `CREATE INDEX` ngoài khối `IF` khi bảng chưa tạo. | (1) Bổ sung `u == "STRING"` vào `isJSONOrJSONBType` trong `update_column_metadata.go` để đồng bộ với `clickhouseDataTypes.ts:241`. (2) Trong `025_create_daily_source_counts.sql`, luôn tạo bảng `cdc_system.daily_source_counts` (có FK khi `sched_rel IS NOT NULL` và không FK khi `sched_rel IS NULL`) để `CREATE INDEX` không bao giờ lỗi. |
| **F5 (HIGH-1)** | `centralized-data-service/internal/service/master/transmuter_gpayid_test.go:409`: lấy cả `id < 0` làm fail test. | Thêm điều kiện `AND id > 0` trong `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift`. |

---

## 2. Code Demo

### 2.1. `recon_multi_shadow_b.go` (F1 — CRITICAL-1)
```go
func isDenormalizedInMongoNestedSubDoc(mrow map[string]any, targetCol string) bool {
	normCol := normReconAlphaNum(targetCol)
	if normCol == "" {
		return false
	}
	for mk, mv := range mrow {
		if mk == "_id" || strings.HasPrefix(mk, "_") || mv == nil {
			continue
		}
		subMap := toStringAnyMap(mv)
		if len(subMap) == 0 {
			continue
		}
		normParent := normReconAlphaNum(mk)
		for ck, cv := range subMap {
			normChild := normReconAlphaNum(ck)
			if normCol == normChild || (normParent != "" && normCol == normParent+normChild) {
				return true
			}
			if subSub := toStringAnyMap(cv); len(subSub) > 0 {
				for sck := range subSub {
					normSubChild := normReconAlphaNum(sck)
					if normCol == normSubChild || normCol == normChild+normSubChild || (normParent != "" && normCol == normParent+normChild+normSubChild) {
						return true
					}
				}
			}
		}
	}
	return false
}
```

### 2.2. `transmuter_subset_array_utils.go` & `mongo_extjson_unwrapper.go` (F2 — CRITICAL-2)
```go
func isValueSubsetCarrierMap(doc bson.M, key string, raw any) bool {
	if key == "" || key == "_id" || strings.HasPrefix(key, "_") || raw == nil {
		return false
	}
	m := decodeRawSubsetMap(raw)
	if len(m) < 2 {
		return false
	}
	rawSubID, hasID := m["_id"]
	if !hasID || rawSubID == nil {
		return false
	}
	if key == "source_field_values" || key == "sourceFieldValues" {
		return true
	}
	if doc != nil && doc["_id"] != nil {
		return normalizeMongoKeyString(rawSubID) == normalizeMongoKeyString(doc["_id"])
	}
	return false
}
```

### 2.3. `snapshot_runner_lifecycle.go` & `snapshot_runner_handler.go` (F3 — CRITICAL-3)
```go
// Trong recoverOrphanedSnapshots:
// Dùng FOR UPDATE SKIP LOCKED và kiểm tra parseSnapshotAutoResumeCount(row.ErrorMsg) >= repository.MaxOrphanAutoResumeAttempts (3)
```

### 2.4. `update_column_metadata.go` & `025_create_daily_source_counts.sql` (F4 — CRITICAL-4)
```go
func isJSONOrJSONBType(dt string) bool {
	u := strings.ToUpper(strings.TrimSpace(dt))
	return u == "JSON" || u == "JSONB" || u == "STRING" || strings.HasPrefix(u, "JSON ") || strings.HasPrefix(u, "JSONB ")
}
```
