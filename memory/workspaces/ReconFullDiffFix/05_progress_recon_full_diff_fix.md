# Audit Log & Tiến độ - Sửa lỗi đối soát Full Search (Full Diff)

## Nhật ký Tiến độ (Audit Log)
- [2026-07-06T10:23:00Z] [Agent:Gemini] Đọc `GEMINI.md` và `lessons.md`. Khởi tạo workspace `ReconFullDiffFix` để quản lý các file theo dõi cho task này.
- [2026-07-06T10:23:30Z] [Agent:Gemini] Tạo file yêu cầu `01_requirements_recon_full_diff_fix.md` và tiến độ `05_progress_recon_full_diff_fix.md`.

## Phân tích Nguyên nhân & Hiện trạng (Root Cause & Status Analysis)
1. **Hiện trạng:** Chế độ đối soát `full_diff` (Full Search) sử dụng khoảng thời gian từ UI truyền xuống qua API để lấy ra danh sách các ID của bản ghi từ Source và Shadow DB, sau đó so khớp để tìm ra các bản ghi bị lệch hoặc bị thiếu ở Shadow DB.
2. **Nguyên nhân gốc rễ (Root Cause):**
   - Hàm `TimeBoundedDiffMissingFromShadow` giải quyết trường timestamp đích (`dstTS`) thông qua cơ chế fallback. Đối với nguồn dữ liệu MongoDB, cột `dstTS` trong Shadow DB (Postgres) thường là `_source_ts` (kiểu `BIGINT` lưu trữ epoch millisecond).
   - Khi thực hiện truy vấn SELECT lấy ID từ Shadow DB, tham số thời gian (`startTime`, `endTime`) được truyền trực tiếp dưới dạng `time.Time`.
   - Postgres không hỗ trợ so sánh trực tiếp kiểu `BIGINT` với `TIMESTAMP WITH TIME ZONE` (hoặc định dạng timestamp string tương đương do GORM tự động map), điều này làm câu lệnh SQL bị lỗi kiểu dữ liệu hoặc trả về tập rỗng (empty results).
   - Vấn đề tương tự xảy ra nếu nguồn dữ liệu là PostgreSQL và trường timestamp là kiểu `BIGINT` hoặc `INTEGER`.
3. **Giải pháp:**
   - Tạo logic phân tích kiểu dữ liệu của cột timestamp (`dstTS` hoặc `timestampField` / `tsField`).
   - Nếu kiểu dữ liệu của cột trong DB chứa `int` hoặc `num` (hoặc là cột mặc định `_source_ts`), chuyển đổi `startTime` và `endTime` từ `time.Time` thành epoch milliseconds (hoặc epoch seconds tùy thuộc vào dải giá trị trong DB).
   - Cập nhật logic này tại 3 vị trí chính trong `internal/service/recon/`:
     1. `TimeBoundedDiffMissingFromShadow` trong `recon_tier_a.go`.
     2. `listIDsInWindowPostgres` trong `recon_stream.go`.
     3. `streamIDsPostgresInTimeRange` trong `recon_stream.go`.

- [2026-07-07T03:18:00Z] [Agent:Gemini] Lập kế hoạch triển khai chi tiết và đồng bộ lên `implementation_plan.md` cùng `12_implementation_plan_recon_full_diff_fix.md`, yêu cầu User xem xét.
- [2026-07-07T03:54:15Z] [Agent:Gemini] Tiếp nhận phản hồi từ User yêu cầu loại bỏ khái niệm phân loại 'tier' cũ để dùng 'type_recon' tường minh xuyên suốt FE, CMS và CDS. Đã lập kế hoạch refactor chi tiết tại '09_tasks_solution_refactor_type_recon.md', đồng bộ sang 'implementation_plan.md' và '12_implementation_plan_recon_full_diff_fix.md', yêu cầu User duyệt.
- [2026-07-07T11:15:25+07:00] [Antigravity:Muscle-Execute] Bắt đầu thực thi Phase 1: Refactor CDS backend (centralized-data-service).
- [2026-07-07T11:23:45+07:00] [Antigravity:Muscle-Execute] Hoàn thành Phase 1: Thay thế payload.Tier bằng payload.TypeRecon và switch-case logic trong recon_check_handler.go, cập nhật TimeBoundedDiffMissingFromShadow trả về destCount, compile & test pass.
- [2026-07-07T11:24:15+07:00] [Antigravity:Muscle-Execute] Bắt đầu thực thi Phase 2: Refactor cdc-cms-service (CMS backend).



