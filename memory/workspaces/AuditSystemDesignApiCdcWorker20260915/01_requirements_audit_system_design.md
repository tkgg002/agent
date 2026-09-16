# 01_requirements_audit_system_design.md

## Yêu cầu Chi tiết (Requirements Specs)
1. **Khảo sát & Đọc Source Code Thực Tế (Không suy diễn, không đoán mò):**
   - Đọc cấu trúc thư mục, package, file cấu hình, router, handler, service, repository, worker handlers, pipeline runner, event bus của `cdc-cms-service` và `centralized-data-service`.
2. **Thống kê Cấu trúc System Design:**
   - Mô hình tổng quan hệ thống CDC Platform (Control Plane vs Data Plane/Worker Engine).
   - Luồng giao tiếp qua NATS JetStream / REST API / Database / Kafka Connect.
   - Luồng CDC End-to-End: Source DB (Mongo/Postgres) -> Debezium/Kafka -> CDC Worker -> Shadow Table -> Transmuter -> Master Table -> Recon / Self-Healing.
3. **Thống kê Design Patterns:**
   - Screaming / Hexagonal Architecture (Domain - Ports - Adapters).
   - Creational Patterns (Factory, Builder, Singleton/Connection Manager).
   - Structural Patterns (Adapter, Facade, Decorator/Tracing).
   - Behavioral Patterns (Strategy, Command/NATS Handler, Observer/Event-driven, Chain of Responsibility/Middleware, Pipeline Pattern, Worker Pool Pattern).
4. **Tạo file phân tích vật lý `13_analysis_system_design_api_cdc_worker.md` trong workspace.**
