# audit_plan_multi_connection_refinement.md — Phản Biện & Thẩm Định Kỹ Thuật (Adversarial Audit)

> **Thời điểm**: 2026-09-14 17:05:00 +07:00  
> **Người thực hiện**: Role Brain (System Architect)  
> **Mục tiêu**: Áp dụng tư duy phản biện Staff Engineer để "bới lông tìm vết" toàn bộ bản kế hoạch Master Multi-Connection, bảo đảm tuân thủ nghiêm ngặt các nguyên tắc **Simplicity First, Minimal Impact, Zero Regressions**.

---

## I. 5 Câu Hỏi Phản Biện Cốt Lõi (Adversarial Challenge)

### 1. Phản biện 1: "Có cần thiết phải sửa cấu trúc DB (Migration) cho `cdc_system.transmute_jobs` không?"
* **Góc nhìn phản biện**: Bất kỳ migration nào cũng tiềm ẩn rủi ro lock bảng, lệch môi trường prod vs dev, hoặc tăng độ phức tạp không đáng có. Liệu có thể giải quyết mà KHÔNG cần migration?
* **Thẩm định kỹ thuật**:
  - `cdc_system.transmute_jobs` là bảng log lịch sử trạng thái job (không phải bảng chứa dữ liệu nghiệp vụ khách hàng).
  - Hiện tại bảng này chỉ có `master_table VARCHAR(128)` dạng text. Khi 2 Master Binding cùng trỏ về 1 tên bảng ở 2 DB khác nhau, việc lưu text làm mất hoàn toàn tính toàn vẹn quan hệ (Relational Integrity).
  - Tuy nhiên, để đảm bảo **Minimal Impact**:
    1. Lệnh DDL bổ sung cột: `ALTER TABLE cdc_system.transmute_jobs ADD COLUMN IF NOT EXISTS master_binding_id BIGINT;` là non-blocking và hoàn toàn an toàn trên Postgres (metadata-only update, không rewrite bảng).
    2. Cập nhật mã nguồn GORM struct: `MasterBindingID *int64 gorm:"column:master_binding_id"`.
    3. Mã nguồn đọc (`ListEnriched`) hỗ trợ dual-mode: Ưu tiên khớp `tj.master_binding_id = mb.id`; nếu job cũ chưa có binding ID (`tj.master_binding_id IS NULL`), chỉ match cho `default_master` nếu bảng đó `approved`.
  - **Kết luận**: Hợp lý, chuẩn kiến trúc DDD, an toàn 100%.

---

### 2. Phản biện 2: "Tại sao không đổi toàn bộ URL từ `/masters/:name/...` thành `/masters/:id/...`?"
* **Góc nhìn phản biện**: Đổi path param từ `:name` thành `:id` nghe có vẻ triệt để nhất?
* **Thẩm định kỹ thuật**:
  - Nếu đổi thẳng `/api/v1/masters/:id/approve` sẽ **gây breaking change** cho toàn bộ API client, swagger doc, script tự động, test suite hiện hữu đang gọi theo format `POST /api/v1/masters/{schema.table}/approve`.
  - **Giải pháp Simplicity First & Minimal Impact**:
    - Giữ nguyên path `/api/v1/masters/:name/approve`.
    - Trong Fiber handler:
      * Nếu client truyền query `?binding_id=123` hoặc body `{"binding_id": 123}` ➔ Ưu tiên xử lý trực tiếp theo `binding_id`.
      * Nếu `:name` là chuỗi số nguyên (vd `/api/v1/masters/123/approve`) ➔ Parse thành `binding_id`.
      * Nếu `:name` là `schema.table` và không có `binding_id`:
        - Nếu DB chỉ có duy nhất 1 bản ghi ➔ Tự động resolve bản ghi đó (hoàn toàn tương thích ngược 100% với các bảng cũ).
        - Nếu DB có > 1 bản ghi ➔ Báo lỗi `ambiguous_master_name` kèm hướng dẫn rõ ràng truyền `binding_id`.
  - **Kết luận**: Giữ nguyên Route, mở rộng khả năng tiếp nhận `binding_id`. Không gây gãy bất kỳ caller nào!

---

### 3. Phản biện 3: "Worker `MasterDDLHandler` và `TransmuterModule` sẽ xử lý NATS payload thế nào để không gãy backward compatibility?"
* **Góc nhìn phản biện**: Nếu NATS payload thêm trường `master_binding_id`, các message cũ hoặc các trigger khác (như realtime post-ingest) có bị lỗi Unmarshal không?
* **Thẩm định kỹ thuật**:
  - Trong Go, struct `json.Unmarshal` bỏ qua các trường không khai báo hoặc nhận `omitempty`:
    ```go
    type masterCreateRequest struct {
        MasterTable     string `json:"master_table"`
        MasterBindingID int64  `json:"master_binding_id,omitempty"`
        ...
    }
    ```
  - Trong hàm `loadBinding` (DDL Generator) và `loadMaster` (Transmuter):
    ```go
    // Ưu tiên 1: Tra cứu theo ID nếu có
    if bindingID > 0 {
        return loadByID(ctx, bindingID)
    }
    // Fallback: Tra cứu theo schema.table như cũ
    return loadByName(ctx, masterName)
    ```
  - Cả hai luồng mới và cũ đều hoạt động trơn tru.

---

### 4. Phản biện 4: "Lỗi hiển thị `Synced (509 / 509)`: Sửa ở Backend hay Frontend?"
* **Góc nhìn phản biện**: Chỉ sửa ở Frontend (chặn `if schema_status !== 'approved'`) có đủ không?
* **Thẩm định kỹ thuật**:
  - **Không đủ**. Nếu chỉ sửa Frontend, API `GET /api/v1/masters` vẫn trả về `last_transmute_status = 'COMPLETED'` và `last_transmute_rows = 509` trong JSON payload. Bất kỳ client nào khác đọc API này (như Grafana, CLI, automated tests) đều nhận dữ liệu rác/sai lệch.
  - **Nguyên tắc Core Systems**: Dữ liệu phải đúng từ tầng phát ra (Single Source of Truth).
  - Do đó, **bắt buộc sửa cả hai**:
    1. Backend `ListEnriched`: Thêm điều kiện `ON (mb.schema_status = 'approved')` và khớp `master_binding_id`. Khi bảng chưa approved, backend trả về `NULL`.
    2. Frontend `TransmuteStatusBadge`: Thêm phòng vệ bổ sung (defense-in-depth), nếu `record.schema_status !== 'approved'` luôn hiển thị Tag "Chưa sync".

---

### 5. Phản biện 5: "Khi tạo Schedule Sync (Manual / Cron / Realtime), mối liên kết giữa Schedule và MasterBinding có bị sai không?"
* **Góc nhìn phản biện**: Hiện tại khi bấm Sync trên UI, hàm `Save` của schedule query `SELECT id FROM master_binding WHERE master_table = ? ORDER BY id DESC LIMIT 1`.
* **Thẩm định kỹ thuật**:
  - Đây là lỗ hổng cực kỳ nghiêm trọng. `ORDER BY id DESC LIMIT 1` sẽ luôn chọn bản ghi có ID lớn nhất. Nếu User thao tác trên Master 1 (ID nhỏ hơn), Schedule sẽ bị gán nhầm vào Master 2 (ID lớn hơn)!
  - **Khắc phục**:
    - `cdc_system.transmute_schedule` đã có sẵn cột `master_binding_id BIGINT UNIQUE (master_binding_id, mode)`.
    - API `POST /api/v1/schedules` nhận `master_binding_id` từ client (`row.id`).
    - Hàm `Save` thực hiện `INSERT ... (master_binding_id, mode, ...) VALUES (?, ?, ...)` trực tiếp, triệt tiêu hoàn toàn câu query `LIMIT 1` mò mẫm.

---

## II. Danh Mục Chi Tiết Các File Cần Sửa (Minimal Impact Scope)

```
├── cdc-cms-service/
│   ├── internal/infra/persistence/master/master_repo_gorm.go           # Chuẩn hoá Resolve theo ID/Name, sửa ApproveSchemaTx, RejectSchemaTx, RevertSchemaTx
│   ├── internal/infra/persistence/master/master_read_repo_gorm.go      # Sửa query ListEnriched (ON approved & match binding_id)
│   ├── internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go # Sửa Save nhận trực tiếp master_binding_id
│   ├── internal/infra/persistence/transmute_job_repo.go                # Bổ sung master_binding_id vào struct và query
│   ├── internal/api/master/master_registry_handler_approve.go          # Nhận binding_id từ query/body/path
│   ├── internal/api/master/master_registry_handler_toggle.go           # Nhận binding_id
│   ├── internal/api/master/master_registry_handler_update_spec.go      # Nhận binding_id
│   ├── internal/app/commands/governance/approve_master.go              # Đính kèm master_binding_id vào NATS cdc.cmd.master-create
│   └── internal/app/commands/scheduler/run_now.go                      # Đính kèm master_binding_id vào NATS cdc.cmd.transmute
├── centralized-data-service/
│   ├── internal/handler/master/master_ddl_handler.go                   # Đọc master_binding_id từ cdc.cmd.master-create
│   ├── internal/service/master/master_ddl_generator.go                 # loadBinding hỗ trợ tra cứu theo ID
│   ├── internal/handler/master/transmute_handler.go                    # Đọc master_binding_id từ cdc.cmd.transmute
│   ├── internal/service/master/transmuter.go                           # loadMaster hỗ trợ tra cứu theo ID
│   └── internal/repository/transmute_job_repo.go                       # Hỗ trợ master_binding_id
└── cdc-cms-web/
    └── src/pages/MasterRegistry.tsx                                    # Truyền binding_id vào các mutation & TransmuteStatusBadge guard
```

---

## III. Kết Luận Thẩm Định
Bản kế hoạch sau khi được phản biện và tinh gọn:
1. **Không tạo migration phức tạp** (chỉ bổ sung 1 cột `master_binding_id` an toàn vào bảng log `transmute_jobs`).
2. **Không breaking change bất kỳ API nào** (vẫn giữ nguyên format đường dẫn, mở rộng tiếp nhận `binding_id`).
3. **Giải quyết triệt để 100% cả 2 lỗi** (Lỗi hiển thị Synced chéo và Lỗi 409 ambiguous_master_name).
4. **Bảo đảm tính toàn vẹn DDL và Data**: Bảng Master trên database đích `master_2` sẽ nhận đúng DDL và được ghi đúng dữ liệu khi transmute.
