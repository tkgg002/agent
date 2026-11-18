# 01. Requirements — `review-export-GP2-4633`

- **tier:** Audit (Propose-Only, skill `diff-review`)
- **suffix:** `gp2_4633`
- **repo:** `/Users/trainguyen/Documents/work/centralized-export-service`
- **range:** `origin/develop` (`3eef807`) .. `origin/feat/GP2-4633` (`7d9bcf1`) — 1 commit, 14 file, +863/-1
- **manifest:** không có `.agent-project.json` → nạp L1 Base + L2 pack `fintech`; Design lens đánh dấu finding layer/chokepoint là `HYPOTHESIS`.
- **Ngày:** 2026-11-17 | **Agent:** Claude Fable 5.1

## Yêu cầu
Review diff qua 4 lens (Security, Logic, Design, Contract), mọi finding có `REF: path:start-end` + trích code; verdict `BLOCK | REQUEST_CHANGES | APPROVE`; qua `laya_gate.py review` (Mode B) + `run --stage dg3` (Mode A); xuất `review_report_gp2_4633.md`.

## Bối cảnh liên quan (VERIFIED)
- Workspace `review-central-export-gp2-4633-2026-11-17` (Gemini-4-Carbon-High + lượt bổ sung của em lúc 15:28) đã có audit F1–F6 chờ APPROVE và claims `claims_dg3_gp2_4633_dr.json` (DG3 PASS `06_laya_summary_dg3_20261117T162230.json`). Lượt này tạo workspace mới theo đúng tên User chỉ định, **tái sử dụng bằng chứng đã có** (EV-, DG3) và chỉ chạy lại các bước còn thiếu (R0 scope vào ws mới, Mode B review, report), không lặp R1–R4 lần 2 (SKILL.md:66).
- Worktree cô lập đã tồn tại tại `~/.gemini/antigravity/brain/e65b1bd5-.../scratch/wt_gp2_4633` (`7d9bcf1`) → dùng để chạy test nếu cần, không checkout repo chính (`feat/GP2-5081`).

## Ràng buộc Thực thi (từ lessons)
- `L001`: kiểm tồn tại cấp **file** trước khi tạo 01/05/08 (đã làm, 0 file trùng).
- `L049`/`L030`: Propose-Only — không sửa source, không `APPROVE` thay User.
- `L010`/`L012`: mọi claim có `REF:` absolute path khớp file thật; Mode A cần ≥2 control_false + ≥1 control_true.
- `L062`: không suy diễn từ MongoMemoryServer 5 bản ghi ra hệ thống thật.
- `L066`: Direct Summary trước, bảng Executive Issues (`L042`).
