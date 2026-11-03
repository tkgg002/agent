---
workspace: feature-clickhouse-docker-dev-2026-11-03
suffix: clickhouse_docker_dev
tier: T2
created_at: 2026-11-03T10:05:00+07:00
---

# 09. Tasks Solution — Thêm ClickHouse (`gpay-clickhouse`) vào `cdc-docker-dev`

## 1. Macro Runtime & Deployment Topology (`L030`)
- **Docker Compose Project:** `cdc-docker-dev` (thư mục cấu hình vật lý trên đĩa: `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/`).
- **Container Name:** `gpay-clickhouse` (sử dụng image có sẵn trên máy `clickhouse/clickhouse-server:25.5.6` — không tốn băng thông tải lại).
- **Network & DNS:** Gắn vào mạng ngoài `cdc-bridge` (`external: true`) để toàn bộ các container trong `centralized-data-service` (`gpay-cdc-worker`) và `cdc-docker-dev` phân giải trực tiếp qua hostname `gpay-clickhouse:9000` (hoặc `clickhouse:9000`), đồng thời ánh xạ ra máy host tại `localhost:9000` (Native TCP cho Go driver `clickhouse-go/v2` / DBeaver) và `localhost:8123` (HTTP interface & healthcheck).
- **Credentials & Database mặc định:**
  - User: `${CLICKHOUSE_USER:-default}` (mặc định `default`)
  - Password: `${CLICKHOUSE_PASSWORD:-root}` (mặc định `root`, đồng nhất với `gpay-mysql` và `gpay-mariadb` trong cùng `cdc-docker-dev`)
  - Database khởi tạo sẵn: `${CLICKHOUSE_DATABASE:-goopay_clickhouse}` (cùng database `default` có sẵn của ClickHouse)
  - Quyền quản trị: `CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1`

## 2. Bảng Ánh xạ Điểm Gọi & Vòng đời Kích hoạt (`L012`)

| Thành phần (`file:line`) | Vai trò trong luồng vận hành | Thời điểm kích hoạt (`Lifecycle Stage`) |
| :--- | :--- | :--- |
| `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml:162-187` | Khai báo service `clickhouse` (`container_name: gpay-clickhouse`), cổng `9000:9000` / `8123:8123`, healthcheck `/ping`, mạng `cdc-bridge` và volume `clickhouse_data` (`cdc_docker_dev_clickhouse_data`) | Khi chạy `docker compose -p cdc-docker-dev -f /Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml up -d clickhouse` ở bước `S4` |
| `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/.env.example:28-32` | Khai báo biến môi trường mẫu `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`, `CLICKHOUSE_DATABASE` | Khi lập trình viên sao chép hoặc tham chiếu thông tin đăng nhập của nhóm `cdc-docker-dev` |
| `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/README.md:10-18` | Cập nhật bảng danh mục service/port của `cdc-docker-dev` | Tài liệu tra cứu cổng của `cdc-docker-dev` |
| `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler_connections_probe_sql.go:72-89` | Hàm `buildClickHouseMasterConnectionDSN` & `probeClickHouseMasterConnection` kết nối tới `gpay-clickhouse` qua giao thức `clickhouse://` cổng `9000` | Khi Admin tạo hoặc Test Connection cho Master ClickHouse trên CMS |
| `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager_sql_masters.go:110-118` | Hàm `buildClickHouseMasterDSNFromFields` tạo kết nối `clickhouse://...:9000/...` cho `TransmuterModule` & `ReconDestAgent` | Khi `cdc-worker` thực thi đồng bộ (`transmute`) hoặc đối soát (`recon`) sang ClickHouse Master |

---

## 3. Phân tích Phát hiện & Giải pháp Ánh xạ 1-1 (`F1 -> F2`)

### Phát hiện `F1`: Thiếu định nghĩa service `clickhouse` (`gpay-clickhouse`) và volume `clickhouse_data` trong `cdc-docker-dev/docker-compose.yml`
- **Bằng chứng (`file:line`):** Tại `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml:134-187`, tệp compose chỉ mới định nghĩa đến mục `# 6. MariaDB SOURCE` (`gpay-mariadb`) và danh sách `volumes:` kết thúc ở `mariadb_data`.
- **Giải pháp cho `F1`:**
  - Bổ sung mục `# 7. ClickHouse MASTER/DEST` với service `clickhouse`, `image: clickhouse/clickhouse-server:25.5.6`, `container_name: gpay-clickhouse`, ánh xạ cổng `"9000:9000"` và `"8123:8123"`, biến môi trường `CLICKHOUSE_DB`, `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`, `CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1`, `ulimits` (`nofile: 262144`), `healthcheck` (`wget --spider -q 0.0.0.0:8123/ping`), gắn mạng `cdc-bridge`.
  - Bổ sung volume `clickhouse_data` với tên cố định `name: cdc_docker_dev_clickhouse_data` (theo đúng chuẩn đặt tên của `pg_master_2_data` và `pg_shadow_data` tại dòng `170-173`).

#### Code Demo cho `F1` (`/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml:150-187`)
```diff
--- a/cdc-docker-dev/docker-compose.yml
+++ b/cdc-docker-dev/docker-compose.yml
@@ -150,6 +150,28 @@ services:
     ports:
       - "13307:3306"
     volumes:
       - mariadb_data:/var/lib/mysql
       - ./init/mariadb:/docker-entrypoint-initdb.d:ro
     healthcheck:
       test: [ "CMD", "healthcheck.sh", "--connect", "--innodb_initialized" ]
       interval: 10s
       timeout: 5s
       retries: 6
     networks:
       - cdc-bridge
 
+  # 7. ClickHouse MASTER/DEST — analytics OLAP destination.
+  clickhouse:
+    image: clickhouse/clickhouse-server:25.5.6
+    container_name: gpay-clickhouse
+    environment:
+      CLICKHOUSE_DB: ${CLICKHOUSE_DATABASE:-goopay_clickhouse}
+      CLICKHOUSE_USER: ${CLICKHOUSE_USER:-default}
+      CLICKHOUSE_PASSWORD: ${CLICKHOUSE_PASSWORD:-root}
+      CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1
+    ports:
+      - "9000:9000"
+      - "8123:8123"
+    volumes:
+      - clickhouse_data:/var/lib/clickhouse
+    ulimits:
+      nofile:
+        soft: 262144
+        hard: 262144
+    healthcheck:
+      test: [ "CMD", "wget", "--spider", "-q", "0.0.0.0:8123/ping" ]
+      interval: 10s
+      timeout: 5s
+      retries: 5
+    networks:
+      - cdc-bridge
+
 volumes:
   pg_source_data:
     #external: true
@@ -174,6 +196,8 @@ volumes:
   mariadb_data:
     #external: true
     name: centralized-data-service_mariadb_data
+  clickhouse_data:
+    name: cdc_docker_dev_clickhouse_data
 
 networks:
   cdc-bridge:
```

---

### Phát hiện `F2`: Thiếu biến môi trường mẫu và tài liệu cổng cho `gpay-clickhouse` trong `.env.example` & `README.md`
- **Bằng chứng (`file:line`):** Tại `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/.env.example:24-32` và `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/README.md:10-18`, mới chỉ liệt kê tới `MariaDB`.
- **Giải pháp cho `F2`:** Bổ sung khối `# ---------- ClickHouse ----------` vào `.env.example` và thêm dòng `gpay-clickhouse` vào bảng dịch vụ trong `README.md`.

#### Code Demo cho `F2` (`/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/.env.example:28-32` & `README.md:10-18`)
```diff
--- a/cdc-docker-dev/.env.example
+++ b/cdc-docker-dev/.env.example
@@ -28,5 +28,10 @@ MYSQL_DATABASE=goopay_db
 # ---------- MariaDB ----------
 MARIADB_ROOT_PASSWORD=root
 MARIADB_DATABASE=goopay_legacy_maria
 MARIADB_USER=cdc
 MARIADB_PASSWORD=cdc_pass
+
+# ---------- ClickHouse (gpay-clickhouse on :9000 TCP / :8123 HTTP) ----------
+CLICKHOUSE_USER=default
+CLICKHOUSE_PASSWORD=root
+CLICKHOUSE_DATABASE=goopay_clickhouse
```

```diff
--- a/cdc-docker-dev/README.md
+++ b/cdc-docker-dev/README.md
@@ -15,3 +15,4 @@
 | `gpay-mongo` | 17017 | demo MongoDB source (replSet rs0) |
 | `gpay-mysql` | 13306 | demo MySQL source |
 | `gpay-mariadb` | 13307 | demo MariaDB source |
+| `gpay-clickhouse` | 9000 / 8123 | demo ClickHouse destination (`goopay_clickhouse`) |
```
