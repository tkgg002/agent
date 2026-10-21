# 01 — Requirements: Audit độ phủ quy trình REA Clean-Room & Harness² trong `/agent` (`rea_cleanroom_gap`)

tier: T2
type: Audit (Propose-Only)
suffix: rea_cleanroom_gap
workspace: audit-rea-cleanroom-harness2-2026-10-21
status: PROPOSED (Waiting for User APPROVE)

## Yêu cầu User (2026-10-21)
Kiểm tra `/agent` hiện tại đã đảm bảo 2 thứ chưa:
1. Quy trình Clean-Room kiểu REA: REA điều tra → SPEC.md mỗi rule có evidence → Jev duyệt, trả rule thiếu bằng chứng → session builder mới tinh chỉ thấy SPEC.md → Haiku viết test + 3 mutant → /goal lặp tới khi rebuild PASS & 3 mutant FAIL; hiển thị SPEC.md trước, chỉ build khi User gõ "BẮT ĐẦU". Clean-room rule: *model thấy code gốc không viết code; model viết code không thấy code gốc*.
2. Tích hợp `github.com/google-research/harness2` (Propose–Probe–Compose + improvement harness).

## Phạm vi
Read-only. Không sửa source/hook. Đầu ra: `13_analysis_rea_cleanroom_gap.md` + DG3.
