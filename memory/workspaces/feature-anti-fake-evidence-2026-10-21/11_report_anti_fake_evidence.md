# 11 — Report `anti_fake_evidence`

Đã làm: `tooling/evidence_run.py` (biên lai), H4c trong `agy_guard.py` (câu khẳng định phải có biên lai thật, mới, đúng số, đúng loại), `mutation_gate.py --base` (mutant phải đánh vào dòng vừa đổi) + `CRASHED` (crash không tính là bị bắt).

Kết quả: xem `06_validation_anti_fake_evidence.md` (mọi số liệu đều kèm EV).

## Còn mở
- Mutation gate sửa file tại chỗ → nên chạy mutant trên bản sao (worktree tạm).
- Chưa cập nhật skill `muscle-execute`/`qa-agent` hướng dẫn dùng `evidence_run.py` và `--base`.
