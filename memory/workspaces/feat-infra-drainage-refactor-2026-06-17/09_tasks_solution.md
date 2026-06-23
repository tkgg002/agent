# 09_tasks_solution: Technical Solution for DB & NATS Drainage

## 1. Thiết kế Ports & Interfaces (Hexagonal Boundaries)

Để cô lập tầng API và Core Application khỏi thư viện hạ tầng cụ thể (GORM, NATS client), chúng tôi định nghĩa các port interface độc lập:

### Ports cho Messaging (NATS)
Định nghĩa tại `internal/app/ports/publisher.go`:
```go
type ReloadPublisher interface {
	PublishReload(ctx context.Context, table, userID, action, field string) error
}
```

### Mapping GORM Sentinel Errors
Các repository adapter sẽ chặn `gorm.ErrRecordNotFound` và chuyển đổi thành lỗi domain-port tại `internal/app/ports/repository.go`:
```go
var ErrRecordNotFound = errors.New("record not found")
```

## 2. Infrastructure Adapter Concrete Implementations

### NATS Publisher Adapter
Thực thi tại `internal/infra/messaging/nats_publisher.go`:
```go
type natsPublisher struct {
	conn *nats.Conn
}

func NewReloadPublisher(conn *nats.Conn) ports.ReloadPublisher {
	return &natsPublisher{conn: conn}
}

func (p *natsPublisher) PublishReload(ctx context.Context, table, userID, action, field string) error {
	if p.conn == nil {
		return nats.ErrConnectionClosed
	}
	payload, _ := json.Marshal(map[string]string{
		"table":     table,
		"user_id":   userID,
		"action":    action,
		"field":     field,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	return p.conn.Publish("schema.config.reload", payload)
}
```

### GORM Error Mapping
Tại các GORM repository (ví dụ: `bridge_status_repo_gorm.go`, `job_repo_gorm.go`), toàn bộ các lỗi `gorm.ErrRecordNotFound` được chuyển đổi trước khi trả về tầng gọi:
```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, ports.ErrRecordNotFound
}
```

## 3. Dependency Injection Wiring (Composition Root)
Đấu nối các component tại `internal/server/server.go`:
```go
// Khởi tạo adapter
reloadPublisher := messaging.NewReloadPublisher(natsClient.Conn)

// Truyền vào service / handler qua port interface
approvalSvc := persistenceGov.NewApprovalService(db, pendingRepo, schemaLogRepo, reloadPublisher, logger)
h.Shadow.Mapping = apishadow.NewMappingRuleHandler(reloadPublisher, cmdBus, listMappingRulesH, resolveMappingScopeH, mappingRuleRepoV2)
```
