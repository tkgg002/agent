---
workspace: feature-clickhouse-docker-dev-2026-11-03
suffix: clickhouse_docker_dev
tier: T2
created_at: 2026-11-03T10:00:40+07:00
---

# 01. Requirements — Thêm ClickHouse Container (`gpay-clickhouse`) vào `cdc-docker-dev`

## 1. Yêu cầu của User (Verbatim)
> "tạo cho anh cái clickhouse ở docker cho anh, nằm ở Container cdc-docker-dev tên gpay-clickhouse"

## 2. Phạm vi & Mục tiêu (Scope & Goals)
- **Mục tiêu:** Khai báo và khởi tạo service ClickHouse trong nhóm Docker Compose `cdc-docker-dev` với tên container cố định là `gpay-clickhouse`, kết nối chung mạng `cdc-bridge` và tương thích hoàn toàn với tầng Master/Destination ClickHouse của hệ thống `data-hub` (`cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`).
- **Vị trí tệp cấu hình thực tế của `cdc-docker-dev`:**
  - `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/docker-compose.yml` (nơi khai báo `gpay-postgres-source`, `gpay-postgres-shadow`, `gpay-postgres-dest`, `gpay-postgres-master-2`, `gpay-mongo`, `gpay-mysql`, `gpay-mariadb`).
  - `/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev/.env.example` (nếu có).

## 3. Ràng buộc Kỹ thuật từ Bài học Kinh nghiệm (Technical Constraints from Global Lessons)
- **L013 (Propose-Only Discipline & Explicit Approval Gate):** Giữ chế độ Read-Only / Propose-Only tại bước `S1 -> S3`, trình bày đầy đủ Giải pháp + Code Demo trực tiếp trên khung chat và chờ lệnh `APPROVE` trước khi sửa file hoặc chạy `docker compose up -d`.
- **L019 (Pre-Plan Adversarial Critique Wave DG0):** Đối chiếu thực tế đường dẫn thư mục `cdc-docker-dev` (`/Users/trainguyen/Documents/work-backup/cdc-system/cdc-docker-dev`), image ClickHouse có sẵn trên máy (`clickhouse/clickhouse-server:25.5.6`), cổng lắng nghe (`9000` native TCP, `8123` HTTP) và chuẩn kết nối ClickHouse trong `data-hub`.
- **L029 (Honest Process State & Environment Lock):** Kiểm tra trạng thái cổng vật lý (`lsof`), image Docker local và xác minh healthcheck thực tế sau khi khởi chạy ở bước `S4`.
- **L030 (Direct-Answer Summary First & 1-to-1 Finding Mapping):** Mở đầu phản hồi bằng khối `Direct Summary` ngắn gọn, ánh xạ giải pháp `1-1` theo từng phát hiện `F1..Fn`.
