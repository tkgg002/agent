# Plan G-11 master_binding hyphen — REV2 (max-Brain iter#40)

> **Author**: max-Brain | **Date**: 2026-05-07 ICT | **Workspace**: `feature-cdc-system-refactor`
> **Source REV1**: `02_plan_g11_master_bind_hyphen_2026-05-07.md` (iter#10, đề xuất Phương án X = thi công helper `normalizePGIdent` tại `provisioning_orchestrator.go:409`).
> **REV2 trigger**: iter#34 phát hiện shadow_binding cũng có row hyphen (id=53) + iter#40 phát hiện helper `naming.NormalizeIdentifier` ĐÃ có sẵn trong source và mod hôm nay 13:55 ICT.

---

## §1 Real-evidence iter#40

### §1.1 Helper đã tồn tại và đúng

File: `centralized-data-service/internal/naming/naming.go` (modified 2026-05-07 13:55)

```go
func NormalizeIdentifier(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    var b strings.Builder
    lastUnderscore := false
    for _, r := range s {
        isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
        if isAlphaNum {
            b.WriteRune(r); lastUnderscore = false; continue
        }
        if !lastUnderscore { b.WriteByte('_'); lastUnderscore = true }
    }
    out := strings.Trim(b.String(), "_")
    if out == "" { return "unknown" }
    if len(out) > 63 { out = out[:63] }
    return out
}
```

→ `refund-requests` → `refund_requests` ✓ ĐÚNG.

### §1.2 Callsite trace

| Path | Line | Status | Risk |
|---|---|---|---|
| `service/provisioning_orchestrator.go:411` (master_binding INSERT) | `masterTable := naming.NormalizeIdentifier(src.SourceObjectName)` | ✅ NORMALIZED | LOW |
| `admin/source_register.go:172` (shadow_binding INSERT initial) | `shadowTable := naming.NormalizeIdentifier(req.SourceObjectName)` | ✅ NORMALIZED | LOW |
| `handler/provisioning_step_handlers.go:317` (`upsertShadowBinding`) | `t.SchemaName, t.TableName` (caller-supplied) | ⚠️ DEPENDENT on caller | MEDIUM — cần verify shadowTarget construction |

### §1.3 Running binary state

- Worker binary `/tmp/cdc-worker-host` PID 90006 build 11:22 ICT
- `naming.go` modified 13:55 ICT (= 2h33m sau worker build)
- → **Worker đang chạy KHÔNG có helper fix**. Stale rows id=31 (master) + id=53 (shadow) inserted before fix landed.

---

## §2 Phương án X — REV2

**Cũ (REV1 iter#10)**: Thi công helper `normalizePGIdent` tại boundary `provisioning_orchestrator.go:409`.

**Mới (REV2 iter#40)**: Helper đã có. Còn 3 bước hoàn tất:

### §2.1 Verify shadowTarget construction (1 file read)

Trace caller của `upsertShadowBinding` để xác định:
- `shadowTarget.TableName` được build như thế nào?
- Có pre-normalize không?
- Nếu KHÔNG → cần thêm 1 callsite normalize hoặc thay đổi line 329 thành `naming.NormalizeIdentifier(t.TableName)`.

**Effort**: 5 phút source trace + 1 patch nếu cần.

### §2.2 Worker rebuild + restart (gated)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go build -o /tmp/cdc-worker-host.new ./cmd/worker

# Boss verb `swap worker` (analogous to swap cms):
cp /tmp/cdc-worker-host /tmp/cdc-worker-host.preG11.bak
kill -TERM 90006 && sleep 2
mv /tmp/cdc-worker-host.new /tmp/cdc-worker-host
nohup /tmp/cdc-worker-host > /tmp/cdc-worker-host.log 2>&1 &
sleep 3 && curl -s http://127.0.0.1:8082/health
```

**Effort**: 5 phút. Boss-gated (kill shared service).

### §2.3 Backfill SQL — stale rows

```sql
-- fix_g11_stale_rows_2026-05-07.sql

BEGIN;

-- 1. master_binding id=31 hyphen → underscore
UPDATE cdc_system.master_binding
SET master_table = 'refund_requests',
    physical_table_fqn = REPLACE(physical_table_fqn, 'refund-requests', 'refund_requests'),
    updated_at = NOW()
WHERE id = 31 AND master_table = 'refund-requests';

-- 2. shadow_binding id=53 hyphen → underscore
UPDATE cdc_system.shadow_binding
SET shadow_table = 'refund_requests',
    physical_table_fqn = REPLACE(physical_table_fqn, 'refund-requests', 'refund_requests'),
    updated_at = NOW()
WHERE id = 53 AND shadow_table = 'refund-requests';

-- 3. Reset src 44 from `failed` → `master_pending` để retry
UPDATE cdc_system.source_object_registry
SET provisioning_state = 'master_pending',
    last_step_error = NULL,
    updated_at = NOW()
WHERE id = 44 AND provisioning_state = 'failed'
  AND last_step_error LIKE '%invalid master_name%';

-- 4. Verify
SELECT id, object_code, provisioning_state, last_step_error
  FROM cdc_system.source_object_registry WHERE id = 44;
SELECT id, master_table FROM cdc_system.master_binding WHERE id = 31;
SELECT id, shadow_table FROM cdc_system.shadow_binding WHERE id = 53;

COMMIT;
```

**Effort**: 1 phút. Idempotent (WHERE clause guards re-run). Boss-gated (modifies shared DB).

---

## §3 Acceptance criteria (7)

1. ✅ `naming.NormalizeIdentifier("refund-requests")` returns `"refund_requests"` (verified iter#40 by code read).
2. ⏳ `upsertShadowBinding` caller trace — confirm shadowTarget pre-normalized OR add normalize at line 329.
3. ⏳ Worker rebuild emits binary `/tmp/cdc-worker-host.new` with naming.go ≥13:55 import.
4. ⏳ Worker restart `/health=ok` post-swap.
5. ⏳ Backfill SQL runs, master_binding id=31 → `refund_requests`, shadow_binding id=53 → `refund_requests`.
6. ⏳ src 44 advances from `master_pending` → `master_active` after worker re-process.
7. ⏳ Smoke: insert new Mongo source with hyphen → state machine completes without `failed`.

---

## §4 Comparison REV1 ↔ REV2

| Aspect | REV1 | REV2 |
|---|---|---|
| Helper status | "thi công mới" | "đã ship 13:55, cần verify scope" |
| Effort | ~30 min code | ~10 min verify+backfill |
| Scope | master_binding only | master_binding + shadow_binding |
| Risk | medium (new code) | low (existing code + data fix) |
| Boss verbs | `ship g11` | `ship g11` (worker rebuild + backfill) hoặc `defer g11` |

---

## §5 Recommendation

**Recommend `ship g11` SAU `swap` cms binary**:

1. Boss approve `swap` (D.1) — Flow 1 P1 happy-path mới (PG source) advance Boss output 1720 → +N.
2. Sau khi P1 happy-path xanh → Boss approve `ship g11` (D.3 REV2) — fix Mongo refund-requests stale flow (90 sec total: 5 phút verify + 5 phút worker rebuild + 1 phút SQL).

D.1 vẫn là gate critical (cms cần shadow DB để advance Path B). D.3 REV2 là post-Flow-1 cleanup, nhỏ và risk-low hơn REV1.

Hoặc `defer g11` tới phase 2 nếu Boss chỉ ưu tiên Flow 1 PG happy-path (G-11 không block PG source register, chỉ block Mongo refund-requests).

---

## §6 Pre-flight check

- §0 Vietnamese ✓
- §1 Brain Chairman only ✓ (zero source code edit)
- §3 Plan & Verify ✓ (real-evidence từ source read line 411, naming.go 41-67, helper trace)
- §11 APPEND-only ✓ (NEW file, not edit REV1)
- §12 Brain Code Prohibition ✓ (kế hoạch, không thi công)
- §14 Pre-flight ✓

---

— max-Brain (iter#40 — G-11 plan revision based on source trace; REV1 carry-able, REV2 actionable)
