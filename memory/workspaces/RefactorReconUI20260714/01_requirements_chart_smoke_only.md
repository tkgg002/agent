# Yêu cầu Chi tiết: Chỉ hiển thị Smoke Check trên Biểu đồ Biến động

## 1. Bối cảnh
Biểu đồ "Biến động số lượng theo phiên recon" trong chi tiết pipeline đối soát (`ReconPipelineGrid.tsx`) hiện hiển thị tất cả các loại phiên đối soát (Smoke Check, Hash Window, Full Diff, Deep Check). Khách hàng yêu cầu biểu đồ này chỉ vẽ dữ liệu của các phiên **Smoke Check**.

## 2. Phạm vi điều chỉnh (Scope)
- **Component điều chỉnh:** `ReconPipelineGrid.tsx`
- **Logic điều chỉnh:**
  - Lọc dữ liệu đầu vào của biểu đồ (`chartData`) để chỉ giữ lại các phiên có `check_type === 'smoke'` hoặc `check_type === 'segment_b_smoke'`.
  - Cập nhật logic tính toán trục Y (`yDomain`) tương ứng chỉ dựa trên các phiên Smoke Check để đảm bảo tỷ lệ hiển thị chính xác.
