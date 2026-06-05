# 01 — Requirements: DLQ Startup Audit

## User request (verbatim)
> "audit xem sao start nó quăng mớ log này"

## Constraints (CLAUDE.md / GEMINI.md)
- KHÔNG sửa code khi user yêu cầu audit
- KHÔNG cheat DB / config để che symptom
- Làm theo core systems, root-cause first
- Plan rõ ràng + solution cụ thể (kèm code demo nếu cần)
- Verify thực tế trước khi báo
- Phải tạo `report_*.md` ghi files changed + LOC

## Acceptance criteria
- [x] Đọc `lessons.md` trước (lesson #820 startup verify, #866 symptom vs upstream, #989 cron-driven replayer)
- [x] Đọc `GEMINI.md` xác định role (Muscle — Chief Engineer)
- [x] Phân tích full source `dlq_state_machine.go` + caller `worker_server.go`
- [ ] Trả lời 3 câu trong `00_context.md` với evidence (file:line refs)
- [ ] Liệt kê tối thiểu 2 phương án cải thiện + ưu/nhược, KHÔNG tự apply
- [ ] Tạo `report_dlq_startup_log_spam.md` (rỗng about changes vì là audit-only)
- [ ] Update `05_progress.md` (append-only)
