# Kế hoạch triển khai: Thêm Số lượng lệch & Thời gian xử lý vào FE

## 1. Phương án sửa đổi
Sửa đổi trực tiếp file `cdc-cms-web/src/components/ReconPipelineGrid.tsx` trong component `DrillDown`.

### Cột Số lượng lệch:
- Thêm cột `Lệch` vào mảng `columns` của bảng lịch sử đối soát.
- Thuộc tính `render` của cột sẽ nhận bản ghi `r: ReconReport`.
- Dùng lại hàm `fmtDrift(r.diff)` đã được khai báo ở đầu file `ReconPipelineGrid.tsx` để tận dụng styling và logic có sẵn (màu xanh lá khi khớp, đỏ khi thiếu, vàng khi thừa).

### Cột Thời gian xử lý:
- Thêm cột `Thời gian xử lý` vào mảng `columns`.
- Định nghĩa helper `fmtDuration(ms: number | null | undefined)` trong block component hoặc ở file scope:
  ```typescript
  const fmtDuration = (ms: number | null | undefined) => {
    if (ms == null) return '—';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };
  ```
- Thuộc tính `render` của cột sẽ hiển thị kết quả từ `fmtDuration(r.duration_ms)`.

### Tối ưu hóa layout:
- Điều chỉnh width của cột `Chi tiết` và các cột khác để tổng thể giao diện Drawer (width 860px) hiển thị hài hòa, không bị tràn hay cuộn ngang quá mức.

## 2. Kế hoạch kiểm thử & Verify
- Kiểm tra tính đúng đắn của code typescript thông qua static checking hoặc chạy build:
  ```bash
  npm run build
  ```
  trong thư mục `cdc-cms-web`.
- Chạy governance linter:
  ```bash
  python3 agent/tooling/verify_governance.py
  ```
