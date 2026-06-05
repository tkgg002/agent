# 00_context.md — Snapshot.v2 Mapping Cache Stale (source 66)

| Field | Value |
|---|---|
| **Workspace** | `bug-snapshot-v2-mapping-cache-stale-2026-06-02` |
| **Date** | 2026-06-02 |
| **Severity** | 🔴 HIGH — snapshot.v2 ghi data vào shadow nhưng **0 business column được map** → master layer trống/sai → operator phải re-snapshot manual |
| **Owner Brain** | Antigravity / Claude Opus 4.7 |
| **Scope** | **Brain plan + spec only** (CLAUDE.md §12). Muscle execute sau User approve. |
| **Target service** | `data-hub/cdc-cms-service` (control plane — approve handler) + `data-hub/centralized-data-service` (worker — cache reload subscriber) |

---

## 1. Triệu chứng (User report)

> "snapshot.v2 source 66, dù đã có field approve nhưng khi snapshot nó vẫn không mapping qua. Có vẻ nó đang dính cache cũ. Anh đã nói là vụ cache này nó rất vớ vẩn."

User ngữ cảnh: đây là vấn đề **lặp lại** (user dùng từ "rất vớ vẩn" + "đã nói") → tồn tại pattern cache invalidation chưa robust, không phải lỗi 1-lần.

---

## 2. Bản chất bug — **3 bug đan xen, 1 root cause chính + 2 amplifier**

### L1 — ROOT CAUSE: Conditional NATS publish skip khi `shadow_table` resolve nil

**File:** `data-hub/cdc-cms-service/internal/app/commands/update_mapping_rule.go:177-179`

```go
if h.nats != nil && rule.ShadowTable != nil && *rule.ShadowTable != "" {
    h.nats.PublishReload(*rule.ShadowTable, cmd.UpdatedBy, "mapping_status_update", "")
}
```

- `rule.ShadowTable` resolve qua `LEFT JOIN cdc_system.shadow_binding sb ON (mr.shadow_binding_id IS NOT NULL AND ...) OR (mr.shadow_binding_id IS NULL AND sb.source_object_id = ... AND sb.is_active = TRUE)` (line 83-89).
- Nếu `mapping_rule_v2.shadow_binding_id IS NULL` AND không active binding match → `ShadowTable = nil` → **publish skip hoàn toàn**.
- Worker subscribe `schema.config.reload` (`worker_server.go:334`) → callback `ReloadAll()` → **không bao giờ được trigger** → cache stale.
- Matches lesson **L-3110 (2026-05-18)** "Conditional Subscriber Registration Causes Silent NATS Drops" — chính pattern producer-conditional, consumer luôn bật.

### L2 — Amplifier: Pre-flight `ReloadAll` race tại worker

**File:** `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go:328-335`

`runSnapshot()` gọi `registrySvc.ReloadAll(ctx)` ngay đầu — đáng lẽ defense-in-depth cho L1. Nhưng:
- User approve mapping rule và click "Snapshot Now" gần như cùng lúc.
- NATS dispatch latency ~ms, transaction commit visible time ~ms.
- Worker chạy `ReloadAll` query DB **TRƯỚC khi** approve transaction commit visible (replica lag hoặc cùng ms với commit) → query `WHERE status='approved'` không trả approved rule mới → cache populate **thiếu rule**.
- Toàn bộ cursor loop sau đó dùng cache stale → 0 mapping được apply.

### L3 — Risk amplifier: Cache key type migration cùng ngày

**Commit `0289fe4` (2026-06-02, hôm nay):** đổi `mappingCache map[string]...` → `map[int64]...`, đổi `MapData(targetTable string)` → `MapData(bindingID int64)`. 23 files, 2127 LOC.
- Nếu worker đang chạy image cũ (chưa rolling restart sau deploy) → lookup `string` key mà cache populate `int64` key → 100% silent miss.
- Pre-existing rules cũng miss, không chỉ approved-mới.

---

## 3. Chain of failure (source 66 cụ thể)

```
User approve rule cho source_object_id=66
  ↓
UpdateMappingRuleCommand.Handle (cdc-cms-service)
  ↓ JOIN resolve shadow_table → NULL (shadow_binding_id NULL hoặc binding inactive)
  ↓ ❌ skip h.nats.PublishReload  ← L1 ROOT CAUSE
  ↓
[không có signal nào tới worker]
  ↓
User click "Snapshot Now V2" cho source 66
  ↓
POST /api/v1/source-objects/66/snapshot-v2
  ↓ SnapshotV2Command → NATS cdc.cmd.snapshot.v2
  ↓
Worker SnapshotRunner.Handle → runSnapshot()
  ↓ ReloadAll() ← race với approve commit ← L2 AMPLIFIER
  ↓ mappingV2Repo.ListActiveBySourceObject(66)
  ↓    SELECT * FROM mapping_rule_v2 WHERE source_object_id=66 AND status='approved' AND is_active=true
  ↓    ❌ approved rule chưa visible → kết quả thiếu
  ↓ mappingCache[binding_id_for_66] = [rules cũ, không có rule mới approve]
  ↓
Cursor loop: mongo.Find → EventHandler.HandleRaw → dynamicMapper.MapData(bindingID, data)
  ↓ rules := registry.GetMappingRules(bindingID)  ← cache stale
  ↓ ❌ rule mới approve KHÔNG trong cache
  ↓ return MappedData{Columns: {...rules cũ...}, RawJSON: rawJSON}
  ↓
Shadow table: chỉ insert column theo rules cũ → field user vừa approve KHÔNG MAPPING
```

---

## 4. Đối chiếu lessons.md

| Lesson | Match | Áp dụng |
|---|---|---|
| **L-557 (2026-04-06)** "Indexing Mismatch in Mapping Cache (X-to-Y Pattern)" | Partial — cache key shape thay đổi `string → int64` (commit `0289fe4`) | L3 amplifier |
| **L-3110 (2026-05-18)** "Conditional Subscriber Registration Causes Silent NATS Drops" | **Direct match** — producer conditional, consumer luôn on, silent drop | L1 root cause |
| **L-3164 (2026-05-19)** "Caller-Resolver Wiring" — enumerate mọi write/read site | Áp dụng — phải audit mọi nơi publish reload | Spec |

→ Hai lessons đã ghi nhận pattern này. Workspace này là **third strike** → cần **kill pattern** ở tầng codebase chứ không phải patch tactical.

---

## 5. Vì sao user nói "cache vớ vẩn"

User đã trải qua bug tương tự nhiều lần. Hệ thống có nhiều cache layer + nhiều invalidation gate conditional:
- `mappingCache` (worker `metadata_registry_service.go`) — TTL=∞, invalidate qua NATS conditional
- Redis cache (CMS layer) — TTL configurable
- `schema_cache` (sinkworker) — invalidate qua hot reload
- Transmute cache — separate

Mỗi cache có cơ chế invalidate khác nhau, không có **single source of truth invalidation signal**. Mỗi lần thay đổi handler quên publish → cache nào đó stale → bug "vớ vẩn".

---

## 6. Mục tiêu workspace

1. **Fix L1 root cause**: ALWAYS publish reload signal khi mapping rule status thay đổi, **không gate bởi shadow_table resolution**.
2. **Fix L2 amplifier**: Pre-flight `ReloadAll` trong `runSnapshot` thêm **post-reload sanity check** — verify cache count khớp DB count, retry 1 lần nếu drift.
3. **Document L3 risk**: ghi deploy procedure rolling restart bắt buộc sau commit cache-key change.
4. **Kill pattern**: thiết lập invariant "mọi mutation mapping_rule_v2 PHẢI publish reload" — CI gate / lint rule.
5. **Test cover**: reproduce race + regression guard.

---

## 7. Out-of-Scope

- ❌ Refactor toàn bộ cache architecture sang Redis pub/sub (over-engineer, NFR violate).
- ❌ Đổi `status='approved'` semantics — đã đúng intent.
- ❌ Đụng V1 legacy `cdc_table_registry` mapping path — chỉ V2 `mapping_rule_v2`.
- ❌ Refactor `MetadataRegistryService` interface (commit `0289fe4` đã làm).
- ❌ Bug `_gpay_id NULL` (workspace `bug-gpay-id-trigger-contract-2026-06-02` riêng).

---

## 8. Tham chiếu file (forensics evidence)

| File | Vai trò |
|---|---|
| `cdc-cms-service/internal/app/commands/update_mapping_rule.go:83-89, 177-179` | L1 — conditional publish skip |
| `cdc-cms-service/internal/api/source_object_actions_handler.go:561` | SnapshotV2 dispatch handler |
| `cdc-cms-service/internal/router/router.go:382` | Route `POST /api/v1/source-objects/{id}/snapshot-v2` |
| `centralized-data-service/internal/handler/snapshot_runner_handler.go:118, 166, 328-335` | Worker entry + pre-flight ReloadAll |
| `centralized-data-service/internal/service/metadata_registry_service.go:74, 112, 156, 303-305, 388-391` | Cache struct + ReloadAll + GetMappingRules |
| `centralized-data-service/internal/service/mapping_rule_v2_repo.go:37-44` | SQL filter `status='approved'` |
| `centralized-data-service/internal/server/worker_server.go:334, 357` | NATS subscribe `schema.config.reload` |
| `centralized-data-service/internal/handler/event_handler.go:128, 210` | processEvent → MapData |
| `centralized-data-service/internal/service/dynamic_mapper.go:71` | GetMappingRules call site |
| Commit `0289fe4` (2026-06-02) | L3 — cache key type migration |
| Commit `0d66fc5` (2026-06-02) | L1 amplifier — thêm shadow_binding_id logic + conditional publish |
| Commit `019bd0e` (2026-05-29) | Snapshot.v2 binding_id scope |
