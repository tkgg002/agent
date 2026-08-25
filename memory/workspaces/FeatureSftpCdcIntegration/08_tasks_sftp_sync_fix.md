# Danh sách công việc: SFTP Sync to Shadow Fix

- [ ] Sửa đổi logic parser trong `kafka_consumer.go`:
  - Phát hiện `IsSFTPTopic(msg.Topic)`.
  - Thực hiện validate schema trực tiếp bằng flat JSON unmarshal.
  - Bypass Debezium decoding và chuyển trực tiếp raw payload sang `eventHandler.HandleRaw`.
- [ ] Bổ sung engine type `"sftp"` vào `buildDSNFromFieldsPatched` của `metadata_registry_service.go` để giải quyết lỗi không parse được URI.
- [ ] Chạy unit test xác minh backend.
- [ ] Restart 2 dịch vụ local backend.
- [ ] Xác minh kết quả sync E2E.
