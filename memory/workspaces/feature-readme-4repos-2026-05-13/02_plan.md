# 02 Plan — README refresh

## Sequence
1. Khởi tạo workspace (DONE bằng file này).
2. Scan parallel 4 repo: Makefile, cmd/, config sample, docs/, package.json.
3. Viết README per repo (1 file/step, không batch).
4. Verify (Read lại từng README).
5. APPEND `05_progress.md`.
6. Pre-flight check CLAUDE.md (§14).

## Risks
- Khác biệt thực tế binary vs project_context.md cũ (snapshot 2026-05-04, dự án có refactor 2026-05-07 Đợt J).
  → Mitigation: đọc thẳng cmd/ + Makefile để xác nhận binary thay vì copy số liệu cũ.
- README đè default Vite có thể mất `npm run` hint chuẩn.
  → Mitigation: giữ scripts từ package.json thực tế.

## Template skeleton (dùng cho 4 README)

```md
# <service-name>

> <One-line role trong CDC pipeline>

## Overview
…

## Tech stack
…

## Repository layout
…

## Prerequisites
…

## Run locally
…

## Configuration
…

## API / Surfaces
…

## Tests
…

## References
- Root architecture: `../architecture.md`
- Project rules: `../CLAUDE.md`
- BRD: `../brd-cdc-system.md`
```
