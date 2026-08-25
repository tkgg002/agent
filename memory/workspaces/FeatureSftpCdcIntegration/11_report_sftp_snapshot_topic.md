# Báo cáo thay đổi: Di chuyển tạo Topic Kafka SFTP sang nút Snapshot

Tài liệu báo cáo chi tiết các file đã sửa đổi, số lượng dòng code và mô tả thay đổi.

---

## 1. Danh sách tệp sửa đổi và số dòng thay đổi

| File | Số dòng thay đổi | Mô tả chi tiết thay đổi |
| :--- | :--- | :--- |
| [`update_shadow_binding.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/shadow/update_shadow_binding.go) | ~78 dòng (revert) | Revert hoàn toàn về trạng thái nguyên bản. Xoá bỏ logic khởi tạo connector SFTP trì hoãn tại đây. |
| [`debezium_connector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go) | ~30 dòng | Loại bỏ logic rẽ nhánh skip connector `Create` đối với SFTP. Trả về luồng tạo connector lập tức cho SFTP khi kết nối được tạo. |
| [`source_object_actions_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/source_object_actions_handler.go) | ~100 dòng | Bổ sung dependency `db *gorm.DB`, thêm hàm helper `autoCreateKafkaTopic`, và rẽ nhánh tại `SnapshotV2` khi `engine == "sftp"`. |
| [`server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go) | ~10 dòng | Cập nhật đăng ký dependency cho `NewSourceObjectActionsHandler` và `NewUpdateShadowBindingHandler`. |
| [`source_object_actions_handler_test.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/api/source_object_actions_handler_test.go) | ~15 dòng | Cập nhật constructor mock và assertion test tương thích với thay đổi. |

---

## 2. Mô tả logic chi tiết đã triển khai

1. **Khi tạo SFTP Connection:** `DebeziumConnectorHandler` gọi Kafka Connect `Create` để tạo SFTP Connector ngay lập tức. Nhưng không gọi tạo Topic Kafka.
2. **Khi bật Active trên UI:** Chỉ thay đổi trạng thái `is_active` trong CSDL bình thường (cho cả table registry và shadow binding).
3. **Khi nhấn Snapshot:** API `POST /v1/source-objects/:id/snapshot-v2` kiểm tra nếu nguồn là `sftp`:
   - Lấy tên topic và bootstrap từ raw config của connector trong bảng `cdc_system.sources`.
   - Gọi `autoCreateKafkaTopic` để khởi tạo Topic trên Kafka.
   - Khi topic xuất hiện, SFTP Connector đang chạy sẽ tự động bắt đầu quét file và đồng bộ.
   - Trả về response trực tiếp cho client, ghi nhận activity log mà không bắn command `snapshot.v2` sang NATS.
