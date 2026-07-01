# Context: Investigate Master Sync New Field

## Bối cảnh
Người dùng đặt câu hỏi về cơ chế đồng bộ hóa dữ liệu sang master khi thêm trường mới:
* Hiện tại khi thêm một field mới vào shadow, nếu dữ liệu đã tồn tại trong `_raw_data`, khi chạy transform hệ thống sẽ cập nhật dữ liệu từ `_raw_data` về các field mới của shadow.
* Câu hỏi đặt ra: Ở phía data master, dữ liệu vẫn "im lìm" (không tự cập nhật cho field mới đó) đúng không?
* Yêu cầu kiểm tra: Đã có chức năng đồng bộ dữ liệu cho các field mới được thêm vào master chưa (ví dụ: một nút click đồng bộ thêm dữ liệu cho các trường này trên master)?

## Mục tiêu
1. Xác định hành vi hiện tại của master khi shadow được transform cập nhật field mới.
2. Tìm kiếm trong codebase xem có chức năng đồng bộ/backfill dữ liệu từ shadow sang master (hoặc ngược lại) khi cấu hình master thay đổi (thêm field mới) hay chưa.
3. Giải thích rõ cơ chế và trả lời chi tiết cho người dùng.
