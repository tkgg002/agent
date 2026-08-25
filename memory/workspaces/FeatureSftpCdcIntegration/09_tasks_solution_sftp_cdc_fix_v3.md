# Kịch bản giải pháp: Sửa lỗi PK INT8, SFTP Recursive Polling & MongoDB Topic Naming

Tài liệu này đặc tả chi tiết các phần mã nguồn cần sửa đổi và trình tự thực thi của Muscle.

---

## 1. Các file cần chỉnh sửa (Surgical Changes)

### A. Centralized Data Service

#### [base_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/base_handler.go)
Sửa `reTypeWhitelist` (Line 194) để chấp nhận kiểu dữ liệu `INT2`, `INT4`, `INT8`.

Target Content:
```go
var reTypeWhitelist = regexp.MustCompile(`^(SMALLINT|INTEGER|BIGINT|REAL|DOUBLE PRECISION|NUMERIC|DECIMAL|BOOLEAN|DATE|TIME|TIMESTAMP|TIMESTAMPTZ|INTERVAL|JSON|JSONB|UUID|INET|CIDR|MACADDR|BYTEA|TEXT|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|DECIMAL\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|(SMALLINT|INTEGER|BIGINT|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$`)
```

Replacement Content:
```go
var reTypeWhitelist = regexp.MustCompile(`^(SMALLINT|INTEGER|BIGINT|INT2|INT4|INT8|REAL|DOUBLE PRECISION|NUMERIC|DECIMAL|BOOLEAN|DATE|TIME|TIMESTAMP|TIMESTAMPTZ|INTERVAL|JSON|JSONB|UUID|INET|CIDR|MACADDR|BYTEA|TEXT|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|DECIMAL\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|(SMALLINT|INTEGER|BIGINT|INT2|INT4|INT8|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$`)
```

---

### B. CDC CMS Web (Frontend)

#### [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx)

**1. Kích hoạt đệ quy cho SFTP Connector (Line 310)**

Target Content:
```typescript
      'policy.regexp':                            values.inputFilePattern || '^.*\\.csv$',
      'policy.recursive':                         'false',
      'file_reader.class':                        'com.github.mmolimar.kafka.connect.fs.file.reader.CsvFileReader',
```

Replacement Content:
```typescript
      'policy.regexp':                            values.inputFilePattern || '^.*\\.csv$',
      'policy.recursive':                         'true',
      'file_reader.class':                        'com.github.mmolimar.kafka.connect.fs.file.reader.CsvFileReader',
```

**2. Điều chỉnh MongoDB topicPrefix fallback khi parse seed (Line 375)**

Target Content:
```typescript
      connectorName: source.connector_name,
      topicPrefix: source.topic_prefix || cfg['topic.prefix'] || `cdc.${database}`,
      connectionUrl,
```

Replacement Content:
```typescript
      connectorName: source.connector_name,
      topicPrefix: source.topic_prefix || cfg['topic.prefix'] || `cdc.goopay.${slugifyForShadow(source.connector_name || 'connector')}`,
      connectionUrl,
```

**3. Điều chỉnh MongoDB topicPrefix auto-set khi điền form (Line 467-469)**

Target Content:
```typescript
    } else {
      form.setFieldValue('topicPrefix', TOPIC_PREFIX_BY_DB[dbKind]);
    }
```

Replacement Content:
```typescript
    } else if (dbKind === 'mongodb') {
      const name = slugifyForShadow(String(connectorNameValue || 'connector'));
      form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);
    } else {
      form.setFieldValue('topicPrefix', TOPIC_PREFIX_BY_DB[dbKind]);
    }
```

---

## 2. Các bước thực thi dành cho Muscle

1. Chạy sửa đổi [base_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/base_handler.go).
2. Chạy sửa đổi [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx).
3. Chạy lệnh verify unit test của backend:
   ```bash
   cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service
   go test ./...
   ```
4. Chạy lệnh verify build frontend:
   ```bash
   cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web
   npm run build
   ```
5. Đăng ký tiến độ vào `05_progress_sftp_cdc_connector.md`.
