# 🚀 Master Implementation Plan V2: Core CDC SFTP Ingestion Engine

## 1. Tổng Quan Kế Hoạch V2 (Context & Scope Adjustment)

Theo phản hồi từ KTS / Admin, hệ thống **CDC SFTP Engine** là hạ tầng CDC tổng quát (General Data Synchronization Protocol) dùng để ingest dữ liệu từ bất kỳ file CSV/SFTP nào vào Bảng Shadow (`shadow_table`), `reconcile` chỉ là 1 ví dụ cụ thể.

Do đó, Kế hoạch V2 điều chỉnh chuẩn hóa ranh giới trách nhiệm (Separation of Concerns) theo 5 điểm cốt lõi:

1. **Host/URI (Tripwire 2):** Khởi tạo linh hoạt từ CMS UI. Connector nhận passthrough 100% SFTP URI do User/Admin nhập, không hardcode.
2. **Near Realtime Ingestion (Tripwire 3):** Đặt `policy.sleepy.sleep = 3000` (3 giây) hoặc `5000` (5 giây), duy trì `policy.recursive = false` (quét thư mục phẳng). Kết hợp với offset tracking byte-level của plugin `kafka-connect-fs` giúp quét cực nhanh (~10ms SSH packet) mà KHÔNG gây DDoS SFTP Server.
3. **Cơ Chế State & Byte Offset Tracking (Tripwire 4):**
   - Plugin `kafka-connect-fs` (`mmolimar`) **tự động lưu vết Byte Offset & Last Modified Timestamp** của từng tệp tin vào Kafka Connect Offset Topic (`connect-offsets`).
   - CDC Worker / Kafka Connect **CHỈ ĐỌC VÀ SYNC (Read-Only)**, tuyệt đối không can thiệp hay bắt SFTP Server thực hiện nghiệp vụ rename/nhãn file.
   - Khi tệp tin mới xuất hiện hoặc tệp tin cũ được append thêm dữ liệu (tăng file size), Connector tự động đọc **Incremental (đọc tiếp từ byte offset cũ)** đến byte cuối cùng.
4. **Trích Xuất Primary Key Động (Tripwire 6):**
   - Loại bỏ việc gán cứng `transaction_id`.
   - Tận dụng Kiến trúc CDC Engine: Kafka Connect đẩy raw JSON record sang CDC Worker (`centralized-data-service`).
   - CDC Worker (`sftp_adapter.go`) tự động tra cứu **Table Registry & Mapping Rules V2** trong database control-plane để trích xuất Primary Key động (`id`, `_id`, hoặc composite key) và ghi vào Shadow Table.
5. **Ranh Giới Quản Trị Hạ Tầng SFTP:**
   - Việc Dọn rác / Archive file trên máy chủ SFTP thuộc trách nhiệm riêng của Hệ thống Nguồn / Ops SFTP Server. CDC Engine hoàn toàn độc lập và không can thiệp.

---

## 2. Bản Cấu Hình Core CDC SFTP Connector Specification V2

```json
{
  "name": "cdc-sftp-source-v2",
  "config": {
    "connector.class": "com.github.mmolimar.kafka.connect.fs.FsSourceConnector",
    "tasks.max": "1",

    "fs.uris": "${file:/etc/kafka-connect/secrets/sftp-credentials.properties:SFTP_URI}",

    "policy.class": "com.github.mmolimar.kafka.connect.fs.policy.SleepyPolicy",
    "policy.sleepy.sleep": "3000",
    "policy.recursive": "false",
    "policy.regexp": "^.*\\.csv$",

    "file_reader.class": "com.github.mmolimar.kafka.connect.fs.file.reader.CsvFileReader",
    "file_reader.delimited.header": "true",

    "topic": "cdc.sftplocal.${file:/etc/kafka-connect/secrets/sftp-credentials.properties:CONNECTOR_NAME}.${file:/etc/kafka-connect/secrets/sftp-credentials.properties:TABLE_NAME}",

    "key.converter": "org.apache.kafka.connect.storage.StringConverter",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "false",

    "errors.tolerance": "all",
    "errors.deadletterqueue.topic.name": "dlq.cdc.sftplocal.errors",
    "errors.deadletterqueue.topic.replication.factor": "1",
    "errors.deadletterqueue.context.headers.enable": "true",
    "errors.log.enable": "true",
    "errors.log.include.messages": "true"
  }
}
```

---

## 3. Lộ Trình Triển Khai Kiến Trúc V2 (5-Phase Roadmap V2)

### Phase 0: Re-align Architecture & Context (Done 🟢)
- [x] Làm rõ ranh giới trách nhiệm: CDC Worker chỉ đọc & sync, không can thiệp SFTP server.
- [x] Xác nhận cơ chế Byte-Offset Incremental Read của `kafka-connect-fs`.
- [x] Chuyển đổi trích xuất Primary Key sang tầng CDC Worker (Dynamic Table Registry Mapping).

### Phase 1: Near-Realtime Poll & Secret Provisioning
- [ ] Cấu hình `policy.sleepy.sleep = 3000` (3s Near Realtime).
- [ ] Khai báo SFTP URI động qua CMS UI/API passthrough.

### Phase 2: Kafka Stream & CDC Worker Dynamic Mapping
- [ ] CDC Worker lắng nghe Topic Pattern `cdc.sftplocal.*`.
- [ ] `sftp_adapter.go` đọc record, resolve Dynamic Primary Key từ Mapping Rules V2.
- [ ] Thực hiện Upsert vào bảng Shadow tương ứng (`shadow_{collection}`).

### Phase 3: Failure Isolation & DLQ Routing
- [ ] Định tuyến dòng CSV hỏng format vào Dead Letter Queue Topic `dlq.cdc.sftplocal.errors`.
- [ ] Đảm bảo Connector giữ trạng thái RUNNING 24/7.

### Phase 4: Integration Verification & Load Test
- [ ] Test Near-Realtime: Đẩy 1 dòng mới vào file CSV -> CDC Worker sync sang Shadow Table trong < 3 giây.
- [ ] Test Incremental Append: Ghi thêm 500 dòng vào file CSV hiện có -> Connector đọc tiếp từ byte offset cũ, không duplicate dòng cũ.
