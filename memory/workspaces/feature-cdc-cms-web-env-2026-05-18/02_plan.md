# 02_plan.md — cdc-cms-web env config riêng theo môi trường

## Interpretation analysis (lesson P-scope-creep step 1+4)

User message: *"thêm config env riêng cho từng biến môi trường env trên repo cdc-cms-web"*

Re-read 2 lần. Phân tích:
- "thêm config env riêng" = thêm các file config env (file vật lý).
- "cho từng biến môi trường" — đọc tự nhiên tiếng Việt: "biến môi trường" thường = "environment" (dev/staging/prod), KHÔNG phải "variable". Ngữ cảnh + "từng" → 1 file cho mỗi môi trường.
- "env trên repo" — env (mode) trên repo.

**Interpretation chính (A — Vite multi-mode env files)**: Tạo 1 file env cho mỗi môi trường (dev, staging, prod) theo Vite native `--mode` pattern. Đây là pattern mainstream cho Vite project.

**Alternative B (typed env wrapper)**: Tạo `src/config/env.ts` module với schema TypeScript + Zod validation, đọc `import.meta.env` an toàn hơn. → KHÔNG match wording user ("config env riêng"). Bỏ.

**Alternative C (per-VAR file)**: Mỗi biến (`VITE_AUTH_API_URL`) có 1 file riêng. → Anti-pattern, không có framework nào support. Bỏ.

**Lựa chọn final**: A. Trình bày ngắn cho user duyệt trước khi implement (theo rule scope-creep "> 2 file mới → check user").

## Plan implementation (chỉ thực thi sau user approve)

### Phase 1 — Tạo env files

**Decision điểm A1**: Giữ `.env` hiện tại làm gì?

| Option | Hành vi | Pros | Cons |
|---|---|---|---|
| A1a — giữ `.env` làm fallback chung (default localhost) | Mọi mode đều load `.env` + `.env.[mode]` đè lên trên | Backward compat tuyệt đối, dev không thay đổi gì | Có 2 file commit cho dev (`.env` + `.env.development`) — duplicate |
| A1b — convert `.env` → `.env.development` (rename) | Dev mode đọc trực tiếp `.env.development` | Mỗi mode 1 file rõ ràng, không duplicate | `npm install` lần đầu cần `cp .env.example .env.local` nếu muốn override |

**Khuyến nghị: A1a** — giữ `.env` làm fallback. Lý do: `.env` đã trong `.gitignore` (line 17) → developer hiện có file `.env` local sẽ không bị conflict; multi-mode chỉ là enhancement, không break flow hiện tại.

→ Revert: `.gitignore` line 17 ignore `.env` → mâu thuẫn với "commit fallback". Re-check.

```
# Env files (URL config / secret)
.env
.env.local
.env.*.local
```

`.env` bị gitignored → nếu giữ A1a, file `.env` không commit, dev mới phải tự copy từ `.env.example`. OK, đây là pattern chuẩn.

**Final decision A1a (refined)**:
- `.env` → giữ nguyên (gitignored, developer local copy).
- `.env.example` → MỚI, commit, template copy-paste-friendly.
- `.env.development` → MỚI, commit, value `http://localhost:*` (cho dev mặc định khi không có `.env.local`).
- `.env.staging` → MỚI, commit, value placeholder hoặc staging URL.
- `.env.production` → MỚI, commit, value placeholder rỗng (production URL inject build-time qua Docker ARG).

### Phase 2 — File content details

#### `.env.example` (commit)
```bash
# Vite-baked env (build-time, prefix VITE_ exposes to client bundle).
# Copy file này → `.env.local` để override personal mọi mode.
VITE_AUTH_API_URL=http://localhost:8081
VITE_CMS_API_URL=http://localhost:8083
VITE_WORKER_API_URL=http://localhost:8090
```

#### `.env.development` (commit, mode=development)
```bash
# Mode `development` — npm run dev (Vite default cho `vite` CLI).
VITE_AUTH_API_URL=http://localhost:8081
VITE_CMS_API_URL=http://localhost:8083
VITE_WORKER_API_URL=http://localhost:8090
```

#### `.env.staging` (commit, mode=staging)
```bash
# Mode `staging` — vite build --mode staging.
# URL placeholder cho staging cluster; override bằng Docker --build-arg khi CI build.
VITE_AUTH_API_URL=https://auth.staging.cdc-system.internal
VITE_CMS_API_URL=https://cms.staging.cdc-system.internal
VITE_WORKER_API_URL=https://worker.staging.cdc-system.internal
```

#### `.env.production` (commit, mode=production)
```bash
# Mode `production` — npm run build / npm run build:prod.
# Để rỗng; Dockerfile inject qua --build-arg VITE_*_API_URL khi build prod image.
VITE_AUTH_API_URL=
VITE_CMS_API_URL=
VITE_WORKER_API_URL=
```

> Lý do `.env.production` để rỗng: Dockerfile đã ARG/ENV truyền vào trước `npm run build:prod`. Nếu hardcode URL vào file commit → prod image carry URL cố định, không deploy được multi-cluster. Pattern này khớp lesson 1934 (Dockerfile bake config-local.yml only = prod ship DEV creds anti-pattern).

### Phase 3 — `.gitignore` update

`.gitignore` hiện tại đã đủ an toàn (đã ignore `.env`, `.env.local`, `.env.*.local`). Pattern `.env.*.local` đảm bảo `.env.development.local`, `.env.staging.local`, `.env.production.local` đều bị ignore.

→ **KHÔNG cần sửa `.gitignore`**. Verify bằng `git check-ignore` trong phase verify.

### Phase 4 — README update

Section "Configuration" hiện tại:
- Sai tên var (`VITE_AUTH_BASE` thay vì `VITE_AUTH_API_URL`).
- Thiếu hướng dẫn multi-mode.

Plan rewrite:
```markdown
## Configuration

Vite bake env vào build-time (immutable sau build). Env vars phải có prefix `VITE_` để expose ra client bundle.

### Env vars

| Tên | Mặc định | Mô tả |
|---|---|---|
| `VITE_AUTH_API_URL` | `http://localhost:8081` | Base URL `cdc-auth-service` |
| `VITE_CMS_API_URL`  | `http://localhost:8083` | Base URL `cdc-cms-service` |
| `VITE_WORKER_API_URL` | `http://localhost:8090` | Base URL worker admin |

### Multi-mode

| Mode | File | Trigger |
|---|---|---|
| development | `.env.development` | `npm run dev` |
| staging | `.env.staging` | `vite build --mode staging` |
| production | `.env.production` | `npm run build` / `npm run build:prod` |

Override per-developer: copy `.env.example` → `.env.local` (gitignored).
Override per-mode: tạo `.env.[mode].local` (gitignored).

Build-time inject (CI/CD): Dockerfile `ARG VITE_*_API_URL` → `ENV` → `npm run build:prod`. ARG/ENV ưu tiên cao hơn file `.env.[mode]`.
```

### Phase 5 — Verify

| # | Command | Expected |
|---|---|---|
| V1 | `npm run dev` (background, kill sau 5s) | Server start `http://localhost:5173`, không log error env. Vite log "using .env.development" |
| V2 | `npm run build` | Exit 0; `dist/` artifact; check `dist/assets/*.js` chứa string `localhost:8083` (KHÔNG — vì `.env.production` rỗng → fallback hardcoded `api.ts` 8083) HOẶC chứa empty string |
| V3 | `npm run build:prod` | Exit 0; bundle giống V2 |
| V4 | `vite build --mode staging` | Exit 0; bundle chứa `staging.cdc-system.internal` |
| V5 | `git check-ignore -v .env .env.local .env.development.local .env.staging.local .env.production.local` | Tất cả MATCHED |
| V6 | `git check-ignore -v .env.example .env.development .env.staging .env.production` 2>&1 | KHÔNG ignored (commit được) |
| V7 | `npm run lint` | Exit 0 (lint không liên quan env nhưng smoke test) |

### Phase 6 — Report file

Tạo `migrations/report_env_2026-05-18.md`? KHÔNG — `migrations/` là từ repo `cdc-cms-service`. Repo `cdc-cms-web` chưa có folder report convention. Quyết định: tạo `report_env_2026-05-18.md` ở **root repo** cdc-cms-web (đơn giản, dễ tìm).

Nội dung:
- Summary: 4 file env mới + README section rewrite.
- Behavior matrix: mode → file → vars.
- Verify evidence: command output exit code + grep bundle.
- Files-touched list.
- Rationale `.env.production` để rỗng (lesson 1934 anti-pattern).

### Phase 7 — APPEND `05_progress.md`

Audit log entry với timestamp + agent + action. Verify evidence từ Phase 5.

## Risk register

| Risk | Mitigation |
|---|---|
| Dev đã có `.env` local với value khác → load thứ tự override sai | Vite load: `.env.[mode].local` > `.env.[mode]` > `.env.local` > `.env`. Dev `.env` cũ vẫn được load nhưng sẽ bị `.env.development` đè (cùng key) → OK nếu giá trị giống nhau. Khác → dev phải merge thủ công |
| `.env.staging` URL placeholder không tồn tại real → staging build có URL sai | Document trong README: staging URL phải đúng cluster thật. Build CI override qua `--build-arg` nếu cần |
| `.env.production` rỗng → bundle prod gọi API về `localhost` (fallback `api.ts`) | Dockerfile đã ARG → ENV truyền real URL. Nếu deploy ngoài Docker → CI phải set env trước `npm run build` |
| Vite không load file `.env.staging` vì mode "staging" không phải mode built-in | Vite cho phép custom mode. Cần test V4 |

## Verification checklist (pre-DONE)

- [ ] 4 env file tồn tại + commit-able (V6 PASS)
- [ ] `.env.local`, `.env.[mode].local` ignored (V5 PASS)
- [ ] `npm run dev` start không error (V1 PASS)
- [ ] `npm run build` + `build:prod` exit 0 (V2, V3 PASS)
- [ ] `vite build --mode staging` exit 0 + bundle có URL staging (V4 PASS)
- [ ] README updated với tên var đúng + multi-mode section
- [ ] `report_env_2026-05-18.md` tồn tại với verify evidence thực
- [ ] `05_progress.md` APPEND entry
- [ ] KHÔNG đụng `src/`, Dockerfile, nginx (lesson P-scope-creep compliance)
