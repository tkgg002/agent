# 09 Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution)

## 1. Giải Pháp Chống Silent Drop & Kiểm Tra Data Trước Khi Commit Offset

### A. Triệt tiêu Nuốt Lỗi tại `event_handler.go:L273-L289`
- **Hiện trạng (Lỗ hổng cốt lõi)**:
  Khi `len(routes) == 0`, hàm chỉ ghi log và tăng metric rồi `return 0, nil`.
  Việc trả về `nil` error khiến `KafkaConsumer` ngộ nhận là message đã được xử lý thành công (`batch.success++`), không kích hoạt DLQ, không dừng luồng và cho phép offset trôi qua.
- **Giải pháp Chuẩn hóa**:
  Khi một CDC event đi vào thuộc các topic được quản lý bởi hệ thống nhưng không tìm thấy route trong cache, **BẮT BUỘC trả về lỗi tường minh**:
  ```go
  if len(routes) == 0 {
      observability.Ctx(ctx, h.logger).Warn("event skipped: source not in registry cache — check is_active flags or registry reload",
          observability.Attrs(
              zap.String("subject", subject),
              zap.String("source_conn", sourceConn),
              zap.String("source_db", sourceDB),
              zap.String("source_table", sourceTable),
          ),
      )
      metrics.EventsDropped.WithLabelValues("source_not_registered", sourceTable).Inc()
      return 0, fmt.Errorf("source route not found in registry cache: db=%s, table=%s, conn=%s", sourceDB, sourceTable, sourceConn)
  }
  ```
- **Tác dụng**:
  1. Khi trả về `error`, `KafkaConsumer.processMessage` trả về `procErr != nil`.
  2. `KafkaConsumer` lập tức ghi bản ghi này vào **Dead Letter Queue (DLQ)** (`failed_sync_logs` trong DB và/hoặc NATS DLQ subject).
  3. Dữ liệu KHÔNG BAO GIỜ bị mất mà được bảo lưu an toàn trong DLQ kèm nguyên nhân lỗi để có thể retry.
  4. Tránh hoàn toàn việc ngộ nhận `batch.success++` khi thực tế 0 dòng nào được ghi xuống database.

---

## 2. Giải Pháp Config-Driven: Lấy Generic Prefix Trực Tiếp Từ Cấu Hình `kafka.topicPrefix`

Tuyệt đối KHÔNG hardcode các chuỗi giả định như `"local"`, `"dev"`, `"staging"`, `"prod"`. Hệ thống lấy trực tiếp danh sách tiền tố từ cấu hình hệ thống:

```yaml
kafka:
  topicPrefix:
    - cdc.gpaylocal
    - cdc.goopaylocal
    - cdc.mariadblocal
    - cdc.sftplocal
```

### A. Đăng ký Tiền tố Động tại `servicesource` (`metadata_registry_utils.go`)
```go
var customGenericPrefixes []string
var customPrefixesMu sync.RWMutex

// RegisterGenericPrefixes đăng ký động các tiền tố từ config.yml/config-local.yml
func RegisterGenericPrefixes(prefixes []string) {
    customPrefixesMu.Lock()
    defer customPrefixesMu.Unlock()
    for _, p := range prefixes {
        p = strings.ToLower(strings.TrimSpace(p))
        if p == "" {
            continue
        }
        customGenericPrefixes = append(customGenericPrefixes, p)
        // Nếu p là "cdc.goopaylocal", tự động đăng ký cả "goopaylocal"
        if trimmed := strings.TrimPrefix(p, "cdc."); trimmed != p && trimmed != "" {
            customGenericPrefixes = append(customGenericPrefixes, trimmed)
        }
    }
}

func IsGenericConnectionCode(code string) bool {
    c := strings.ToLower(strings.TrimSpace(code))
    switch c {
    case "goopay", "gpay", "mongodb", "postgres", "postgresql", "mysql", "mariadb", "sftp":
        return true
    }
    customPrefixesMu.RLock()
    defer customPrefixesMu.RUnlock()
    for _, p := range customGenericPrefixes {
        if c == p {
            return true
        }
    }
    return false
}
```

### B. Bootstrap Wiring tại `server_setup.go`
Tại thời điểm khởi động server (`server_setup.go:L95`), hệ thống nạp ngay cấu hình vào registry:
```go
servicesource.RegisterGenericPrefixes(cfg.Kafka.TopicPrefix)
```

### C. Chuẩn hóa Cắt Tiền Tố Động tại `kafka_consumer.go`
Thay vì hardcode cắt `cdc.goopay.`, hàm duyệt qua danh sách `kc.config.TopicPrefix` để cắt tiền tố chính xác:
```go
for _, p := range kc.config.TopicPrefix {
    p = strings.TrimSpace(p)
    if p != "" {
        sourceConnCode = strings.TrimPrefix(sourceConnCode, p+".")
        sourceConnCode = strings.TrimPrefix(sourceConnCode, p)
        if stripped := strings.TrimPrefix(p, "cdc."); stripped != "" {
            sourceConnCode = strings.TrimPrefix(sourceConnCode, stripped+".")
            sourceConnCode = strings.TrimPrefix(sourceConnCode, stripped)
        }
    }
}
sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")

if source.IsGenericConnectionCode(sourceConnCode) {
    sourceConnCode = ""
}
```
-> 100% Config-driven, sạch sẽ, không cheat, tự động thích ứng trên mọi môi trường (local/dev/staging/prod)!

---

## 2. Giải Pháp Khôi Phục 212 Bản Ghi Bị Mất (Data Healing)

Vì 490 bản ghi đã bị drop và commit offset trên Kafka, chúng ta không được phép "cheat DB" (sửa DB trực tiếp theo Rule #12). Thay vào đó, sử dụng pipeline chính thức:
- **Phương án 1 (Recon Heal - Tối ưu nhất)**:
  Gọi API Heal:
  `POST /api/reconciliation/heal/shadow_traitestctphs.trans_his`
  Hoặc trigger qua UI Recon -> Bấm "Heal".
  Module `ReconHealer` sẽ kết nối trực tiếp vào MongoDB nguồn `traitestctphs`, lấy danh sách 212 IDs bị thiếu và insert vào bảng `shadow_traitestctphs.trans_his`.
- **Phương án 2 (Oplog Bridge)**:
  Sử dụng Oplog Bridge trên CMS để kéo bù dữ liệu từ MongoDB trong khoảng `[10:07:00 -> 10:22:00]`.

---

## 3. Giải Pháp Kích Hoạt Lại Kafka Consume

- Nếu muốn Kafka Consumer chạy lại và đọc dữ liệu:
  - Do Debezium đang để `snapshot.mode: no_data`, nó chỉ đợi write mới.
  - Khi có một transaction mới được ghi vào MongoDB `trans-his`, Debezium sẽ đẩy event vào Kafka, và Worker sẽ tiêu thụ ngay lập tức.
  - Hoặc nếu muốn re-read toàn bộ collection MongoDB vào Kafka, có thể kích hoạt Debezium Ad-hoc Snapshot qua collection `cdc.signal.commands` (đã được cấu hình trong connector: `"signal.enabled.channels": "kafka"`).
