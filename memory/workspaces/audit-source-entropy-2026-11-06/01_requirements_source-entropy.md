---
tier: T2
mode: audit
suffix: source-entropy
---
# 01 Requirements — Source Entropy & False-OK Audit (data-hub)

## Yêu cầu User (2026-11-06)
"Làm feature này thì lỗi cái cũ, xong báo OK. Source càng làm càng rối, design pattern càng làm càng phá. Đọc cây thư mục + file, lên hướng giải quyết."

## Phạm vi
- `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web` (branch `recon-heal`).
- Audit Propose-Only: KHÔNG sửa source code cho tới khi User APPROVE.

## Technical Constraints (từ lessons)
- L018 No Goalpost Moving / L034 Zero-Mockup Evidence / L037 Honest Process State: mọi "OK" phải có lệnh + output full-suite.
- L015 No Silent Live-DB Test Skip: test live-DB fail phải báo rõ, không giấu.
- L024 Single-Responsibility Module / L039 Strict Scope Focus: 1 task = 1 scope, không lan.
- L023 Strict Workspace & Git: restore-point commit theo từng task.

## Definition of Done (của bản audit này)
- Có baseline đỏ/xanh thật của 3 repo, có root-cause gắn bằng chứng, có 1 phương án khắc phục duy nhất.
