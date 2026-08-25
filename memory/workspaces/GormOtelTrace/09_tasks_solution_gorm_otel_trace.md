# Technical Solution: Enable GORM OpenTelemetry Tracing with Query Obfuscation

Tài liệu hướng dẫn chi tiết code thay đổi để tích hợp OpenTelemetry plugin cho GORM cùng tính năng che giấu tham số SQL.

## 1. centralized-data-service/pkgs/database/multi.go
**File**: `pkgs/database/multi.go`

- Thêm import `"gorm.io/plugin/opentelemetry/tracing"`.
- Đăng ký plugin trong `openGorm`:
  ```go
  	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
  		Logger: NewCDCLogger(logLevel),
  	})
  	if err != nil {
  		return nil, fmt.Errorf("gorm open: %w", err)
  	}

  	var tracePlugin gorm.Plugin
  	if role == RoleControlPlane {
  		tracePlugin = tracing.NewPlugin()
  	} else {
  		tracePlugin = tracing.NewPlugin(
  			tracing.WithoutQueryVariables(),
  		)
  	}

  	if err := db.Use(tracePlugin); err != nil {
  		return nil, fmt.Errorf("gorm otel plugin: %w", err)
  	}
  ```

## 2. centralized-data-service/cmd/admin-api/main.go
**File**: `cmd/admin-api/main.go`

- Thêm import `"gorm.io/plugin/opentelemetry/tracing"`.
- Đăng ký plugin:
  ```go
  	dbDSN := getEnvOr("ADMIN_DB_URL", cfg.SystemDB.PgxDSN())
  	db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
  	if err != nil {
  		// Sử dụng hàm min() built-in của Go 1.21+
  		logger.Fatal("open control-plane db", zap.Error(err), zap.String("dsn_prefix", dbDSN[:min(len(dbDSN), 30)]))
  	}

  	if err := db.Use(tracing.NewPlugin()); err != nil {
  		logger.Fatal("gorm otel plugin", zap.Error(err))
  	}
  ```

## 3. cdc-cms-service/pkgs/database/postgres.go
**File**: `pkgs/database/postgres.go`

- Thêm import `"gorm.io/plugin/opentelemetry/tracing"`.
- Cập nhật hàm `NewPostgresConnection` nhận thêm `role string`:
  ```go
  func NewPostgresConnection(dbCfg config.DBConfig, role string) (*gorm.DB, error) {
      ...
  	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
  		Logger:      gormLogger,
  		PrepareStmt: true,
  	})
  	if err != nil {
  		return nil, fmt.Errorf("failed to connect postgres: %w", err)
  	}

  	var tracePlugin gorm.Plugin
  	if role == "cdc" {
  		tracePlugin = tracing.NewPlugin()
  	} else {
  		tracePlugin = tracing.NewPlugin(
  			tracing.WithoutQueryVariables(),
  		)
  	}

  	if err := db.Use(tracePlugin); err != nil {
  		return nil, fmt.Errorf("gorm otel plugin: %w", err)
  	}
  ```

## 4. cdc-cms-service/internal/server/server.go
**File**: `internal/server/server.go`

- Cập nhật các lời gọi `database.NewPostgresConnection`:
  - Dòng 84:
    ```go
    	db, err := database.NewPostgresConnection(cfg.DB, "cdc")
    ```
  - Dòng 96:
    ```go
    		sdb, err := database.NewPostgresConnection(cfg.ShadowDB, "shadow")
    ```
