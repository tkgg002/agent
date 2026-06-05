# 08_tasks.md — Danh Sách Nhiệm Vụ Delegate

Tài liệu này lưu trữ danh sách nhiệm vụ chi tiết được delegate từ Brain (Chairman) sang Muscle (Chief Engineer) để thực thi.

---

## Task 1: Database Migration & Sửa Lỗi Schedules API
- **Phase**: GĐ0 (Rà soát & DB Setup)
- **Service Group**: Utilities
- **Service(s)**: cdc-cms-service
- **Mô tả**: Tạo file migration cho bảng master mapping rules mới và sửa lỗi API schedules bị 500 do thay đổi schema chuẩn hóa V2.
- **Trạng thái**: [/] IN_PROGRESS

### [Context]
- Bảng `cdc_system.transmute_schedule` đã đổi từ cột `master_table` sang cột `master_binding_id`. API handler `CreateTransmuteScheduleHandler` vẫn đang insert trực tiếp cột legacy `master_table`.
- Cần tạo bảng mới `cdc_system.mapping_rule_master` để cô lập mapping rules của master.

### [Definition of Done]
- [ ] Tạo file migration `migrations/schema/cdc_system_model/074_v2_mapping_rule_master.sql` chứa câu lệnh tạo bảng `cdc_system.mapping_rule_master` và index như thiết kế trong plan.
- [ ] Sửa `CreateTransmuteScheduleHandler` trong `cdc-cms-service/internal/app/commands/create_schedule.go`:
  - Trước khi insert schedule, thực hiện SELECT query lookup `master_binding_id` từ `cdc_system.master_binding` bằng `MasterTable`.
  - Thực hiện UPSERT sử dụng `master_binding_id` thay cho `master_table`.
- [ ] CMS service build thành công không lỗi biên dịch (`cd cdc-cms-service && go build ./...`).
- [ ] **[QA Gate]**: Không có lỗi build.
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.

---

## Task 2: Backend CMS Service — Logic Clone Rules & REST API cho Master Mappings
- **Phase**: GĐ3 (Architecture)
- **Service Group**: Utilities
- **Service(s)**: cdc-cms-service
- **Mô tả**: Viết logic tự động clone shadow rules sang master mapping rules khi tạo/approve master. Xây dựng REST API Handlers hoàn chỉnh cho bảng `mapping_rule_master`, bao gồm API tự động scan & flatten JSON fields.
- **Trạng thái**: [ ] TODO

### [Context]
- Khi tạo master table, ta cần clone các active mapping rules của shadow tương ứng sang master, loại bỏ các system/internal columns.
- Cần cung cấp đầy đủ các API CRUD, batch update, và flatten để Frontend và operator quản lý mapping rules cho master.

### [Definition of Done]
- [ ] Sửa logic tạo master (`CreateMasterHandler` hoặc `ApproveMasterHandler` trong `cdc-cms-service`) để auto-clone default rules:
  - Query active shadow rules của shadow binding từ `mapping_rule_v2`.
  - Filter loại bỏ các system columns (`_gpay_id`, `_raw_data`, `_synced_at`, `progress`, `status`, `params`, `error`, `_id`, v.v.).
  - Insert các bản ghi này vào bảng `mapping_rule_master` liên kết với `master_binding_id` mới tạo.
- [ ] Tạo Domain Model `MasterMappingRule` và interface `MasterMappingRuleRepo` trong `cdc-cms-service/internal/domain/mapping/`.
- [ ] Tạo persistence repository GORM cho `mapping_rule_master` trong `cdc-cms-service/internal/infra/persistence/`.
- [ ] Tạo API Handlers trong `cdc-cms-service/internal/api/` hỗ trợ:
  - `GET /api/v1/master-mapping-rules?master_binding_id=X` (List rules)
  - `POST /api/v1/master-mapping-rules` (Create/Update rule)
  - `PATCH /api/v1/master-mapping-rules/:id` (Update fields/status)
  - `PATCH /api/v1/master-mapping-rules/batch` (Batch approve/reject)
  - `DELETE /api/v1/master-mapping-rules/:id` (Delete rule)
  - `POST /api/v1/master-mapping-rules/flatten` (Auto-flatten JSON logic):
    - Đọc mẫu dữ liệu từ shadow table tương ứng (SELECT 5 dòng).
    - Parse JSON trích xuất các nested keys.
    - Tạo các rule `pending` trong `mapping_rule_master`.
- [ ] Đăng ký routes mới trong `internal/api/router.go`.
- [ ] CMS service build thành công không lỗi biên dịch (`go build ./...`).
- [ ] **[QA Gate]**: Viết unit test cơ bản cho logic clone và logic flatten.
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.

---

## Task 3: Worker Service — Cấu Hình Load Rules Từ mapping_rule_master
- **Phase**: GĐ3 (Architecture)
- **Service Group**: Utilities
- **Service(s)**: centralized-data-service (transmuter)
- **Mô tả**: Cập nhật worker service để load rules trực tiếp từ bảng `mapping_rule_master` dựa theo `master_binding_id`, loại bỏ fallback logic cũ.
- **Trạng thái**: [ ] TODO

### [Context]
- Transmuter trong worker load mapping rules để map và sync shadow -> master. Trước đây nó query từ `mapping_rule_v2` với logic fallback phức tạp.
- **CẢNH BÁO AN TOÀN**: Tuyệt đối không thay đổi bất kỳ logic hay database query nào của pipeline Source -> Shadow (sử dụng `mapping_rule_v2`). Chỉ thực hiện sửa đổi trong `TransmuterModule.loadRules` (trong file `internal/service/transmuter.go`) chuyên biệt cho việc biến đổi dữ liệu từ Shadow -> Master.

### [Definition of Done]
- [ ] Sửa hàm `loadRules` trong `centralized-data-service/internal/service/transmuter.go`:
  - Query rules trực tiếp từ bảng `cdc_system.mapping_rule_master` lọc theo `master_binding_id = ?` và `is_active = true` và `status = 'approved'`.
  - Loại bỏ hoàn toàn fallback query `master_binding_id IS NULL`.
- [ ] Worker service build thành công (`cd centralized-data-service && go build ./...`).
- [ ] **[QA Gate]**: Worker build PASS.
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.


---

## Task 4: Frontend Web — Nâng Cấp UI MasterMappingsPanel
- **Phase**: GĐ4 (Observability & UI)
- **Service Group**: Utilities
- **Service(s)**: cdc-cms-web
- **Mô tả**: Xây dựng trình quản lý mapping rules đầy đủ (10 cột, modal, add/edit/delete, batch approve/reject, flatten action) trên UI Master Registry.
- **Trạng thái**: [ ] TODO

### [Context]
- UI expandable row hiện tại của Master Registry chỉ hiển thị list mappings read-only, cần nâng cấp lên đầy đủ tính năng như trang mappings của Shadow.

### [Definition of Done]
- [ ] Cập nhật API client services trong `cdc-cms-web/src/services/api.ts` hỗ trợ gọi các API mới của `master-mapping-rules`.
- [ ] Nâng cấp component `MasterMappingsPanel` trong `MasterRegistry.tsx`:
  - Table hiển thị 10 cột mapping đúng yêu cầu (Source Field, Target Column, Source Data Type, Data Type Target, Rule Type, Status, Active Switch, Sensitive Switch, Mask Strategy Select, Actions).
  - Tích hợp Modal Add/Edit mapping rule.
  - Tích hợp nút **Flatten** cho các cột có kiểu dữ liệu JSON/JSONB.
  - Tích hợp Modal nhập Explode Path cho Flatten.
  - Tích hợp Batch actions (Approve, Reject) phía trên bảng.
- [ ] Frontend Web build thành công không lỗi typescript (`cd cdc-cms-web && npm run build`).
- [ ] **[QA Gate]**: Không có lỗi type.
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.
