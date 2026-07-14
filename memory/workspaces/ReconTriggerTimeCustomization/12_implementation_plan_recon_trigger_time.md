# Kế Hoạch Triển Khai Thực Tế - AI Implementation Plan

Kế hoạch và quá trình thực hiện sửa đổi frontend cho tác vụ Tùy chỉnh Thời gian Đối soát.

## 1. Các bước chuẩn bị (Pre-flight)
- [x] Đọc `GEMINI.md` để hiểu Core Rules và vai trò Muscle.
- [x] Đọc `lessons.md` để tránh các lỗi đã xảy ra trong quá khứ.
- [x] Tạo file backup `ConfirmDestructiveModal.tsx.bak-before-recon-trigger-time-20260713` làm restore-point, tuân thủ kỷ luật VCS và không tự ý commit khi chưa được lệnh.

## 2. Thực thi mã nguồn
- [x] Chỉnh sửa `ConfirmDestructiveModal.tsx`:
  - [x] Thêm helper `getRoundedEndTime`.
  - [x] Cập nhật logic trong `handleCheckModeChange`.
  - [x] Cập nhật logic khởi tạo trong `useEffect` khi open modal.
  - [x] Cập nhật logic tính toán `startMs`, `endMs` trước khi gọi `onConfirm` trong `handleOk`.

## 3. Xác minh chất lượng (Verification)
- [x] Chạy kiểm tra tĩnh TypeScript bằng `npx tsc --noEmit` ở thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`.
- [x] Kiểm tra xem có lỗi cú pháp hoặc kiểu dữ liệu nào không.

## 4. Hoàn tất tài liệu và Governance
- [x] Cập nhật audit log vào `05_progress_recon_trigger_time.md`.
- [x] Cập nhật trạng thái task trong `08_tasks_recon_trigger_time.md`.
- [x] Viết báo cáo thay đổi trong `11_report_recon_trigger_time.md`.
- [ ] Chạy Process Linter qua script `verify_governance.py` (sẽ thực hiện ở pre-flight check cuối turn).
