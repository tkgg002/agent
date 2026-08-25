# Nhật ký tiến độ: SFTP Sync to Shadow Fix

- **2026-08-12 11:54:00 [Agent:Gemini-2.5-Pro]**: Khởi tạo workspace fix lỗi đồng bộ dữ liệu SFTP sang Shadow DB.
  - Phát hiện lỗi: `kafka_consumer.go` cố gắng giải mã Debezium envelope với message phẳng của SFTP dẫn đến `event["after"]` là `nil` và bị drop.
  - Phát hiện lỗi: Thiếu parser cho engine type `"sftp"` trong `metadata_registry_service.go`.
