# 11 Báo Cáo Thay Đổi (Change Report)

## Danh Sách Tệp Tin Đã Thay Đổi
1. `internal/service/source/metadata_registry_utils.go`:
   - Dòng thêm: ~55 dòng code.
   - Nội dung: Bổ sung `customGenericPrefixes []string`, `customPrefixesMu sync.RWMutex`, hàm `RegisterGenericPrefixes(prefixes []string)`, hàm `stripGenericPrefixes(conn string) []string`, và nâng cấp `IsGenericConnectionCode` để kiểm tra cả base engines và các dynamic generic prefixes từ config.
2. `internal/server/server_setup.go`:
   - Dòng thêm: 2 dòng code (tại dòng 97).
   - Nội dung: Nạp danh sách tiền tố `servicesource.RegisterGenericPrefixes(cfg.Kafka.TopicPrefix)` ngay tại bootstrap pipeline.
3. `internal/handler/shadow/kafka_consumer.go`:
   - Dòng sửa: 15 dòng code (tại dòng 671-683).
   - Nội dung: Thay thế việc hardcode cắt `cdc.goopay.` bằng vòng lặp duyệt qua `kc.config.TopicPrefix` để cắt tiền tố động; tự động reset `sourceConnCode = ""` nếu là generic prefix.
4. `internal/handler/shadow/event_handler.go`:
   - Dòng sửa: 1 dòng code (tại dòng 288).
   - Nội dung: Thay `return 0, nil` thành `return 0, fmt.Errorf("source route not found in registry cache: db=%s, table=%s, conn=%s", sourceDB, sourceTable, sourceConn)` để không nuốt lỗi, bảo vệ dữ liệu và kích hoạt ghi vào Dead Letter Queue (DLQ).
5. `internal/service/source/metadata_registry_service.go`:
   - Dòng sửa: 12 dòng code (tại dòng 578-592).
   - Nội dung: Sử dụng `stripGenericPrefixes(conn)` trong `ResolveSourceRoutes` để đối soát danh sách ứng viên connection code thay vì hardcode `cdc.goopay.`.
6. `internal/service/source/metadata_registry_service_test.go`:
   - Dòng thêm: 20 dòng code (tại dòng 528-545).
   - Nội dung: Bổ sung unit test `TestRegisterGenericPrefixes` kiểm chứng việc đăng ký động các tiền tố `cdc.goopaylocal`, `cdc.mariadblocal`, `cdc.sftplocal` và khẳng định các connection cụ thể như `traitestctphs`, `traitestmongodevct` không bị nhầm là generic.
