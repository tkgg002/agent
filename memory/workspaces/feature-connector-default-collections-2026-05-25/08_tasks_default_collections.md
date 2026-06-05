# 08_tasks_default_collections — Task Checklist

> **Phase**: `default_collections`
> **Owner Muscle**: claude-sonnet-4-6 (default) hoặc claude-opus-4-7 (nếu user override)
> **Execution order**: TUẦN TỰ (M0 → M6)
> **Pre-condition**: User approve plan, ra verb "execute" / "go" / "muscle thực thi"

---

## M0 — Pre-flight audit verify

- [ ] **T0.1** Đọc `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx` lines 125-175 và 950-1000. Quote exact code vào `05_progress.md`. Verify `compactConfig` filter empty + `buildConnectorConfig` không inject default.
- [ ] **T0.2** Đọc `data-hub/cdc-cms-service/internal/api/system_connectors_handler.go` lines 150-210. Verify Create + Update handler không inject default `collection.include.list`, không validate field này.
- [ ] **T0.3** Tra connector class trong code: `grep -rn "MongoDbConnector\|io.debezium.connector.mongodb" data-hub/cdc-cms-service/` và `data-hub/cdc-cms-web/src/`. Note class FQDN + version (qua go.mod hoặc Kafka Connect plugin manifest).
- [ ] **T0.4** API test (cần local stack BE + Kafka Connect up): `curl POST /api/system-connectors` với body không có `collection.include.list`. Verify 200 OK + Kafka Connect `GET /connectors/<name>/config` không có key.
- [ ] **T0.5** Mongo + Kafkacat verify: insert doc vào collection mới → kafkacat consume topic mới → có event.
- [ ] **T0.6** APPEND `05_progress.md` "M0 done — hypothesis confirmed: runtime correct, gap = UX only. Debezium connector class = <FQDN>, version = <X.Y.Z>".

**Exit gate**: Nếu T0.4 hoặc T0.5 fail → STOP, escalate Brain re-plan (hypothesis sai).

---

## M1 — Audit list view + form rendering

- [ ] **T1.1** `grep -rn "collectionNames\|collection.include.list\|Collections" data-hub/cdc-cms-web/src/` — list tất cả call site.
- [ ] **T1.2** Đọc các file trong T1.1 — identify file/component nào hiển thị connector list / detail view. Note `file:line` vào `03_implementation_default_collections.md` Section 2.2.
- [ ] **T1.3** Đọc `data-hub/cdc-cms-web/package.json` — verify Antd version (`antd` dependency). Confirm `Form.Item` API `extra` + `tooltip` available trong version đó (Antd ≥ 4.x).
- [ ] **T1.4** Check `data-hub/cdc-cms-web/src/i18n/` hoặc `src/locales/` HOẶC `grep -rn "useTranslation\|i18next" data-hub/cdc-cms-web/src/`. Identify i18n setup.
- [ ] **T1.5** APPEND `05_progress.md` "M1 done — list view file: <path:line>, Antd version: <X>, i18n: <yes/no/convention>".

---

## M2 — Implementation: FE form hint + list display

- [ ] **T2.1** Edit `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx` Form.Item Collections (line ~966). Thêm `extra={...}` prop theo Edit #1 trong `09_tasks_solution_default_collections.md`. Tùy chọn: update `placeholder`.
- [ ] **T2.2** (Conditional T1.4 = yes) Thêm i18n key `connector.form.collections.extra` + `connector.list.collections.all` vào file locale tương ứng. Nếu T1.4 = no → hardcode tiếng Việt.
- [ ] **T2.3** Edit list view component (path từ T1.2). Wrap render function field `collection.include.list` với fallback `(All collections)` theo Edit #2 trong `09_tasks_solution_default_collections.md`.
- [ ] **T2.4** (Optional) Audit edit form (nếu component edit là cùng `SourceConnectors.tsx` thì T2.1 đã cover; nếu component khác → cũng thêm `extra`).
- [ ] **T2.5** APPEND `05_progress.md` "M2 done — files edited: <list>, diff lines: <count>".

---

## M3 — Build + lint verify

- [ ] **T3.1** `cd data-hub/cdc-cms-web && pnpm install` (idempotent). Lưu log `/tmp/default_collections_install.log`.
- [ ] **T3.2** `pnpm build 2>&1 | tee /tmp/default_collections_build.log`. Exit 0, NO new warning.
- [ ] **T3.3** `pnpm lint 2>&1 | tee /tmp/default_collections_lint.log` (nếu có script). Exit 0.
- [ ] **T3.4** `pnpm tsc --noEmit 2>&1 | tee /tmp/default_collections_tsc.log` (nếu TS). Exit 0.
- [ ] **T3.5** APPEND `05_progress.md` "M3 done — build PASS, lint PASS, tsc PASS, logs: <paths>".

**Exit gate**: Bất kỳ gate fail → fix root cause, KHÔNG bypass (no `--no-verify`, no skip).

---

## M4 — Smoke test E2E

- [ ] **T4.1** Spin up local stack (verify): `docker compose ps` HOẶC project-specific command. Healthcheck: FE dev, cdc-cms-service, Kafka Connect, Mongo, PG đều UP.
- [ ] **T4.2** Start FE dev: `pnpm dev` (background). Note port.
- [ ] **T4.3** Open browser, navigate Create Connector page. Fill required: name, connector.class = MongoDbConnector, mongodb.hosts, database.include.list. **Để TRỐNG Collections**. Submit.
- [ ] **T4.4** Verify toast success + connector trong list. Verify list view Collections cell hiển thị `(All collections)`.
- [ ] **T4.5** Verify Kafka Connect config: `curl -s http://localhost:8083/connectors/<name>/config | jq` — assert KHÔNG có key `collection.include.list`. Save output `/tmp/default_collections_smoke_create.log`.
- [ ] **T4.6** Mongo insert doc collection mới:
      ```
      mongosh "<uri>" --eval 'db.brand_new_test_coll.insertOne({_id:"smoke_2026-05-25", marker:"default_collections_smoke"})'
      ```
- [ ] **T4.7** Kafkacat verify topic mới: `kafkacat -b localhost:9092 -t cdc.<server>.<db>.brand_new_test_coll -C -e -o -1 | head -3`. Save `/tmp/default_collections_smoke_cdc.log`.
- [ ] **T4.8** Backward compat: tạo connector thứ 2 với `Collections = users,orders`. Verify Kafka Connect config có key `collection.include.list: "users,orders"`. Mongo insert vào `brand_new_test_coll` của DB này → verify topic `brand_new_test_coll` KHÔNG có event (chỉ topic users + orders có).
- [ ] **T4.9** Screenshot: (a) Form Create với hint visible, (b) list view với `(All collections)` cell, (c) list view với explicit list. Save `/tmp/default_collections_screenshots/`.
- [ ] **T4.10** Edge case TC-E-06: edit connector empty → nhập explicit → save. Verify Kafka Connect config update.
- [ ] **T4.11** Edge case TC-E-07: edit connector explicit → clear field → save. Verify Kafka Connect config bị xóa key.
- [ ] **T4.12** APPEND `05_progress.md` "M4 done — TC-E-01..07 PASS, evidence: <log paths + screenshot paths>".

**Exit gate**: Nếu TC-E-02 fail (CDC không capture collection mới) → hypothesis sai, STOP, re-audit Debezium version + config.

---

## M5 — Security review

- [ ] **T5.1** Run `/security-agent` trên `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx` + file list view (từ T1.2). Save `/tmp/default_collections_security.log`.
- [ ] **T5.2** Manual review: hint text không leak credential / DSN / secret. Render function không inject `dangerouslySetInnerHTML`.
- [ ] **T5.3** XSS check: nếu config map có giá trị chứa `<script>` → Antd auto-escape, không render execute. Test bằng input value `<script>alert(1)</script>` ở field Collections (nếu BE accept) → verify chỉ hiển thị text plain.
- [ ] **T5.4** APPEND `05_progress.md` "M5 done — security PASS / N findings: <list, severity>".

---

## M6 — Report + memory update

- [ ] **T6.1** Fill `report_default_collections_2026-05-25.md`:
  - Section 1 executive summary
  - Section 2 files changed (file path + diff line count + diff summary)
  - Section 3 verify gates results (M0..M5 với evidence)
  - Section 4 behavior changes (before/after table)
  - Section 5 screenshots inline hoặc link
  - Section 6 rollback plan
  - Section 7 lessons learned
  - Section 8 open items / defer
- [ ] **T6.2** APPEND `agent/memory/global/lessons.md` Global Pattern nếu có (vd: "FE-only UX fix khi BE đã đúng — verify pipeline contract end-to-end BEFORE edit").
- [ ] **T6.3** Update `agent/memory/global/active_plans.md` workspace `feature-connector-default-collections-2026-05-25` row → status DONE.
- [ ] **T6.4** APPEND `05_progress.md` "M6 done — phase default_collections COMPLETE — report at <path>".
- [ ] **T6.5** Pre-flight §14: list tất cả file trong workspace (`ls -la agent/memory/workspaces/feature-connector-default-collections-2026-05-25/`) — verify tồn tại vật lý đầy đủ. Cross-check với `00_context` Section 5 References + plan M6 deliverables.
- [ ] **T6.6** Report user verb hoàn thành. KHÔNG tự ý merge PR / push remote nếu user chưa yêu cầu (CLAUDE.md "Executing actions with care").

---

## Skip conditions

- Nếu local stack không lên được (KC down): M4 partial — chỉ verify FE rendering + payload qua mock. Defer CDC topic verify, note trong report.
- Nếu i18n setup phức tạp (T1.4 lớn): T2.2 fallback hardcode tiếng Việt.
- Nếu list view component không tồn tại / quá phức tạp: defer R3 sang sub-phase, chỉ làm R1+R2+R5 trong phase này. Note ADR-006 mới.

## Escalation

- Stuck > 3 lần ở 1 task → APPEND `05_progress.md` "STUCK at T#.#, escalate Brain re-plan", chờ user verb.
- Phát hiện gap mới ngoài scope → APPEND "Open items" trong report, KHÔNG tự ý expand scope.

## Code change guard

- TUYỆT ĐỐI: trước khi edit bất kỳ file `.tsx` / `.ts` / `.go` → re-read CLAUDE.md §12 (Brain Code Prohibition). Đảm bảo actor là Muscle (CC CLI), không phải Brain.
- TUYỆT ĐỐI: KHÔNG cheat DB / config / migration để pass smoke (CLAUDE.md §6, L-cheat-DB-ALTER-in-report).
