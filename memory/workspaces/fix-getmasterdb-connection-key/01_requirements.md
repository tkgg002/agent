# 01_requirements.md — Detailed Specs & Requirements

## I. YÊU CẦU CHỨC NĂNG (FUNCTIONAL REQUIREMENTS)

### [REQ-01] Schema-Qualified Master Identification
- Mọi định danh bảng Master khi giao tiếp qua NATS, lưu trữ Runtime State, và trigger Transmute BẮT BUỘC phải ở định dạng đầy đủ FQN: `<master_schema>.<master_table>` (Ví dụ: `master_bidv_connector_service.bank_requests`).
- Nếu `master_schema` rỗng hoặc NULL trong database cũ, BẮT BUỘC fallback về `'public'` hoặc schema mặc định được cấu hình.

### [REQ-02] CMS API & DTO Completeness
- Endpoint `POST /api/v1/schedules` và struct `ScheduleCreateRequest` BẮT BUỘC hỗ trợ field `master_schema`.
- Regex validation `schedNameRe` được áp dụng độc lập cho cả `master_schema` (optional) và `master_table` (required).
- Controller `Create()` phải parse `req.MasterSchema` và đưa vào `CreateTransmuteScheduleCommand`.

### [REQ-03] Repository & Persistence Safety
- Method `Save()` trong `TransmuteScheduleRepository` phải nhận cả `masterSchema` và `masterTable`, thực hiện query an toàn với `NULL`:
  ```sql
  WHERE master_table = ? AND COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')
  ```
- Method `GetHeaderByID()` phải SELECT `mb.master_schema` cùng `mb.master_table`.

### [REQ-04] CDS Worker & Scheduler Accuracy
- `TransmuteScheduler.tick()` phải query `COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn` để chống lỗi `NULL` concatenation trong PostgreSQL.
- `MasterBindingRepo.ListMasterTablesByShadowTable` và `ListMasterTablesByShadowIdentity` phải trả về FQN an toàn với `NULL`.

---

## II. YÊU CẦU PHI CHỨC NĂNG (NON-FUNCTIONAL & GOVERNANCE)

### [NFR-01] Zero DB Tampering (Anti-DB Cheat)
- Nghiêm cấm mọi hành vi can thiệp xóa/sửa dữ liệu thủ công trong `cdc_system.sync_runtime_state`.
- Việc reset cursor phải tuân theo vòng đời tự nhiên của `TransmuterModule` khi kết thúc Full Sync.

### [NFR-02] Backward Compatibility
- Nếu hệ thống nhận được tên bảng thuần không có dấu `.` từ các client cũ, `loadMaster()` vẫn fallback an toàn nhưng ghi log WARN để truy vết.
