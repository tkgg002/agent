# 09_tasks_solution_sftp_docker_host.md — Giải pháp Khởi tạo Docker SFTP Host cho Kiểm thử

> **Ngày tạo**: 2026-08-11  
> **Thư mục target**: `/Users/trainguyen/Documents/work/data-hub/docker`  
> **Mục tiêu**: Xây dựng một SFTP Server giả lập chạy bằng Docker container phục vụ kiểm thử end-to-end luồng SFTP Source Connector.

---

## 1. Yêu cầu & Cấu hình Docker SFTP Server

- **Docker Image**: `atmoz/sftp:latest` (Image tiêu chuẩn, nhẹ, ổn định cho SFTP testing).
- **Port Mapping**: Host Port `2022` ➔ Container Port `22`.
- **Tài khoản mặc định**:
  - **Username**: `gp-reconcile-admin`
  - **Password**: `sftp_password`
  - **User ID / Group ID**: `1001:1001`
- **Cấu trúc Thư mục SFTP Server**:
  - `/home/gp-reconcile-admin/goopay/reconcile_final` (thư mục chứa file CSV nguồn)
  - `/home/gp-reconcile-admin/goopay/reconcile_final/processed` (thư mục chuyển file sau khi đọc thành công)
  - `/home/gp-reconcile-admin/goopay/reconcile_final/error` (thư mục chứa file lỗi)

---

## 2. Danh sách Files sẽ khởi tạo tại `data-hub/docker/`

### 1. `docker-compose.yml`
```yaml
version: '3.8'

services:
  sftp-host:
    image: atmoz/sftp:latest
    container_name: sftp-host
    hostname: sftp-host
    ports:
      - "2022:22"
    command: gp-reconcile-admin:sftp_password:1001:1001:goopay/reconcile_final,goopay/reconcile_final/processed,goopay/reconcile_final/error
    volumes:
      - ./data/reconcile_final:/home/gp-reconcile-admin/goopay/reconcile_final
      - ./data/processed:/home/gp-reconcile-admin/goopay/reconcile_final/processed
      - ./data/error:/home/gp-reconcile-admin/goopay/reconcile_final/error
    restart: always
```

### 2. Sample Data File: `data/reconcile_final/reconcile_final_20260811.csv`
```csv
transaction_id,amount,status,partner_code,created_at
TX1001,150000.00,SUCCESS,MOMO,2026-08-11T08:00:00Z
TX1002,250000.00,SUCCESS,ZALOPAY,2026-08-11T08:05:00Z
TX1003,50000.00,FAILED,VIETCOMPAY,2026-08-11T08:10:00Z
```

### 3. `README.md`
Tài liệu hướng dẫn khởi chạy, lệnh `docker compose up -d`, kiểm tra kết nối bằng `sftp -P 2022 gp-reconcile-admin@localhost` và lệnh test đẩy file.

---

## 3. Kế hoạch Phân công Thực thi (Brain / Muscle Protocol)

1. **Brain**: Lập giải pháp, xuất tài liệu và nộp `implementation_plan.md` chờ User **APPROVE**.
2. **Muscle**: Uỷ quyền cho `muscle_executor` tạo toàn bộ cấu trúc file tại `data-hub/docker/` và khởi chạy container.
3. **Verification**: 
   - Kiểm tra container `sftp-host` đang ở trạng thái `Up` (chạy `docker ps`).
   - Kiểm tra kết nối TCP tới port 2022 (`nc -zv localhost 2022`).
