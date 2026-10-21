# 06 — Validation `anti_fake_evidence`

- Hook tests: Ran 60 tests, OK (EV-20261021T103714-c9a2)
- Tooling tests: Ran 47 tests, OK (EV-20261021T103714-9f02)
- Mutation gate --base HEAD: gate=PASS killed=7/7 (EV-20261021T103719-b403); M1–M7 phủ claim-không-EV, EV cũ, sai số, runtime=test runner, diff-anchor, CRASHED, biên lai tự viết.
- H4c chạy thật qua `agy_guard.py pre-write` với transcript thật (EV-20261021T103738-aa1a): không biên lai → deny; biên lai đúng → ask; sai số 99/99 → deny.

## Vòng 2 (H7 stop-audit 10:39) — sửa fingerprint theo cwd của biên lai
- Hook tests: Ran 61 tests, OK (EV-20261021T103937-6721)
- Tooling tests: Ran 47 tests, OK (EV-20261021T103938-a369)
- Mutation gate --base 74840c8: gate=PASS killed=7/7 (EV-20261021T103953-1f17)
- Runtime check validate-surface: status VALIDATED (EV-20261021T103943-5711)
- H4c runtime với transcript thật: deny / ask / deny (EV-20261021T104002-6b42)

## Chưa kiểm được
- IDE có gọi H4c ở mọi lần ghi hay không: không tự chứng minh từ trong phiên.
- `mutation_gate.py` sửa file tại chỗ: test chạy song song (hook H6, phiên khác) thấy code đang bị làm hỏng — đã gặp 2 lần trong phiên (H6 báo fail giả lúc gate đang chạy).
- Tổng kết sau sửa test: `06_mutation_summary_anti_fake_evidence_20261021T104510.json` (EV-20261021T103953-1f17).
