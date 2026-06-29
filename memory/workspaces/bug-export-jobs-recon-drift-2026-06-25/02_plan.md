# Plan: Investigating and Fixing Export Jobs Reconciliation Drift

## Phase 1: Research & Diagnose
- **Mục tiêu**: Tìm ra nguyên nhân tại sao `segment_b_window` đối soát báo `destCount = 0` trên master table, trong khi `count_total` báo `ok` (đều bằng 452).
- **Các bước thực hiện**:
  1. Brain giao nhiệm vụ cho Muscle chạy script kiểm tra (hoặc trực tiếp truy vấn qua psql/go run) để xem giá trị `_source_ts` thực tế của các dòng trong bảng `shadow_testdel.export_jobs` (ở shadow DB) và `master.export_jobs` (hoặc schema tương ứng ở dest DB).
  2. Xác minh xem giá trị `_source_ts` ở master table có bị NULL, 0, hay lệch timezone/lookback window hay không.
  3. Nếu `_source_ts` bị NULL hoặc 0, kiểm tra mã nguồn `transmuter.go` (nơi ghi dữ liệu từ shadow sang master) để xem logic gán `_source_ts` có bị lỗi/skip hay không.
  4. Nếu `_source_ts` có giá trị đầy đủ nhưng lệch window, rà soát lại cách tính `lower` và `upper` watermark trong `recon_tier_b.go`.

## Phase 2: Implementation & Fix
- **Mục tiêu**: Khắc phục triệt để lỗi ghi nhận `_source_ts` hoặc logic filter window.
- **Các bước thực hiện**:
  1. Cập nhật code tương ứng (trong `transmuter.go` hoặc `recon_tier_b.go`).
  2. Đảm bảo dữ liệu cũ (backfill) được xử lý nếu cần thiết.

## Phase 3: Verification
- **Mục tiêu**: Đảm bảo không còn drift giả hoặc drift thật giữa shadow và master.
- **Các bước thực hiện**:
  1. Chạy lại bộ tests tích hợp và unit tests.
  2. Chạy thử tiến trình recon segment B để xác thực kết quả ghi nhận status `ok` cho `segment_b_window`.
