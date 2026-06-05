# 02_plan_buildtime.md — Phase 3: Build-time bake same-origin path

**Trigger**: User frustrated với pattern envsubst trên k8s (URL internal DNS không resolve từ browser, app fail silently). Yêu cầu: "tao muốn api.ts lấy 3 cái này từ dockerfile luôn".

## Diagnose vấn đề Phase 2 (envsubst) trên k8s thực tế

User test deploy với Helm values:
```yaml
env:
  VITE_AUTH_API_URL: "http://cdc-auth-service-v2.data-hub:8081"
  ...
```

3 lỗi chồng:
1. Lỗi YAML: `env: {}` rồi viết tiếp keys → empty object, k8s không nhận env. (User đã fix)
2. URL k8s internal DNS (`<svc>.<ns>:port`) → browser không resolve được (SPA chạy ở browser, không trong cluster).
3. Image tag cũ `1.0.0-ccf8e52f` trước Phase 2 → chưa có entrypoint sed. (User đã build tag mới `1.0.0-91de9294`)

→ Pattern envsubst conceptually đúng, nhưng trade-off với SPA: phải có URL **public** qua Ingress, đụng tới architecture k8s networking.

## Decision: Phase 3 = Build-time bake **same-origin path**

Thay vì runtime envsubst, bake **path tương đối** (`/api/auth`, `/api/cms`, `/api/worker`) trực tiếp vào bundle. Browser gọi same-origin → Ingress cluster route path tới backend service.

### Ưu điểm

- 1 image / N cluster (same-origin path domain-agnostic)
- Không cần env runtime, không cần entrypoint script
- KHÔNG CORS (same-origin)
- Không cần subdomain riêng / wildcard cert
- Backend hiện tại không cần đổi (chỉ cần ingress rewrite `/api/auth(/|$)(.*)` → `/$2`)

### Nhược điểm

- Mỗi cluster phải có 4 path trên cùng 1 ingress host
- Nếu cần đổi path prefix → rebuild image

## Implementation steps

### Step 1 — Sửa Dockerfile

Builder stage thêm 3 ENV (đè `.env.production` build-time):
```dockerfile
ENV VITE_AUTH_API_URL=/api/auth
ENV VITE_CMS_API_URL=/api/cms
ENV VITE_WORKER_API_URL=/api/worker
```

Runtime stage:
- XOÁ `COPY docker-entrypoint.sh` + `RUN chmod +x`
- XOÁ `ENTRYPOINT ["/docker-entrypoint.sh"]`
- THÊM lại `CMD ["nginx", "-g", "daemon off;"]`

### Step 2 — Sửa `.env.production`

Bỏ magic placeholder, set path:
```bash
VITE_AUTH_API_URL=/api/auth
VITE_CMS_API_URL=/api/cms
VITE_WORKER_API_URL=/api/worker
```

(Dockerfile ENV vẫn override khi build container, file `.env.production` này phục vụ `npm run build:prod` local.)

### Step 3 — Xoá `docker-entrypoint.sh`

Không cần nữa.

### Step 4 — `api.ts` giữ nguyên

`import.meta.env.VITE_*` đọc từ ENV builder stage. Không đụng src/.

### Step 5 — Helm values yaml (user apply ở repo helm)

```yaml
# BỎ env block:
# env:
#   VITE_AUTH_API_URL: ...

# THÊM 3 ingress path:
ingress_nginx:
  enabled: true
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
  rules:
    - host: testing-cdc.goopay.vn
      paths:
        - path: /api/auth(/|$)(.*)
          pathType: ImplementationSpecific
          serviceName: cdc-auth-service-v2
          servicePort: 8081
        - path: /api/cms(/|$)(.*)
          pathType: ImplementationSpecific
          serviceName: cdc-cms-service-v2
          servicePort: 8083
        - path: /api/worker(/|$)(.*)
          pathType: ImplementationSpecific
          serviceName: centralized-data-service-v1
          servicePort: 8082
        - path: /
          pathType: Prefix
          serviceName: cdc-cms-web-v1
          servicePort: 80
```

### Step 6 — Verify

| V | Command | Expected |
|---|---|---|
| V1 | `npm run build:prod` + grep bundle | 3 path `/api/auth`, `/api/cms`, `/api/worker`; KHÔNG localhost, KHÔNG magic |
| V2 | `docker build` (user CI) | image build OK |
| V3 | k8s deploy + browser test | request gửi `https://testing-cdc.goopay.vn/api/auth/login` → 401 hoặc 200 (reach backend) |

## Risk register

| Risk | Mitigation |
|---|---|
| Ingress rewrite-target syntax sai (annotation phụ thuộc ingress-nginx version) | Test path-based với regex group capture `(.*)` |
| Backend nhận path `/api/auth/login` thay vì `/login` | Annotation `rewrite-target: /$2` strip prefix |
| Cluster prod dùng cùng image cần ingress prod có 4 path tương tự | Document trong README |
| User đổi backend port → ingress sai | Doc port + service name trong README |

## Pre-DONE checklist

- [x] Dockerfile: 3 ENV + bỏ ENTRYPOINT
- [x] `.env.production` set path
- [x] Xoá `docker-entrypoint.sh`
- [x] `api.ts` giữ nguyên
- [x] README updated
- [x] V1 PASS (build local + grep verify)
- [ ] User apply Helm yaml changes (out-of-scope repo)
- [x] APPEND 05_progress
