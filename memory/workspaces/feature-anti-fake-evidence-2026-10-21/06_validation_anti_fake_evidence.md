# 06 — Validation `anti_fake_evidence`

- Hook tests: Ran 60 tests, OK (EV-20261021T103714-c9a2)
- Tooling tests: Ran 47 tests, OK (EV-20261021T103714-9f02)
- Mutation gate --base HEAD: gate=PASS killed=7/7 (EV-20261021T103719-b403); M1–M7 phủ claim-không-EV, EV cũ, sai số, runtime=test runner, diff-anchor, CRASHED, biên lai tự viết.
- H4c chạy thật qua `agy_guard.py pre-write` với transcript thật (EV-20261021T103738-aa1a): không biên lai → deny; biên lai đúng → ask; sai số 99/99 → deny.

## Chưa kiểm được
- IDE có gọi H4c ở mọi lần ghi hay không: không tự chứng minh từ trong phiên.
- `mutation_gate.py` sửa file tại chỗ: test chạy song song (hook H6, phiên khác) thấy code đang bị làm hỏng — đã gặp 2 lần trong phiên (H6 báo fail giả lúc gate đang chạy).
