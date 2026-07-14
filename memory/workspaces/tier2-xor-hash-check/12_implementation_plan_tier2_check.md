# Kế Hoạch Triển Khai Chi Tiết của AI (AI Session Implementation Plan)

## 1. Mục tiêu phiên làm việc
- Nghiên cứu và xác minh luồng đối soát Tier 2 (window-based XOR-hash) của centralized-data-service có chỉ đọc (read-only) và không ghi dữ liệu hay không.

## 2. Kế hoạch chi tiết của AI (Đã thực hiện)
- **Bước 1: Khởi động & Định vị**: Đọc `lessons.md` và `GEMINI.md` để nắm bắt các quy tắc cốt lõi của hệ thống.
- **Bước 2: Khởi tạo Workspace**: Tạo thư mục workspace `tier2-xor-hash-check` trong `work/agent/memory/workspaces/` và thiết lập các file context (`00_context.md`), yêu cầu (`01_requirements_tier2_check.md`), task list (`08_tasks_tier2_check.md`), và nhật ký tiến độ (`05_progress_tier2_check.md`).
- **Bước 3: Định vị mã nguồn**: Tìm kiếm mã nguồn thực tế của dịch vụ `centralized-data-service` trong thư mục workspace `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/`.
- **Bước 4: Phân tích mã nguồn**:
  - Đọc hàm `RunTier2` trong `recon_tier_a.go` để hiểu luồng chạy tổng quan (chia window, tính XOR-hash song song, drill-down list ID khi lệch hash, post-processing cross-check).
  - Đọc hàm `HashWindow` trong `recon_hash.go` (Source MongoDB/PostgreSQL) và `recon_dest_hash.go` (Destination Shadow DB PostgreSQL).
  - Đọc hàm `ListIDTsInWindow` trong `recon_stream.go` (Source) và `recon_dest_query.go` (Destination).
- **Bước 5: Xác minh tính chất Chỉ Đọc (Read-only)**:
  - Kiểm tra helper `readOnlyDB(ctx)` của `ReconDestAgent` trong `recon_dest_agent.go`. Xác nhận việc sử dụng `SET TRANSACTION READ ONLY` cấp độ PostgreSQL DB và rollback transaction.
  - Xác nhận không có câu lệnh INSERT, UPDATE, DELETE hay trigger heal dữ liệu nào được thực hiện trong suốt tiến trình Tier 2.
- **Bước 6: Tạo báo cáo phân tích**: Viết báo cáo chi tiết dưới dạng artifact `tier2_xor_hash_analysis.md`.
- **Bước 7: Sửa sai quy tắc ngôn ngữ**: Dịch toàn bộ các tệp tin trong workspace từ tiếng Anh sang tiếng Việt để tuân thủ nghiêm ngặt quy tắc "Luôn trả lời bằng tiếng Việt".
- **Bước 8: Đồng bộ phiên**: Tạo file `12_implementation_plan_tier2_check.md` để ghi nhận chi tiết kế hoạch triển khai của AI.
- **Bước 9: Phân tích Khóa Bảng (withTableLock)**: Xem xét cơ chế ghim connection database để giữ Postgres Advisory Lock, xác định rủi ro cạn kiệt connection pool dưới quy mô lớn (200 bảng * 50tr records) và đề xuất phương án chuyển dịch sang Redis Distributed Lock.
- **Bước 10: Xác thực ánh xạ ID và Timestamp**: Kiểm tra kiểu dữ liệu, định dạng (Epoch Ms vs Timezone), cơ chế tự động dịch Camel sang Snake case của tên cột và cơ chế fallback ObjectID của MongoDB.
- **Bước 11: Phân tích Đầu ra & Luồng Heal**: Điều tra bảng lưu trữ báo cáo (`cdc_system.cdc_reconciliation_report`) và cơ chế heal an toàn qua Debezium Signal (Segment A) và Transmute pipeline (Segment B).

## 3. Kế hoạch triển khai phiên hiện tại (Thực thi Code bởi Muscle)
- **Bước 1**: Đọc và phân tích mã nguồn hiện tại của Frontend (React) và Backend (Go) để xác định điểm cần sửa đổi.
- **Bước 2**: Thực hiện sửa đổi Frontend (cdc-cms-web):
  - `src/hooks/useReconStatus.ts`: Thêm tham số `lookback?: string` vào POST `/api/reconciliation/heal` mutation payload.
  - `src/components/ConfirmDestructiveModal.tsx`:
    - Ẩn time input (`startTime`/`endTime`) khi `mode === 'window'`.
    - Thêm radio buttons cho lookback mode: 'hot' (Hot Mode - 2h lookback) / 'cold' (Cold Lookback - 7d lookback), default 'cold'.
    - Chỉ hiển thị time input khi `mode === 'full_diff'`.
  - `src/pages/DataIntegrity.tsx`: Truyền tham số `lookback` vào `heal.mutateAsync`.
- **Bước 3**: Thực hiện sửa đổi Backend (centralized-data-service):
  - `internal/handler/recon/recon_handler_run.go`: Unmarshal thêm `lookback` từ NATS message payload và truyền vào `healSegmentA`.
  - `internal/handler/recon/recon_heal_v4.go`: Nhận `lookback` trong `healSegmentA`. Nếu `lookback == "hot"` thì dùng ctx gốc, nếu `lookback == "cold"` thì dùng context chứa key `"cold_lookback" = true` trước khi gọi `RunTier2`.
  - `internal/service/recon/recon_stream.go`: Kiểm tra `StreamIDsInTimeRange` MongoDB filter, đảm bảo có `$or` để lọc cả Date (time.Time) và Epoch Ms (int64).
- **Bước 4**: Xác minh và kiểm thử:
  - Chạy `npm run build` trên Frontend (`cdc-cms-web`) để đảm bảo build compile thành công.
  - Chạy `go test -v ./internal/handler/recon/...` và `go test -v ./internal/service/recon/...` để đảm bảo tất cả unit tests của Backend đều pass.
- **Bước 5**: Bổ sung tính năng cho luồng "Kiểm tra ID" (Tier 2):
  - Frontend:
    - `ConfirmDestructiveModal.tsx`: Thêm prop `isCheckTier2`. Nếu `true`, chỉ hiện radio buttons chọn Hot Mode / Cold Lookback (không hiện mode Window/Full-diff hay inputs date). Cập nhật `onConfirm` để trả về `lookback`.
    - `DataIntegrity.tsx`: Truyền prop `isCheckTier2` cho modal check Tier 2 và gửi `lookback` qua `checkTable.mutateAsync`.
    - `useReconStatus.ts`: Thêm `lookback` vào `useCheckTableMutation` payload và gửi lên body POST `/api/reconciliation/check`.
  - Backend:
    - `recon_handler_run.go`: Mở rộng unmarshal struct của NATS `HandleReconCheck` thêm `lookback`. Tùy theo `lookback == "cold"` để wrap context `"cold_lookback" = true` trước khi gọi `RunTier2` cho Tier 2 check.
- **Bước 8**: Giải quyết triệt để lỗi "kẹp upper lùi về quá khứ" khi chạy check/heal thủ công:
  - Backend (centralized-data-service):
    - `recon_handler_run.go`: Khi chạy check Tier 2, luôn set `"manual_lookback" = true` vào context.
    - `recon_heal_v4.go`: Khi chạy heal segment A, luôn set `"manual_lookback" = true` vào context cho cả Hot và Cold modes.
    - `recon_tier_a.go`: Trong `pickScanRangeWithLag`, kiểm tra nếu context có `"manual_lookback" = true`, bỏ qua việc kẹp `upper` lùi về `srcMax` và `dstMax` quá khứ, giữ `upper = nowFreeze` (thời gian thực tế) để quét đúng 7 ngày hoặc 2 giờ gần nhất.
  - API Gateway (cdc-cms-service):
    - `internal/app/commands/recon/recon_check.go`: Mở rộng struct `ReconCheckCommand` thêm trường `lookback` để unmarshal từ HTTP request body và truyền qua NATS.
    - `internal/app/commands/recon/recon_async.go`: Mở rộng struct `ReconHealCommand` thêm các trường `mode`, `start_time`, `end_time`, `lookback` để unmarshal và truyền qua NATS.
    - `internal/api/recon/reconciliation_handler_commands.go`: Tại cả 2 hàm `TriggerCheck` và `TriggerCheckAll`, unmarshal trường `lookback` từ HTTP request body và truyền vào `ReconCheckCommand`.
    - `internal/api/recon/reconciliation_handler_heal.go`: Tại hàm `TriggerHeal`, unmarshal các trường `mode`, `start_time`, `end_time`, `lookback` từ HTTP request body và truyền vào `ReconHealCommand`.
    - Khởi động lại process `cdc-cms-service` để load code mới.



- **Bước 9**: Xác minh và kiểm thử frontend compile + backend unit tests.
- **Bước 10**: Cập nhật nhật ký tiến độ và tạo walkthrough.

## 4. Kết quả thực tế phiên làm việc (Muscle Code Execution)
- Đã thực hiện thay đổi Frontend và Backend Go đầy đủ để hỗ trợ chọn lookback (Hot/Cold) cho đối soát Tier 2 (Check Table).
- Đã build thử Frontend (`npm run build`) và tất cả các module compile hoàn hảo.
- Đã chạy tests của Backend (`go test -v ./internal/handler/recon/...`) và kết quả là PASS 100%.
- Nhật ký tiến độ và danh sách thay đổi đã được ghi chép vào `05_progress_tier2_check.md` và `11_report_tier2_check.md` trong workspace.
- TUYỆT ĐỐI KHÔNG thực hiện chạy `git commit` hay `git add` theo chỉ định của User.
