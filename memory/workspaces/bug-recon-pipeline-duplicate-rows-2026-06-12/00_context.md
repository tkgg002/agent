# Context: Lỗi trùng lặp (duplicate) dòng pipeline trong bảng ReconPipelineGrid

## Mô tả lỗi
- Người dùng phát hiện các dòng trong bảng pipelines bị trùng lặp (duplicate).
- Có thể do logic ghép `aRows` và `bRows` trong hàm `buildPipelines` của `ReconPipelineGrid.tsx` bị trùng, hoặc logic nhóm/hiển thị `flatData` khiến các dòng bị nhân đôi trong quá trình expand/collapse hoặc do key của row bị trùng.

## Phân tích Governance & Root Cause
- **Governance Audit**: Khởi tạo workspace `bug-recon-pipeline-duplicate-rows-2026-06-12` trước khi chỉnh sửa code.
- **Root Cause Analysis**: Sẽ thực hiện kiểm tra dữ liệu đầu vào và logic tạo `flatData`/`buildPipelines` để xác định nguyên nhân chính xác.
