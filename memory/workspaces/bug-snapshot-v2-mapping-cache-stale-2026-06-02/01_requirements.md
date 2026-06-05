# 01_requirements.md — Functional / Non-functional / AC

> Tham chiếu `00_context.md` + `10_gap_analysis.md` (G1 root + G2 amplifier + G6 observability).

---

## 1. Functional Requirements

| ID | Requirement | Layer | Acceptance signal |
|---|---|---|---|
| **FR-1** | Mọi mutation `mapping_rule_v2` (status / is_active / source_field / target_column / jsonpath / transform_fn) PHẢI publish reload signal, **không gate bởi shadow_table resolution** | CMS approve handler | `grep` codebase: 0 instance `if ... ShadowTable != nil ... PublishReload`; 100% mutation paths covered |
| **FR-2** | Reload signal PHẢI chứa `source_object_id` (không chỉ shadow_table) → worker resolve binding tự | NATS contract | NATS msg payload có field `source_object_id int64` |
| **FR-3** | Worker subscribe `schema.config.reload` PHẢI re-load `mappingCache` đầy đủ khi nhận signal | Worker | `ReloadAll()` được call trên mỗi msg |
| **FR-4** | `runSnapshot` pre-flight `ReloadAll` PHẢI verify cache count ≥ DB count cho source_object_id; nếu drift → retry 1 lần | Worker snapshot runner | Post-reload count check log line, retry counter metric |
| **FR-5** | Approve handler PHẢI log explicit khi publish reload (success / skip lý do) | Observability | Log line `mapping_rule.reload_signal action=approved source_object_id=X published=true/false reason=...` |
| **FR-6** | Worker PHẢI expose metric `mapping_cache_size{source_object_id}` + `mapping_cache_last_reloaded_seconds` | Observability | `/metrics` endpoint show 2 metric mới |
| **FR-7** | Snapshot.v2 dispatch endpoint (CMS) PHẢI publish reload TRƯỚC khi dispatch snapshot command (defense-in-depth) | CMS snapshot handler | Order: publish reload → wait 200ms → publish snapshot |
| **FR-8** | Test reproduce race: approve + snapshot dispatch trong < 50ms PHẢI có rule mới trong shadow | Integration test | `TestSnapshotV2_AfterApprove_NoRace` pass |

---

## 2. Non-functional Requirements

| ID | Requirement | Rationale |
|---|---|---|
| **NFR-1** | Patch tối thiểu — chỉ chạm file cần thiết (CMS approve + worker reload + 1 test mới) | Simplicity First |
| **NFR-2** | KHÔNG refactor cache architecture (TTL, fallback DB, multi-cache SSoT) | Out-of-scope, future workspace |
| **NFR-3** | KHÔNG đổi `mapping_rule_v2` schema | Schema stable |
| **NFR-4** | NATS publish thêm KHÔNG vượt > 5ms latency cho approve handler | Approve UX responsive |
| **NFR-5** | Worker `ReloadAll` thêm count check KHÔNG > +10ms cho snapshot pre-flight | Snapshot startup fast |
| **NFR-6** | Reload signal PHẢI backwards-compat — worker cũ nhận msg payload mới không crash | Deploy rolling safety |
| **NFR-7** | Log line phải có `trace_id` / `request_id` để debug được race scenario | Observability |
| **NFR-8** | Metric mới phải có label `source_object_id` đủ low-cardinality (source_object_id < 10k) | Prometheus safe |

---

## 3. Acceptance Criteria

### AC-1: Reproduce race (TRƯỚC fix)
```bash
# Setup: testcontainers Postgres + NATS + Redis
# 1. Insert source_object_id=66, shadow_binding active, 0 mapping rules
# 2. Goroutine A: approve mapping rule (set status='approved')
# 3. Goroutine B: dispatch snapshot.v2 cho source 66 (delay 5ms sau goroutine A)
# 4. Wait snapshot done
# 5. Assert: shadow table có column theo rule mới approved
# EXPECT (before fix): FAIL — column missing
```

### AC-2: Fix pass (SAU fix)
Cùng test scenario AC-1 — EXPECT PASS.

### AC-3: Conditional skip eliminated
```bash
# Audit codebase
grep -rn "ShadowTable.*nil.*PublishReload\|nats.*Publish.*Reload" cdc-cms-service/internal/
# EXPECT: 0 match patterns kiểu "if x != nil then publish"
# Tất cả publish phải unconditional (hoặc gated bởi NATS client init only)
```

### AC-4: Log line on every reload event
```bash
# Trigger 5 mapping rule mutations
# Tail log
grep "mapping_rule.reload_signal" cms.log | wc -l
# EXPECT: ≥ 5 (mỗi mutation 1 log line)
```

### AC-5: Worker post-reload count check
```bash
# Trigger snapshot.v2 với DB có 10 approved rules
# Tail worker log
grep "snapshot.preflight.reload" worker.log
# EXPECT: log line có "cache_count=10 db_count=10 drift=0"
# Nếu drift > 0 → log "RETRY" + counter metric ++
```

### AC-6: Metrics exposed
```bash
curl localhost:9090/metrics | grep mapping_cache
# EXPECT (≥ 2 line):
# mapping_cache_size{source_object_id="66"} 3
# mapping_cache_last_reloaded_seconds 5.234
```

### AC-7: Defense-in-depth dispatch order
```bash
# Trace cdc.cmd.snapshot.v2 dispatch
# EXPECT order in NATS log:
# 1. schema.config.reload (from snapshot dispatch endpoint)
# 2. cdc.cmd.snapshot.v2 (200ms sau)
```

### AC-8: Backwards-compat
Deploy worker mới với CMS cũ (chỉ publish `shadow_table` không `source_object_id`):
- Worker không crash khi field missing.
- Fall back behavior: ReloadAll() đầy đủ (không filter theo source).

---

## 4. Out-of-Scope (re-confirm)

| Out | Lý do |
|---|---|
| Refactor cache architecture sang Redis pub/sub | Over-engineer, NFR-2 |
| Đổi schema `mapping_rule_v2` | Stable, NFR-3 |
| Đụng V1 `cdc_table_registry` mapping | V1 không có bug này |
| Future SSoT invalidation registry (G5) | Future workspace |
| Bug `_gpay_id NULL` | Workspace riêng `bug-gpay-id-trigger-contract-2026-06-02` |

---

## 5. Risk Register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| R1 | Always-publish gây spam NATS subject khi bulk approve | LOW | NATS throughput thoải mái, worker idempotent ReloadAll |
| R2 | Worker chưa rolling restart sau commit `0289fe4` → cache key string vs int64 | HIGH | Doc deploy: bắt buộc restart all worker pod sau merge commit |
| R3 | Post-reload count check tăng DB load | LOW | 1 query COUNT(*) extra mỗi snapshot, negligible |
| R4 | Snapshot dispatch endpoint publish reload trước → tăng 1 NATS msg | LOW | Acceptable cho UX |
| R5 | Race chưa eliminate 100% (replica lag vẫn tồn tại) | MED | Post-reload count check retry 1 lần; nếu vẫn drift → log WARN + 503 |
| R6 | Test integration race khó deterministic | MED | Use `chan` để synchronize goroutine, không sleep |
| R7 | Lesson L-3110 cần update (cover producer-conditional) | LOW | Append global lessons.md sau User confirm |

---

## 6. Definition of Done

- [ ] AC-1 reproduce FAIL trước fix
- [ ] AC-2 → AC-7 pass sau fix
- [ ] AC-8 backwards-compat verified với deploy mock
- [ ] G1 + G2 fix merged
- [ ] G6 observability landed
- [ ] Lesson L-3110 updated (append global, không overwrite)
- [ ] CI grep gate (AC-3) thêm vào pipeline
- [ ] `05_progress.md` log đầy đủ
- [ ] User sign-off
