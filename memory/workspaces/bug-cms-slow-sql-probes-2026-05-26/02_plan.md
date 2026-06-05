# Kế hoạch thực hiện (Implementation Plan) - EN/VI

## Mục tiêu / Goal
Khắc phục triệt để lỗi SLOW SQL (>= 200ms) tại cdc-cms-service cho các câu truy vấn định kỳ của healthcheck.
Thoroughly fix the SLOW SQL (>= 200ms) issues in cdc-cms-service for periodic healthcheck queries.

## Các bước thực hiện (Phân chia vai trò Brain/Muscle) / Execution Steps (Brain/Muscle Roles)

### Giai đoạn 1: Nghiên cứu & Thiết kế (Brain) / Phase 1: Research & Design (Brain)
1. [x] Phân tích log SQL chậm và xác định các hàm/file liên quan. / Analyze slow SQL logs and identify related functions/files.
2. [x] Kiểm tra query plan của các SQL liên quan trực tiếp trên database. / Inspect query plans of related SQL directly in the database.
3. [x] Đề xuất hai hướng tối ưu chính: / Propose two main optimizations:
   - Thay thế hàm `NOW()` trong SQL bằng mốc thời gian tĩnh được tính toán từ Go (`time.Now()`) và truyền qua placeholder `?`. / Replace `NOW()` in SQL with static timestamps computed in Go and passed via placeholders `?`.
   - Vô hiệu hóa `PrepareStmt` của GORM cho các truy vấn healthcheck/probes qua `db.Session(&gorm.Session{PrepareStmt: false})`. / Disable GORM's `PrepareStmt` for healthcheck/probe queries via `db.Session(&gorm.Session{PrepareStmt: false})`.

### Giai đoạn 2: Thực thi (Muscle) / Phase 2: Execution (Muscle)
1. [ ] Cập nhật file `internal/infra/observability/probes/postgres.go` để chạy Session với `PrepareStmt: false`. / Update `internal/infra/observability/probes/postgres.go` to use Session with `PrepareStmt: false`.
2. [ ] Cập nhật file `internal/infra/observability/system_health_queries.go` để: / Update `internal/infra/observability/system_health_queries.go` to:
   - Chạy các truy vấn thông qua Session `PrepareStmt: false`. / Run queries via Session with `PrepareStmt: false`.
   - Tính toán các giá trị timestamp trong Go và truyền vào câu lệnh SQL thay vì dùng `NOW() - INTERVAL '...'`. / Compute timestamp values in Go and pass them to SQL instead of using `NOW() - INTERVAL '...'`.
3. [ ] Build và chạy thử cục bộ service để đảm bảo không lỗi cú pháp hay runtime crash. / Build and run the service locally to verify no syntax errors or runtime crashes.

### Giai đoạn 3: Kiểm chứng & Báo cáo (QA & Verify) / Phase 3: Verify & Report (QA & Verify)
1. [ ] Thực hiện chạy thử background collector và theo dõi log để xác nhận không còn cảnh báo SLOW SQL >= 200ms. / Run the background collector and monitor logs to verify no more SLOW SQL >= 200ms warnings.
2. [ ] Cập nhật tài liệu tiến độ (`05_progress.md`), quyết định (`04_decisions.md` nếu có), và lessons learned (`lessons.md`). / Update progress logs (`05_progress.md`), decisions (`04_decisions.md` if any), and lessons learned (`lessons.md`).
