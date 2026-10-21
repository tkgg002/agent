# Audit Report `anti_fake_evidence`

## 1. Từng dòng git diff
`git diff HEAD -- hooks tooling`: `agy_guard.py` (+CLAIM_RE/EV_RE, `claim_issues`, `current_fingerprint`, H4c), `mutation_gate.py` (`diff_anchor_issues`, `--base`, CRASHED, dòng MUTATION_GATE), `evidence_run.py` (mới), 2 file test mới.
- H4c chỉ lint dòng mới (trừ dòng có trong TargetContent) — không chặn tài liệu cũ.
- Khớp biên lai ↔ transcript đổi từ "nguyên văn cmd" sang "`evidence_run.py --kind <kind>` + không ghi tay 06_evidence" sau khi runtime cho false deny (biến shell/nháy).
- `CRASHED`: bổ sung khi thấy mutation_gate tính mọi exit≠0 là KILLED.

## 2. Chống mockup / smoke — trace runtime
Test dùng repo git thật trong tmp, file thật, subprocess thật. H4c chạy qua đúng lệnh hook với transcript thật (EV-20261021T103738-aa1a): deny/ask/deny đúng 3 ca.

## 3. Concurrency / timeout / continuity
`mutation_gate.py` sửa file tại chỗ: hook H6 chạy test song song đã báo fail giả 2 lần khi gate đang chạy (đã xác minh: chạy tuần tự sau đó đều xanh). Rủi ro cho phiên song song; chưa sửa trong scope này. Fingerprint không đổi khi commit, đổi khi sửa code.

## 4. Suy diễn / báo cáo láo
Mọi dòng khẳng định trong 06_validation trích biên lai EV; ca chưa kiểm được ghi riêng. Lần đầu H4c runtime cho false deny — đã sửa, không che.

## 5. Self-Improvement Loop
S5 → S3 → S4 hai lần (plan mục 4, 5). Lesson: biên lai chỉ giữ đuôi output nên verdict phải in ở dòng cuối (đã thêm `MUTATION_GATE ...`).

## 6. laya_router DG3
Xem `05_progress_anti_fake_evidence.md`.
