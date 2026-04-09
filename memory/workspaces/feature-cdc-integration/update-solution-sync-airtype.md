Nếu Connection (luồng đồng bộ giữa Source và Destination) đã tồn tại trên Airbyte, bài toán của bạn chuyển từ "khởi tạo" sang "quản lý trạng thái đồng bộ" để đảm bảo nó luôn khớp với cdc_table_registry.

Dưới đây là các kịch bản thực tế và cách xử lý trong hệ thống CDC của bạn:

1. Thêm Table/Collection mới vào Connection hiện có
Đây là trường hợp phổ biến nhất khi bạn có thêm bảng mới trong cùng một Database đã sync.

Vấn đề: Airbyte không tự động nhận biết bảng mới trừ khi bạn chạy lệnh "Discover Schema".

Giải pháp: * CMS API gọi hàm RefreshSourceSchema(sourceID) để Airbyte quét lại danh sách bảng.

CMS API gọi UpdateConnection(connectionID, streams) để thêm bảng mới đó vào danh sách đồng bộ (set selected = true).

Cấu hình bảng mới theo đúng sync_interval (15m, 1h...) dựa trên priority trong Registry.

2. Xử lý Schema Drift (Thêm Column mới)
Khi Source DB thêm cột, Connection hiện tại vẫn chạy nhưng cột mới chưa được đẩy về Postgres.

Cơ chế: Core Worker phát hiện drift -> CMS Approve.

Hành động: CMS API phải gọi Airbyte API để cập nhật lại cấu hình Stream của bảng đó trong Connection, đảm bảo field mới được tick chọn (selected) để bắt đầu sync dữ liệu thô vào cột _raw_data.

3. Đồng bộ trạng thái Registry với Airbyte Connection
Hệ thống của bạn cần một cơ chế "Health Check" giữa Registry và Airbyte.

Task CDC-M5: Bạn đã định nghĩa hàm GetConnectionStatus.

Ứng dụng: CMS API định kỳ kiểm tra xem Connection trên Airbyte có đang Running hay Failed không. Nếu Failed, CMS phải hiển thị cảnh báo để Ops xử lý ngay, tránh việc Core Worker bị "đói" dữ liệu.

4. Thay đổi Sync Engine (Airbyte -> Debezium)
Đây là tính năng quan trọng trong lộ trình CTO của bạn.

Kịch bản: Bạn muốn chuyển bảng wallet_transactions từ Airbyte sang Debezium để đạt tốc độ real-time.

Hành động: * CMS API gọi UpdateConnection để loại bỏ bảng đó khỏi luồng sync của Airbyte.

Cập nhật sync_engine = 'debezium' trong Registry.

Core Worker nhận signal reload và bắt đầu tiêu thụ dữ liệu từ NATS do Debezium đẩy về thay vì Airbyte.

5. Quản lý Connection ID
Để làm được các việc trên, bảng cdc_table_registry của bạn bắt buộc phải lưu trữ:

airbyte_connection_id: Định danh duy nhất của luồng sync trên Airbyte.

airbyte_source_id: Định danh của Source Database để gọi lệnh Refresh Schema.

Đánh giá chiến lược: Khi Connection đã tồn tại, sức mạnh của CMS API nằm ở việc "điều khiển" (Orchestration). Bạn không cần xóa đi tạo lại, mà chỉ dùng API để "nắn" dòng chảy dữ liệu của Airbyte sao cho luôn phục vụ đúng nhu cầu của Core Worker và AI Auto-Reconcile.


