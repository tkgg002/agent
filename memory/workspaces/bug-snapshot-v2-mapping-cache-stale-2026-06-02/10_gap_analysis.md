# 10_gap_analysis.md — Cache Invalidation Gap Evidence

> Đo bằng `Read` + `Grep` + commit log forensics. Mỗi gap có file:line evidence.

---

## Gap matrix

| Gap | Severity | Layer | File:Line | Loại |
|---|---|---|---|---|
| **G1** | 🔴 ROOT | CMS approve handler | `update_mapping_rule.go:177-179` | Conditional publish skip |
| **G2** | 🟡 AMPLIFIER | Worker snapshot runner | `snapshot_runner_handler.go:328-335` | Race window pre-flight |
| **G3** | 🟠 RISK | Cache key migration | Commit `0289fe4` (today) | Type change deploy mismatch |
| **G4** | 🟡 MED | Worker cache invalidate | `metadata_registry_service.go` (whole file) | TTL=∞, no post-write verify |
| **G5** | 🟢 LOW | Single signal source | Multi-cache architecture | No SSoT invalidation contract |
| **G6** | 🟢 LOW | Observability | No metric `cache_miss_ratio` | Silent stale invisible |

---

## G1 — Conditional NATS publish skip (ROOT CAUSE)

### Evidence
```go
// cdc-cms-service/internal/app/commands/update_mapping_rule.go:83-89
err := h.db.WithContext(ctx).Raw(`
    SELECT mr.*, sb.shadow_table AS shadow_table_resolved
    FROM cdc_system.mapping_rule_v2 mr
    LEFT JOIN cdc_system.shadow_binding sb ON
        (mr.shadow_binding_id IS NOT NULL AND sb.id = mr.shadow_binding_id)
        OR (mr.shadow_binding_id IS NULL AND sb.source_object_id = mr.source_object_id AND sb.is_active = TRUE)
    WHERE mr.id = ?
`, cmd.ID).Scan(&rule).Error
```

```go
// cdc-cms-service/internal/app/commands/update_mapping_rule.go:177-179
if h.nats != nil && rule.ShadowTable != nil && *rule.ShadowTable != "" {
    h.nats.PublishReload(*rule.ShadowTable, cmd.UpdatedBy, "mapping_status_update", "")
}
// ❌ ELSE branch: NOTHING — không log, không stub publish, silent skip
```

### Khi nào trigger
1. `mapping_rule_v2.shadow_binding_id IS NULL` (rule chưa được bind explicit) AND
2. `shadow_binding` table không có row active match `source_object_id` → JOIN trả NULL.

### Tác động
- Worker không nhận signal → `ReloadAll()` không chạy.
- Cache giữ rules cũ vĩnh viễn cho đến khi:
  - Worker restart, HOẶC
  - Một mutation khác publish reload thành công (race), HOẶC
  - Snapshot.v2 trigger (pre-flight reload — nhưng L2 race).

### Lesson match
**L-3110 (2026-05-18)** Direct pattern:
> "Pattern [A registers a NATS subscriber S for subject J inside a conditional block ... producer P remains enabled unconditionally] → silent drop"

Hiện tại pattern đảo ngược: **producer conditional, consumer luôn on** — vẫn cùng kết quả (silent drop). Lesson chưa cover trường hợp đảo ngược → cần update lesson.

---

## G2 — Race window pre-flight ReloadAll

### Evidence
```go
// centralized-data-service/internal/handler/snapshot_runner_handler.go:328-335
func (r *SnapshotRunner) runSnapshot(ctx context.Context, p Payload, jobID int64) error {
    // ... metadata setup ...
    if err := r.registrySvc.ReloadAll(ctx); err != nil {
        log.Error("snapshot pre-flight reload failed", "err", err)
        // KHÔNG return — vẫn chạy tiếp với cache stale có sẵn
    }
    routes, err := r.registrySvc.ResolveSourceRoutes(srcDB, srcColl)
    // ... cursor loop ...
}
```

### Race scenario
```
t=0ms    User click "Approve" rule cho source 66
t=2ms    UpdateMappingRuleCommand.Handle BEGIN transaction
t=5ms    UPDATE mapping_rule_v2 SET status='approved' WHERE id=X
t=6ms    [L1: skip publish vì shadow_table nil]
t=7ms    User click "Snapshot Now V2" (gần như cùng lúc)
t=8ms    POST /api/v1/source-objects/66/snapshot-v2
t=9ms    NATS publish cdc.cmd.snapshot.v2 {source_object_id: 66}
t=10ms   COMMIT approve transaction
t=11ms   Worker SnapshotRunner.Handle nhận msg
t=12ms   runSnapshot → ReloadAll → query mapping_rule_v2
         ❌ Có 2 scenarios:
         (a) Query trước commit visible → thiếu rule
         (b) Replica DB lag → cùng kết quả
t=13ms   mappingCache populate KHÔNG có rule mới
t=14ms+  Cursor loop chạy với cache thiếu → snapshot xong → 0 mapping field mới
```

### Quan trọng
Pre-flight ReloadAll **một mình** đáng lẽ đủ defense — bug L1 (publish skip) khiến cache "trắng" từ trước. Snapshot.v2 chỉ là **occasion** bóc lộ, không phải nguồn.

### Tác động
- Snapshot scan toàn bộ collection → ghi shadow → master transmute trống mapping field mới.
- User phải tự nhận ra → thực thi `ReloadAll` thủ công (qua restart hoặc trigger lại mapping update) → snapshot lại → 2x cost.

---

## G3 — Cache key migration deploy risk

### Evidence
```
git log --all --oneline --since=2026-06-01 -- internal/service/metadata_registry_service.go
0289fe4 refactor: migrate mappingCache key from string targetTable to int64 bindingID
```

### Trước vs Sau
| Aspect | Before `0289fe4` | After `0289fe4` |
|---|---|---|
| `mappingCache` type | `map[string][]MappingRule` | `map[int64][]MappingRule` |
| Key | `targetTable string` (table name) | `bindingID int64` (shadow_binding.id) |
| `GetMappingRules` signature | `(targetTable string)` | `(bindingID int64)` |
| `MapData` signature | `(targetTable string, ...)` | `(bindingID int64, ...)` |
| Call sites changed | — | 23 files, 2127 LOC |

### Tác động deploy mismatch
- Worker pod đang chạy image `pre-0289fe4`:
  - Cache populate key = `int64` (nếu pull image mới một phần)
  - Lookup key = `string`
  - → 100% silent miss cho ALL rules, không chỉ approve-mới
- Có thể giải thích "tất cả mapping field" không apply, không chỉ field mới.

### Mitigation đã có
Nếu deploy đúng quy trình (rolling restart, all pods cùng image) → không impact.
Nhưng nếu local dev chạy binary cũ + DB schema mới → trigger.

---

## G4 — Worker cache TTL=∞, no post-write verify

### Evidence
```go
// centralized-data-service/internal/service/metadata_registry_service.go:74
type metadataRegistryService struct {
    mappingCache map[int64][]model.MappingRule  // ❌ no TTL, no last_loaded_at
    routeCache   map[string]Route               // same
    // ...
}
```

```go
// metadata_registry_service.go:388-391
func (rs *metadataRegistryService) GetMappingRules(bindingID int64) []model.MappingRule {
    rs.mu.RLock()
    defer rs.mu.RUnlock()
    return rs.mappingCache[bindingID]  // ❌ no check stale, no fallback DB query
}
```

### Tác động
- Sống vĩnh viễn cho đến `ReloadAll` được gọi explicit.
- Không có check "khi rule thiếu, có nên fall back DB query?" → silent return empty.
- Không có metric expose cache.size, cache.last_reloaded → operator mù.

---

## G5 — Multi-cache architecture không SSoT invalidation

### Cache layers hiện tại
| Cache | Location | Invalidate trigger |
|---|---|---|
| `mappingCache` | Worker `metadata_registry_service` | NATS `schema.config.reload` |
| `routeCache` | Worker `metadata_registry_service` | Cùng signal |
| `shadow_binding cache` | CMS `cdc-cms-service` Redis | TTL + explicit set/del |
| `schema_cache` | Worker `sinkworker/schema_manager` | Hot reload trigger riêng |
| `transmute_schedule cache` | Worker scheduler | NATS `cdc.cmd.scheduler.reload` |

### Vấn đề
- Mỗi cache có invalidation signal riêng → mutation handler PHẢI nhớ publish đúng signal.
- Không có **registry** cho biết "khi mutate table T, phải publish signals [S1, S2, ...]".
- Mutation thêm sau code không biết phải publish gì → cache stale.

### Recommended (out-of-scope nhưng note)
Future: invariant registry `mutationSignals[table] = [subjects...]`, ORM hook auto-publish.

---

## G6 — No observability cho cache miss / stale ratio

### Evidence
- `grep -rn "cache_miss\|cache_hit\|cache_stale\|mapping_cache_size" centralized-data-service/internal/` → 0 hits.
- Metrics export Prometheus `/metrics` không có cache-related counter.

### Tác động
- Operator chỉ phát hiện cache stale khi user complain → quá muộn.
- Không có alert rule "mapping_cache.last_reloaded > 30min" hay "cache_miss_ratio > 5%".

---

## Tổng kết: 1 root + 2 amplifier + 3 systemic

| Gap | Status hiện tại | Sau fix |
|---|---|---|
| G1 ROOT | Conditional publish skip → silent | Always publish, log every event |
| G2 AMPLIFIER | Pre-flight ReloadAll race | Post-reload count sanity check + retry |
| G3 RISK | Deploy mismatch silent miss | Deploy procedure doc + version pin |
| G4 SYSTEMIC | TTL=∞, no fallback | Optional: cache.last_reloaded metric expose |
| G5 SYSTEMIC | No SSoT invalidation | Future workspace (out-of-scope) |
| G6 SYSTEMIC | No observability | Add 2 metrics + 1 log line |

**Brain recommendation:** Fix G1 + G2 + add G6 instrumentation. G3 = deploy doc. G4 + G5 = future refactor.
