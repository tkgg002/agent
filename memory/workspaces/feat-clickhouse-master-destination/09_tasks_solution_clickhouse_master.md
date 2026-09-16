# 09 Technical Solutions: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`

---

## 1. Giải pháp TASK-CH-01: Cấu hình Docker Service ClickHouse

Thêm block sau vào `docker/docker-compose.yml`:

```yaml
  clickhouse:
    image: clickhouse/clickhouse-server:24.3-alpine
    container_name: cdc-clickhouse-master
    ports:
      - "8123:8123"   # HTTP Client / Web UI / DBeaver
      - "9000:9000"   # Native TCP Client
    environment:
      CLICKHOUSE_DB: cdc_master_dw
      CLICKHOUSE_USER: default
      CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1
    volumes:
      - clickhouse_data:/var/lib/clickhouse
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
    restart: always

volumes:
  clickhouse_data:
```

---

## 2. Giải pháp TASK-CH-02: Package Client ClickHouse Driver

Tệp: `centralized-data-service/pkgs/clickhouse/client.go`

```go
package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

type Config struct {
	Addr     string `mapstructure:"addr"`     // "localhost:9000"
	Database string `mapstructure:"database"` // "cdc_master_dw"
	Username string `mapstructure:"username"` // "default"
	Password string `mapstructure:"password"` // ""
	Debug    bool   `mapstructure:"debug"`
}

func NewConnection(ctx context.Context, cfg Config, logger *zap.Logger) (driver.Conn, error) {
	opts := &clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		DialTimeout: 5 * time.Second,
		MaxOpenConns: 16,
		MaxIdleConns: 4,
		ConnMaxLifetime: 10 * time.Minute,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := conn.Ping(ctxPing); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	logger.Info("ClickHouse connected",
		zap.String("addr", cfg.Addr),
		zap.String("database", cfg.Database))

	return conn, nil
}
```

---

## 3. Giải pháp TASK-CH-04: ClickHouse DDL Generator Logic

```go
func BuildClickHouseDDL(dbName, tableName string, rules []MappingRule) string {
	var cols []string
	cols = append(cols, "_gpay_id Int64")
	cols = append(cols, "_source_id String")
	cols = append(cols, "_source_ts DateTime64(3, 'UTC')")
	cols = append(cols, "_deleted UInt8 DEFAULT 0")
	cols = append(cols, "_version UInt64")

	for _, r := range rules {
		chType := MapToClickHouseType(r.DataType)
		if r.IsNullable {
			chType = fmt.Sprintf("Nullable(%s)", chType)
		}
		cols = append(cols, fmt.Sprintf("%s %s", r.TargetColumn, chType))
	}

	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s (
    %s
) ENGINE = ReplacingMergeTree(_version, _deleted)
ORDER BY (_gpay_id)
PRIMARY KEY (_gpay_id)
SETTINGS index_granularity = 8192;`,
		dbName, tableName, strings.Join(cols, ",\n    "))
}
```
