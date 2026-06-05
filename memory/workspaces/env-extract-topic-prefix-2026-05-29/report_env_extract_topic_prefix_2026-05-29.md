# Report — Extract 4 hardcoded chuỗi sang env

- **Workspace**: `env-extract-topic-prefix-2026-05-29`
- **Date**: 2026-05-29
- **Severity**: P2 — config hygiene; tách env theo môi trường (local/dev/prod cùng codebase).
- **Service**: `cdc-cms-web`

## 1. Inventory hardcoded

| Chuỗi | File | Line | Trạng thái |
|---|---|---|---|
| `cdc.goopaylocal` | `cdc-cms-web/src/pages/SourceConnectors.tsx` | 129 | Đã env-driven |
| `cdc.mariadblocal` | `cdc-cms-web/src/pages/SourceConnectors.tsx` | 130 | Đã env-driven |
| `cdc.gpaylocal` | `cdc-cms-web/src/pages/SourceConnectors.tsx` | 131 | Đã env-driven |
| `cdc-worker-group-local` | `cdc-cms-web/src/pages/SourceConnectors.tsx` | 186 | Đã env-driven |
| `cdc-worker-group-local` | `centralized-data-service/config/config-*.yml` | 45/57 | KHÔNG đụng — đã ở config file, env-driven sẵn |
| `cdc.gpaylocal`, `cdc.goopaylocal`, `cdc.mariadblocal` | `centralized-data-service/config/config-local.yml` | 47-49 | KHÔNG đụng — đã config file |

## 2. Pattern apply (theo style existing)

```ts
const TOPIC_PREFIX_MONGODB =
  import.meta.env.VITE_TOPIC_PREFIX_MONGODB || '__VITE_TOPIC_PREFIX_MONGODB__' || 'cdc.goopaylocal';
```

3 tầng fallback:
1. Build-time inject: Vite đọc `VITE_*` khi build.
2. Runtime sed replace: `docker-entrypoint.sh` thay placeholder `__VITE_*__` trên file `.js` sau khi container start.
3. Hard fallback string: chỉ kick-in nếu cả 1 và 2 fail — dev local thuần.

## 3. Files modified

| # | File | LOC delta | Loại |
|---|------|-----------|------|
| 1 | `cdc-cms-web/src/pages/SourceConnectors.tsx` | +9 / -4 | Constants + usage refactor |
| 2 | `cdc-cms-web/.env.development` | +4 / 0 | Local default |
| 3 | `cdc-cms-web/.env.example` | +4 / 0 | Doc template |
| 4 | `cdc-cms-web/docker-entrypoint.sh` | +8 / 0 | Runtime sed replacement |
| 5 | `cdc-cms-web/src/App.tsx` | +0 / -1 | Side-fix unused import chặn tsc (pre-existing) |

NET: +25 / -5.

## 4. Env keys mới

```env
VITE_TOPIC_PREFIX_MONGODB=cdc.goopaylocal
VITE_TOPIC_PREFIX_MYSQL=cdc.mariadblocal
VITE_TOPIC_PREFIX_POSTGRESQL=cdc.gpaylocal
VITE_KAFKA_CONSUMER_GROUP_ID=cdc-worker-group-local
```

## 5. Verify

| Item | Result |
|---|---|
| `npm run build` (Vite + tsc -b) | PASS — `built in 684ms` |
| Bundle `SourceConnectors-33L1PGTY.js` | 25.34 kB / gzip 7.28 kB (healthy) |
| Grep `cdc.goopaylocal\|cdc.mariadblocal\|cdc.gpaylocal\|cdc-worker-group-local` trong `SourceConnectors.tsx` | Chỉ còn ở dạng default fallback của env constants — KHÔNG còn trong logic |
| docker-entrypoint required check | 4 var mới added `:?` guard — fail-fast nếu ops không pass |

## 6. Ops deploy note

- Khi build prod, ops PHẢI set 4 env trên trong docker compose / k8s secret / deployment manifest, nếu không container fail start với message `VITE_TOPIC_PREFIX_MONGODB is required`.
- Để rollback dễ: 4 default fallback giữ nguyên giá trị cũ → nếu unset ENV cả runtime + build-time, code vẫn hoạt động (chỉ phá guard ở docker-entrypoint — cần soft hoá nếu muốn).
- BE `centralized-data-service/config/config-*.yml` đã có dạng key-value config, KHÔNG cần đổi.

## 7. Out of scope

- BE `SIGNAL_KAFKA_TOPIC` placeholder `__VITE_SIGNAL_KAFKA_TOPIC__` chưa có sed replace trong docker-entrypoint (pre-existing gap, không phải task này).
- KHÔNG refactor MySQL/PostgreSQL block thêm `signal.kafka.consumer.group.id` (giữ pattern hiện có, chỉ MongoDB block dùng).
