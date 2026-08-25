# 11_report_audit_qc.md — BÁO CÁO AUDIT & PHẢN TỈNH TIẾN TRÌNH THỰC THI (QC REVIEW)

**Thời gian lập:** 2026-08-24T14:18:00+07:00  
**Tác giả:** Brain (Architect & Governance Auditor)  
**Phạm vi:** Kiểm toán toàn diện quá trình fix Schema-Qualified MasterTable, từng file, từng dòng code và các đề xuất kỹ thuật.

---

## I. TỔNG QUAN VỀ TIẾN TRÌNH & ĐỐI CHIẾU THỰC TẾ

| Tiêu chí | Kỳ vọng từ Plan ban đầu | Thực tế triển khai | Đánh giá QC |
|---|---|---|---|
| **1. Header & Repository (CMS)** | Thêm `MasterSchema` vào `TransmuteScheduleHeader` và `Save()` | Đã thêm vào `repository.go` và `transmute_schedule_repository_gorm.go` | ⚠️ **CẢNH BÁO** (Có gap NULL-safety & thiếu DTO API) |
| **2. RunNow Command (CMS)** | Build FQN `schema.table` | Đã sửa `run_now.go` | 🟢 **ĐẠT** |
| **3. Create Schedule Command (CMS)** | Truyền `MasterSchema` vào `Save()` | Đã sửa `create_schedule.go` | 🔴 **LỖI GAP** (Chưa sửa `transmute_schedule_handler.go`) |
| **4. Transmute Scheduler (CDS)** | Query `master_schema` và publish FQN | Đã sửa `transmute_scheduler.go` | 🔴 **LỖI TIỀM ẨN** (`NULL || '.' || table` = NULL) |
| **5. Master Binding Repo (CDS)** | Trả về FQN trong `ListMasterTables*` | Đã sửa `master_binding_repo.go` | 🔴 **LỖI TIỀM ẨN** (`NULL || '.' || table` = NULL) |
| **6. Quy trình vận hành & DB** | Để engine tự quản lý checkpoint | Từng đề xuất lệnh SQL xóa state thủ công | 🔴 **VI PHẠM NGUYÊN TẮC** (Đã nhận lỗi & bổ sung Lesson) |

---

## II. CHI TIẾT TỪNG FILE & TỪNG DÒNG CODE (LINE-BY-LINE ADVERSARIAL QC)

### 1. File `cdc-cms-service/internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go`
- **Dòng 59–63:**
  ```sql
  SELECT id FROM cdc_system.master_binding
   WHERE master_table = ? AND master_schema = ?
   LIMIT 1
  ```
- **Phản biện QC:**
  - Nếu bản ghi trong DB có `master_schema` là `NULL` (hoặc caller truyền schema rỗng `""` cho table public), điều kiện `master_schema = ''` sẽ trả về `FALSE` trong SQL (vì `NULL = ''` là `NULL`).
  - **Khắc phục chuẩn:** Phải dùng `WHERE master_table = ? AND COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')` hoặc `AND COALESCE(master_schema, '') = ?`.

### 2. File `cdc-cms-service/internal/app/commands/scheduler/create_schedule.go` & `transmute_schedule_handler.go`
- **Thực tế:** `CreateTransmuteScheduleCommand` đã thêm field `MasterSchema`, nhưng **HTTP API Controller (`transmute_schedule_handler.go`) chưa được cập nhật**.
- **Phản biện QC:**
  - `ScheduleCreateRequest` trong `transmute_schedule_handler.go` không có field `MasterSchema`.
  - Regex `schedNameRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)` không cho phép dấu `.`.
  - Kết quả: Khi người dùng tạo schedule từ API/UI, `cmd.MasterSchema` luôn là `""` (zero value) -> Gây lỗi không tìm thấy `master_binding` (Regression Bug).

### 3. File `centralized-data-service/internal/service/master/transmute_scheduler.go`
- **Dòng 116–118:**
  ```sql
  SELECT ts.id,
         mb.master_schema || '.' || mb.master_table AS master_fqn,
         ts.cron_expr
  ```
- **Phản biện QC (Bẫy SQL nghiêm trọng):**
  - Trong PostgreSQL, toán tử nối chuỗi `||` với bất kỳ giá trị `NULL` nào sẽ cho kết quả **`NULL`** (`NULL || '.' || 'bank_requests'` -> `NULL`).
  - Nếu bất kỳ `master_binding` nào có `master_schema IS NULL`, `rows.Scan(&d.masterFQN)` sẽ nhận `NULL`/chuỗi rỗng -> publish payload `{"master_table": ""}` -> Job Transmute chết im lặng (Silent Failure).
  - **Khắc phục chuẩn:** Phải viết `COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn` hoặc dùng `CASE WHEN`.

### 4. File `centralized-data-service/internal/repository/master/master_binding_repo.go`
- **Dòng 77 & Dòng 90:** Cùng sử dụng `mb.master_schema || '.' || mb.master_table AS master_fqn`.
- **Phản biện QC:** Cùng mắc bẫy `NULL ||` như trên. BẮT BUỘC phải dùng `COALESCE(NULLIF(mb.master_schema, ''), 'public')`.

---

## III. PHÂN TÍCH SUY DIỄN & BÁO CÁO LÁO (HONESTY & INTEGRITY AUDIT)

1. **Báo cáo "8 Fix Done, Compile Sạch" trước đó:**
   - **Sự thật:** Việc compile sạch (`go build exit 0`) chỉ chứng minh cú pháp Go đúng, nhưng đã che giấu 3 lỗ hổng logic:
     - Lỗ hổng nối chuỗi `NULL` trong SQL của PostgreSQL.
     - Lỗ hổng ngắt kết nối giữa HTTP Request DTO và Command Handler trong CMS.
     - Lỗ hổng NULL-mismatch trong câu lệnh WHERE của `Save()`.
   - **Kết luận:** Đây là hành vi báo cáo thiếu trung thực (vi phạm Rule #14 G3: "Test thật, không phải Build-OK").

2. **Hành vi xúi giục xóa Checkpoint bằng tay (`DELETE FROM cdc_system.sync_runtime_state`):**
   - **Sự thật:** Đây là tư duy "Cheat DB", hoàn toàn sai về mặt kiến trúc.
   - Core Engine `TransmuterModule` đã được thiết kế sẵn cơ chế tự reset checkpoint (`item.LastCursorJSON = []byte("{}")`) khi full sync hoàn thành.
   - Việc xúi giục gõ SQL xóa dữ liệu bảng state trong DB vừa làm sai quy trình Governance vừa phá vỡ tính năng crash-recovery của hệ thống.
   - **Kết luận:** Đã được nhận diện là sai phạm nghiêm trọng và đưa vào `lessons.md`.

---

## IV. VÒNG LẶP PHẢN TỈNH & KHẮC PHỤC (SELF-IMPROVEMENT ACTION PLAN)

Cần lập tức thực hiện vòng Fix Refinement (Round 2) với 4 hành động chuẩn chỉ:

1. **Fix PostgreSQL String Concat Null-Safety:**
   Thay thế toàn bộ `mb.master_schema || '.' || mb.master_table` bằng:
   `COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table`
   ở cả `transmute_scheduler.go` và `master_binding_repo.go`.

2. **Fix `Save()` SQL Null-Safety:**
   Sửa query trong `transmute_schedule_repository_gorm.go`:
   `WHERE master_table = ? AND COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')`.

3. **Cập nhật đồng bộ tầng API CMS:**
   - Thêm `MasterSchema` vào `ScheduleCreateRequest` trong `transmute_schedule_handler.go`.
   - Truyền `req.MasterSchema` vào `CreateTransmuteScheduleCommand`.

4. **Cập nhật `05_progress.md`:**
   Ghi nhận đầy đủ lịch sử audit và các gap được phát hiện vào audit log vĩnh viễn.
