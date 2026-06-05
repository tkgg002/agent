# 05_progress.md — cdc-cms-web env config — Audit Log (APPEND-only)

> CLAUDE.md §11: TUYỆT ĐỐI CẤM overwrite. Chỉ APPEND vào cuối file.

---

## [2026-05-18] [Muscle:claude-opus-4-7] Workspace khởi tạo

Tạo 4 file plan (CLAUDE.md §7 Full Doc Set):
- `00_context.md` — scope + stack + DoD
- `01_requirements.md` — R1-R7 functional + N1-N4 non-functional + out-of-scope explicit
- `02_plan.md` — Interpretation analysis (A/B/C) + Phase 1-7 implementation + Risk register
- `05_progress.md` — this file (audit log placeholder)

State của repo `cdc-cms-web` đã survey:
- 1 file `.env` (gitignored) với 3 var localhost
- `package.json` đã có `build:prod` (`--mode production`)
- `.gitignore` đã ignore `.env*.local` an toàn
- Dockerfile pattern ARG → ENV → build:prod đã đúng
- `src/services/api.ts:3-5` đọc 3 var với fallback hardcoded
- README section Configuration sai tên var (`VITE_AUTH_BASE` cũ vs `VITE_AUTH_API_URL` thực tế)

**Status**: Chờ user approve Interpretation A (Vite multi-mode env files) trước khi implement.

---

## [2026-05-18] [Muscle:claude-opus-4-7] User approve "làm đi" — Implement

User reply: "làm đi" → approve Interpretation A.

**Files tạo MỚI tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/`**:
1. `.env.example` — template copy → `.env.local`. Default 3 var localhost (8081/8083/8090).
2. `.env.development` — mode `development`, load khi `npm run dev`. Value giống `.env.example`.
3. `.env.staging` — mode `staging`, load khi `vite build --mode staging`. Value `https://auth.staging.cdc-system.internal` + cms + worker.
4. `.env.production` — mode `production`, load khi `npm run build`/`build:prod`. Value **RỖNG** (Dockerfile inject runtime).
5. `report_env_2026-05-18.md` — report root repo, ghi nhận thay đổi + verify evidence.

**Files SỬA**:
- `README.md` section Configuration — rewrite: tên var đúng (`VITE_AUTH_API_URL` vs cũ `VITE_AUTH_BASE`), multi-mode table, override hierarchy explanation.

**Files KHÔNG đụng** (compliance lesson `P-scope-creep`):
- `.env` (gitignored, giữ nguyên)
- `.gitignore` (đã đúng pattern)
- `src/**` (toàn bộ)
- `Dockerfile`, `nginx.conf`, `vite.config.ts`, `package.json`

## [2026-05-18] [Muscle:claude-opus-4-7] Verify thực tế

| # | Command | Exit | Evidence |
|---|---|---|---|
| V2 | `npm run build` | 0 | grep dist/assets/api-*.js: `localhost:8081`, `:8082`, `:8083` (fallback hardcoded vì `.env.production` rỗng) |
| V3 | `npm run build:prod` | 0 | Identical V2 |
| V4 | `vite build --mode staging` | 0 | grep: `auth.staging.cdc-system.internal`, `cms.staging.cdc-system.internal`, `worker.staging.cdc-system.internal` |
| V8 | `vite build --mode development` | 0 | grep: `localhost:8081`, `:8083`, `:8090` (worker `:8090` từ `.env.development` đè fallback `:8082` trong api.ts) → chứng minh `.env.development` được load |
| V5 | `git check-ignore .env*.local` | 0 | All MATCHED (5 file: `.env`, `.env.local`, `.env.[dev/staging/prod].local`) |
| V6 | `git check-ignore .env.[mode]` non-local | non-zero | NOT ignored → commit-able |
| V7 | `npm run lint` | non-zero | 32 errors pre-existing trong `src/components` + `src/pages` (react-hooks/set-state-in-effect, no-empty). KHÔNG do env files (eslint không lint `.env*`). Ngoài scope. |
| V1 | `npm run dev` | skip | Port 5173 busy. Resolution evidence đã có ở V8 (build mode=development). |

**Kết luận verify**: 4/4 mode bake URLs đúng vào bundle. Vite native `--mode` resolution hoạt động đúng pattern.

## [2026-05-18] [Muscle:claude-opus-4-7] Definition of Done compliance

- [x] R1: 3 file env mode commit-able (`.env.development`, `.env.staging`, `.env.production`)
- [x] R2: `.env.example` template với 3 var + comment header 1 dòng (lesson 1898)
- [x] R3: Vite resolve đúng theo `--mode` (V2/V3/V4/V8 PASS)
- [x] R4: `.env.production` rỗng → tránh leak prod URL vào commit
- [x] R5: `.gitignore` chặn `.env.[mode].local` (V5 PASS)
- [x] R6: Dockerfile ARG pattern không break (Vite ưu tiên `process.env` > `.env.[mode]`)
- [x] R7: README cập nhật tên var đúng + multi-mode table

- [x] N1: `.env.example` actionable style
- [x] N2: multi-env config no DEV creds leak
- [x] N3: minimal impact (chỉ env + README + report)
- [x] N4: dùng Vite native, không custom loader

- [x] Report file `report_env_2026-05-18.md` ở root repo
- [x] Workspace audit log APPEND-only

**Status**: COMPLETED. Task chain #32-#37 closed. Refactor env cdc-cms-web 2026-05-18 hoàn tất.

---

## [2026-05-18] [Muscle:claude-opus-4-7] Phase 2: Runtime envsubst pattern

**Trigger**: User share Helm values `cdc-cms-web.yaml` với 3 env runtime đang comment + hỏi "k8s muốn add env vào yaml, cái này là add khi build docker phải ko". Đề xuất 3 Options → user chốt **Option 2 (envsubst)**.

**Plan đã document**: `02_plan_envsubst.md` cùng workspace.

### Files thay đổi (7 file)

**NEW**:
1. `docker-entrypoint.sh` — Shell script `sh` alpine, fail-fast `:?` check 3 env required + `find` + `sed -i` thay magic placeholder + `exec nginx`.
2. `report_envsubst_2026-05-18.md` — root repo, evidence verify thực.
3. `02_plan_envsubst.md` — workspace plan Phase 2.

**EDIT**:
4. `Dockerfile` — xoá `ARG VITE_*` + `ENV VITE_*` build-time; thêm `COPY docker-entrypoint.sh` + `chmod +x`; đổi `CMD` → `ENTRYPOINT ["/docker-entrypoint.sh"]`.
5. `.env.production` — rỗng → magic `__VITE_*__`.
6. `.env.staging` — rỗng → magic `__VITE_*__`.
7. `.dockerignore` — `.env.*` (over-broad) → `.env.local` + `.env.*.local` (bug pre-existing, fixed mid-implementation).
8. `README.md` — Section Configuration update: envsubst pattern + Helm values demo + fail-fast note.

### Files KHÔNG đụng (compliance)

- `src/services/api.ts` — fallback `|| 'localhost:*'` giữ nguyên
- `src/**`, `nginx.conf`, `vite.config.ts`, `package.json`
- Helm values `cdc-cms-web.yaml` (repo khác, operator quản lý)
- Port worker `:8082` vs `:8090` discrepancy (out-of-scope)

### Verify thực tế (4/4 PASS)

| V | Command | Result |
|---|---|---|
| V1 | `rm -rf dist && npm run build:prod` + grep magic | exit 0; chunk `api-BwOxAo4R.js` chứa nguyên 3 magic `__VITE_AUTH_API_URL__`, `__VITE_CMS_API_URL__`, `__VITE_WORKER_API_URL__` |
| V2 | `docker build --no-cache -t cdc-cms-web-envsubst-test:dev .` | exit 0, image built |
| V3 | `docker run -e VITE_*_API_URL=https://*.prod.test -p 8088:80 -d` + `docker exec cat chunk \| grep URL` | URL trong container chunk: `https://auth.prod.test`, `https://cms.prod.test`, `https://worker.prod.test` (magic được sed thay đúng); HTTP /index.html = 200 |
| V4 | `docker run --rm cdc-cms-web-envsubst-test:dev` (không env) | container exit 2; log: `/docker-entrypoint.sh: line 10: VITE_AUTH_API_URL: VITE_AUTH_API_URL is required` |

### Discovery mid-implementation

V3 lần 1 fail: chunk hash trong container = `api-CIcU52lJ.js` (cũ) chứa localhost fallback thay vì magic. Root cause: `.dockerignore:6` pattern `.env.*` exclude TẤT CẢ env files khỏi docker context → Vite trong builder stage không tìm thấy `.env.production` → bundle fallback `api.ts:3-5` hardcoded localhost.

Fix: Edit `.dockerignore` `.env.*` → `.env.local` + `.env.*.local`. Rebuild `--no-cache` → V3 PASS đầy đủ với chunk hash mới `api-BwOxAo4R.js` (khớp local dist).

Lesson: `.dockerignore` patterns dạng `.env.*` quá rộng — luôn explicit `.env.local` + `.env.*.local` để không vô tình exclude commit-able env files. Sẽ APPEND lesson global sau.

### Definition of Done compliance

- [x] V1-V4 PASS evidence thực từ docker exec
- [x] `docker-entrypoint.sh` exist + chmod +x
- [x] Dockerfile ENTRYPOINT thay CMD
- [x] `.env.production` + `.env.staging` magic placeholder
- [x] `.dockerignore` bug pre-existing fixed
- [x] README updated
- [x] `report_envsubst_2026-05-18.md` ở root repo với pipeline diagram + verify evidence + Helm values guide
- [x] APPEND 05_progress (entry này)
- [x] KHÔNG đụng src/, Helm values, Ingress

**Status**: Phase 2 envsubst hoàn tất. Operator có thể:
1. Build image mới push harbor (tag bump).
2. Uncomment `env:` trong Helm values với **public URL** (không internal DNS).
3. Verify Ingress 3 backend service đã có host tương ứng.
4. Deploy.
