# Phạm vi & Thành phần (Scope & Context) - Luồng Chữa Lành Tương Tác

## 1. Thành phần liên quan
* **Frontend (cdc-cms-web)**: Thay đổi giao diện popup chữa lành để hiển thị chi tiết các ca chênh lệch và cung cấp checkboxes chọn lựa hành động.
* **API Gateway (cdc-cms-service)**: Cung cấp API truyền tải thông tin report và dispatch lệnh heal với các cờ lựa chọn granular sang NATS.
* **CDC Worker (centralized-data-service)**: Đọc thông tin từ report đã lưu trong DB và thực thi heal/prune cụ thể theo các cờ đã chọn, ngắt luồng tự động đối soát khi heal.

## 2. Ranh giới hệ thống (Boundaries)
* Chỉ tác động đến chặng Ingest (Source ➔ Shadow) và Transmute (Shadow ➔ Master) của đối soát dữ liệu (Segment A & B).
* Không làm thay đổi cấu trúc bảng báo cáo hay cơ sở dữ liệu hiện có.
