# 09 - Technical Architecture & Solution Design: Connection-Scoped Namespace (Brain)

## 1. Bản chất Kỹ thuật của Vấn đề (Root Architectural Problem)

Khi hệ thống mở rộng đa nguồn (Multi-Source / Multi-Cluster):
- **Nguồn 1:** MongoDB Cluster A (Core) -> DB: `payment-services`, Collection: `payments`
- **Nguồn 2:** PostgreSQL Cluster B (RDB) -> DB: `payment-services`, Table: `payments`
- **Nguồn 3:** MongoDB Cluster C (Merchant) -> DB: `payment-services`, Collection: `payments`

### Tại sao lại xảy ra xung đột?
1. **Ở Kafka Broker:** Nếu cả Nguồn 1 và Nguồn 3 đều dùng `topic.prefix = cdc.goopay`, Debezium sẽ đẩy cả 2 luồng dữ liệu vào CÙNG MỘT Topic `cdc.goopay.payment-services.payments`. Payload của Debezium chỉ chứa `source.db: "payment-services"` và `source.collection: "payments"` -> Consumer không có cách nào phân biệt record nào thuộc Cluster A hay Cluster C -> **GÂY NHIỄM ĐỘC DỮ LIỆU (Data Pollution)**.
2. **Ở Frontend CMS:** Code cũ tự ý nối `connector_name` vào prefix (`cdc.goopay.payment_service`) trong khi DB name cũng là `payment-service` -> Sinh ra topic 5 segments bị lặp từ: `cdc.goopay.payment_service.payment-service.payments`.
3. **Ở Backend Worker (`centralized-data-service`):** `ResolveSourceRoutes` chỉ tìm theo `sourceDB:sourceTable` (`payment-services|payments`). Nếu cả 2 connection cùng đăng ký `payment-services|payments`, router sẽ gộp chung route hoặc ghi sai Shadow Table.

---

## 2. GIẢI PHÁP TỐI ƯU DUY NHẤT (THE SINGLE BEST APPROACH)

### Mô hình: `Connection-Scoped Namespace Architecture` (Định danh theo Mã Kết Nối)

Mỗi Connector thực chất được sinh ra từ một **Source Connection** trong bảng `cdc_system.connection_registry` (có `connection_code` duy nhất, ví dụ: `mongo_core`, `mongo_merchant`, `pg_main`).

Quy tắc đặt `topic.prefix` tự động (Khóa cứng 100% trên CMS):
$$\text{topic.prefix} = \text{cdc} . \langle\text{connection\_code}\rangle$$

### Bảng đối chiếu Topic sinh ra sau chuẩn hóa:
| Nguồn | Connection Code | DB Name | Table/Coll | Topic.prefix | Kafka Topic sinh ra (Chuẩn Debezium) |
|---|---|---|---|---|---|
| **Mongo 1** | `mongo_core` | `payment-services` | `payments` | `cdc.mongo_core` | `cdc.mongo_core.payment-services.payments` |
| **Mongo 2** | `mongo_merchant` | `payment-services` | `payments` | `cdc.mongo_merchant` | `cdc.mongo_merchant.payment-services.payments` |
| **Postgres** | `pg_main` | `payment-services` | `payments` | `cdc.pg_main` | `cdc.pg_main.payment-services.public.payments` |
| **SFTP** | `sftp_bank` | `sftp_data` | `reconcile` | `cdc.sftp.sftp_bank` | `cdc.sftp.sftp_bank.reconcile` |

---

## 3. Code Demo chi tiết cho từng Tầng

### Tầng 1: Frontend CMS Web (`SourceConnectors.tsx`)
Khi người dùng chọn Connection hoặc nhập `connectorName`, `topicPrefix` tự động sinh theo `connection_code` và **LOCKED 100%**:

```typescript
// Auto-fill topicPrefix chuẩn hóa theo Connection Code / Connector Name:
useEffect(() => {
  if (!editorOpen || editorMode !== 'create') return;
  const connSlug = slugifyForShadow(String(connectorNameValue || 'connector'));
  if (dbKind === 'sftp') {
    form.setFieldValue('topicPrefix', `cdc.sftp.${connSlug}`);
  } else {
    // Tự động gán: cdc.<connection_code> (VD: cdc.mongo_core, cdc.mongo_merchant)
    form.setFieldValue('topicPrefix', `cdc.${connSlug}`);
  }
}, [dbKind, editorOpen, editorMode, form, connectorNameValue]);
```

### Tầng 2: Backend Worker Router (`event_handler.go`)
Khi nhận topic `cdc.mongo_core.payment-services.payments`, Worker trích xuất `connection_code = mongo_core` từ topic để định tuyến chính xác vào đúng Shadow Binding:

```go
// Trích xuất Connection Code từ topic prefix (segment thứ 2 sau 'cdc'):
parts := strings.Split(subject, ".")
var connCode, db, table string
if len(parts) >= 4 && parts[0] == "cdc" {
    connCode = parts[1] // "mongo_core"
    db = parts[2]       // "payment-services"
    table = parts[3]    // "payments"
}

// Định tuyến chính xác 100% theo Connection Code + DB + Table:
routeKey := fmt.Sprintf("%s:%s|%s", connCode, db, table) // "mongo_core:payment-services|payments"
routes := h.registrySvc.ResolveSourceRoutes(routeKey)
```
