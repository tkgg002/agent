# 01 — Requirements: Chặn "test láo" tận gốc (`anti_fake_evidence`)

tier: T2
suffix: anti_fake_evidence
workspace: feature-anti-fake-evidence-2026-10-21
status: PROPOSED (Waiting for User APPROVE)

## Yêu cầu User (2026-10-21T10:23)
"tao thấy vụ test láo rất hay xảy ra. mày có giải pháp ko"

## Định nghĩa "test láo" (từ bằng chứng thật)
Báo cáo khẳng định PASS/chạy thật/KILLED mà không có output thật tương ứng, hoặc output có nhưng đến từ test giả (mock/stub/đường dẫn giả), hoặc output cũ trước lần sửa code cuối, hoặc mutant chọn né code vừa sửa.

## Phạm vi
Core agnostic `/agent` (hooks + tooling). Không sửa dự án đích. Propose-only tới khi User APPROVE.
