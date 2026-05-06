# 08 — Tasks (Phase B1 + B2)

| ID | Task | Owner | Depends | Status |
|---|---|---|---|---|
| B1.1 | Commit 5 file Phase F1/F3 pending | Brain (git) | — | pending |
| B1.2 | Update architecture.md gỡ Airbyte mention | Brain (.md) | — | pending |
| B1.3 | Fill project_context.md + tech_stack.md | Brain (.md) | — | pending |
| B1.4 | Update active_plans.md | Brain (.md) | — | pending |
| B1.5 | APPEND lesson L-input-fallback-pattern | Brain (.md) | — | pending |
| B2.1.a | Inventory cdc-auth-service config + Makefile | Brain (read) | — | pending |
| B2.1.b | Build + start cdc-auth-service local | Brain (bash) | B2.1.a | pending |
| B2.1.c | Smoke /healthz + login endpoint | Brain (curl) | B2.1.b | pending |
| B2.2.a | Inventory cdc-cms-web env + API base | Brain (read) | — | pending |
| B2.2.b | npm run dev background | Brain (bash) | B2.2.a | pending |
| B2.2.c | Smoke FE root | Brain (curl) | B2.2.b | pending |
| B2.3 | E2E operator path 7 step | Brain (curl + psql) | B2.1.c, B2.2.c | pending |
| B2.4 | scripts/dev-up.sh | Brain (.sh) | B2.1.b, B2.2.b | pending |
| B-final | report_phase_b_completed_*.md + APPEND 05_progress | Brain (.md) | all above | pending |

## Note governance

- Nếu B2.1.b hoặc B2.2.b fail vì cần edit `.go/.ts/.js/.py/.sql` → STOP, escalate Muscle, ghi rõ điểm dừng vào `05_progress.md`.
- Mỗi step PASS → APPEND progress + mark task completed.
- Mỗi step FAIL → APPEND progress + diagnose + đề xuất fix (Muscle owner), không tự fix code.
