# 🚀 Detailed Code Implementation Plan: Core CDC SFTP Fixes

## 1. Phân Tích & Demo Code Từng Tripwire Cho Hệ Thống `data-hub`

### Tripwire 1: Hardcode Mật khẩu Plain-text
- **Đánh giá:** Đúng. Lộ secret nếu viết thẳng trên URI.
- **Giải pháp hệ thống:** Mật khẩu do Admin nhập từ UI (`values.fsUris`).
- **File cần sửa:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Vị trí:** Dòng 305
- **Code thay đổi:** Thêm placeholder gợi ý dùng FileConfigProvider `${file:...}` khi Admin nhập UI.

### Tripwire 2: Môi trường `host.docker.internal`
- **Đánh giá:** Không cần lo bẫy này vì Admin nhập FQDN/IP thực tế từ UI CMS khi khởi tạo Connector.
- **File cần sửa:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Vị trí:** Dòng 305
- **Code thay đổi:** Cập nhật Placeholder chuẩn Production `sftp://user:pass@sftp.yourdomain.com:22/path`.

### Tripwire 3: DDoS SFTP Server (`sleep=3000` & `recursive=true`)
- **Đánh giá:** Cần Near-Realtime 3s (`sleep=3000`). Bẫy thực sự là `policy.recursive=true` làm duyệt đệ quy cây thư mục sâu.
- **Giải pháp:** Tắt đệ quy `policy.recursive = 'false'`, giữ `sleep = 3000`. Quét thư mục phẳng chỉ tốn ~10ms SSH packet, vừa đạt Near-Realtime 3s vừa không gây DDoS.
- **File cần sửa:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Vị trí:** Dòng 310
- **Code thay đổi:** Đổi `'policy.recursive': 'true'` thành `'policy.recursive': 'false'`.

### Tripwire 4: Đọc file dở dang & Cơ chế State Offset
- **Đánh giá:** CDC Worker là service Read-Only, không làm nghiệp vụ rename file trên SFTP. `kafka-connect-fs` tự lưu vết Byte-Offset vào offset topic (`connect-offsets`) để đọc nối tiếp (Incremental). Bổ sung DLQ để cách ly dòng CSV rác mà không crash Connector.
- **File cần sửa:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Vị trí:** Sau dòng 320
- **Code thay đổi:** Bổ sung khối `errors.tolerance = all` và `errors.deadletterqueue.*`.

### Tripwire 5: Cấu hình Header Rác (Shotgun Config)
- **Đánh giá:** Đúng. Dọn dẹp 3 dòng header thừa đè nhau.
- **File cần sửa:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Vị trí:** Dòng 314–316
- **Code thay đổi:** Xóa `file_reader.csv.header` và `file_reader.delimited.settings.header`, chỉ giữ lại duy nhất `'file_reader.delimited.header': 'true'`.

### Tripwire 6: Kafka Key = Null & Mất Thứ Tự Dữ Liệu
- **Đánh giá:** CDC SFTP là core tổng quát từ SFTP vào Bảng Shadow. CDC Worker (`sftp_adapter.go`) tự trích xuất Primary Key động từ Mapping Rules V2. Để giữ đúng thứ tự dòng CSV, Kafka Topic SFTP tự động khởi tạo với `partitions = 1`.
- **File cần sửa:** `cdc-cms-service/internal/api/source/system_connectors_handler.go`
- **Vị trí:** Dòng 455
- **Code thay đổi:** Đảm bảo `autoCreateKafkaTopic` cho SFTP luôn set `numPartitions = 1`.

---

## 2. Code Diff Chi Tiết 100%

### Code Diff 1: `cdc-cms-web/src/pages/SourceConnectors.tsx`

```diff
<<<<
       'connector.class':                          'com.github.mmolimar.kafka.connect.fs.FsSourceConnector',
       'tasks.max':                                '1',
       'fs.uris':                                  values.fsUris || '',
       'topic':                                    sftpTopic,
       'policy.class':                             'com.github.mmolimar.kafka.connect.fs.policy.SleepyPolicy',
       'policy.sleepy.sleep':                      String(values.sleepMs ?? 3000),
       'policy.regexp':                            values.inputFilePattern || '^.*\\.csv$',
-      'policy.recursive':                         'true',
+      'policy.recursive':                         'false',
       'file_reader.class':                        'com.github.mmolimar.kafka.connect.fs.file.reader.CsvFileReader',
       'file_reader.batch_size':                   '1000',
       'policy.batch_size':                        '1000',
-      'file_reader.csv.header':                   'true',
       'file_reader.delimited.header':             'true',
-      'file_reader.delimited.settings.header':    'true',
       'value.converter':                          'org.apache.kafka.connect.json.JsonConverter',
       'value.converter.schemas.enable':           'false',
       'key.converter':                            'org.apache.kafka.connect.storage.StringConverter',
+      'errors.tolerance':                         'all',
+      'errors.deadletterqueue.topic.name':        'dlq.cdc.sftplocal.errors',
+      'errors.deadletterqueue.topic.replication.factor': '1',
+      'errors.deadletterqueue.context.headers.enable': 'true',
+      'errors.log.enable':                        'true',
+      'errors.log.include.messages':              'true',
====
```

---

### Code Diff 2: `cdc-cms-service/internal/api/source/system_connectors_handler.go`

```diff
<<<<
 	if isSFTP {
-		// Auto create topic with default partitions
-		autoCreateKafkaTopic(ctx, topicName, 3)
+		// Auto create topic with 1 partition for strict CSV row ordering
+		autoCreateKafkaTopic(ctx, topicName, 1)
 	}
====
```
