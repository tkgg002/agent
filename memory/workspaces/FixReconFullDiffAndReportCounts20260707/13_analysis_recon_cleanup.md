# Phân tích kỹ thuật - muscle_execute: Recon Cleanup Impact Analysis

Tài liệu phân tích tác động của các thay đổi dọn dẹp reconciliation report metadata lên các thành phần trong hệ thống Data Hub.

## 1. Tác động Database Schema (cdc_reconciliation_report)
- **Cột `tier`**: Bị loại bỏ hoàn toàn trong code struct. Ở tầng database, cột này có thể được giữ lại tạm thời (tương thích ngược) hoặc drop sau thông qua migration. Việc loại bỏ struct field đảm bảo code không còn ghi hoặc đọc trường này, tránh nhầm lẫn logic check level.
- **Cột `target_table`**: Được đổi tag sang `gorm:"-"` ở centralized-data-service. Tầng này sẽ không còn ghi đè cột này bằng shadow_table bare nữa, tránh xung đột tên. Thay vào đó, trường `shadow_table` và `master_table` sẽ làm khóa duy nhất để quản lý các pipeline.
- **Các cột metadata nguồn (`source_type`, `source_host`, `source_table`)**: Được bổ sung trực tiếp vào database model. Khi chạy Segment A (Source ↔ Shadow), hệ thống sẽ trích xuất host từ connection URL của source registry để điền vào `source_host`, và điền loại database vào `source_type` (ví dụ: `mongodb`, `mysql`). Khi chạy Segment B, `source_type` được gán mặc định là `"postgresql"` và `source_host` là `"shadow_plane"`. Điều này cải thiện khả năng quan sát (observability) và dòng chảy dữ liệu (lineage tracing) trực tiếp từ database report.

## 2. Tác động API Contract & Web UI
- API `/api/reconciliation/report` trả về danh sách report sẽ không còn trường `tier` mà thay vào đó là các trường `source_type`, `source_host`, `source_table`.
- Web UI sử dụng hook `useReconStatus.ts` đã được cập nhật interface để đón nhận cấu trúc mới này.
- Grid hiển thị pipeline `ReconPipelineGrid.tsx` sử dụng helper `getSourceDisplayName` để tự động gộp các trường này thành một nhãn thân thiện: `[source_type] source_host / source_db . source_table`. Nhãn này hiển thị rõ ràng thông tin nguồn hơn so với việc chỉ hiển thị `source_db.source_table` bare như trước, đặc biệt là khi có nhiều nguồn kết nối khác nhau.

## 3. Độ an toàn của các Test cases
- Tất cả các test cases kiểm tra flow trong `internal/service/recon` và `internal/handler/recon` đều đã compile và chạy thành công. Điều này chứng minh rằng việc loại bỏ Smoke Check khỏi flow đối soát chính (ReconciliationReport) không phá vỡ bất kỳ kịch bản kiểm thử tích hợp (integration tests) hay kiểm thử đơn vị (unit tests) nào của hệ thống.
- TypeScript compiler (`tsc --noEmit`) hoàn thành không lỗi đảm bảo sự khớp nối 100% về mặt type contract giữa API Client (frontend hooks) và các component hiển thị UI React.
