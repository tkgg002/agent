# Implementation Plan: Phase 1.8 - CMS Frontend Completion

Hoàn thiện giao diện quản trị (CMS Frontend) cho hệ thống CDC để đóng vòng lặp tự động hóa: Phát hiện (Drift) → Duyệt (Approve) → Đồng bộ (Sync) → Nạp (Load).

## User Review Required

> [!IMPORTANT]
> **Governance Violation RCA (Rule 7)**: 
> - **Lỗi**: Bản kế hoạch ban đầu thiếu các bước bắt buộc về duy trì tài liệu và bộ não dự án theo Quy tắc số 7. 
> - **Nguyên nhân**: Tập trung quá mức vào nghiệp vụ kỹ thuật (React/AntD) mà quên mất quy trình Governance bắt buộc. 
> - **Khắc phục**: Bổ sung phần "Governance & Memory Maintenance" vào kế hoạch này và tạo workflow kiểm tra tự động.

> [!IMPORTANT]
> **API Ports Consistency**: CMS API hiện đang chạy trên cổng `:8080` (theo backend), nhưng Frontend đang cấu hình mặc định `:8090` trong `src/services/api.ts`. Cần thống nhất về `:8080`.
> **Environment Variables**: Sẽ chuyển cấu hình URL sang `.env` để dễ dàng thay đổi giữa các môi trường (Dev/Staging/Prod).

## Proposed Changes

### [Component] Frontend Infrastructure & Auth

#### [MODIFY] [api.ts](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/services/api.ts)
- Sửa `CMS_API` default từ `8090` thành `8080`.
- Chuyển sang sử dụng `import.meta.env.VITE_CMS_API_URL` làm ưu tiên.

#### [NEW] [.env](file:///Users/trainguyen/Documents/work/cdc-cms-web/.env)
- Khai báo các biến môi trường:
  ```env
  VITE_AUTH_API_URL=http://localhost:8081
  VITE_CMS_API_URL=http://localhost:8080
  ```

### [Component] Feature: Schema Approval Workflow

#### [MODIFY] [SchemaChanges.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/SchemaChanges.tsx)
- Kiểm tra và đảm bảo payload gửi lên `/approve` đúng định dạng (`target_column_name`, `final_type`, `approval_notes`).
- Thêm loading states cho các nút Approve/Reject để tránh double-click.

### [Component] Feature: Table Registry Management

#### [MODIFY] [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/TableRegistry.tsx)
- Hoàn thiện xử lý lỗi khi click "Standardize" hoặc "Discover" (đảm bảo gọi đúng NATS Command Pattern đã fix ở Task 1.7).
- Cải thiện trải nghiệm người dùng khi "Bulk Import (JSON)".

### [Component] Deployment & Build

#### [MODIFY] [package.json](file:///Users/trainguyen/Documents/work/cdc-cms-web/package.json)
- Thêm script `build:prod` để tối ưu hóa bundle.

---

### [Component] Governance & Memory Maintenance (Rule 7)

#### [MODIFY] [05_progress.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-integration/05_progress.md)
- Cập nhật nhật ký thực thi hàng ngày kèm timestamp và Model ID.
- Ghi nhận Audit Log sau khi hoàn thành mỗi task nhỏ.

#### [MODIFY] [04_decisions.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-integration/04_decisions.md)
- Ghi nhận bất kỳ thay đổi nào về kiến trúc Frontend hoặc quy trình duyệt (nếu có).

#### [MODIFY] [active_plans.md](file:///Users/trainguyen/Documents/work/agent/memory/global/active_plans.md)
- Cập nhật trạng thái workspace khi hoàn thành Giai đoạn 1.

## Open Questions

- Bạn có muốn đổi tên project từ `cdc-cms-web` thành một tên thương hiệu chung của GooPay không?
- Bạn có cần hỗ trợ đa ngôn ngữ (i18n) ngay trong giai đoạn này không? (Hiện tại đang là tiếng Anh).

## Verification Plan

### Automated Tests
- Kiểm tra build thành công bằng `npm run build`.

### Manual Verification
1.  **Login Flow**: Đăng nhập qua `cdc-auth-service` và lưu token vào localStorage.
2.  **Drift Approval**: Thực hiện Approve một field mới và kiểm tra xem database Postgres có được `ALTER TABLE` thành công hay không (thông qua CMS API).
3.  **Config Reload**: Kiểm tra xem sau khi Approve, Worker có tự động load mapping mới mà không cần restart (Verify qua logs của `centralized-data-service`).
