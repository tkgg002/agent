# 09 - Technical Solution & Code Demo (Single Best Approach)

## 1. Phương án kỹ thuật duy nhất (The Single Best Approach)
Triển khai chuẩn hóa Topic Prefix tại file `cdc-cms-web/src/pages/SourceConnectors.tsx`:
1. **SFTP (`kafka-connect-fs`):** Giữ nguyên logic `${TOPIC_PREFIX_SFTP}.${slugify(connector_name)}` (VD: `cdc.sftp.my_connector`).
2. **MongoDB / PostgreSQL / MySQL:** Gán tự động base prefix:
   - MongoDB: `TOPIC_PREFIX_MONGODB` (`cdc.goopay` hoặc từ ENV `VITE_TOPIC_PREFIX_MONGODB`).
   - PostgreSQL: `TOPIC_PREFIX_POSTGRESQL` (`cdc.gpay` hoặc từ ENV `VITE_TOPIC_PREFIX_POSTGRESQL`).
   - MySQL: `TOPIC_PREFIX_MYSQL` (`cdc.mariadb` hoặc từ ENV `VITE_TOPIC_PREFIX_MYSQL`).
3. **Giao diện người dùng (UI):** Trường `Topic Prefix` luôn luôn bị khóa cứng (`<Input disabled />`) cho tất cả các loại database.

---

## 2. Chi tiết Code Demo đề xuất thay đổi

### Điểm 1: Khóa cứng 100% ô nhập liệu Topic Prefix (Dòng 1589-1603)
```tsx
<Col span={12}>
  <Form.Item
    name="topicPrefix"
    label={dbKind === 'sftp' ? 'Kafka Topic (topic)' : 'Topic Prefix'}
    tooltip={dbKind === 'sftp' ? 'Tên Kafka topic đầy đủ mà kafka-connect-fs sẽ ghi vào' : undefined}
    rules={[{ required: true }]}
  >
    <Input disabled />
  </Form.Item>
</Col>
```

### Điểm 2: Logic Auto-Fill khi mở Form tạo mới (Dòng 480-494)
```typescript
useEffect(() => {
  if (!editorOpen || editorMode !== 'create') return;
  const name = slugifyForShadow(String(connectorNameValue || 'connector'));
  if (dbKind === 'sftp') {
    // SFTP (kafka-connect-fs): cần connector name trong prefix vì không tự append database.collection
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_SFTP}.${name}`);
  } else if (dbKind === 'mongodb') {
    // Debezium tự append {database}.{collection} → prefix chỉ cần base (e.g. cdc.goopay)
    form.setFieldValue('topicPrefix', TOPIC_PREFIX_MONGODB);
  } else if (dbKind === 'mysql') {
    form.setFieldValue('topicPrefix', TOPIC_PREFIX_MYSQL);
  } else if (dbKind === 'postgresql') {
    form.setFieldValue('topicPrefix', TOPIC_PREFIX_POSTGRESQL);
  }
}, [dbKind, editorOpen, editorMode, form, connectorNameValue]);
```

### Điểm 3: Parse Seed Fallback cho MongoDB (Dòng 391-399)
```typescript
topicPrefix: source.topic_prefix || cfg['topic.prefix'] || TOPIC_PREFIX_MONGODB,
```

### Điểm 4: Parse Seed Fallback cho PostgreSQL / MySQL (Dòng 432-436)
```typescript
topicPrefix: source.topic_prefix || cfg['topic.prefix'] || TOPIC_PREFIX_BY_DB[dbKind] || TOPIC_PREFIX_POSTGRESQL,
```
