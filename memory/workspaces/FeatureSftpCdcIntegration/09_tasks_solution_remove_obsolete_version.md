# 09_tasks_solution_remove_obsolete_version.md — Loại bỏ Thuộc tính version bị Deprecated trong docker-compose.yml

> **Ngày tạo**: 2026-08-11  
> **Mục tiêu**: Xóa dòng `version: '3.8'` ở đầu file `docker-compose.yml` để loại bỏ cảnh báo `the attribute version is obsolete` của Docker Compose v2.

---

## 1. Phân tích Nguyên nhân

- **Thông báo**: `WARN[0000] the attribute version is obsolete, it will be ignored`
- **Container status**: `✔ Container sftp-host Running` (Container đã khởi chạy thành công).
- **Nguyên nhân**: Docker Compose v2 (Compose Specification mới nhất) khuyến nghị bỏ thuộc tính `version` ở đầu file `.yml` vì Docker Compose hiện tại tự động nhận diện specification mà không cần khai báo `version`.

---

## 2. Chi tiết Thay đổi File `data-hub/docker/docker-compose.yml`

```yaml
services:
  sftp-host:
    image: atmoz/sftp:latest
    platform: linux/amd64
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

---

## 3. Kế hoạch Phân công Thực thi (Brain / Muscle Protocol)

1. **Brain**: Xuất kế hoạch và nộp `implementation_plan.md` chờ User **APPROVE**.
2. **Muscle**: Cập nhật `docker-compose.yml` loại bỏ dòng `version: '3.8'`.
3. **Verification**: 
   - Kiểm tra `docker compose ps` xác nhận container vẫn đang `Running` và không còn warning.
