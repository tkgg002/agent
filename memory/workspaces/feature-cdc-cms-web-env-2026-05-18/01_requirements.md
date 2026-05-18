# 01_requirements.md — cdc-cms-web env config

## Functional requirements

| # | Yêu cầu | Acceptance |
|---|---|---|
| R1 | Mỗi môi trường (dev/staging/prod) phải có file env riêng commit-able | Tồn tại `.env.development`, `.env.staging`, `.env.production` ở root repo |
| R2 | Template tham chiếu (gitignored `.env.local` copy nguồn) | Tồn tại `.env.example` với 3 var + comment header (lesson 1898 style) |
| R3 | Vite resolve đúng file theo `--mode` | `npm run dev` → đọc `.env.development`; `npm run build:prod` → `.env.production`; `vite build --mode staging` → `.env.staging` |
| R4 | Không leak secrets/dev URLs khi `.env.production` commit | Prod env file dùng placeholder rỗng (`VITE_*_API_URL=`) HOẶC production URL công khai. Quyết định ở `02_plan.md` |
| R5 | `.gitignore` chặn `.env.[mode].local` rò rỉ | `.gitignore` đã có pattern `.env.*.local` — verify pattern chính xác |
| R6 | Backward compatible với Dockerfile | Dockerfile truyền ARG vẫn override file env (Vite resolution: ENV > .env.[mode] > .env) → KHÔNG break build pipeline hiện tại |
| R7 | README hướng dẫn rõ ràng | Section Configuration cập nhật tên var đúng + ví dụ chạy từng mode |

## Non-functional

| # | Yêu cầu | Acceptance |
|---|---|---|
| N1 | Lesson 1898 `.env.example` style | Mỗi entry = comment header 1 dòng + env var actionable. KHÔNG block comment prose |
| N2 | Lesson 1934 multi-env config | Mỗi env file là 1 variant; runtime/build chọn qua `--mode` |
| N3 | Lesson `P-scope-creep` minimal impact | Chỉ tạo env files + README + report. KHÔNG đụng `src/`, Dockerfile, nginx |
| N4 | CLAUDE.md §3 Simplicity First | Pattern dùng Vite native multi-mode resolution, KHÔNG custom loader |

## Constraints

- Vite 8 dùng `import.meta.env` với prefix `VITE_` (default) — chỉ env có prefix `VITE_` mới expose ra client bundle. 3 var hiện tại đều đúng prefix.
- Vite bake env vào bundle build-time (immutable sau build). Để runtime override → phải build lại với env khác HOẶC dùng runtime config pattern (out of scope).
- `.env.[mode]` value được commit → KHÔNG được chứa secret/credential. 3 var hiện tại là URL → OK commit.

## Out-of-scope (explicit)

1. Sửa `src/services/api.ts` hardcoded fallback URLs.
2. Fix discrepancy `VITE_WORKER_API_URL=8090` (.env) vs `8082` (api.ts fallback).
3. Thêm env var mới (e.g., feature flags, GA tracking ID) — chỉ làm việc với 3 var hiện tại.
4. Tạo runtime config endpoint (JSON fetched on bootstrap) thay thế Vite build-time env.
5. CI/CD pipeline cho multi-env build.
6. Dockerfile / nginx changes.
