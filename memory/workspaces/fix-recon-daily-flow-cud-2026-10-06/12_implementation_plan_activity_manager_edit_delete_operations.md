# Kế Hoạch Triển Khai: Edit, Delete và Danh Mục Tác Vụ trên Activity Manager

---

## 1. MỤC TIÊU & PHẠM VI
Triển khai hoàn chỉnh 3 chức năng chính trên trang `/activity-manager`:
1. Chức năng Xoá lịch trình (`DELETE /api/worker-schedule/:id`) với xác nhận Popconfirm.
2. Chức năng Chỉnh sửa lịch trình (Modal Edit tải dữ liệu cũ, cho phép sửa `params`, `interval_minutes`, `run_at_time`, `target_table`, `notes`, `is_enabled`).
3. Chuẩn hóa danh sách `ALL_OPERATIONS`: Bỏ `drop-gin-index`, `create-default-columns`; Thêm `reconcile`, `bridge`.

---

## 2. KẾ HOẠCH CÁC BƯỚC THỰC THI (THE SINGLE BEST APPROACH)

```mermaid
flowchart TD
    A["Bước 1: Backend CMS Service"] --> B["1.1 Thêm Delete vào WorkerScheduleRepo"]
    B --> C["1.2 Tạo DeleteWorkerScheduleCommand & Handler"]
    C --> D["1.3 Bổ sung TargetTable vào UpdateScheduleCommand"]
    D --> E["1.4 Bổ sung Delete handler & đăng ký route DELETE"]
    E --> F["Bước 2: Frontend CMS Web"]
    F --> G["2.1 Cập nhật ALL_OPERATIONS: bỏ gin/columns, thêm reconcile/bridge"]
    G --> H["2.2 Mở rộng cột Thao tác: 3 nút Chạy ngay + Sửa + Xoá"]
    H --> I["2.3 Tái cấu trúc Dynamic 5-Tier Cascading Selector"]
    I --> J["2.4 Xây dựng Modal & Form Edit linh hoạt theo từng Operation"]
    J --> K["2.5 Tích hợp handleDeleteSchedule gọi API DELETE"]
    K --> L["Bước 3: Verification & Quality Gates"]
    L --> M["3.1 Compile & Unit Test: cms-service, centralized-data-service, cms-web"]
    M --> N["3.2 Test thực tế: Tạo, Sửa, Xoá 1 lịch trình reconcile/bridge"]
```

---

## 3. PHÂN BỔ TRÁCH NHIỆM ROLE
- **Brain (Chairman & Architect):** Lập kế hoạch, thiết kế kiến trúc, rà soát bảo mật và rủi ro, xuất bản tài liệu giải pháp và chờ `APPROVE` từ User.
- **Muscle (Chief Engineer):** Khi nhận lệnh `APPROVE`, trực tiếp sửa code trên backend và frontend, chạy test, build và nghiệm thu thực tế.
