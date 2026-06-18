# 08_tasks

- [ ] T1. Backup transmute_schedule_handler.go (*.bak)
- [ ] T2. Sửa `Toggle`: prev-state lookup + catch-up dispatch off→on (+revert on fail)
- [ ] T3. `go build ./...` CMS = 0
- [ ] T4. Reproduce red→green: tắt realtime → INSERT shadow record → master miss → bật lại → master tự khớp
- [ ] T5. Restart CMS, smoke PATCH /schedules/:id (verify catchup_dispatched)
- [ ] T6. /security-agent self-review (G7): SQL parameterized, no injection, no secret log
- [ ] T7. 06_validation (bằng chứng) + 05_progress append + report_*.md (file đổi + LOC)
- [ ] T8. Governance pre-flight (Rule 14): file vật lý đủ
