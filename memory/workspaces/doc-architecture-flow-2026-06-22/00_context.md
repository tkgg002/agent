# Workspace Context: doc-architecture-flow-2026-06-22

## Overview
Workspace này chuyên trách việc tạo tài liệu đặc tả kiến trúc phân tầng (Layered Architecture) và các luồng xử lý dữ liệu toàn trình (End-to-End Execution Flows) của hệ thống `centralized-data-service`.

## Scope
- Mô tả 9 layer tầng cao trong dự án: `activity`, `admin`, `handler`, `model`, `naming`, `repository`, `server`, `service`, `sinkworker`.
- Phác thảo sơ đồ liên kết giữa các layer bằng Mermaid Diagram.
- Đặc tả 3 luồng xử lý chính:
  1. Provisioning Flow (Luồng khởi tạo đồng bộ).
  2. Ingestion & Transmutation Flow (Luồng nhận dữ liệu & chuyển đổi).
  3. Reconciliation & Self-healing Flow (Luồng đối soát & tự phục hồi).

## Technical Context
- Dự án: `centralized-data-service`.
- Ngôn ngữ: Go.
- Database: PostgreSQL (DW), MongoDB/PG (Source).
- Giao thức: NATS, Kafka.
