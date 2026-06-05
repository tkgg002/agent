# 04_decisions.md — Architecture Decision Records

> Brain recommendation: 1 hướng duy nhất per ADR (no option 1/2/3 noise per User guideline).

---

## ADR-01 — Bỏ conditional gate `ShadowTable != nil` trong approve handler, ALWAYS publish reload

### Context
`update_mapping_rule.go:177-179` chỉ publish reload khi `rule.ShadowTable` resolve thành non-nil. Nếu `mapping_rule_v2.shadow_binding_id IS NULL` hoặc không có active binding match → resolve nil → silent skip → worker cache stale → bug.

### Decision
**Bỏ điều kiện `ShadowTable != nil && *ShadowTable != ""`**. Always publish reload signal khi mutation success. Đổi payload signal từ `shadow_table` (string, identifier suy diễn) → `source_object_id` (int64, identity nguyên gốc). Worker tự resolve binding từ source_object_id qua ReloadAll.

### Rationale
- **L-3110 lesson direct match**: "every subject must have unconditional producer if consumer is unconditional".
- **Identity-first principle**: `source_object_id` là PK gốc, không suy diễn. `shadow_table` là **derived attribute** — luôn có thể nil/stale.
- **Idempotent on worker**: `ReloadAll()` đã idempotent → publish thừa không hại.

### Consequences
- ✅ Eliminate L1 root cause silent skip
- ✅ Signal robust với mọi shape mapping rule (binding-bound, free-floating, multi-binding)
- ⚠ Tăng NATS message rate marginal (1 msg per mutation, không spam)
- ⚠ Worker phải parse payload field mới — backwards-compat handle (NFR-6)

---

## ADR-02 — Worker pre-flight `ReloadAll` thêm post-reload count sanity check + retry 1 lần

### Context
`snapshot_runner_handler.go:328-335` gọi `ReloadAll` trước cursor loop, nhưng nếu approve commit chưa visible tại thời điểm query → cache populate thiếu rule → snapshot toàn bộ với cache stale.

### Decision
Sau `ReloadAll(ctx)`, ngay lập tức:
1. Query DB `SELECT COUNT(*) FROM mapping_rule_v2 WHERE source_object_id = ? AND status = 'approved' AND is_active = true` → `dbCount`.
2. Đếm `len(mappingCache[bindingID])` cho binding của source → `cacheCount`.
3. Nếu `dbCount > cacheCount` → log WARN, sleep 200ms (đợi replica visibility), gọi `ReloadAll` lần 2.
4. Sau retry, nếu vẫn drift → log ERROR `snapshot.preflight.cache_drift_unresolved`, increment metric, **vẫn tiếp tục** snapshot (defense, không fail-fast).

### Rationale
- **Defense-in-depth** cho L1 fix (nếu CMS gửi signal trễ).
- **1 retry là đủ** — replica lag thông thường < 100ms.
- **Không fail-fast**: snapshot incomplete vẫn tốt hơn 0 snapshot; log + metric đủ để operator phát hiện.

### Consequences
- ✅ Bóc lộ race trong observability thay vì silent.
- ✅ Tự chữa race nhỏ (< 200ms).
- ⚠ +1 DB COUNT query mỗi snapshot — negligible.
- ⚠ Tối đa +200ms snapshot startup latency khi drift.

---

## ADR-03 — Snapshot.v2 dispatch endpoint CMS-side publish reload TRƯỚC khi dispatch snapshot

### Context
Defense ngay tại CMS dispatch layer: trước khi gửi `cdc.cmd.snapshot.v2`, publish `schema.config.reload` cho source_object_id trước, để worker có cơ hội reload trước khi snapshot msg arrive.

### Decision
Trong `source_object_actions_handler.go:561 SnapshotV2()`:
```
1. Validate input
2. nats.PublishReload(source_object_id=X, reason="snapshot_v2_dispatch")
3. time.Sleep(50 * time.Millisecond)   // ngắn, không block UX
4. nats.Publish("cdc.cmd.snapshot.v2", payload)
5. return 202
```

### Rationale
- Race timing approve → reload → snapshot rút ngắn xuống còn race trong **NATS subject ordering** (cùng subject queue đảm bảo FIFO, cross-subject không) + replica lag.
- 50ms là sweet spot: đủ cho NATS deliver reload signal đến worker subscriber, không impact UX (user không thấy).
- Worker side đã có pre-flight ReloadAll (ADR-02) → 3 layer defense.

### Consequences
- ✅ Defense layer 3: dispatch-time signal.
- ✅ UX không cảm nhận (50ms).
- ⚠ Cần đảm bảo dispatch handler có NATS client (đã có).

---

## ADR-04 — Reload signal payload: `source_object_id` thay vì `shadow_table`

### Context
Hiện tại signal mang `shadow_table string` — worker chỉ reload binding match table name. Bug: nếu shadow_table nil ở publisher → không publish.

### Decision
Đổi payload sang `{source_object_id: int64, reason: string, updated_by: string}`. Worker subscribe parse field, gọi `ReloadAll(ctx)` (đơn giản nhất) hoặc tương lai có thể optimize sang `ReloadForSource(source_object_id)`.

### Rationale
- Identity-first (ADR-01 rationale).
- Backwards-compat: worker mới nhận msg cũ (chỉ shadow_table) vẫn fallback `ReloadAll()`.
- Tương lai có thể optimize partial reload bằng source_object_id.

### Consequences
- ✅ Payload robust.
- ⚠ Phải bump NATS message schema version (hoặc nullable field).

---

## ADR-05 — Observability: 2 metric + 1 log line mỗi reload event

### Context
G6 — operator mù với cache stale state.

### Decision
- Metric 1: `mapping_cache_size{source_object_id="X"} gauge` — số rule trong cache cho source.
- Metric 2: `mapping_cache_last_reloaded_seconds gauge` — Unix timestamp last reload.
- Log line: `[INFO] mapping_rule.reload_signal action=<action> source_object_id=<X> published=<bool> reason=<r> trace_id=<id>` mỗi mutation.

### Rationale
- Low-cardinality (source_object_id label, < 10k cardinality OK).
- Alertable: Prometheus rule `(time() - mapping_cache_last_reloaded_seconds) > 1800` → warning.
- Log line giúp grep race scenario thực.

### Consequences
- ✅ Operator phát hiện stale cache trong < 5 phút.
- ⚠ Thêm 1 prometheus dependency nhỏ (đã có).

---

## ADR-06 — Lesson update: append L-3110 với pattern producer-conditional inverse

### Context
L-3110 (2026-05-18) cover "subscriber conditional, producer luôn on". Bug hiện tại là **đảo ngược**: "publisher conditional, subscriber luôn on". Cùng kết quả silent drop. Lesson chưa có entry cho mặt này → repeat bug.

### Decision
Sau khi User confirm fix, APPEND `agent/memory/global/lessons.md` entry mới:
> Pattern [Producer P publishes signal S conditionally gated by derived attribute D, while consumer C subscribes unconditionally]. Khi D resolve nil/empty (do upstream schema null hoặc JOIN fail) → P skip → C never triggered → cache/state stale silent. **Đúng**: Producer publish unconditionally khi mutation success. Payload mang identity primary key (không phải derived attribute). Audit: `grep "if.*derived.*!= nil.*Publish"` toàn codebase.

### Rationale
- L-3110 chỉ cover 1 mặt → bug đảo ngược vẫn xảy ra → cần append.
- Format Global Pattern A/B/X/Y theo CLAUDE.md §13.
- Áp dụng được ≥ 3 dự án: CDC, microservice event bus, cache invalidation pattern general.

### Consequences
- ✅ Codify learning để không repeat lần 4.
- ⚠ Lesson file tăng size — chấp nhận, file là append-only knowledge base.

---

## ADR-07 — CI grep gate cho conditional publish anti-pattern

### Context
Để FR-1 / AC-3 enforce lâu dài, cần CI catch khi dev mới thêm pattern `if X != nil { publish }`.

### Decision
Thêm CI step trong GitHub Actions:
```yaml
- name: Anti-pattern grep gate
  run: |
    # Pattern: conditional publish reload
    if grep -rn "ShadowTable.*nil.*Publish\|TargetTable.*nil.*Publish" \
       cdc-cms-service/internal/ \
       --include="*.go"; then
      echo "FAIL: conditional reload publish detected. See ADR-01."
      exit 1
    fi
```

### Rationale
- Cheap, fast (< 1s grep).
- Specific pattern, low false positive.
- Catch tại review time.

### Consequences
- ✅ Pattern không quay lại.
- ⚠ Maintainer phải hiểu lesson trong commit message khi update grep pattern.

---

## ADR-08 — KHÔNG dùng Redis pub/sub thay NATS

### Context
Có cám dỗ refactor sang Redis pub/sub vì "cache là Redis problem". User và team đã dùng NATS làm command bus.

### Decision
**KHÔNG.** Giữ NATS `schema.config.reload` subject. Lý do:
- NATS đã là command bus chính (`cdc.cmd.*`, `cdc.evt.*`).
- Mixing 2 message broker tăng surface bug + ops cognitive load.
- "Simplicity First" — fix gate, không refactor transport.

### Consequences
- ✅ Patch tối thiểu.
- ✅ Không thay đổi infra topology.

---

## Summary

| ADR | Decision |
|---|---|
| 01 | Bỏ conditional gate, always publish reload |
| 02 | Worker pre-flight post-reload count sanity + retry |
| 03 | CMS dispatch endpoint publish reload trước snapshot 50ms |
| 04 | Payload sang `source_object_id` (identity-first) |
| 05 | 2 metric + 1 log line observability |
| 06 | Append lesson L-3110 với inverse pattern |
| 07 | CI grep gate anti-pattern |
| 08 | KHÔNG đổi sang Redis pub/sub |
