# Yêu cầu loại bỏ handlePrune và Tái cấu trúc Routing Recon Check

## 1. Bối cảnh
Logic `handlePrune` hiện tại trong `recon_check_handler.go` không cần thiết hoặc không còn phù hợp với luồng nghiệp vụ hiện tại. Đồng thời, cấu trúc routing phân mảnh giữa Segment A và Segment B hiện tại quá phức tạp và khó bảo trì.

## 2. Chi tiết yêu cầu
- Loại bỏ nhánh kiểm tra `payload.TypeRecon == TypeReconPrune` và phương thức `handlePrune` khỏi `CheckHandler`.
- Tái cấu trúc logic routing trong `HandleReconCheck`:
  - Gom luồng kiểm tra toàn bộ bảng (`*` hoặc rỗng) cho Segment A và B về cùng một khối.
  - Gom luồng kiểm tra bảng cụ thể về hai hàm riêng biệt: `executeCheckSegmentA` và `executeCheckSegmentB`.
  - Hỗ trợ đầy đủ các loại `TypeRecon` một cách nhất quán cho cả Segment A và B (không quăng `full_diff` ra ngoài luồng check chuẩn).
