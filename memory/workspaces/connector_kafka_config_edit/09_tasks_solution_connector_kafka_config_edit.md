# Hồ sơ giải pháp kỹ thuật - Thêm input Kafka Config khi Edit Connector

Task này sẽ thực hiện tích hợp thêm một trường nhập liệu "Kafka Config" động trên giao diện chỉnh sửa Connector (chấp nhận định dạng JSON), sau đó tự động merge vào cấu hình khi gửi yêu cầu cập nhật lên Backend.

## 1. File cần chỉnh sửa
*   [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx)

## 2. Chi tiết các bước thay đổi code

### Bước 2.1: Bổ sung trường `kafkaConfig` vào interface `ConnectionFormValues`
Tại phần định nghĩa kiểu dữ liệu (gần dòng 99-117), thêm `kafkaConfig?: string;` vào `ConnectionFormValues`:
```diff
 interface ConnectionFormValues {
   dbKind: DbKind;
   connectorName: string;
   topicPrefix: string;
   connectionUrl?: string;
   host?: string;
   port?: number;
   database: string;
   username?: string;
   password?: string;
   collectionNames?: string;
   tableIncludeList?: string;
   schemaIncludeList?: string;
   serverId?: number;
   slotName?: string;
   publicationName?: string;
   reason: string;
+  kafkaConfig?: string;
 }
```

### Bước 2.2: Định nghĩa danh sách các trường cấu hình gốc (Native Config Keys) và hàm trích xuất Kafka Config
Ta định nghĩa một danh sách các key cấu hình mặc định được quản lý bởi form UI để loại trừ chúng khi trích xuất Kafka config custom:
```typescript
const NATIVE_CONFIG_KEYS = new Set([
  'connector.class',
  'mongodb.connection.string',
  'database.include.list',
  'collection.include.list',
  'topic.prefix',
  'capture.mode',
  'snapshot.mode',
  'key.converter',
  'key.converter.schema.registry.url',
  'value.converter',
  'value.converter.schema.registry.url',
  'schema.history.internal.kafka.bootstrap.servers',
  'signal.enabled.channels',
  'signal.data.collection',
  'signal.kafka.topic',
  'signal.kafka.bootstrap.servers',
  'signal.kafka.consumer.group.id',
  'database.hostname',
  'database.port',
  'database.user',
  'database.password',
  'table.include.list',
  'database.server.id',
  'schema.history.internal.kafka.topic',
  'database.dbname',
  'schema.include.list',
  'plugin.name',
  'slot.name',
  'publication.name',
  'publication.autocreate.mode',
]);

function extractKafkaConfig(cfg: Record<string, string>): string {
  const custom: Record<string, string> = {};
  for (const [key, val] of Object.entries(cfg)) {
    if (key.startsWith('producer.') || key.startsWith('consumer.')) {
      custom[key] = val;
    }
  }
  return Object.keys(custom).length > 0 ? JSON.stringify(custom, null, 2) : '';
}
```
Đặt các định nghĩa này ở phía trước hàm `buildConnectorConfig`.

### Bước 2.3: Thay đổi logic merge Kafka Config trong hàm `buildConnectorConfig`
Tại cuối hàm `buildConnectorConfig`, trước khi return config, ta merge JSON từ trường `kafkaConfig` nhập vào:
```typescript
function buildConnectorConfig(values: ConnectionFormValues, mode: EditorMode) {
  let config: Record<string, string> = {};
  // ... (giữ nguyên logic build ban đầu cho các DB kind) ...

  if (values.kafkaConfig) {
    try {
      const customConfig = JSON.parse(values.kafkaConfig);
      if (customConfig && typeof customConfig === 'object') {
        const stringifiedCustom: Record<string, string> = {};
        for (const [key, val] of Object.entries(customConfig)) {
          stringifiedCustom[key] = typeof val === 'object' ? JSON.stringify(val) : String(val);
        }
        config = { ...config, ...stringifiedCustom };
      }
    } catch (e) {
      console.error('Failed to parse kafkaConfig JSON', e);
    }
  }

  return config;
}
```

### Bước 2.4: Trích xuất cấu hình Kafka Config khi click Edit
Cập nhật các hàm khởi tạo Form state khi mở form edit và create:
*   Trong `openCreate`:
    ```typescript
        publicationName: '',
        reason: '',
        kafkaConfig: '',
    ```
*   Trong `openEdit`:
    ```typescript
      const connector = connectorByName.get(source.connector_name);
      const seed = parseConnectionSeed(source, connector);
      const kafkaConfig = connector?.config ? extractKafkaConfig(connector.config) : '';
      // ...
      form.setFieldsValue({
        // ... (giữ các trường cũ) ...
        reason: '',
        kafkaConfig,
      });
    ```
*   Trong `openEditFromConnector`:
    ```typescript
      const dbKind = detectDbKind(connector.connector_class);
      const cfg = connector.config || {};
      const kafkaConfig = extractKafkaConfig(cfg);
      // ...
      form.setFieldsValue({ ...seed, reason: '', kafkaConfig });
    ```

### Bước 2.5: Bổ sung ô nhập liệu Kafka Config vào form giao diện (JSX)
Ngay phía trên trường `Reason` (ở khoảng dòng 1322), thêm:
```typescript
          <Form.Item
            name="kafkaConfig"
            label="Kafka Config"
            tooltip="Cấu hình Kafka tùy chỉnh dưới dạng JSON (ví dụ: { &quot;producer.override.max.request.size&quot;: &quot;10485760&quot; })"
            rules={[
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve();
                  try {
                    JSON.parse(value);
                    return Promise.resolve();
                  } catch (e) {
                    return Promise.reject(new Error('Cấu hình Kafka Config phải là JSON hợp lệ'));
                  }
                },
              },
            ]}
          >
            <Input.TextArea rows={3} placeholder='{"producer.override.max.request.size": "10485760"}' />
          </Form.Item>
```
