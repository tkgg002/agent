# Giải Pháp Kỹ Thuật: Production-Ready SFTP Kafka Connect Configuration

## 1. Golden Production JSON Configuration

```json
{
  "name": "sftp-reconcile-source-v1",
  "config": {
    "connector.class": "com.github.mmolimar.kafka.connect.fs.FsSourceConnector",
    "tasks.max": "1",

    "fs.uris": "sftp://${file:/etc/kafka-connect/secrets/sftp-credentials.properties:SFTP_USER}:${file:/etc/kafka-connect/secrets/sftp-credentials.properties:SFTP_PASS}@sftp-prod-internal.goopay.vn:2022/home/gp-reconcile-admin/goopay/reconcile",

    "policy.class": "com.github.mmolimar.kafka.connect.fs.policy.SleepyPolicy",
    "policy.sleepy.sleep": "60000",
    "policy.recursive": "false",
    "policy.regexp": "^reconcile_.*\\.csv$",

    "file_reader.class": "com.github.mmolimar.kafka.connect.fs.file.reader.CsvFileReader",
    "file_reader.delimited.header": "true",

    "topic": "cdc.sftplocal.reconcile.transactions",

    "key.converter": "org.apache.kafka.connect.storage.StringConverter",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "false",

    "transforms": "createKey,extractField",
    "transforms.createKey.type": "org.apache.kafka.connect.transforms.ValueToKey",
    "transforms.createKey.fields": "transaction_id",
    "transforms.extractField.type": "org.apache.kafka.connect.transforms.ExtractField$Key",
    "transforms.extractField.field": "transaction_id",

    "errors.tolerance": "all",
    "errors.deadletterqueue.topic.name": "dlq.cdc.sftplocal.reconcile.transactions",
    "errors.deadletterqueue.topic.replication.factor": "3",
    "errors.deadletterqueue.context.headers.enable": "true",
    "errors.log.enable": "true",
    "errors.log.include.messages": "true"
  }
}
```

## 2. Quy Chuẩn Vận Hành SFTP Server (Operational Standard)

### A. Atomic File Upload Protocol (Chống Partial Read)
1. **Pusher Process (Nguồn đẩy file):**
   - Upload file vào thư mục `/home/gp-reconcile-admin/goopay/reconcile/` với tên tạm: `reconcile_20260814_01.csv.tmp`
   - Ngay sau khi upload xong 100% (SFTP `close` session), thực hiện SFTP `rename` sang: `reconcile_20260814_01.csv`
2. **Connector Policy:**
   - Chỉ bắt matcher `^reconcile_.*\\.csv$`. Mọi file `.tmp` bị bỏ qua 100%.

### B. SFTP Retention & Cleanup Cronjob (Chống nghẽn Directory Listing)
Chạy cronjob hàng ngày lúc 01:00 AM trên SFTP Server:
```bash
#!/bin/bash
# Clean & Archive SFTP files older than 3 days
SOURCE_DIR="/home/gp-reconcile-admin/goopay/reconcile"
ARCHIVE_DIR="/home/gp-reconcile-admin/goopay/reconcile_archive"

mkdir -p "$ARCHIVE_DIR/$(date +%Y-%m)"
find "$SOURCE_DIR" -maxdepth 1 -name "reconcile_*.csv" -mtime +3 -exec mv {} "$ARCHIVE_DIR/$(date +%Y-%m)/" \;
find "$ARCHIVE_DIR" -type f -mtime +90 -delete
```
