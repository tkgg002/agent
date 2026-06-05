# 02_plan.md — Kế hoạch xây dựng tài liệu so sánh

## Phase 1: Thu thập ground truth (parallel)
- **Explore Agent A** — cdc-control: routes, services, models, templates, schema, encryption, scheduler, auto-restart, CRUD connection_registry, multi-shadow support.
- **Explore Agent B** — cdc-cms-service + cdc-cms-web: handlers, commands, models, components, infra, observability, source/sink CRUD, shadow binding.

## Phase 2: Tổng hợp tài liệu
File chính: `10_gap_analysis.md` — gồm các section:
1. **Tổng quan kiến trúc** — tech stack, layering, runtime.
2. **Domain Model** — bảng entity × repo (Connection, Connector, Source, Sink, Shadow, Audit…).
3. **API/UI Endpoints** — bảng endpoint × repo.
4. **Tính năng connector lifecycle** — create/edit/delete/restart/list/status.
5. **Tính năng connection_registry** — CRUD + encryption + role_type.
6. **Tính năng shadow management** — single vs multi shadow, binding model.
7. **Tính năng schema sync** — Postgres DDL vs MongoDB index/view.
8. **Background jobs / scheduler** — auto-restart, sync status.
9. **Observability / Audit** — logs, alerts, probes.
10. **Security** — auth, encryption, secret management.
11. **Operational ergonomics** — UI/UX, error handling, retry, validation.
12. **Gap matrix** — bảng final feature × repo (CÓ/KHÔNG/PARTIAL).
13. **Điểm cdc-control chi tiết hơn**.
14. **Điểm cdc-cms chi tiết hơn**.

## Phase 3: Audit + Status
- Append `05_progress.md`.
- Update `07_status.md`.

## Definition of Done
- [ ] `00_context.md` — DONE (workspace init).
- [ ] `02_plan.md` — DONE (this file).
- [ ] `10_gap_analysis.md` — chứa đủ 14 section trên, ≥ 5 bảng so sánh.
- [ ] `05_progress.md` — append-only audit log.
- [ ] `07_status.md` — status report.
- [ ] KHÔNG sửa source code 3 repo (verify bằng `git status` hoặc visual diff).
