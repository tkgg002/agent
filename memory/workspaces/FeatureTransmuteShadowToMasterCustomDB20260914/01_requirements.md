# 01_requirements.md - Transmute Shadow to Master with Custom Master Database Connection

## 1. Bối cảnh & Mục tiêu (Context & Objectives)
- **Vấn đề**: Hiện tại hệ thống CDC chỉ hỗ trợ kết nối database mặc định (`RoleDestination` / `defaultMasterConnectionCode`) cho Master Table. Trong `centralized-data-service`, `ConnectionManager.GetMasterDB(ctx, key)` đang discard hoàn toàn tham số `key`. Trong `cdc-cms-service`, `CreateMasterHandler` hardcode `masterConnectionCode := h.defaultMasterConnectionCode`.
- **Yêu cầu của User**:
  1. Thêm chức năng Transmute chỉ từ Shadow sang Master.
  2. Cho phép Master chọn một Database Connection đích độc lập (ví dụ một PostgreSQL khác).
  3. Đảm bảo DDL Lifecycle & Safety Gate: Tuyệt đối không cho phép Transmute nếu Master Table chưa hoàn tất vòng đời DDL (`schema_status = 'approved'`, bảng vật lý và index đã tồn tại trên database đích). Tránh crash runtime với lỗi `relation does not exist`.

## 2. Phạm vi & Ranh giới (Scope & Boundaries)
- **In-Scope**:
  - **CDS Engine (`centralized-data-service`)**:
    - `ConnectionManager`: Thêm connection pool cache `masterDBPool map[string]*gorm.DB` với mutex. Dynamic resolve DSN từ `connectionOverrides` hoặc `cdc_system.connection_registry` (bằng `systemDB`), mở GORM connection pool độc lập cho từng target database.
    - `TransmuterModule.Run()`: Bổ sung DDL Pre-flight Gate. Kiểm tra bảng đích trên target DB (`information_schema.tables`). Nếu bảng chưa tồn tại hoặc schema chưa ready, trả về lỗi rõ ràng và dừng, không thực hiện upsert.
  - **CMS Backend (`cdc-cms-service`)**:
    - `CreateMasterHandler`: Nhận `cmd.MasterConnectionCode`, tra cứu `master_connection_id` từ `connection_registry` thay vì hardcode default. Lưu vào `cdc_system.master_binding`.
    - API Endpoint: Thêm endpoint `GET /api/v1/master-connections` (hoặc reuse API connection query) để trả về danh sách active Master DB connections cho UI.
  - **CMS Web Frontend (`cdc-cms-web`)**:
    - `MasterRegistry.tsx`: Bổ sung chọn Database Connection trong Modal tạo Master Table. Hiển thị thông tin Target Connection trên bảng Master. Nút Transmute tại Master Table chỉ active khi Master đã `approved`.
    - `TableRegistry.tsx`: Không đặt nút Transmute mù quáng trên hàng chính của bảng Shadow. Nếu hiển thị trong sub-table Master Bindings của Shadow, nút Transmute chỉ enable cho binding đã Approved.

- **Out-of-Scope**:
  - Không thay đổi luồng Ingestion / Snapshot từ Source DB vào Shadow DB.
  - Không tự động chạy DDL migration trong Transmuter runtime (DDL phải qua Master DDL flow chuẩn).

## 3. Ràng buộc Kỹ thuật & Bẫy cần tránh (Tripwires & Constraints)
- **Lesson #master-ddl-prerequisite-fallacy**: Transmute chỉ được kích hoạt khi Master Table đã có DDL trên target DB.
- **Dynamic Connection Cache**: Không mở lại kết nối GORM mỗi lần gọi Transmute (tránh connection leak / resource starvation). Phải pool và cache thread-safe.
- **Simplicity First**: Tận dụng bảng `cdc_system.connection_registry` và cấu trúc `master_binding.master_connection_id` đã có trong database schema.
