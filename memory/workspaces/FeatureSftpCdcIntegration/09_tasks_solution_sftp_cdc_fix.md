# Thiết kế kỹ thuật: FIX PLAN sau Audit QC - SFTP CDC Connector

Tài liệu này chi tiết hóa CHÍNH XÁC từng dòng code cần sửa/thêm cho 8 hạng mục fix.

---

## FIX-1: Migration SQL mở rộng Check Constraint

### [NEW] Migration file
**Path**: `cdc-cms-service/migrations/schema/core/060_add_sftp_source_type.sql`

```sql
-- 060_add_sftp_source_type.sql
-- Mở rộng check constraint cho source_type để chấp nhận giá trị 'sftp'
-- cho tính năng tích hợp SFTP Source Connector.

ALTER TABLE cdc_table_registry DROP CONSTRAINT IF EXISTS ctr_check_source_type;
ALTER TABLE cdc_table_registry ADD CONSTRAINT ctr_check_source_type
    CHECK (source_type IN ('mongodb', 'mysql', 'postgresql', 'sftp'));
```

**Lưu ý**: Kiểm tra số thứ tự migration (060_) chưa bị trùng bằng `ls migrations/schema/core/ | tail -5`.

---

## FIX-2: Sửa logic parse `db`/`table` từ SFTP topic

### [MODIFY] `event_handler.go` (centralized-data-service)

**Dòng 132-137 hiện tại (SAI):**
```go
if isSFTP {
    parts := strings.Split(subject, ".")
    if len(parts) >= 2 {
        db = parts[0]
        table = parts[1]
    }
}
```

**Sửa thành:**
```go
if isSFTP {
    // Topic naming convention cho SFTP: sftp.<source_name>
    // VD: sftp.reconcile_final → db="sftp", table="reconcile_final"
    // VD: sftp.reconcile.final → db="sftp", table="reconcile_final" (join các phần còn lại bằng "_")
    parts := strings.Split(subject, ".")
    if len(parts) >= 2 {
        db = parts[0] // "sftp"
        table = strings.Join(parts[1:], "_") // join toàn bộ phần còn lại bằng "_"
    }
}
```

**Giải thích**: `strings.Join(parts[1:], "_")` sẽ biến `sftp.reconcile.final.events` thành `table = "reconcile_final_events"` và `sftp.reconcile_final` thành `table = "reconcile_final"`. Linh hoạt với mọi format topic.

---

## FIX-3: Gán `Source = "sftp-connector"` trong adapter

### [MODIFY] `sftp_adapter.go` (centralized-data-service)

**Dòng 21-29 hiện tại:**
```go
event := &shadow.CDCEvent{
    SpecVersion: "1.0",
    ID:          "", // Sẽ tự sinh uuid ở handler nếu trống
    Type:        "sftp.reconcile.final",
    Data: shadow.CDCEventData{
        Op:    "c",
        After: payload,
    },
    KafkaTopic: topic,
}
```

**Sửa thành:**
```go
event := &shadow.CDCEvent{
    SpecVersion: "1.0",
    ID:          uuid.New().String(),
    Source:      "sftp-connector",
    Type:        topic,
    Data: shadow.CDCEventData{
        Op:    "c",
        After: payload,
    },
    KafkaTopic: topic,
}
```

**Import thêm**: `"github.com/google/uuid"` — kiểm tra `go.mod` xem đã có chưa. Nếu chưa: `go get github.com/google/uuid`.

**Giải thích:**
- FIX-3: `Source: "sftp-connector"` → downstream ghi đúng vào UpsertRecord.Source.
- FIX-4: `ID: uuid.New().String()` → giải quyết comment hứa sinh UUID.
- SMELL-3: `Type: topic` → dynamic thay vì hardcode.

---

## FIX-5: kafka_consumer.go — KHÔNG cần sửa code

**Phân tích thực tế:** `kafka_consumer.go` sử dụng cơ chế **dynamic topic discovery** qua `discoverTopics()` trong `topic_helper.go`:

1. Nó dial vào Kafka broker, đọc toàn bộ partitions.
2. Filter theo `topicPrefix` config (ví dụ: `cdc.goopay`, `cdc.gpaylocal`).
3. **Cross-match** với `registrySvc.GetDebeziumTables()` — chỉ subscribe những topic có tên table nằm trong registry.

**Điều cần làm** (Config, KHÔNG phải code):
- Thêm prefix `sftp.` vào danh sách `topicPrefix` trong config YAML:
  ```yaml
  topicPrefix:
    - cdc.gpaylocal
    - cdc.goopay
    - sftp.         # <-- THÊM
  ```
- Đảm bảo bảng `source_object_registry` đã đăng ký `reconcile_final` (để `GetDebeziumTables()` trả về nó).

**Lưu ý**: `filterMatchingTopics` (topic_helper.go:199-203) yêu cầu topic có ít nhất 4 phần (`len(parts) >= 4`) để trích xuất `tableName`. Nếu topic SFTP chỉ có 2 phần (`sftp.reconcile_final`), `tableName` sẽ rỗng và bị filter ra. **CẦN SỬA** thêm 1 dòng trong `filterMatchingTopics` để xử lý topic ngắn.

### [MODIFY] `topic_helper.go` (centralized-data-service)

**Dòng 199-206 hiện tại:**
```go
parts := strings.Split(topic, ".")
var tableName string
if len(parts) >= 4 {
    tableName = parts[len(parts)-1]
}
if len(debeziumTables) > 0 && !debeziumTables[tableName] {
    continue
}
```

**Sửa thành:**
```go
parts := strings.Split(topic, ".")
var tableName string
if len(parts) >= 4 {
    tableName = parts[len(parts)-1]
} else if len(parts) >= 2 {
    // SFTP topics có ít segment hơn: sftp.reconcile_final
    tableName = strings.Join(parts[1:], "_")
}
if len(debeziumTables) > 0 && !debeziumTables[tableName] {
    continue
}
```

---

## FIX-6: Bổ sung edge-case unit tests

### [MODIFY] `sftp_adapter_test.go` (centralized-data-service)

**Thêm các test case sau:**

```go
func TestSFTPEventAdapter_ConvertToCDCEvent_EmptyJSON(t *testing.T) {
    adapter := NewSFTPEventAdapter()
    event, err := adapter.ConvertToCDCEvent([]byte(`{}`), "sftp.test")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(event.Data.After) != 0 {
        t.Errorf("expected empty After map, got %v", event.Data.After)
    }
}

func TestSFTPEventAdapter_ConvertToCDCEvent_SourceField(t *testing.T) {
    adapter := NewSFTPEventAdapter()
    event, err := adapter.ConvertToCDCEvent([]byte(`{"id":"1"}`), "sftp.reconcile_final")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    src, ok := event.Source.(string)
    if !ok || src != "sftp-connector" {
        t.Errorf("expected Source 'sftp-connector', got %v", event.Source)
    }
}

func TestSFTPEventAdapter_ConvertToCDCEvent_DynamicType(t *testing.T) {
    adapter := NewSFTPEventAdapter()
    event, err := adapter.ConvertToCDCEvent([]byte(`{"id":"1"}`), "sftp.payout_final")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if event.Type != "sftp.payout_final" {
        t.Errorf("expected Type 'sftp.payout_final', got %s", event.Type)
    }
}

func TestSFTPEventAdapter_ConvertToCDCEvent_UUID(t *testing.T) {
    adapter := NewSFTPEventAdapter()
    event, err := adapter.ConvertToCDCEvent([]byte(`{"id":"1"}`), "sftp.test")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if event.ID == "" {
        t.Errorf("expected non-empty UUID ID, got empty string")
    }
}

func TestSFTPEventAdapter_ConvertToCDCEvent_EmptyBytes(t *testing.T) {
    adapter := NewSFTPEventAdapter()
    _, err := adapter.ConvertToCDCEvent([]byte(""), "sftp.test")
    if err == nil {
        t.Fatalf("expected error for empty bytes, got nil")
    }
}

func TestSFTPEventAdapter_ConvertToCDCEvent_JSONArray(t *testing.T) {
    adapter := NewSFTPEventAdapter()
    _, err := adapter.ConvertToCDCEvent([]byte(`[1,2,3]`), "sftp.test")
    if err == nil {
        t.Fatalf("expected error for JSON array, got nil")
    }
}
```

---

## FIX-7: connector_types.go và debezium_connector.go — HOÃN

Sau khi đọc kỹ:
- `connector_types.go` chỉ chứa view-model types cho Kafka Connect status API (không define source types) → **KHÔNG CẦN SỬA** cho phase này.
- `debezium_connector.go` không tồn tại tại path trong plan → plan gốc có lỗi path. Cần xác minh lại. **HOÃN** sang phase tiếp.

---

## Checklist thực thi cho Muscle

1. [ ] Kiểm tra số migration cuối: `ls cdc-cms-service/migrations/schema/core/ | tail -5`
2. [ ] Tạo migration SQL (FIX-1)
3. [ ] Sửa `sftp_adapter.go` (FIX-3 + FIX-4 + SMELL-3): Source, UUID, Type
4. [ ] Sửa `event_handler.go` (FIX-2): parse db/table bằng `strings.Join`
5. [ ] Sửa `topic_helper.go` (FIX-5): xử lý topic ngắn
6. [ ] Thêm edge-case tests (FIX-6)
7. [ ] Chạy `go test -v ./internal/handler/shadow/...` → PASS
8. [ ] Cập nhật unit test cũ nếu assert `Type` bị thay đổi
