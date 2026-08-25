# Checklist SRE Tasks: Tích hợp SFTP Source Connector

Tài liệu quản lý chi tiết các task SRE phục vụ việc triển khai tính năng tích hợp SFTP Source Connector qua plugin `kafka-connect-fs`.

---

## 1. Danh sách Task SRE-1454 -> SRE-1459 (Chi tiết và Tiến độ Thực tế)

### SRE-1454: Evaluate and plan for implementation
* **Trạng thái:** Hoàn thành (Done)
* **Mô tả (Description):** Khảo sát hạ tầng, phân tích yêu cầu nghiệp vụ tích hợp nguồn dữ liệu SFTP/File CSV vào CDC Pipeline. Đưa ra quyết định kiến trúc quan trọng (ADR-005): Thay vì tự viết và duy trì một tiến trình worker riêng (Internal Polling Worker), chuyển sang sử dụng plugin open-source `kafka-connect-fs` (`FsSourceConnector`) chạy trực tiếp trên cụm Kafka Connect sẵn có. Phương án này tối ưu hóa việc phân phối qua Kafka và đồng bộ với kiến trúc Debezium hiện tại.
* **Todo List:**
  * [x] Tạo tài liệu kiến trúc tổng quan (Context, Requirements, ADRs).
  * [x] Thiết kế mô hình dữ liệu, sơ đồ callchain từ: SFTP Server $\rightarrow$ Kafka Connect $\rightarrow$ Kafka Broker $\rightarrow$ CDC Worker $\rightarrow$ Shadow DB.
  * [x] Đánh giá rủi ro về bảo mật credentials khi lưu SFTP URI (chứa password) và phương án xử lý (sanitize mật khẩu).
  * [x] Trình duyệt kế hoạch và nhận approve từ User.

---

### SRE-1455: CMS-FE
* **Trạng thái:** Hoàn thành (Done)
* **Mô tả (Description):** Cập nhật giao diện quản trị (CMS Web React) để hỗ trợ cấu hình nguồn SFTP. Thay đổi luồng nhập từ các trường host/port/credentials riêng lẻ thành một trường nhập duy nhất `fs.uris` (SFTP URI kết hợp credentials) để tương thích với cấu hình của plugin `kafka-connect-fs`.
* **Todo List:**
  * [x] Sửa đổi UI Form trong `SourceConnectors.tsx` để hỗ trợ class connector `com.github.mmolimar.kafka.connect.fs.FsSourceConnector`.
  * [x] Ẩn các trường nhập liệu cũ không liên quan (host, port, username, password, database) khi chọn loại Engine là SFTP, thay bằng ô nhập SFTP URI (`fs.uris`).
  * [x] Chuyển đổi cấu hình tần suất quét file từ số (`InputNumber`) sang chuỗi (`String(values.sleepMs)`) để tránh lỗi type mismatch trên Kafka Connect REST API.
  * [x] Tích hợp logic parse Connection Seed và ánh xạ các config keys đặc thù của plugin (`policy.sleepy.sleep`, `file_reader.delimited.settings.header`, `policy.recursive`).
  * [x] Chạy TypeScript check (`npx tsc --noEmit`) đảm bảo giao diện không lỗi biên dịch.

---

### SRE-1456: CMS-API
* **Trạng thái:** Hoàn thành (Done)
* **Mô tả (Description):** Nâng cấp Backend API của CMS (`cdc-cms-service`) để lưu trữ cấu hình SFTP Connector an toàn, hỗ trợ parse thông tin từ SFTP URI và giải quyết các vấn đề bảo mật lộ thông tin nhạy cảm.
* **Todo List:**
  * [x] Sửa đổi `system_connectors_handler.go` để nhận diện class `FsSourceConnector`.
  * [x] Viết helper trích xuất credentials (user, pass, host, port) từ URI `fs.uris` để phân giải thành dữ liệu kết nối phục vụ việc tạo Connection Registry.
  * [x] Bổ sung hàm `sanitizeSFTPURI()` tại `kafka_connect.go` để che giấu (mask) mật khẩu trong URI khi lưu log hoặc hiển thị cấu hình ra ngoài giao diện (`FilterSafeConfig`), ngăn chặn rò rỉ credential vào DB/Logs.
  * [x] Tích hợp luồng bắn (Dispatch) các NATS commands thông qua NATS Command Bus từ CMS sang Worker (các lệnh `cdc.cmd.provision-shadow-table`, `cdc.cmd.scan-fields`, `cdc.cmd.create-default-columns`, `cdc.cmd.alter-column`).
  * [x] Đăng ký nhận (Subscribe) và xử lý kết quả phản hồi từ Worker qua NATS (`cdc.result.scan-fields`, `cdc.result.alter-column`...) để cập nhật trạng thái database registry / shadow binding tương ứng.
  * [x] Đồng bộ Job ID và Trace ID (correlation header) thông qua NATS Message Header phục vụ log tracing toàn trình.
  * [x] Biên dịch thử nghiệm (`go build`) đảm bảo Backend CMS Service hoạt động ổn định.

---

### SRE-1457: CDC-WORKER
* **Trạng thái:** Hoàn thành (Done)
* **Mô tả (Description):** Cập nhật Worker (`centralized-data-service`) để tự động nhận diện Kafka topic của SFTP, parse thông tin database/table để định tuyến (routing) ghi dữ liệu mẫu, và tự động quét Header/Data của file CSV mẫu trên SFTP để sinh Mapping Rules.
* **Todo List:**
  * [x] Thêm cấu hình prefix topic `- cdc.sftplocal` vào `config-local.yml`.
  * [x] Cập nhật logic `isSFTP` trong `event_handler.go` để nhận diện topic `cdc.sftplocal.*` và tự động phân giải tên bảng (ví dụ: `reconcile_final`).
  * [x] Cập nhật isSFTPTopic trong `topic_helper.go` để lọc chính xác topic của SFTP.
  * [x] Thêm migration SQL cập nhật check constraint `connection_registry_engine_type_check` để cho phép lưu connection loại `'sftp'`.
  * [x] Viết hàm `scanFieldsFileSource` trong `discover_handler_sftp.go` để đọc trực tiếp Header/Data từ file CSV mẫu trên thư mục SFTP nhằm sinh Mapping Rules v2 (giải quyết vòng lặp "bảng shadow trống không scan được field").
  * [x] Disable cơ chế rewind offset trực tiếp (`RewindTopicOffset`) trong `snapshot_runner_handler.go` để chuyển sang luồng điều khiển Connector qua API Connect.
  * [x] Tích hợp luồng chạy Snapshot custom riêng cho nguồn SFTP:
    * Bypass hoàn toàn tiến trình truy vấn cơ sở dữ liệu truyền thống.
    * Tự động khởi tạo Kafka topic nếu chưa có.
    * Gọi API của Kafka Connect để **Resume** connector (`PUT /resume`) và **Restart** connector task (`POST /restart`) để kích hoạt việc quét/ingest file.
    * Ghi nhận trạng thái progress trong bảng `cdc_system.snapshot_progress` là `'done'` với `rows_processed = 0`.
  * [x] Sửa lỗi parse GJSON path (`lastSeg` fallback) khiến worker nhận diện sai kiểu dữ liệu của key.
  * [x] Biên dịch thử nghiệm Worker (`go build`) thành công.

---

### SRE-1458: Selftest in env dev
* **Trạng thái:** Hoàn thành (Done)
* **Mô tả (Description):** Thực hiện chạy thử nghiệm E2E môi trường Local Docker để verify toàn bộ luồng truyền nhận dữ liệu SFTP.
* **Todo List:**
  * [x] Khởi chạy container `sftp-host` đóng vai trò SFTP Server local giả lập, mount sẵn các thư mục CSV mẫu.
  * [x] Đóng gói và cài đặt plugin `kafka-connect-fs` JAR v1.3.0 vào container `gpay-kafka-connect` local.
  * [x] Test luồng Ingestion thực tế: Tạo SFTP Connector từ CMS UI $\rightarrow$ Kafka Connect quét file CSV và đẩy event vào Kafka $\rightarrow$ CDC Worker tiêu thụ event $\rightarrow$ Ghi dữ liệu mẫu thành công vào Postgres Shadow Table `shadow_testsftp20.reconcile`.
  * [x] Verify cấu trúc bảng Shadow sau khi ingest và đảm bảo chỉ có đúng 5 cột nghiệp vụ (`id`, `amount`, `created_at`, `status`, `trans_id`), loại bỏ hoàn toàn các cột rác bị gán nhầm trước đó.

---

### SRE-1459: Deploy & selftest in env testing
* **Trạng thái:** Đang chờ thực hiện (To Do)
* **Mô tả (Description):** Deploy mã nguồn mới lên môi trường Testing (Staging), cấu hình kết nối tới SFTP Server thực tế và thực hiện kiểm thử E2E chặng cuối.
* **Todo List:**
  * [ ] DevOps thực hiện cài đặt plugin `kafka-connect-fs` JAR v1.3.0 lên cụm Kafka Connect của môi trường Testing.
  * [ ] Deploy code CMS Backend, CMS Web Frontend và CDC Worker bản mới nhất lên Testing.
  * [ ] Cấu hình thủ công prefix `cdc.sftplocal` vào file cấu hình môi trường testing (`config.yml`).
  * [ ] Chạy thử nghiệm kết nối SFTP thực tế trên Testing: Kiểm tra xem các cấu hình đường dẫn có bị lỗi phân quyền chroot SSH không. *Lưu ý: Không tự prepend `/home/<user>` vào đường dẫn SFTP, tôn trọng 100% path do Admin nhập.*
  * [ ] Đăng ký Table Registry, scan field sinh Mapping Rules v2, tạo bảng Shadow và verify luồng đối soát (Reconciliation cycle) chặng Source (SFTP) $\leftrightarrow$ Shadow PG $\leftrightarrow$ Master PG hoạt động chính xác.

---

## 2. Các Task Đầu mục Dịch vụ/Operation Lớn còn thiếu

Các task dưới đây phục vụ cho quá trình vận hành, quản trị và kiểm thử chặng cuối, hiện tại chưa được gán mã SRE:

### A. DevOps & Hạ tầng (Infrastructure)
* [ ] **DevOps - Cài đặt Plugin Kafka Connect FS trên Production**
  * **Mô tả:** Đóng gói và cài đặt plugin `kafka-connect-fs` (slim JAR v1.3.0) lên cụm Kafka Connect của môi trường Production và restart để kích hoạt connector.
* [ ] **Cấu hình Firewall và Mạng cho SFTP kết nối đối tác**
  * **Mô tả:** Mở tường lửa (firewall rules) đảm bảo cụm Kafka Connect ở các môi trường kết nối thông suốt tới IP/Port máy chủ SFTP của đối tác/nguồn.
* [ ] **Cấp quyền tài khoản và cấu hình thư mục SFTP của đối tác**
  * **Mô tả:** Đảm bảo tài khoản SFTP có quyền đọc (Read) và quyền ghi/di chuyển file (Write/Rename) trên thư mục, tạo sẵn các folder processed/error trên SFTP.

### B. Quản trị & Cấu hình (Configuration)
* [ ] **Cấu hình whitelist topic prefix trên môi trường Staging/Prod**
  * **Mô tả:** Cập nhật file cấu hình môi trường Production để thêm whitelist `cdc.sftplocal` giúp worker tự động discover topic.
* [ ] **Đăng ký Table Registry và Mapping Rules trên DB Staging/Prod**
  * **Mô tả:** Khai báo metadata registry và duyệt mapping rules trên môi trường thật để hệ thống tự động provision bảng shadow trên database Production.

### C. Kiểm thử Nghiệp vụ & Đối soát (Reconciliation Validation)
* [ ] **Kiểm định và xác thực Đối soát (Reconciliation)**
  * **Mô tả:** Cấu hình Master Binding trên môi trường thật, chạy thử nghiệm chu kỳ đối soát (Reconciliation cycle) và verify đối soát Segment B (Shadow PG $\leftrightarrow$ Master PG) khớp mã băm xxhash.
