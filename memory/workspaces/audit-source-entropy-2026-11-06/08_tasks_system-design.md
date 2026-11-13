# 08 Tasks — system-design
- [x] SD1 Khảo sát centralized-data-service (entrypoint, wiring, pipeline, NATS/Kafka, bảng) — survey/SD1_data_service.md
- [x] SD2 Khảo sát cdc-cms-service (API, command/query, publish NATS, bảng) — survey/SD2_cms_service.md
- [x] SD3 Khảo sát cdc-cms-web + cdc-auth-service + cdc-control + docker (UI→API, auth, hạ tầng) — survey/SD3_web_auth_infra.md
- [x] SD4 Hợp đồng liên service (NATS subject/payload, bảng dùng chung, HTTP) — survey/SD4_contracts.md
- [x] SD5 Tổng hợp overview + design pattern đích + ánh xạ hiện trạng→đích — 13_system_design_system-design.md
- [x] SD6 Thiết kế chi tiết Code-Level Design Patterns (Interfaces, Registries, Lifecycle Hooks & Code Demo) cho 4 trục mở rộng: Multi-Source Engine, Multi-Master Destination, Multi-Transform Type (`transmute.Strategy` mở rộng), và Field-Level Extractor/Transform — cập nhật vào `system_overview_and_design_pattern.md` & `09_tasks_solution_system-design.md`
- [x] SD7 Kiểm kê toàn bộ 99 Domain Actions của hệ thống theo 7 miền chức năng (`Connector`, `Source`, `Shadow`, `Master`, `Recon & Heal`, `Governance`, `Scheduler/Wizard/System`), đánh giá độ phủ đa engine ("Đủ chưa?") và mở rộng các Driver/Strategy Interfaces để bao phủ 100% actions — cập nhật mục 3.3 & 4 vào `system_overview_and_design_pattern.md`
- [x] SD8 Gỡ khóa `recon-heal` (User đã push), hoàn thiện Bản Thiết Kế & Lộ Trình Tổng Thể (`W1 → W6` + Bản đồ sở hữu bảng `cdc_system.*` & Hợp đồng NATS/JetStream/Kafka ở mục 3.4 & mục 6) trong `agent/memory/global/project/data-hub/system_overview_and_design_pattern.md` và tạo/cập nhật 4 file `project_context.md` (`data-hub`, `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) để phiên sau vào đọc ngay
