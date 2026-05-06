# 08 — Tasks (Phase B5)

> Bind cứng vào TaskList trong CLI (#115–#121). Status nguồn = TaskList; file này là human-readable mirror.

| TaskID | Subject | DoD | Files đụng tới |
|---|---|---|---|
| 115 | B5.1 Phase docs | 4 docs B5 + reqs/plan/tasks/solution viết xong | `01_requirements_b5_*`, `02_plan_b5_*`, `08_tasks_b5_*`, `09_tasks_solution_b5_*` |
| 116 | B5.2 cdc-auth-service env + .env.example | `go build ./...` PASS, `.env.example` 8 keys | `cdc-auth-service/config/config.go`, `.env.example` |
| 117 | B5.3 cdc-cms-service env + airbyte purge | `go build ./...` PASS, airbyte block xoá khỏi YAML, `.env.example` 11 keys | `cdc-cms-service/config/config.go`, `config-local.yml`, `.env.example` |
| 118 | B5.4 centralized-data-service SOURCE_DSN | `go build ./...` PASS, 2 SOURCE_DSN_* env override hoạt động | `centralized-data-service/config/config.go`, `.env.example` |
| 119 | B5.5 Docker split | 2 compose lên độc lập, mạng `cdc-bridge` external, không depends_on chéo | `centralized-data-service/docker-compose.yml`, `cdc-docker-dev/docker-compose.yml`, `cdc-docker-dev/.env.example`, `cdc-docker-dev/README.md` |
| 120 | B5.6 Verify exercise-driven | 3× build PASS, 16 containers Up, 3 business endpoint trả đúng | logs, psql output |
| 121 | B5.7 Report + APPEND progress | report file vật lý + 05_progress nhận entry mới | `report_phase_b5_*.md`, `05_progress.md` |
| 122 | B5.8 Delete workspace sai (DONE) | `feature-config-env-extract-2026-05/` đã xoá | — |

## Blockers

- 117 cần 116 PASS (pattern parseIntDefault dùng chung).
- 118 cần 117 (consistency convention `<SVC>_DB_*`).
- 119 cần 116-118 (compose pass ENV xuống service).
- 120 cần 119 up đầy đủ.
- 121 cần 120 PASS.

## Out of scope

- Track E (MongoDB Debezium connector) — vẫn out of scope như đã chốt B3.
- Git history purge (Airbyte secret cũ trong git log) — task riêng, anh quyết.
- Production secret manager (Vault/SSM) — chỉ bandaid bằng env file dev.
- Test integration đầy đủ E2E — verify trong B5.6 theo "smoke" mức 3 endpoint.
