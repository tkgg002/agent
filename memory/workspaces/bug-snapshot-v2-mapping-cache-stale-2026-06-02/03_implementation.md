# 03_implementation.md — Patch tối thiểu (REVISED)

> **Simplicity First**. Snapshot bypass cache, query DB `mapping_rule_v2` trực tiếp. 1 service (worker `centralized-data-service`), ~20 LOC, 2 file.
>
> KHÔNG động CMS. KHÔNG động NATS. KHÔNG động cache invalidation. Cache realtime CDC vẫn giữ nguyên.

---

## Nguyên lý

| Path | Frequency | Cần cache? |
|---|---|---|
| Realtime CDC stream | High (events/sec) | YES — `mappingCache` |
| Snapshot.v2 replay | Low (manual trigger) | **NO** — query DB direct |

Snapshot không có lý do gì xài cache. Query DB 1 lần đầu snapshot job → đảm bảo always fresh.

---

## §1 — Worker snapshot bypass cache, query DB direct

**File**: `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go`

### Before (line ~328-335)
```go
if err := r.registrySvc.ReloadAll(ctx); err != nil {
    log.Error("snapshot pre-flight reload failed", "err", err)
}
routes, err := r.registrySvc.ResolveSourceRoutes(srcDB, srcColl)
// ... cursor loop gọi eventHandler.HandleRaw → dynamicMapper.MapData(bindingID, data)
// → registry.GetMappingRules(bindingID) ← CACHE STALE
```

### After
```go
// Snapshot: bypass cache, query rules trực tiếp từ mapping_rule_v2 (always fresh).
rules, err := r.mappingV2Repo.ListActiveBySourceObject(ctx, sourceObjectID)
if err != nil {
    return fmt.Errorf("snapshot mapping rules query failed: %w", err)
}
log.Info("snapshot.mapping_rules.loaded",
    "source_object_id", sourceObjectID, "count", len(rules))

routes, err := r.registrySvc.ResolveSourceRoutes(srcDB, srcColl)
// ... cursor loop dùng `rules` truyền trực tiếp xuống mapper (xem §2)
```

**Bỏ**: `r.registrySvc.ReloadAll(ctx)` — không cần nữa, snapshot không phụ thuộc cache.

---

## §2 — Mapper accept rules-provided (snapshot scope)

**File**: `data-hub/centralized-data-service/internal/service/dynamic_mapper.go`

### Add method mới (giữ nguyên `MapData` cũ cho realtime CDC)
```go
// MapDataWithRules: snapshot path — rules provided explicit, bypass cache.
func (m *dynamicMapper) MapDataWithRules(rules []model.MappingRule, data map[string]any) MappedData {
    return m.mapInternal(rules, data) // tách core logic ra hàm internal nếu cần
}

// MapData cũ giữ nguyên — realtime CDC vẫn xài cache.
func (m *dynamicMapper) MapData(bindingID int64, data map[string]any) MappedData {
    rules := m.registry.GetMappingRules(bindingID)
    return m.mapInternal(rules, data)
}
```

Nếu logic body `MapData` đơn giản → inline `MapDataWithRules` không cần tách `mapInternal`, copy ~5 LOC body.

---

## §3 — Snapshot cursor loop dùng rules-provided

**File**: `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go` (cursor loop, sau §1)

### Before
```go
for cursor.Next(ctx) {
    var doc bson.M
    cursor.Decode(&doc)
    mapped := r.eventHandler.HandleRaw(bindingID, doc)  // ← cache path
    // ... upsert shadow ...
}
```

### After
```go
for cursor.Next(ctx) {
    var doc bson.M
    cursor.Decode(&doc)
    mapped := r.dynamicMapper.MapDataWithRules(rules, doc)  // ← rules từ §1, fresh DB
    // ... upsert shadow ...
}
```

Nếu `eventHandler.HandleRaw` có thêm logic ngoài map (filter, transform meta) → thêm overload `HandleRawWithRules(rules, doc)` tương tự §2 (~10 LOC).

---

## §4 — Verify

```bash
# 1. Apply patch + restart centralized-data-service.
# 2. Approve mapping rule mới cho source 66 (KHÔNG cần restart CMS, KHÔNG cần publish reload).
# 3. Trigger snapshot.v2:
curl -X POST http://cms/api/v1/source-objects/66/snapshot-v2
# 4. Tail worker log:
kubectl logs deploy/centralized-data-service --since=1m | grep snapshot.mapping_rules.loaded
# Expect: count=N (N = số rule approved hiện tại trong DB)
# 5. Query shadow:
psql -c "SELECT column_name FROM information_schema.columns WHERE table_schema='shadow' AND table_name='<shadow>_66'"
# Expect: column rule mới approved có mặt.
```

---

## §5 — Out (KHÔNG làm)

| Item | Lý do |
|---|---|
| Fix CMS `update_mapping_rule.go` conditional publish | Snapshot không còn phụ thuộc cache → không cần fix producer |
| Cache invalidation post-reload count check | Snapshot path đã không xài cache |
| Prometheus metrics cache | Future workspace (G6) |
| Realtime CDC path | Vẫn xài cache (high-throughput cần cache); bug realtime = workspace khác nếu xảy ra |

---

## §6 — Patch size

| File | LOC changed |
|---|---|
| `snapshot_runner_handler.go` | ~10 (query DB + bỏ ReloadAll + cursor loop dùng rules-provided) |
| `dynamic_mapper.go` | ~10 (thêm method `MapDataWithRules`) |
| **Total** | **~20 LOC, 2 file, 1 service** |

---

## §7 — Rollback

`git revert <commit>` — 20 LOC, an toàn. Realtime CDC không bị động → 0 risk regression.
