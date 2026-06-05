# 02_plan_envsubst.md — Phase 2: Runtime envsubst pattern

**Trigger**: User chia sẻ Helm values `cdc-cms-web.yaml` với 3 env runtime đang comment + hỏi "k8s muốn add env vào yaml, cái này là add khi build docker phải ko". Trả lời: Vite bake build-time → k8s env runtime KHÔNG work với pattern Dockerfile cũ. Đề xuất 3 Options → user chốt Option 2 (envsubst).

## Scope

Refactor `cdc-cms-web` Dockerfile từ build-time per-cluster (Phase 1) sang runtime envsubst (Phase 2):
- 1 image build duy nhất dùng cho mọi cluster (testing/staging/prod)
- k8s Helm `env:` quyết định URL backend runtime
- Container fail-fast nếu thiếu env (CrashLoopBackOff visible)

**KHÔNG trong scope**:
- Đụng `src/services/api.ts` (logic fallback giữ nguyên)
- Sửa Helm values file (file repo khác, operator quản lý)
- Tạo Ingress YAML cho 3 backend service
- Fix port discrepancy `:8082` vs `:8090`
- Setup CI/CD

## Implementation steps

### Step 1 — Tạo `docker-entrypoint.sh` (NEW)

Shell script (`#!/bin/sh` alpine compatible) với 3 phần:
1. Validate 3 env required (`: "${VAR:?msg}"` fail-fast).
2. `find` + `sed -i` thay 3 magic placeholder trong mọi `.js` + `.html` dưới `/usr/share/nginx/html`.
3. `exec nginx -g 'daemon off;'`.

### Step 2 — Sửa `Dockerfile`

Stage 1 (builder):
- XOÁ `ARG VITE_*` + `ENV VITE_*` (không cần build-arg nữa).
- Giữ COPY package.json + npm ci + COPY . + npm run build:prod.

Stage 2 (runtime):
- THÊM `COPY docker-entrypoint.sh /docker-entrypoint.sh` + `RUN chmod +x`.
- ĐỔI `CMD ["nginx", "-g", "daemon off;"]` → `ENTRYPOINT ["/docker-entrypoint.sh"]`.

### Step 3 — Sửa `.env.production` + `.env.staging`

Rỗng → magic placeholder:
```bash
VITE_AUTH_API_URL=__VITE_AUTH_API_URL__
VITE_CMS_API_URL=__VITE_CMS_API_URL__
VITE_WORKER_API_URL=__VITE_WORKER_API_URL__
```

### Step 4 — Sửa `.dockerignore` (discovered mid-implementation)

Bug pre-existing: `.env.*` exclude TẤT CẢ env files khỏi docker build context → Vite không tìm thấy `.env.production` trong builder stage → bundle dùng fallback localhost trong `api.ts:3-5`.

Fix: `.env.*` → `.env.local` + `.env.*.local` (chỉ ignore local override).

### Step 5 — Update README

Section Configuration:
- Thay text `.env.production rỗng` → magic placeholder pattern
- Thêm Helm values demo
- Thêm fail-fast note

### Step 6 — Verify (4 case)

| V | Command | Expected |
|---|---|---|
| V1 | `npm run build:prod && grep magic dist/assets/api-*.js` | 3 magic placeholder bake nguyên vẹn |
| V2 | `docker build --no-cache .` | exit 0, image tagged |
| V3 | `docker run -e VITE_*_API_URL=https://*.prod.test -p 8088:80 -d` + `docker exec cat chunk \| grep URL` | 3 URL substituted; HTTP /index.html = 200 |
| V4 | `docker run` không env | exit 2, log "X is required" |

### Step 7 — Tạo report file + APPEND 05_progress

- `report_envsubst_2026-05-18.md` ở root repo.
- APPEND 05_progress.md với evidence verify thực.

## Pattern diagram

```
[build-time]                                [runtime container start]
.env.production: __VITE_*__       k8s env: VITE_*_API_URL=https://...
       │                                   │
       ▼                                   ▼
vite build → bundle .js              docker-entrypoint.sh
chứa "__VITE_AUTH_API_URL__"          ├─ check 3 env required
       │                              ├─ find .js + .html
       ▼                              ├─ sed thay magic
Docker image (1 image / N cluster)    └─ exec nginx
```

## Risk register

| Risk | Mitigation |
|---|---|
| Vite minifier break magic string across chunks | Test V1 grep — confirm magic literal nguyên vẹn |
| sed performance trên chunk antd 1MB | Acceptable ~50-200ms container start; nginx + sed alpine native nhanh |
| Env empty không trigger fail-fast | sh `:?` operator chỉ check unset, không check empty. Solution: set non-empty default trong Helm `env:` |
| dockerignore vô tình exclude env file | Test V3 (URL substituted) — bonus: discovered ngay khi V3 lần 1 chứa localhost fallback |
| 2 image trong harbor (Phase 1 + Phase 2) chạy đồng thời | Operator phải rebuild + bump tag sau merge Phase 2 |

## Pre-DONE checklist

- [ ] `docker-entrypoint.sh` exist + executable
- [ ] Dockerfile: ENTRYPOINT thay CMD, COPY entrypoint OK
- [ ] `.env.production` + `.env.staging` magic placeholder
- [ ] `.dockerignore` fix `.env.*`
- [ ] README updated
- [ ] V1-V4 PASS với evidence từ docker exec
- [ ] report_envsubst_2026-05-18.md ở root repo
- [ ] APPEND 05_progress
- [ ] KHÔNG đụng src/, Helm values, Ingress
