# 01 Requirements: Kiểm tra & Phân loại `git diff` trên 3 Repo CDC (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`)

tier: T1
suffix: cdc_3repos_diff
date: 2026-10-26

## 1. Yêu cầu của User
- Kiểm tra toàn bộ `git diff` hiện tại trên 3 repository của hệ thống CDC trong `/Users/trainguyen/Documents/work/data-hub`:
  1. `centralized-data-service`
  2. `cdc-cms-service`
  3. `cdc-cms-web`
- Xác định rõ hiện tại các thay đổi uncommitted / untracked đang thuộc những feature / đợt fix nào, phân nhóm chi tiết theo từng feature, từng repo và từng `file:line`.
- Giữ nguyên trạng thái Read-Only (Propose-Only, không tự ý sửa hay commit/reset code).
