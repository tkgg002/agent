# FIX PLAN v2 — Khắc phục BLOCKER-1 (Topic Discovery Filter)

## Phương án: C (Minimal code change)

---

## FIX-B1: Sửa `filterMatchingTopics` bỏ qua `debeziumTables` check cho SFTP topic

### [MODIFY] `topic_helper.go`
**Path**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/topic_helper.go`

**Dòng 199-208 hiện tại:**
```go
		parts := strings.Split(topic, ".")
		var tableName string
		if len(parts) >= 4 {
			tableName = parts[len(parts)-1]
		} else if len(parts) >= 2 {
			tableName = strings.Join(parts[1:], "_")
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
			tableName = strings.Join(parts[1:], "_")
		}
		// SFTP topics bypass debeziumTables filter — chúng dùng sync_engine='custom'
		// và không được liệt kê bởi GetDebeziumTables().
		isSFTPTopic := strings.HasPrefix(topic, "sftp.")
		if !isSFTPTopic && len(debeziumTables) > 0 && !debeziumTables[tableName] {
			continue
		}
```

**Giải thích:** Topic có prefix `sftp.` sẽ luôn pass qua filter, không bị chặn bởi `debeziumTables`. Mọi topic CDC thường (prefix `cdc.*`) vẫn giữ nguyên logic filter cũ.

---

## FIX-B2: Viết unit test cho SFTP topic bypass

### [MODIFY] `kafka_consumer_test.go` hoặc tạo `topic_helper_test.go` mới
**Path**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/topic_helper_test.go`

```go
package shadow

import "testing"

func TestFilterMatchingTopics_SFTPBypassDebeziumFilter(t *testing.T) {
	allTopics := []string{
		"cdc.goopay.db1.users",
		"sftp.reconcile_final",
		"cdc.goopay.db1.orders",
	}
	prefixes := []string{"cdc.goopay", "sftp."}
	debeziumTables := map[string]bool{
		"users":  true,
		"orders": true,
		// Lưu ý: "reconcile_final" KHÔNG nằm trong debeziumTables
	}

	topics, perPrefix, _ := filterMatchingTopics(allTopics, prefixes, debeziumTables)

	// sftp.reconcile_final phải lọt qua dù không nằm trong debeziumTables
	found := false
	for _, t2 := range topics {
		if t2 == "sftp.reconcile_final" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected sftp.reconcile_final to bypass debeziumTables filter, but it was excluded. topics=%v", topics)
	}

	// CDC topics vẫn hoạt động bình thường
	if len(topics) != 3 {
		t.Fatalf("expected 3 topics (2 CDC + 1 SFTP), got %d: %v", len(topics), topics)
	}

	if perPrefix["sftp."] != 1 {
		t.Errorf("expected perPrefix[sftp.]=1, got %d", perPrefix["sftp."])
	}
}

func TestFilterMatchingTopics_SFTPNotMatchedWithoutPrefix(t *testing.T) {
	allTopics := []string{"sftp.reconcile_final"}
	prefixes := []string{"cdc.goopay"} // Không có prefix sftp.
	debeziumTables := map[string]bool{}

	topics, _, _ := filterMatchingTopics(allTopics, prefixes, debeziumTables)

	if len(topics) != 0 {
		t.Fatalf("expected 0 topics when sftp prefix not configured, got %d: %v", len(topics), topics)
	}
}
```

---

## FIX-B3: SQL Seed Script cho Metadata Registry

### [NEW] Seed script (tham khảo, user tự điều chỉnh theo môi trường)
**Path**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/seed/sftp_reconcile_final_seed.sql`

```sql
-- Seed script: Đăng ký SFTP reconcile_final vào metadata registry
-- Cần chạy MANUAL trên môi trường tương ứng (local/staging/production)
-- Điều chỉnh source_connection_id và shadow_connection_id theo env.

-- 1. Đăng ký Source Object
INSERT INTO cdc_system.source_object_registry (
    source_object_name,
    source_object_type,
    primary_key_field,
    source_connection_id,
    source_database,
    sync_engine,
    is_active
) VALUES (
    'reconcile_final',
    'table',
    'transaction_id',       -- Khóa chính của file CSV final
    1,                      -- TODO: Thay bằng source_connection_id thực tế
    'sftp',                 -- source_database = 'sftp' để match route lookup key
    'custom',               -- sync_engine = 'custom' (V2 schema cho phép)
    true
);

-- 2. Đăng ký Shadow Binding (ánh xạ sang Postgres target)
-- Lưu ý: source_object_id phải lấy từ INSERT ở trên
-- INSERT INTO cdc_system.shadow_binding (...) VALUES (...);
```

---

## Checklist thực thi

1. [ ] Sửa `topic_helper.go` (FIX-B1): thêm bypass cho SFTP topics
2. [ ] Tạo `topic_helper_test.go` (FIX-B2): 2 test cases
3. [ ] Tạo seed script SQL (FIX-B3): tham khảo
4. [ ] Chạy `go test -v ./internal/handler/shadow/...` → PASS
5. [ ] Báo cáo kết quả
