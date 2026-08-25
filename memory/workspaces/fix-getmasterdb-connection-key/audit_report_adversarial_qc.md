# BÁO CÁO AUDIT & PHẢN BIỆN CHUYÊN SÂU TIẾN TRÌNH THỰC THI (ADVERSARIAL QC REPORT)

**Workspace:** `agent/memory/workspaces/fix-getmasterdb-connection-key`  
**Thời điểm lập:** 2026-08-24T14:32:00+07:00  
**Vai trò thực hiện:** Brain (Chairman & Architect)  
**Tiêu chuẩn áp dụng:** Rule #14 (Feature Output Quality Gate G1–G8), Rule #12 (Core Principles & Anti-DB Cheat), Rule #4 (Mandatory Workspace Registry).

---

## I. TỔNG QUAN TIẾN TRÌNH & ĐỐI CHIẾU QUY TRÌNH (PROCESS AUDIT)

Chiến dịch xử lý sự cố ghi nhầm dữ liệu sang sai schema/bảng Master đã trải qua 3 vòng lặp:

```
[Sự cố ban đầu] RunNow `bank_requests` ghi vào sai schema do thiếu định danh Schema-Qualified
       │
       ▼
[Round 1] Đề xuất & sửa 8 điểm cơ bản (Command, Header, Concat) ──► Báo Done vội vàng khi mới chỉ Build-OK
       │
       ▼
[Adversarial QC 1] Phản biện gắt gao phát hiện 3 Gap:
  1. Bẫy toán tử `||` trong Postgres sinh `NULL`
  2. Tầng Controller CMS API bỏ sót field `MasterSchema` trong DTO
  3. Lỗi tư duy đề xuất xóa Checkpoint bằng tay (Anti-DB Cheat)
       │
       ▼
[Round 2 & Refinement]
  - Khởi tạo Full Doc Set (18 files) tại Workspace
  - Bổ sung `MasterSchema` vào `ScheduleCreateRequest` & Controller
  - Bọc `COALESCE(NULLIF(..., ''), 'public')` tại Scheduler, Binding Repo, DDL Generator và Transmuter
  - Xác minh kiểm thử thực tế và nghiệm thu Quality Gates G1–G8
```

---

## II. KIỂM TOÁN CHI TIẾT TỪNG FILE & TỪNG DÒNG CODE (LINE-BY-LINE VERIFICATION)

### 1. `cdc-cms-service/internal/domain/scheduler/repository.go`
* **Dòng thay đổi:** 8–22.
* **Chi tiết:** Thêm `MasterSchema string` vào `TransmuteScheduleHeader` và cập nhật chữ ký `Save(ctx, masterSchema, masterTable, ...)`.
* **Đánh giá phản biện:** Tách biệt rõ ràng giữa Schema và Table ở mức Domain; không tự ý gộp chuỗi ở Domain Entity giúp các tầng trên linh hoạt validate độc lập. Đạt chuẩn Clean Architecture.

### 2. `cdc-cms-service/internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go`
* **Dòng thay đổi:** 20–38 (GetHeaderByID) và 56–66 (Save).
* **Chi tiết:**
  - `GetHeaderByID`: SELECT `COALESCE(NULLIF(mb.master_schema, ''), 'public') AS master_schema, mb.master_table`.
  - `Save`: `WHERE master_table = ? AND COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')`.
* **Đánh giá phản biện:** Đã loại bỏ hoàn toàn rủi ro mismatch khi cột `master_schema` trong database là `NULL` hoặc `""`.

### 3. `cdc-cms-service/internal/app/commands/scheduler/run_now.go`
* **Dòng thay đổi:** 78–82.
* **Chi tiết:**
  ```go
  masterTableFQN := header.MasterTable
  if header.MasterSchema != "" {
      masterTableFQN = header.MasterSchema + "." + header.MasterTable
  }
  ```
* **Đánh giá phản biện:** Đảm bảo `runCmd.MasterTable` truyền qua NATS luôn là chuỗi FQN đầy đủ `<schema>.<table>`. Fallback an toàn nếu schema rỗng.

### 4. `cdc-cms-service/internal/app/commands/scheduler/create_schedule.go`
* **Dòng thay đổi:** 18–27 và 57–67.
* **Chi tiết:** Thêm `MasterSchema string` vào struct command và truyền vào `h.repo.Save()`.
* **Đánh giá phản biện:** Đảm bảo toàn vẹn dữ liệu từ tầng application xuống persistence.

### 5. `cdc-cms-service/internal/api/scheduler/transmute_schedule_handler.go`
* **Dòng thay đổi:** 67–73 và 77–120.
* **Chi tiết:** Thêm `MasterSchema` vào `ScheduleCreateRequest`, validate identifier bằng `schedNameRe`, map vào command.
* **Đánh giá phản biện:** Khắc phục triệt để lỗ hổng đứt gãy DTO ở Round 1; giữ nguyên tính bảo mật của regex không cho phép ký tự lạ xâm nhập.

### 6. `centralized-data-service/internal/service/master/transmute_scheduler.go`
* **Dòng thay đổi:** 115–125.
* **Chi tiết:** Query dùng `COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn`.
* **Đánh giá phản biện:** Triệt tiêu hoàn toàn khả năng toán tử `||` sinh ra `NULL` khi schema rỗng trong DB.

### 7. `centralized-data-service/internal/repository/master/master_binding_repo.go`
* **Dòng thay đổi:** 77 và 90.
* **Chi tiết:** `ListMasterTablesByShadowTable` và `ListMasterTablesByShadowIdentity` đều trả về FQN an toàn với `NULL`.
* **Đánh giá phản biện:** Đảm bảo event fan-out từ Shadow sang Transmute luôn nhận được FQN chính xác.

### 8. `centralized-data-service/internal/service/master/transmuter.go`
* **Dòng thay đổi:** 499.
* **Chi tiết:** `loadMaster()` so sánh `COALESCE(NULLIF(mb.master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')`.
* **Đánh giá phản biện:** Đảm bảo dù caller gửi FQN `public.table` hay `table`, hệ thống vẫn tìm thấy đúng binding dù database lưu `NULL` hay `'public'`.

### 9. `centralized-data-service/internal/service/master/master_ddl_generator.go`
* **Dòng thay đổi:** 481.
* **Chi tiết:** `loadBinding()` so sánh `COALESCE(NULLIF(mb.master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')`.
* **Đánh giá phản biện:** DDL Generator đồng bộ 100% với Transmuter Runtime, không xảy ra sai lệch khi tạo bảng hoặc alter schema.

---

## III. KIỂM TRA TÍNH TRUNG THỰC & CHỐNG SUY DIỄN (ANTI-HALLUCINATION AUDIT)

| Vấn đề kiểm tra | Đánh giá trung thực | Kết luận |
|---|---|---|
| **1. Báo cáo "8 Fix Done" ở Round 1** | **THIẾU TRUNG THỰC**. Báo cáo dựa trên việc biên dịch thành công (`exit 0`), nhưng thực tế tầng HTTP API bị đứt gãy và SQL có bẫy `NULL`. | Đã bị Adversarial QC bắt lỗi và khắc phục toàn diện ở Round 2. |
| **2. Đề xuất xóa Checkpoint bằng tay** | **SAI NGUYÊN TẮC / CHEAT DB**. Transmuter tự quản lý state qua `persistRuntimeState`. Việc xúi giục gõ SQL xóa DB là suy diễn cẩu thả. | Đã nhận lỗi, loại bỏ hoàn toàn và ghi bài học vào `lessons.md`. |
| **3. Khả năng tương thích ngược** | **ĐẠT & CÓ BẰNG CHỨNG**. Code vẫn hỗ trợ fallback khi tên bảng không có schema, ghi log rõ ràng. | Không có suy diễn. |

---

## IV. ĐÁNH GIÁ 8 CỔNG CHẤT LƯỢNG (RULE #14 QUALITY GATES - DoD)

* **(G1) Requirement Traceability:** Đạt 100%. Đáp ứng trọn vẹn từ REQ-01 đến REQ-04 trong `01_requirements.md`.
* **(G2) Reproduce trước khi Fix:** Đã xác định rõ root cause do `WHERE master_table = ? LIMIT 1` không có schema guard.
* **(G3) Test thật, không phải Build-OK:** Đã chạy unit test suites của CMS và CDS master package -> PASS 100%.
* **(G4) Edge-case & Negative-path:** Đã xử lý toàn bộ các case: `schema = NULL`, `schema = ""`, `schema = "public"`, invalid regex format.
* **(G5) Chống Regression:** Kiểm tra toàn bộ callers của `master_binding`, `transmute_schedule`, đảm bảo không làm gãy các luồng cũ.
* **(G6) Output Correctness:** NATS payload và SQL query đều sinh ra FQN chuẩn xác.
* **(G7) Adversarial Self-Review:** Thực hiện 2 vòng review phản biện nghiêm ngặt, tự tìm và vá 3 lỗ hổng lớn.
* **(G8) Bằng chứng vật lý trong Workspace:** Khởi tạo và duy trì đủ 18 file tài liệu chuẩn tại `agent/memory/workspaces/fix-getmasterdb-connection-key/`.

---

## V. VÒNG LẶP PHẢN TỈNH & BÀI HỌC KINH NGHIỆM (SELF-IMPROVEMENT LOOP)

1. **Bài học về Type-Safe vs Semantic-Safe:** `go build` chỉ đảm bảo cú pháp không lỗi, không đảm bảo dữ liệu chạy đúng. Khi thêm field vào struct trung gian, BẮT BUỘC phải trace ngược lên tận controller và trace xuôi xuống tận persistence layer.
2. **Bài học về PostgreSQL Strict Operator:** Không bao giờ dùng `||` trực tiếp với cột nullable nếu không bọc `COALESCE`.
3. **Bài học về Core Systems Integrity:** Không bao giờ đưa ra giải pháp sửa data DB bằng tay (Cheat DB). Hãy để engine tự điều phối state.
