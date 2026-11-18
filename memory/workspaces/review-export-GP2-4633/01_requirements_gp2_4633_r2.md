# 01. Requirements — `review-export-GP2-4633` (lượt chạy lại, suffix `gp2_4633_r2`)

- **tier:** T2
- **suffix:** `gp2_4633_r2`
- **repo:** `/Users/trainguyen/Documents/work/centralized-export-service`
- **range:** `origin/develop` (`3eef807`) .. `origin/feat/GP2-4633` (`7d9bcf1`)
- **Ngày:** 2026-11-18 | **Agent:** Claude Sonnet 5.5

## Yêu cầu
User yêu cầu chạy lại `/diff-review` **như chưa có gì** (không tái dùng finding/claims/EV cũ của suffix `gp2_4633`), trong đó Design lens (R3) phải đối chiếu từng file mới với file canonical của repo (B-DES-03), không chỉ lấy lại F3/F3b. Đồng thời đo thời gian từng bước để tìm điểm gây chậm của 1 task.

## Ràng buộc Thực thi (từ lessons)
- `L001`: kiểm tồn tại file trước khi tạo (suffix `_r2` chưa tồn tại — đã kiểm).
- `L049`/`L030`: Propose-Only — không sửa source repo.
- `L010`/`L012`: mọi claim có `REF:` absolute path khớp file thật tại head `7d9bcf1`.
- `L062`: không suy diễn từ test in-memory.

## Definition of Done
- `scope_gp2_4633_r2.json`, 4 lens có finding kèm REF, `claims_dg3_gp2_4633_r2.json`, `06_laya_summary_*` (Mode A + B), `EV-` mới cho lint/unit/integration, `audit_report_gp2_4633_r2.md`, `verify_governance.py` PASSED.
- Bảng thời gian từng bước (`timing`) trong `audit_report`.
