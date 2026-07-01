# Yêu cầu: Phân tích Luồng Đối soát Tier 2 XOR-Hash

## Mục tiêu
Xác minh luồng đối soát Tier 2 trong dịch vụ `cdc-system/centralized-data-service`.

## Yêu cầu chi tiết
1. **Phân tích Logic**: Nghiên cứu cách luồng Tier 2 thực hiện đối chiếu XOR-hash dựa trên cửa sổ thời gian (window-based) ở cả hai bên Source và Destination.
2. **Xác minh Chỉ Đọc (Read-only)**: Xác thực xem luồng này có strictly read-only và hoàn toàn không ghi hay sửa đổi dữ liệu ở cơ sở dữ liệu Source và Destination hay không.
3. **Truy vết Luồng thực thi**: Cung cấp tài liệu walkthrough chi tiết về đường đi của code đối với Tier 2.
4. **Phát hiện lỗi/vấn đề**: Ghi nhận bất kỳ bug tiềm ẩn, sự bất nhất, hoặc điểm sai lệch nào so với đặc tả chỉ đọc hoặc tính chính xác của thuật toán XOR-hash.
