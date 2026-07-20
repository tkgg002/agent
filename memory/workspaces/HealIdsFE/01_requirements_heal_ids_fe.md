# Requirements: Hiển thị danh sách IDs đã heal trong tab Phiên đã xử lý

Thêm icon/nút xem chi tiết danh sách IDs đã heal (đã xử lý) tại tab "Phiên đã xử lý" trên giao diện Chữa lành đối soát.

## Chi tiết yêu cầu:
1. Thêm một cột mới hoặc tích hợp vào tab "Phiên đã xử lý" hiển thị danh sách IDs đã heal tương tự như cột "ID lệch" ở tab "Phiên chưa xử lý".
2. Cột này sẽ giải nén/parse các IDs từ `missing_ids` và `stale_ids` của report tương ứng (sử dụng hàm helper `getDiffIDs`).
3. Hiển thị tối đa 2 ID đầu dưới dạng tag màu xanh lá cây (`green`) để phân biệt với tag màu đỏ của IDs lệch chưa xử lý.
4. Nếu số lượng IDs > 2, hiển thị một nút hình tròn chứa icon con mắt (`EyeOutlined`). Khi người dùng nhấn vào nút này, một Popover sẽ hiển thị danh sách đầy đủ tất cả IDs đã heal cùng với nút "Copy" để copy nhanh.
