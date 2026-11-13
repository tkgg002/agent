# Project Context

> **Last Updated**: 2026-11-09
> **Maintained by**: Brain (Antigravity)

## ⚠️ INVARIANT BẤT BIẾN (PHẢI NHỚ TRƯỚC MỌI TASK)
- **HỆ THỐNG ĐANG CHẠY VẬN HÀNH THỰC TẾ (LIVE PRODUCTION) — KHÔNG PHẢI ĐANG BỊ LỖI.**
- Toàn bộ luồng dữ liệu cốt lõi (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`, `cdc-auth-service`) đang hoạt động trên môi trường thực tế.
- Tuyệt đối **CẤM** dựa vào lỗi thiếu môi trường local (`go test` thiếu DB/NATS local) hoặc các thư mục lưu trữ phụ (`docs/*_bk*`, `scratch/`) hay nhánh code cũ không gọi để phán hệ thống đang hỏng.
- Mọi tính năng mới hoặc chỉnh sửa bắt buộc phải **bảo toàn 100% tương thích ngược (Backward Compatibility)** với luồng đang chạy và tuân thủ đúng **Design Pattern hiện hữu** của từng service.
- **TÀI LIỆU TỔNG THỂ BẮT BUỘC ĐỌC TRƯỚC MỌI TÁC VỤ (SINGLE SOURCE OF TRUTH):**
  👉 [`/Users/trainguyen/Documents/work/agent/memory/global/project/data-hub/system_overview_and_design_pattern.md`](file:///Users/trainguyen/Documents/work/agent/memory/global/project/data-hub/system_overview_and_design_pattern.md)
  *(Chứa đầy đủ: Kiến trúc 4 tầng × 7 nhóm chức năng, 5 Shared Chokepoints `CP-1..CP-5`, Ma trận Đa hình 3 Chiều, Danh mục 99 Domain Actions & đánh giá "Đủ chưa?", Code-Level Design Patterns & Go Interface Contracts cho Multi-Source / Multi-Master / Multi-Transform Type, 5 bước kiểm tra trước khi sửa code, và Master Roadmap `W1 → W6`).*

## Overview

**cdc-system (`data-hub`)** — Hệ thống Change Data Capture (CDC) đồng bộ dữ liệu thay đổi từ
nguồn vận hành (MongoDB / PostgreSQL / MariaDB / SFTP) sang đích (PostgreSQL / MongoDB / MySQL / ClickHouse),
qua 2 tầng `shadow` → `master`, có schema evolution kiểm soát, reconciliation (Segment A & B, Smoke, Stream Bucket),
self-healing (SegA & SegB), DLQ retry, và operator UI (`cdc-cms-web`).

- **Scale**: multi-repo trong `data-hub/` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`, `cdc-auth-service`, `cdc-control`).
- **Users**: operator/admin vận hành qua CMS UI; developer & DevOps.
- **Stage**: **LIVE / PRODUCTION (Đang vận hành thực tế)**.