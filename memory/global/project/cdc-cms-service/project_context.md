# Project Context — cdc-cms-service

> **Last Updated**: 2026-11-09
> **Maintained by**: Brain (Antigravity)

## 📌 TÀI LIỆU TỔNG THỂ BẮT BUỘC ĐỌC TRƯỚC MỌI TASK
Trước khi phân tích, thiết kế hay sửa code trên `cdc-cms-service`, **BẮT BUỘC** đọc tài liệu kiến trúc & Design Pattern tổng thể của `data-hub` tại:

👉 **`/Users/trainguyen/Documents/work/agent/memory/global/project/data-hub/system_overview_and_design_pattern.md`**

- **Kiến trúc chuẩn của `cdc-cms-service`:** **Hexagonal + CQRS + Screaming Architecture** (`internal/api/<group>` → `internal/app/commands|queries/<group>` → `internal/domain/<group>` & `internal/app/ports` ← `internal/infra/{persistence,messaging,http}`) chia theo 7 nhóm chức năng (`source`, `shadow`, `master`, `governance`, `recon`, `scheduler`, `system`).
- **Danh mục đầy đủ 99 Actions, Hợp đồng NATS/DB với `centralized-data-service`, và Lộ trình `W1 → W6`:** Xem Mục 3, 4, 5, 6 trong `../data-hub/system_overview_and_design_pattern.md`.
