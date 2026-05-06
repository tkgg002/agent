# P3 Plan Critique — Gap Analysis (Boss Review 2026-05-06)

## Status: critique RECEIVED → fact-verified → recommendations đề xuất, CHỜ boss approve các trade-off architectural trước khi tiếp tục T3.6+.

---

## 1. Facts — Verified

| ID | Plan claim | Thực tế (verified) | Status |
|----|------------|--------------------|--------|
| F1 | `master_registry_handler.go:630` inline `ALTER ... RENAME` | File 616 LOC, line 598 gọi `h.swap.Swap(ctx, ...)` qua `service/master_swap.go`. **Đã extract — KHÔNG inline.** DoD `grep "ALTER TABLE.*RENAME" internal/api/` = 0 đã PASS từ trước. | ✅ Boss đúng |
| F2 | `registry_handler.go:148/268/316` SyncFromLegacy inline | Thực tế ở `:168 / :288 / :338`. Claim "inline synchronous" đúng — handler block request đợi sync xong. | ✅ Boss đúng (line off, claim đúng) |
| F3 | T3.1 — migration `052_create_cdc_jobs.sql` "mới" | File **đã tồn tại** trong `centralized-data-service/migrations/cdc/052_create_cdc_jobs.sql`. Worker đang chạy nên đã apply rồi. | ✅ Boss đúng — đã có |
| F4 | Item 14 evt = NEW cho 12 event subjects | `cdc.evt.provisioning.step_completed` đã có (`worker/provisioning_emit.go`), `cdc.evt.transmute.completed` đã có (`worker/transmute_handler.go:142`). | ✅ Boss đúng — REUSE thay NEW cho 2 cái |
| F5 | T3.9 "subscribe 12 evt subject mới" | `JobMonitor` (`worker/service/job_monitor.go`) đã hoạt động cho 2 evt. **Wildcard `cdc.evt.>` là chuẩn duy nhất hợp lý** — list 12 subject = duplication, scale issue khi thêm. | ✅ Boss đúng — đổi sang wildcard |

### Action items (facts)
- [F1] Re-frame plan section "Master Swap": "Swap synchronous → async (job model)", **không phải** "extract inline". Câu hỏi thiết kế: có cần move sang worker không? — xem G3 + Q1.
- [F2] Cập nhật line numbers trong plan: 168 / 288 / 338 thay 148 / 268 / 316.
- [F3] Bỏ T3.1 khỏi pending list. Mark là **DONE upstream** trong worker repo.
- [F4] Section "Companion 12 evt": tag rõ những cái nào REUSE (provisioning.step_completed, transmute.completed) vs NEW (≤10 còn lại).
- [F5] Đổi T3.9 từ "subscribe 12 subject" → **`cdc.evt.>` wildcard subscription**. Pattern-match `subject` field trong handler để dispatch logic. Code hiện tại của worker JobMonitor đã subscribe theo subject riêng cho 2 cái — nên unify: **CMS-side JobMonitor (mới, ở `cdc-cms-service/internal/service/job_monitor_cms.go`)** subscribe `cdc.evt.>`.

---

## 2. Lỗ hổng thiết kế — đề xuất fix

### G1 — Stuck job recovery (CRITICAL, phải có)

**Vấn đề**: `NATSCommandBus.Dispatch` insert job (status=pending) xong → publish NATS. Nếu publish fail (network blip / NATS down), job kẹt status=pending vĩnh viễn. Comment hiện tại "let JobMonitor flag stuck job" sai — JobMonitor chỉ subscribe `cdc.evt.>`, không quét timeout.

**Đề xuất**: Thêm task **T3.12 — `StuckJobReaper`** (cron, chạy cms-side, in-process):
- Mỗi 30s, query `SELECT id FROM cdc_system.cdc_jobs WHERE status='pending' AND created_at < NOW() - INTERVAL '30 seconds'`.
- Flip → `status='failed', error='stuck_pending: no NATS ack within 30s'`.
- Log warn với job_id để operator điều tra.
- Idempotent — race-safe vì worker handler chỉ flip pending → running, không touch failed.

**Effort**: ~0.5d. **Critical** vì là hard floor cho SLO "không kẹt job".

---

### G2 — Idempotency workflow (BLOCKER cho T3.4)

**Vấn đề**: DDL `cdc_jobs.idempotency_key TEXT UNIQUE` đã có nhưng plan không định nghĩa:
- (a) Client retry cùng key → ON CONFLICT trả `job_id` cũ HAY tạo mới?
- (b) Quan hệ với `middleware/idempotency.go` (Redis 1h cache response) — replace hay coexist?

**Đề xuất**:
```sql
INSERT INTO cdc_system.cdc_jobs (..., idempotency_key)
VALUES (..., $key)
ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL
DO UPDATE SET id = cdc_jobs.id  -- no-op write trick to RETURNING old id
RETURNING id, status, created_at;
```
- Replay an toàn: client retry → cùng job_id, cùng status (status đã advance qua pending nếu lần trước thành công).
- **Coexist với Redis middleware**: Redis cache full response body (200ms TTL → 1h) cho **idempotent GETs**; cdc_jobs UNIQUE chuyên cho **write operations**. Different layer — không conflict.
- Document trong `04_decisions.md` để FE team biết retry contract.

**Effort**: ~0.25d (chỉ thay INSERT + 1 unit test).

---

### G3 — Worker DDL permission (BLOCKER cho T3.6 master-swap)

**Vấn đề**: T3.6 đề xuất move master-swap (`ALTER ... RENAME`) sang worker. Hiện tại worker connect DB user nào? Có OWNERSHIP của `public.<master>` không?

**Đề xuất**: Verify GRANT trước khi viết handler. 
```sql
SELECT grantee, privilege_type FROM information_schema.table_privileges
 WHERE table_schema='public' AND table_name LIKE 'master_%'
   AND grantee IN ('cdc_worker', 'gpay_admin');
```
Nếu worker chỉ có `INSERT/UPDATE/SELECT` → ALTER fail run-time. **Có 3 đường thoát**:
1. (Quick) Giữ Swap ở `cdc-cms-service` + wrap sync→async qua `cdc_jobs` table (KHÔNG publish NATS). HTTP returns 202 + job_id ngay; goroutine chạy Swap nền + UPDATE job khi xong. → **Đề xuất default**.
2. (Heavy) Grant worker `OWNER` trên master tables. Risk: worker process crash giữa ALTER → master table bị orphan name.
3. (Compromise) Worker yêu cầu DDL qua connection pool dùng admin user (separate pool). Phức tạp + double permission surface.

**Recommendation**: **Đường 1**. Architectural consistency (Q1 = b) thì wrap qua `cdc_jobs` đủ — không cần publish NATS. Boss confirm Q1 = a hay b sẽ chốt đường này.

---

### G4 — Item 14 collision

**Vấn đề**: `cdc.cmd.master-create` (legacy approve) share subscriber với `cdc.cmd.master.bind` (provisioning) tại `worker/worker_server.go:331`. Nếu thêm `cdc.evt.master-create.completed` riêng + giữ `provisioning.step_completed` → 1 master → 2 evt → JobMonitor double-update → race.

**Đề xuất**: **Unify**. Bỏ `cdc.evt.master-create.completed` mới. Cả 2 path emit cùng `cdc.evt.provisioning.step_completed` với `step_type='master_create'` discriminator. JobMonitor pattern-match step_type → update đúng row.

**Effort**: ~0.25d (worker handler điều chỉnh emit subject; CMS JobMonitor read step_type).

---

### G5 — Sync 7 commands cost > value (RE-DESIGN T3.4)

**Vấn đề**: Mapping create/update, wizard create/patch, master create/reject, alert ack — chạy sync trả 200 ngay (50ms thuần CPU). Nếu wrap qua `commandBus.Dispatch` → tạo job row + insert + return job_id → FE phải GET `/jobs/:id` để xem result → +1 round-trip cho mỗi action 50ms. **Gánh nặng > giá trị.**

**Đề xuất**: Tách 2 nhóm trong `app/ports/command_bus.go`:
```go
type SyncCommand interface {
    Type() string
    Validate() error
    Execute(ctx context.Context) (any, error)  // CHẠY in-request, không tạo job row
}

type AsyncCommand interface {
    Type() string
    Validate() error
    // Bus tạo job row + publish NATS, return job_id luôn
}
```
- `commandBus.Dispatch(ctx, cmd)` switch theo type interface — sync chạy thẳng, async đi qua job table.
- 7 sync commands thi công T3.4 KHÔNG tạo job row → không thay đổi UX.
- Pattern chuẩn industrial (Java/Spring tách `CommandHandler` / `AsyncCommandHandler`).

**Effort**: ~0.5d (đã có hybrid logic rồi — chỉ cần tách interface rõ ràng).

---

### G6 — Companion 12 evt regression risk

**Vấn đề**: T3.8 nói "extend mọi cmd handler hiện có → emit `cdc.evt.X.completed`". Thay đổi 12 handler một lúc → regression dễ.

**Đề xuất**: Tách thành **12 sub-task riêng + canary** (deploy 1 handler → JobMonitor verify event arrived + cdc_jobs row updated → merge tiếp). Effort thực tế: 12 × 0.25d = ~3d (plus buffer).

**Risk mitigation**: Mỗi handler thay đổi tối thiểu — chỉ thêm `defer publishCompleted(...)` ở cuối, không rewrite logic.

---

## 3. Effort revised: 5d → 7d (chấp nhận)

| Task | Effort | Note |
|------|--------|------|
| ~~T3.1 (migration)~~ | ~~0~~ | DONE upstream |
| T3.2 JobRepo | 0.5d | DONE |
| T3.3 NATSBus | 0.5d | DONE (+ HOTFIX wire-format) |
| T3.4 sync 7 cmd | 1d | Sau khi G5 split sync/async |
| T3.5 async 14 cmd | **1.5d (tăng từ 1d)** | Hiện đang ~75% — còn 3-4 site + tests |
| T3.6+T3.7 master-swap | 1d | Đường 1 (G3) — không cần worker permission |
| T3.8 companion evt | **2d (tăng từ 0.5-1d, canary 12 handler)** | G6 |
| T3.9 wildcard subscribe | 0.5d | F5 — `cdc.evt.>` |
| T3.10 GET /jobs/:id | 0.5d | DONE |
| T3.11 verify + report | 0.5d | |
| **T3.12 StuckJobReaper** (NEW) | **0.5d** | G1 |
| Buffer review | 0.5d | |
| **Total** | **~7-7.5d** | match boss estimate |

---

## 4. Trả lời 3 câu hỏi của boss

### Q1: Master Swap → worker async — purpose là (a) decouple HTTP timeout khỏi 3s lock, hay (b) kiến trúc nhất quán?

**Đề xuất**: **(b) kiến trúc nhất quán** — vì 3s lock không phải HTTP timeout pain point thực tế (Fiber default 30s read timeout). Nếu (b), **giữ Swap ở cdc-cms-service + wrap qua cdc_jobs** mà KHÔNG publish NATS (đường 1 trong G3). Lợi ích:
- Không cần worker DDL permission.
- Không cần migrate logic — chỉ thêm async wrapper.
- Worker repo workspace giảm scope (-1 task).
- HTTP returns 202 + job_id; goroutine chạy `h.swap.Swap` nền + UPDATE `cdc_jobs`.

Câu hỏi follow-up: nếu boss chốt (a), thì cần verify worker user OWNERSHIP trước (G3 đường 2/3).

---

### Q2: GET `/api/jobs/:id` — tier nào?

**Đề xuất**: **Shared (admin + operator)**. Operator cần xem progress của action mình trigger (vd: bấm "Backfill" thấy job_id → poll endpoint xem done chưa). Admin cũng cần để debug. Không có lý do gate riêng — endpoint chỉ trả status text, không expose secret.

Implementation: middleware `requireAuth()` (bất kỳ role logged-in), không phải `requireRole("admin")`.

---

### Q3: Idempotency-Key middleware (Redis 1h) — giữ song song hay rút?

**Đề xuất**: **Giữ song song** (coexist). Lý do:
- **Redis middleware**: cache **full HTTP response body** (idempotent GETs primarily, hoặc retry POST với cùng request body). TTL 1h. Layer **transport-level**.
- **`cdc_jobs.idempotency_key` UNIQUE**: dedup **business operation** (đảm bảo 1 backfill chỉ chạy 1 lần dù client retry 100 lần). Layer **business/domain-level**.

Khác mục đích → giữ cả 2. Document rõ trong `04_decisions.md`:
- Client gửi `Idempotency-Key: X-abc` header → middleware check Redis trước (response cache hit thì return ngay).
- Cache miss → handler chạy → `commandBus.Dispatch` → INSERT cdc_jobs với `idempotency_key=X-abc` → ON CONFLICT trả job_id cũ.
- Response cache vào Redis.

Trade-off: Redis miss + ON CONFLICT trùng → 2 lookup. Chấp nhận được vì cả 2 là indexed UNIQUE/key-based, mỗi cái <1ms.

---

## 5. TL;DR cho boss

**Plan vẫn đúng hướng** (commandBus + job tracker = chuẩn industrial). Boss chỉ ra **5 facts off-by-line/đã-có** + **6 design gaps cần fill trước T3.6**:

| Action | Owner | Status |
|--------|-------|--------|
| Update plan với F1-F5 (line numbers, REUSE evt, wildcard) | Brain | TODO |
| T3.12 StuckJobReaper (G1) thêm vào pending tasks | Muscle | TODO |
| G2 idempotency workflow → `04_decisions.md` | Brain | TODO |
| G3 worker permission verify TRƯỚC T3.6 hoặc chuyển đường 1 | Boss decide Q1 | BLOCKED |
| G4 unify provisioning.step_completed cho master-create | Brain | TODO |
| G5 split SyncCommand / AsyncCommand interface | Brain | TODO trước T3.4 |
| G6 12 sub-task canary cho companion evt | Brain | TODO planning |
| Effort 5d → 7-7.5d | Brain | UPDATE 02_plan |

**Boss approve** 3 câu trả lời (Q1=b, Q2=shared, Q3=coexist) thì Muscle tiếp tục T3.5 final sites + T3.12 reaper.
