# 00_context.md — cdc-cms-web env config riêng theo môi trường

**Ngày khởi tạo**: 2026-05-18
**Owner**: Muscle (CC CLI)
**Repo**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`

## Scope

Thêm cấu hình `.env` riêng cho từng môi trường (Vite multi-mode: development / staging / production) trên repo `cdc-cms-web`. Khắc phục tình trạng hiện tại chỉ có 1 file `.env` duy nhất với value localhost — không deploy được sạch lên multi-env (dev/staging/prod) mà không sửa thủ công.

**KHÔNG trong scope** (tuân thủ lesson `P-scope-creep`):
- KHÔNG sửa `src/services/api.ts` (logic đọc env).
- KHÔNG sửa Dockerfile (build-arg pattern đã đúng).
- KHÔNG sửa `nginx.conf`.
- KHÔNG fix discrepancy fallback `VITE_WORKER_API_URL` (8082 trong `api.ts` vs 8090 trong `.env`) — đây là bug khác, ngoài request.
- KHÔNG setup CI/CD pipeline / build hooks.
- KHÔNG đụng `src/` source code.

## Stack / Context kỹ thuật

| Mục | Giá trị |
|---|---|
| Framework | Vite 8 + React 19 + TS 5.9 |
| Mode mặc định | `dev` (npm run dev) → mode=`development`; `build` → mode=`production` |
| Script đã có | `dev`, `build`, `build:prod` (`--mode production` explicit), `lint`, `preview` |
| Env hiện tại | 1 file `.env` với 3 var: `VITE_AUTH_API_URL`, `VITE_CMS_API_URL`, `VITE_WORKER_API_URL` |
| Env consumer | `src/services/api.ts` (3 dòng đọc với fallback hardcoded), `src/main.tsx` (1 dòng `import.meta.env.DEV`) |
| Dockerfile | Truyền 3 ARG → ENV trước `npm run build:prod` → Vite bake build-time |
| .gitignore | Đã ignore `.env`, `.env.local`, `.env.*.local` → an toàn commit `.env.example`, `.env.[mode]` non-local |

## Vite multi-mode load order (built-in)

Vite tự động load env files theo thứ tự ưu tiên (cao đè thấp):
1. `.env.[mode].local`  ← gitignored, dev override per-mode
2. `.env.[mode]`         ← commit, value default per-mode
3. `.env.local`          ← gitignored, dev override mọi mode
4. `.env`                ← commit, fallback chung mọi mode

Khi chạy:
- `npm run dev` → `mode=development` → load `.env.development[.local]` + `.env[.local]`.
- `npm run build` → `mode=production` (Vite default) → load `.env.production[.local]` + `.env[.local]`.
- `npm run build:prod` → explicit `--mode production` → giống `build`.
- `vite build --mode staging` → load `.env.staging[.local]` + `.env[.local]`.

## Definition of Done

1. 4 file env commit-able: `.env.example`, `.env.development`, `.env.staging`, `.env.production`.
2. `.env` (single file hiện tại) giữ nguyên vai trò fallback (default localhost) hoặc convert sang `.env.development`. Quyết định ở `02_plan.md`.
3. `.gitignore` đảm bảo không leak `.env.[mode].local`.
4. README cập nhật: tên env var đúng (`VITE_AUTH_API_URL` thay vì `VITE_AUTH_BASE` cũ), hướng dẫn chạy từng mode.
5. Verify: `npm run dev`, `npm run build`, `npm run build:prod`, `vite build --mode staging` đều PASS (build exit 0, không error env-related).
6. Report file `migrations/report_env_2026-05-18.md` (đặt trong repo, gốc hoặc docs) ghi lại thay đổi.
