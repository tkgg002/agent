# 08 — Tasks: Tối ưu hoá quy trình Agent

Chi tiết giải pháp: `13_analysis_process_optimization.md`. Mỗi phase cần User APPROVE riêng.

## Phase 0 — Chuẩn bị
- [ ] T0.1 `git commit` local repo `agent/` (restore-point `pre-optimization`) — KHÔNG push
- [ ] T0.2 Cài hook log `matcher: "*"` để đo tên tool / args / passthrough decision thực tế

## Phase 1 — Rules + Skills (U1, U2)
- [ ] T1.1 Tách `GEMINI.md` → Core ≤ 12KB; chuyển chi tiết sang skills
- [ ] T1.2 Global rule → pointer ≤ 1KB (bỏ bản copy trùng)
- [ ] T1.3 Tạo `~/.gemini/config/skills.json` trỏ `agent/skills`
- [ ] T1.4 Migrate 10 skill lõi, sửa: bỏ `git push`, bỏ `npm` cho Go, sửa path memory, thống nhất format log
- [ ] T1.5 Archive workflows còn lại → `workflows/_archive/`

## Phase 2 — Hooks (U3)
- [ ] T2.1 `~/.gemini/config/hooks.json` + `hooks/agy/{memory_protect,secret_scan,startup_reminder,done_gate}.sh`
- [ ] T2.2 Test 3 kịch bản deny/continue

## Phase 3 — Lessons (U4)
- [ ] T3.1 Script sinh `lessons/INDEX.md` + `lessons/topics/*.md` (không ghi đè gốc)
- [ ] T3.2 Đối soát 237 = 237; thăng cấp pattern tái phạm vào Core

## Phase 4 — Linter, Tier, Vệ sinh (U5–U7)
- [ ] T4.1 Sửa `verify_governance.py` (bắt buộc `--workspace`, tier, `12_implementation_plan*`)
- [ ] T4.2 Deprecate `organize_skills.sh`; archive skill ngoài domain
- [ ] T4.3 `.gitignore` cho `models.env`; Permission Grant cho `agent/memory`
