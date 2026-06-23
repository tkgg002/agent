# Kế hoạch Refactor dọn dẹp các file lớn và không rõ ràng trong centralized-data-service

Kế hoạch này tập trung vào việc xác định các file code quá dài, phức tạp, thiếu phân tách rõ ràng về mặt trách nhiệm trong `centralized-data-service`, từ đó tiến hành refactor để cải thiện chất lượng code mà không phá vỡ tính đúng đắn của hệ thống.

## Giai đoạn 1: Khảo sát và Phân tích (Research & Audit)
1. Liệt kê toàn bộ các file Go trong `centralized-data-service` cùng với số dòng code của chúng.
2. Xác định các file "God objects" hoặc các file chứa quá nhiều logic nghiệp vụ trộn lẫn (ví dụ: handler chứa logic của service, repository chứa logic nghiệp vụ, v.v.).
3. Lập danh sách các file cần refactor (Priority List).

## Giai đoạn 2: Lập Kế hoạch Chi tiết và Thiết kế (Design & Planning)
1. Với mỗi file cần refactor, phân tích các struct, interface, function/method chứa trong đó.
2. Đề xuất phương án chia nhỏ:
   - Di chuyển các struct model sang file model riêng.
   - Di chuyển các hàm xử lý phụ trợ (helpers/utils) sang file utils riêng.
   - Phân tách handlers và services nếu chúng đang bị gộp chung.
   - Phân rã các function quá dài thành các function nhỏ hơn, tập trung hơn.
3. Tạo Implementation Plan chi tiết để trình bày với User.

## Giai đoạn 3: Thực hiện Refactor (Execution)
1. Thực hiện refactor từng file theo thứ tự ưu tiên bằng cách chia nhỏ code sang các file mới có tên gọi tường minh, đúng chuẩn cấu trúc dự án.
2. Duy trì tính tương thích ngược về mặt API và contract của các struct/interface để không ảnh hưởng đến các phần khác của hệ thống.

## Giai đoạn 4: Xác nhận và Kiểm thử (Verification & Testing)
1. Biên dịch dự án sau mỗi bước refactor: `go build ./...`
2. Chạy toàn bộ unit tests: `go test ./...`
3. Đảm bảo chất lượng code và tuân thủ các quy tắc bảo mật.
