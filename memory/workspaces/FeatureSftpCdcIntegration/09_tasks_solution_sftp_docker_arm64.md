# 09_tasks_solution_sftp_docker_arm64.md — Khắc phục Lỗi Platform Mismatch (ARM64 vs AMD64) trên macOS Apple Silicon

> **Ngày tạo**: 2026-08-11  
> **Mục tiêu**: Bổ sung chỉ thị `platform: linux/amd64` vào `docker-compose.yml` để Docker Desktop trên macOS Apple Silicon (ARM64) tự động sử dụng cơ chế Rosetta 2 / QEMU emulation chạy container `atmoz/sftp` mượt mà không bị cảnh báo platform.

---

## 1. Phân tích Nguyên nhân

- **Thông báo lỗi**: `The requested image's platform (linux/amd64) does not match the detected host platform (linux/arm64/v8)`
- **Nguyên nhân**: Máy Mac của bạn sử dụng chip Apple Silicon (`arm64`), trong khi Docker image `atmoz/sftp` được build gốc trên kiến trúc `amd64` (x86_64).
- **Khắc phục**: Khai báo cờ `platform: linux/amd64` trong file `docker-compose.yml` để Docker Engine ép chạy dưới chế độ tương thích Rosetta 2 / x86_64 emulation.

---

## 2. Chi tiết Thay đổi File `data-hub/docker/docker-compose.yml`

```yaml
version: '3.8'

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

## 3. Kế hoạch Thực thi (Brain / Muscle Protocol)

1. **Brain**: Xuất kế hoạch và nộp `implementation_plan.md` chờ User **APPROVE**.
2. **Muscle**: Cập nhật `docker-compose.yml` với `platform: linux/amd64`.
3. **Verification**: 
   - Hướng dẫn User chạy lại `docker compose up -d`.
   - Kiểm tra `docker ps` container `sftp-host` trạng thái `Up`.
