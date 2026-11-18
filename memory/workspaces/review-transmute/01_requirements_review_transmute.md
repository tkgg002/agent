---
tier: Audit
suffix: review_transmute
skill: diff-review
repo: /Users/trainguyen/Documents/work/data-hub/centralized-data-service
base: origin/develop (b0ff47cd)
head: recon-heal (e82ed6e5)
---
# Requirements — diff-review recon-heal vs origin/develop
- Propose-Only: chỉ đọc & báo cáo, không sửa source.
- 4 lens (Security/Logic/Design/Contract) x 3 lớp luật (L1 base, L2 fintech, L3 profile data-hub).
- Finding bắt buộc có REF path:start-end + trích code; verdict BLOCK/REQUEST_CHANGES/APPROVE.
- DoD: scope_review_transmute.json, review_report_review_transmute.md, claims_dg3, 06_laya_summary, EV-*.
- Lưu ý: working tree repo đang dirty (transmute_handler.go, ... chưa commit) — review chỉ dựa trên ref, KHÔNG gồm thay đổi chưa commit.
