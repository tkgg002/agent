# Thiết kế kỹ thuật chi tiết (Tasks Solution) - Tích hợp SFTP Source Connector

Tài liệu này chi tiết hóa các file cần sửa đổi, bổ sung code cụ thể để tích hợp SFTP Source Connector trên cả `cdc-cms-service` và `cdc-worker`.

---

## 1. cdc-cms-service (Control Plane)

### [MODIFY] [system_connectors_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go)

#### Nhánh parse SFTP Fingerprint:
Cập nhật hàm `parseFingerprint` để nhận diện class SFTP Source Connector và trích xuất các metadata liên quan:
```go
func parseFingerprint(cfg map[string]string) fingerprint {
	fp := fingerprint{topicPrefix: cfg["topic.prefix"]}
	cls := cfg["connector.class"]
	switch {
	case strings.Contains(cls, "MongoDb"):
		fp.sourceType = "mongodb"
		fp.serverAddress = cfg["mongodb.connection.string"]
		fp.dbList = cfg["database.include.list"]
		fp.collectionList = cfg["collection.include.list"]
	case strings.Contains(cls, "MySql"):
		fp.sourceType = "mysql"
		fp.serverAddress = joinHostPort(cfg["database.hostname"], cfg["database.port"])
		fp.dbList = cfg["database.include.list"]
		fp.collectionList = cfg["table.include.list"]
	case strings.Contains(cls, "Postgres"):
		fp.sourceType = "postgresql"
		fp.serverAddress = joinHostPort(cfg["database.hostname"], cfg["database.port"])
		fp.dbList = cfg["database.dbname"]
		fp.collectionList = cfg["table.include.list"]
	case strings.Contains(cls, "SftpSourceConnector") || strings.Contains(cls, "Sftp"):
		fp.sourceType = "sftp"
		fp.serverAddress = joinHostPort(cfg["sftp.host"], cfg["sftp.port"])
		fp.dbList = cfg["input.path"]
		fp.collectionList = cfg["input.file.pattern"]
	default:
		fp.sourceType = "unknown"
	}
	return fp
}
```

#### Trích xuất credentials SFTP:
Cập nhật hàm `extractCredentialsAsOptions` để lưu trữ user/password của SFTP:
```go
func extractCredentialsAsOptions(cfg map[string]string) []byte {
	opts := make(map[string]string)
	// Postgres / Mysql / SQL Server
	if user, ok := cfg["database.user"]; ok {
		opts["username"] = user
	}
	if pass, ok := cfg["database.password"]; ok {
		opts["password"] = pass
	}
	if sslmode, ok := cfg["database.sslmode"]; ok {
		opts["sslmode"] = sslmode
	}
	// MongoDB
	if user, ok := cfg["mongodb.user"]; ok {
		opts["username"] = user
	}
	if pass, ok := cfg["mongodb.password"]; ok {
		opts["password"] = pass
	}
	// SFTP
	if user, ok := cfg["sftp.username"]; ok {
		opts["username"] = user
	}
	if pass, ok := cfg["sftp.password"]; ok {
		opts["password"] = pass
	}
	if len(opts) == 0 {
		return nil
	}
	b, _ := json.Marshal(opts)
	return b
}
```

---

## 2. cdc-worker (Data Plane)

### [NEW] [sftp_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/sftp_adapter.go)

Adapter này sẽ convert cấu trúc message phẳng dạng JSON (từ CSV do SFTP Connector parse) thành `shadow.CDCEvent` tiêu chuẩn:
```go
package shadow

import (
	"encoding/json"
	"fmt"
)

type SFTPEventAdapter struct{}

func NewSFTPEventAdapter() *SFTPEventAdapter {
	return &SFTPEventAdapter{}
}

func (a *SFTPEventAdapter) ConvertToCDCEvent(flatJSON []byte, topic string) (*CDCEvent, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(flatJSON, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse SFTP flat JSON: %w", err)
	}

	event := &CDCEvent{
		SpecVersion: "1.0",
		ID:          "", // Sẽ tự sinh uuid ở handler nếu trống
		Type:        "sftp.reconcile.final",
		Data: CDCEventData{
			Op:    "c",       // Giả lập là phép Create
			After: payload,  // Toàn bộ record phẳng ném vào After
		},
		KafkaTopic: topic,
	}
	return event, nil
}
```

### [MODIFY] [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)

Cập nhật `HandleRaw` để tự động chuyển tiếp event từ SFTP:
```go
// HandleRaw processes a CDC event from any transport (NATS, Kafka, etc.)
func (h *EventHandler) HandleRaw(ctx context.Context, subject string, data []byte) (rows int, err error) {
	start := time.Now()

	// Check if subject is from sftp source
	isSFTP := strings.HasPrefix(subject, "sftp.") || strings.Contains(subject, ".sftp.")

	var event shadow.CDCEvent
	if isSFTP {
		adapter := shadow.NewSFTPEventAdapter()
		cdcEv, errAdapter := adapter.ConvertToCDCEvent(data, subject)
		if errAdapter != nil {
			return 0, fmt.Errorf("SFTP event convert failed: %w", errAdapter)
		}
		event = *cdcEv
	} else {
		if err = json.Unmarshal(data, &event); err != nil {
			return 0, fmt.Errorf("parse CDC event: %w", err)
		}
	}

	// ... phần code cũ phân tách database và table ...
	var db, table string
	// Với SFTP, parse từ subject: sftp.reconcile.final.events ➔ db = "sftp", table = "reconcile_final"
	if isSFTP {
		parts := strings.Split(subject, ".")
		if len(parts) >= 2 {
			db = parts[0]
			table = parts[1]
		}
	} else {
		// code cũ extract db/table từ debezium source
	}
	
	// ...
	rows, err = h.processEvent(ctx, start, &event, subject, db, table)
	return rows, err
}
```
