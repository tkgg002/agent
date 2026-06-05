# 02_plan_default_collections — Milestone Roadmap

> **Phase**: `default_collections`
> **Execution order**: TUẦN TỰ (M0 → M6)
> **Effort estimate**: 2h work + 30% buffer ≈ 2h40m
> **Risk level**: LOW (UI-only change, no runtime change)

---

## Strategy decision (xem `04_decisions_default_collections.md` ADR-001)

**Chọn**: **Phương án A — FE-only hint UX improvement** (không đụng BE / Debezium).

**Lý do**: Audit đã xác nhận runtime đúng. Gap duy nhất là UX. Minimal-impact fix theo CLAUDE.md §6 "Simplicity First".

---

## M0 — Pre-flight audit verify

> Mục đích: xác nhận hypothesis trước khi code. Nếu gãy assumption → re-plan ngay.

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T0.1 | Đọc lại `SourceConnectors.tsx:130-170, 960-985` xác nhận behavior `compactConfig` + form schema | Muscle | Quote exact code lines, không hypothesize |
| T0.2 | Đọc `system_connectors_handler.go:160-200` xác nhận BE không inject default / không validate field | Muscle | Quote exact code lines |
| T0.3 | Tra Debezium Mongo connector class trong code (tìm `connector.class` config template hoặc helper). Verify version qua go.mod / kafka-connect manifest | Muscle | Note version + connector class name vào `05_progress.md` |
| T0.4 | API test trên local: `curl POST /api/system-connectors` với body không có `collection.include.list` → verify response 200, connector created | Muscle | Status RUNNING, log output |
| T0.5 | Mongo doc: insert doc vào 1 collection bất kỳ → verify CDC event qua `kafkacat -t cdc.<topic>` hoặc Kafka UI | Muscle | Topic có message |

**Exit gate**: Nếu bất kỳ T0.x fail → STOP, escalate Brain re-plan. KHÔNG proceed M1.

---

## M1 — Audit list view + form rendering

> Mục đích: tìm chính xác file / component / field cần edit ở M2.

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T1.1 | `grep -rn "collectionNames\|collection.include.list\|Collections" data-hub/cdc-cms-web/src/` tìm tất cả call site liên quan | Muscle | List file:line |
| T1.2 | Đọc các file tìm được, identify component nào hiển thị connector list / detail | Muscle | Note vào `03_implementation_default_collections.md` |
| T1.3 | Verify Antd version (`package.json`) → confirm `Form.Item` API `extra` / `tooltip` available | Muscle | Note version vào progress |
| T1.4 | Check i18n setup (`src/i18n/` hoặc `src/locales/`). Nếu có → identify key naming convention | Muscle | Note convention |

**Exit gate**: Đầy đủ context để M2 viết edit chính xác.

---

## M2 — Implementation: FE form hint + list display

> Reference code-level: xem `09_tasks_solution_default_collections.md`.

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T2.1 | Edit `SourceConnectors.tsx:966-969`: thêm `extra` text vào `<Form.Item>` Collections | Muscle | Diff applied, file builds |
| T2.2 | (Optional, nếu T1.4 có i18n) Thêm key i18n thay vì hardcode | Muscle | i18n file updated |
| T2.3 | Edit list view component (path xác định ở T1.2): khi render field collections, nếu empty/null → hiển thị `(All collections)` italic gray | Muscle | Diff applied |
| T2.4 | (Optional) Cập nhật placeholder thành `users,orders (để trống = tất cả)` HOẶC giữ placeholder cũ và rely on `extra` text | Muscle | Wording decided in ADR-002 |

**Exit gate**: M2 build + lint pass cục bộ.

---

## M3 — Build + lint verify

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T3.1 | `cd data-hub/cdc-cms-web && pnpm install` (nếu chưa) | Muscle | Exit 0 |
| T3.2 | `pnpm build` HOẶC `pnpm run build` | Muscle | Exit 0, no NEW warnings |
| T3.3 | `pnpm lint` (nếu có script) | Muscle | Exit 0 |
| T3.4 | `pnpm tsc --noEmit` (typecheck) | Muscle | Exit 0 |

**Exit gate**: Tất cả gates PASS, log save vào `/tmp/default_collections_*.log`.

---

## M4 — Smoke test E2E

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T4.1 | Spin up local stack (FE dev server + cdc-cms-service + Kafka Connect + Mongo) | Muscle | All services healthy |
| T4.2 | Open FE form Create Connector kiểu Mongo, fill required, để TRỐNG Collections, submit | Muscle | Connector created, 200 OK |
| T4.3 | Verify trên Kafka Connect REST: `curl http://<connect>:8083/connectors/<name>/config` — assert KHÔNG có key `collection.include.list` | Muscle | jq verify |
| T4.4 | Mongo insert doc vào collection MỚI (chưa từng được listed): `db.brand_new_coll.insertOne({...})` | Muscle | Insert success |
| T4.5 | Kafkacat consume topic `cdc.<server>.<db>.brand_new_coll` → verify có event | Muscle | Event present, payload đúng |
| T4.6 | Backward compat: tạo connector thứ 2 với explicit `users,orders` → verify chỉ topic users + orders có event | Muscle | brand_new_coll NOT capture |
| T4.7 | List view UI: verify connector empty hiển thị `(All collections)`, connector explicit hiển thị `users, orders` | Muscle | Screenshot |

**Exit gate**: All A1..A6 acceptance criteria PASS.

---

## M5 — Security review

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T5.1 | Run `/security-agent` trên `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx` + list view file | Muscle | No HIGH/CRITICAL |
| T5.2 | Manual check: hint text không leak credentials, không inject HTML/script | Muscle | XSS-safe (Antd `extra` auto-escape) |

---

## M6 — Report + memory update

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T6.1 | Fill `report_default_collections_2026-05-25.md` với evidence thực tế: file changed, diff summary, screenshot, smoke output | Muscle | Report complete |
| T6.2 | APPEND lesson vào `agent/memory/global/lessons.md` nếu có pattern mới (vd "FE-only fix khi BE đã đúng — verify pipeline contract first") | Muscle | Lesson abstracted thành Global Pattern |
| T6.3 | Update `agent/memory/global/active_plans.md` workspace row → DONE | Muscle | Row updated |
| T6.4 | APPEND `05_progress.md` "M6 done — phase default_collections COMPLETE" | Muscle | Audit log entry |
| T6.5 | Pre-flight §14: list tất cả file workspace, verify tồn tại vật lý, không có "shadow file" thảo luận trong chat mà thiếu file | Muscle | Checklist trong report |

---

## Decision tree (nếu lệch khỏi plan)

```
M0 fail (hypothesis sai)
  → BE thực ra reject empty? → Re-scope thành phase backend-default-injection
  → Debezium version khác default? → Re-scope thành phase debezium-config-default

M1 fail (không tìm được list view component)
  → Defer R3 (display) sang sub-phase
  → M2 chỉ làm hint (R1 + R2 + R5)

M3 build fail
  → Đọc error → fix → re-run
  → Stuck 3 lần → STOP, escalate

M4 smoke fail (CDC ko capture collection mới)
  → Hypothesis M0 sai
  → STOP, re-audit Debezium config

M5 security HIGH
  → Fix, re-review, không skip
```

## Effort breakdown

| Milestone | Estimate |
|---|---|
| M0 audit | 30m |
| M1 grep + read | 20m |
| M2 edit | 30m |
| M3 build/lint | 15m |
| M4 smoke | 30m |
| M5 security | 10m |
| M6 report | 25m |
| **Total** | **2h40m** |

## Skip conditions

- Nếu local stack không lên được (Kafka Connect down) → M4 smoke partial: chỉ verify FE rendering + config payload, defer CDC topic verify khi infra available. Note trong report.
- Nếu i18n setup phức tạp (M1.4) → fallback hardcode tiếng Việt theo CLAUDE.md §0.

## Escalation

- Stuck > 3 lần ở 1 task → APPEND `05_progress.md` "STUCK at T#.#, escalate Brain re-plan", chờ user verb.
- Nếu phát hiện thêm gap ngoài scope → KHÔNG tự ý expand, APPEND vào "Open items / Defer" trong report.
