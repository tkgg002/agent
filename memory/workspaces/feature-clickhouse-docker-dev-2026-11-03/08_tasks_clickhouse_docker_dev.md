---
workspace: feature-clickhouse-docker-dev-2026-11-03
suffix: clickhouse_docker_dev
tier: T2
---

# 08. Tasks — Thêm `gpay-clickhouse` vào `cdc-docker-dev`

- [x] **TASK-CH-01:** Khai báo service `clickhouse` (`container_name: gpay-clickhouse`) và volume `clickhouse_data` (`name: cdc_docker_dev_clickhouse_data`) trong `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml` (`lines 162-187`) cùng các biến môi trường mặc định trong `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/.env.example`.
  - **Context:** `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml:77-187`, `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager_sql_masters.go:110-118`, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/master/masterEngineConfig.ts:53-62`.
  - **Logs/Test:** Kiểm tra cấu hình bằng `docker compose -f /Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml config --quiet`, khởi chạy `docker compose -p cdc-docker-dev -f /Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml up -d clickhouse`, xác minh trạng thái `Up (healthy)` và kết nối TCP/HTTP (`:9000`, `:8123`).
  - **Definition of Done (DoD):** Container `gpay-clickhouse` chạy trong project `cdc-docker-dev`, gắn mạng `cdc-bridge`, mở cổng `9000:9000` (Native TCP) và `8123:8123` (HTTP), tự khởi tạo database mặc định `goopay_clickhouse` (user `default` / hoặc `gpay_admin`), đạt `healthy` và phản hồi truy vấn kiểm tra kết nối thành công.
