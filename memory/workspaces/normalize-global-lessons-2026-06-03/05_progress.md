# 05_progress — Normalize Global Lessons (APPEND-ONLY AUDIT LOG)

## [2026-06-03] Khởi tạo
- Nhận yêu cầu: thống kê + tổng hợp + sắp xếp + chuẩn hoá `lessons.md` theo hướng global.
- Thống kê sơ bộ: 530KB / 5.061 dòng / ~194–210 lesson / 134 chuẩn `## [DATE]` + ~76 lệch chuẩn / 750 tag / Fix-marker 15, Lesson-marker 5.
- Phát hiện xung đột Rule 11 (cấm overwrite Memory) → xác nhận User chọn: xuất file mới + append index; chuẩn hoá toàn bộ.
- Tính 9 chunk boundary tại separator `---`.
- Khởi tạo workspace + 00_context, 01_requirements, 02_plan, 08_tasks, 05_progress.

## [2026-06-04] Hoàn tất chuẩn hoá (execution)
- Dispatch 9 sub-agent song song đọc 9 chunk (ranh giới tại `---`), rewrite từng lesson → canonical @@-block, ghi /tmp/norm_part_NN.md, trả summary nhỏ (giữ context chính sạch).
- Tổng hợp lần 1: 226 block; marker LESSON=CAT=END cân bằng mọi part; 0 malformed.
- **PHÁT HIỆN MID-SESSION**: lessons.md tăng 5061→5109 dòng TRONG lúc chạy → user APPEND 3 lesson mới (Master-UI 06-03, Regex-DDL 06-03, VCS-granularity 06-04) nằm NGOÀI chunk 9 (≤5061). Gap-analysis bằng token độc nhất (`monorepo-of-repos`, `ddlIdentRe`, `_synced_at`) xác nhận 3 lesson thiếu → chuẩn hoá bổ sung part_10 → tổng **229**.
- Assembly (Python): group theo taxonomy 8 nhóm, sort ngày giảm dần → `lessons_global_normalized.md` (229 pattern, 2190 dòng, 100% Rule 13).
- APPEND index vào lessons.md (append-only); cập nhật số liệu index 226→229 trên block phái sinh. VERIFY: `head -5109 | md5 = cdbbc29722f23683f6b707b3991a8033` KHÔNG ĐỔI → Rule 11 OK (5109 dòng lesson gốc nguyên vẹn).
- Security scan file phái sinh: 0 raw password / 0 live connstring / 0 API key / 0 private IP.
- Lưu `tooling_assemble_lessons.py` vào workspace để tái lập.

## [2026-06-05] Cập nhật GEMINI.md — thêm Rule 15 (Check lại & Bảo trì Lesson)
- Backup `GEMINI.md` → workspace (`GEMINI.md.bak-before-recheck-rule-2026-06-05`) trước khi sửa (restore-point, repo không có git).
- Thêm **Rule 15**: (1) đọc bản normalized đầu phiên; (2) coverage-verification snapshot-during-mutation; (3) format lesson tại nguồn chống drift; (4) định kỳ re-generate; (5) check cuối phiên. Cập nhật cross-ref `/context-manager` (#7,#15). GEMINI.md 120→127 dòng; rule 0–15 đủ & đúng thứ tự (verified).
- Đề xuất Part 2 (gap analysis) chờ User chọn: A) sync GEMINI↔CLAUDE · B) VCS/restore-point · C) secret/PII trong memory · D) sub-agent context hygiene.

## [2026-06-05] Thêm Rule 16 — Feature Output Quality Gate (Definition of Done)
- Làm rõ ý User: rule cũ giữ nguyên; chỉ thêm rule lesson (Rule 15 ✓) + rule đảm bảo đầu ra mỗi feature chính xác/không bug.
- Backup `GEMINI.md.bak-before-quality-gate-2026-06-05` trước khi sửa (restore-point).
- Thêm **Rule 16** gồm 8 gate: G1 Requirement traceability · G2 Reproduce trước fix (red→green) · G3 Test thật không build-ok · G4 Edge-case/negative-path · G5 Chống regression · G6 Output correctness trên dữ liệu thật · G7 Adversarial self-review · G8 Bằng chứng vật lý trong workspace.
- Cập nhật cross-ref `/qa-agent` (#4,#16), `/security-agent` (#8,#16-G7). GEMINI.md 127→~146 dòng; rule 0–16 đủ & đúng thứ tự (verified).
- KHÔNG đụng CLAUDE.md (theo ý User giữ nguyên rule). Đề xuất sync để mở (optional).

## [2026-06-05] Sync GEMINI.md → CLAUDE.md
- Backup `CLAUDE.md.bak-before-sync-2026-06-05` (restore-point).
- Theo ý User (rule cũ giữ nguyên): CHỈ thêm Rule 15 & Rule 16 (condensed) vào CLAUDE.md, KHÔNG sửa rule 0–14 cũ. CLAUDE.md 82→~108 dòng.
- Cross-check: cả GEMINI.md và CLAUDE.md đều có Rule 15 (Check lại & Bảo trì Lesson) + Rule 16 (Feature Output Quality Gate, đủ 8 gate). 2 file đã đồng bộ ở phần rule mới.

## [2026-06-05] Thêm Rule 17–20 (gap analysis A/B/C/D) + sync 2 file
- User duyệt cả 4 đề xuất. Backup `*.bak-before-ABCD-rules-2026-06-05` cho cả GEMINI.md & CLAUDE.md.
- Thêm vào CẢ 2 file (GEMINI full + CLAUDE condensed): Rule 17 Constitution Sync (GEMINI↔CLAUDE) · Rule 18 VCS/Restore-point discipline · Rule 19 Secret/PII trong memory files · Rule 20 Sub-agent context hygiene.
- Verify: GEMINI 138→~210 dòng, CLAUDE 100→~125 dòng; cả 2 đều có rule 0–20; cross-check rule set khớp (tuân thủ chính Rule 17 vừa tạo).

## [2026-06-05] Triển khai Governance Enforcement Hooks (cú nhảy L4→L5)
- Tạo 5 hook script `agent/hooks/*.sh` (bash+jq) + `.claude/settings.json` (project) gọi script (qua skill update-config).
  - rule11 (PreToolUse/Write): BLOCK overwrite memory file đang tồn tại → ép APPEND.
  - rule19 (PreToolUse/Write|Edit): BLOCK ghi secret thô vào memory (high-precision, masked *** & prose "password" không bị false-positive).
  - rule7 (PostToolUse/Write|Edit): reminder cập nhật 05_progress khi sửa source.
  - rule14 (Stop): checklist pre-flight; session_start (SessionStart): tiêm startup protocol.
- VERIFY: pipe-test 5/5 PASS (cả block lẫn allow). settings.json valid JSON + schema-path ✓.
- PROVE LIVE trong CC thật: (a) rule7 PostToolUse fired khi Write `.go`; (b) rule11 PreToolUse DENY thật khi Write đè `/tmp/agent/memory/.../_denytest.md` (file giữ nguyên nội dung gốc). Test artifacts đã dọn.
- Residual gap đã ghi nhận: Bash `>`/`>>` bypass hook (chỉ cover tool Write/Edit); Stop hook fire mỗi turn (tắt qua /hooks nếu nhiễu). Chưa có KPI/metrics (cần cho full L5).

## [2026-06-05] KPI/Metrics + phát hiện layout thay đổi mid-session
- Xây `agent/tooling/governance_metrics.sh` (collector tự tính KPI từ dữ liệu thật) + trend-log append-only `agent/memory/global/governance_metrics.md`.
- **PHÁT HIỆN (Rule 15 snapshot-during-mutation)**: user đã tái cấu trúc giữa kỳ (mtime 11:29): `lessons.md` GIỜ = bản chuẩn hoá (229 lesson, `### [date]`); raw audit-log cũ archived thành `ls_old.md`; file `lessons_global_normalized.md` riêng đã xoá. → Run metrics đầu tiên ra số rác (đọc file cũ không còn) → đã sửa script theo layout mới (catalog=lessons.md, raw=ls_old.md), xoá snapshot rác, chạy lại sạch.
- KPI snapshot đầu: patterns=229, tags=499, catalog-fmt=99%, workspaces=110 (progress 91%), rules 21/21 sync OK, hooks=5.
- **Recidivism top**: #process-governance(97) #root-cause(74) #verification(72) #cdc(57) #observability(31) #silent-drop(27) #coupling(26) → lớp lỗi tái diễn lớn nhất là PROCESS/kỷ luật, không phải kỹ thuật.
- ⚠️ FLAG cần reconcile governance theo layout mới: (1) Rule 15 & session_start hook trỏ `lessons_global_normalized.md` (đã xoá); (2) Rule 15 nói "lessons.md = raw audit-log append-only" nhưng giờ lessons.md là catalog (raw = ls_old.md); (3) rule11 hook chặn Write-overwrite lessons.md → xung đột nếu muốn re-generate catalog vào lessons.md. → Chờ user duyệt hướng sửa.

## [2026-06-05] Reconcile governance theo layout mới
- Backup 3 file trước khi sửa (Rule 18): GEMINI.md, CLAUDE.md, session_start_protocol.sh.
- Rule 15 (GEMINI + CLAUDE, đồng bộ Rule 17) viết lại theo layout mới: lessons.md = CATALOG chuẩn hoá (đọc đầu phiên); ls_old.md = raw audit-log archived; lesson mới CHÈN bằng Edit theo `### [date]` (không Write đè — hợp rule11); re-sort dùng write-temp→mv; tích hợp chạy governance_metrics.sh vào loop.
- session_start hook trỏ lessons.md (catalog) + ls_old.md thay cho file đã xoá.
- Giải quyết 3 flag: (1)(2) hết ref hỏng — chỉ còn note "đã gộp" cố ý; (3) tension rule11 giải bằng Edit-insert + write-temp→mv.
- VERIFY: rule sync 21/21; hook JSON hợp lệ trỏ đúng; metrics xanh (229 patterns, 99% fmt, health ✓).
- Còn lại (optional, chưa làm): tiêu đề nội bộ của lessons.md vẫn ghi "# lessons_global_normalized.md" + mục Maintenance cũ — cosmetic, để user quyết.
