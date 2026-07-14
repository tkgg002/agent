# Context: Phân Tích Kiến Trúc Luồng Đối Soát & Chữa Lành

## Phạm vi
- Phân tích toàn bộ luồng Đối soát (Reconciliation) và Chữa lành (Heal) trong hệ thống CDC
- Bao gồm 3 thành phần: CMS Frontend, API Gateway (cdc-cms-service), CDC Worker (centralized-data-service)
- Bao gồm 3 giai đoạn: Kích hoạt đối soát → Truy vấn danh sách lỗi → Thực thi chữa lành

## Loại task
- Phân tích / Tài liệu hóa (Analysis/Documentation)
- KHÔNG thay đổi source code

## Ngày khởi tạo
- 2026-07-03

## Người yêu cầu
- User (trainguyen) yêu cầu phân tích sequence diagram luồng đối soát
