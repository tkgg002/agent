# 05_progress — APPEND-ONLY

> CLAUDE.md §7 + §11: TUYỆT ĐỐI KHÔNG xóa/sửa entry cũ. CHỈ APPEND.

---

## [2026-05-26 15:00] [Agent:claude-opus-4-7] Phase init + plan

**Actor**: Brain → Muscle (cùng session)
**Trigger**: User verb `1,2,3 làm luôn đi` (apply cả 3 option fix OTel log sampling).

**Pre-work**:
- ✅ Diagnose root cause tại `pkgs/observability/otel.go:161-180` + `config-local.yml:60`.
- ✅ Audit 5 call site milestone log via grep.
- ✅ Tạo workspace docs (00_context, 01_requirements, 02_plan, 05_progress).

**Next**: M1 → M2 → M3 → M4 → M5.

---

## [2026-05-26 15:25] [Agent:claude-opus-4-7] M1+M2+M3+M4 thực thi xong

**Actor**: Muscle (claude-opus-4-7)
**Tasks completed**: T1 (workspace), T2 (Option 1 config), T3 (Option 3 audit bypass), T4 (Option 2 milestone tags), T5 (verify build/vet).

**Files changed**:
- `centralized-data-service/config/config-local.yml` — `info: 0.1 → 1.0` (line 60). 1 LOC.
- `centralized-data-service/config/config-sample.yml` — `info: 0.1 → 1.0` (line 63). 1 LOC.
- `centralized-data-service/config/config-production.yml` — KHÔNG đụng (giữ `info: 0.05` cho cost).
- `centralized-data-service/pkgs/observability/otel.go`:
  - Thêm field `audited bool` vào `severityAwareCore` struct (line 104+).
  - Refactor `Check`: register chính `s` qua `ce.AddCore(ent, s)` thay vì gọi `s.Core.Check`. Sampling defer sang Write.
  - Mới `Write(ent, fields)`: apply sampling trừ khi `hasAuditField(fields)` hoặc `s.audited` (đã set qua With).
  - Mới `Sync()`: delegate inner.
  - Mới helper `hasAuditField([]zapcore.Field) bool` — detect `audit=true` boolean field.
  - Update `With(fields)`: propagate `audited` flag (`s.audited || hasAuditField(fields)`).
  - +50 LOC net.
- `centralized-data-service/internal/handler/command_handler.go` — thêm `zap.Bool("audit", true)` vào 4 log:
  - line 1195 `introspect.mongo.databases.start`
  - line 1209 `introspect.mongo.databases.ok`
  - line 1266 `introspect.mongo.collections.start`
  - line 1321 `introspect.mongo.collections.ok`
- `centralized-data-service/internal/service/schema_inspector.go` — thêm `zap.Bool("audit", true)` vào `schema drift detected (batch summary)` (line 162).

**Verification**:
- Build: `go build ./...` → EXIT=0 (log /tmp/otel_fix_build.log).
- Vet: `go vet ./...` → EXIT=0 (log /tmp/otel_fix_vet.log).
- Test: `go test ./pkgs/observability/...` → no test files (pre-existing, không trong scope).

**Design notes**:
- Audit bypass KHÔNG bypass mute: degraded sink vẫn drop audit log. Operator phải fix sink trước.
- Production giữ `info: 0.05` — milestone log vẫn 100% qua (do audit bypass). Noisy log (batch upsert ok, V2 reload, kafka discover) vẫn sample 5%.
- Performance impact: hasAuditField iterate fields O(n) — n thường < 10, negligible.

**Issues / Open items**:
- KHÔNG đụng `derived cores muted state` issue (atomic.Bool không share giữa parent + With-derived). Out of scope — workspace observation only.
- Chưa add unit test cho hasAuditField + Write audit bypass — defer khi user yêu cầu.
- Worker restart cần thiết để config-local mới apply. User tự run.

**Next** (chờ verb):
- `restart worker` → user tự restart, verify SigNoz nhận đầy đủ Info log.
- `unit tests` → Muscle add TC cho audit bypass + sampling.
- `commit` → stage + commit (1 repo: centralized-data-service).

---

