# Technical Solution: Lazy Create SFTP Connector On Snapshot (Creation & snapshot_runner_handler.go Demo)

## 1. Scope Restriction
- **Chỉ áp dụng cho:** Nguồn loại `sftp` / `file` (`connector.class` = `com.github.mmolimar.kafka.connect.fs.FsSourceConnector` hoặc tương đương).
- **Tuyệt đối KHÔNG đụng vào:** MongoDB, PostgreSQL, MySQL, SQL Server, Oracle... (Các DB này tiếp tục giữ nguyên 100% luồng khởi tạo Eager Creation hiện tại).
- **File thực thi Snapshot chuẩn:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go`. KHÔNG đụng vào `TriggerSnapshot` trong `cdc-cms-service`.

---

## 2. PART 1: Demo Luồng Tạo Connection (Creation Side - `cdc-cms-service`)

Trong `cdc-cms-service/internal/app/commands/source/debezium_connector.go`:

```go
func (h *CreateSystemConnectorHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
	cmd, ok := c.(CreateSystemConnectorCommand)
	if !ok {
		return nil, errors.New("system-connector.create: command type mismatch")
	}

	cls := cmd.Config["connector.class"]
	sourceType := ""
	if cmd.Fingerprint != nil {
		sourceType = cmd.Fingerprint.SourceType
	}
	isSFTP := sourceType == "sftp" || strings.Contains(cls, "Sftp") ||
		strings.Contains(cls, "FsSourceConnector") || strings.Contains(cls, "kafka.connect.fs") ||
		strings.Contains(cmd.Config["topic"], "cdc.sftplocal")

	// =========================================================================
	// NHÁNH 1: DÀNH RIÊNG CHO SFTP -> LAZY CREATION FLOW
	// =========================================================================
	if isSFTP {
		/*
		   // LEGACY SFTP EAGER CREATION & PAUSE FLOW - PRESERVED FOR REFERENCE
		   if h.writer == nil {
		   		return nil, errors.New("kafka connect client not ready")
		   }
		   r, err := h.writer.Create(ctx, cmd.Name, cmd.Config)
		   if err != nil { return nil, err }
		   if errPause := h.writer.Lifecycle(ctx, cmd.Name, "pause"); errPause != nil {
		   		h.logger.Warn("sftp connector pause failed", zap.Error(errPause))
		   }
		*/

		// Chỉ lưu Fingerprint & Cấu hình SFTP vào Database với Status = "configured"
		if h.sourceRepo != nil && cmd.Fingerprint != nil {
			cmd.Fingerprint.Status = "configured"
			if err := h.sourceRepo.Upsert(ctx, cmd.Fingerprint); err != nil {
				h.logger.Error("failed to persist sftp source fingerprint in DB", zap.String("connector", cmd.Name), zap.Error(err))
				return nil, fmt.Errorf("persist_sftp_fingerprint_failed: %w", err)
			}
		}

		resp := map[string]any{
			"name":    cmd.Name,
			"status":  "configured",
			"message": "SFTP Connection configuration saved in DB. Connector will be created dynamically on Kafka Connect upon snapshot.",
		}
		body, _ := json.Marshal(resp)
		return body, nil
	}

	// =========================================================================
	// NHÁNH 2: TẤT CẢ CÁC DATABASE TYPES KHÁC (MongoDB, Postgres, MySQL...) 
	// -> GIỮ NGUYÊN 100% LUỒNG TẠO CONNECTOR CŨ (Eager Flow)
	// =========================================================================
	if h.writer == nil {
		return nil, errors.New("kafka connect client not ready")
	}

	var resp map[string]any
	runner := saga.New("connector.create", h.logger,
		saga.Step{
			Name: "http-create-connector",
			Execute: func(ctx context.Context) error {
				r, err := h.writer.Create(ctx, cmd.Name, cmd.Config)
				if err != nil {
					return err
				}
				resp = r
				return nil
			},
			Compensate: func(ctx context.Context) error {
				return h.writer.Delete(ctx, cmd.Name)
			},
		},
		saga.Step{
			Name: "db-upsert-fingerprint",
			Execute: func(ctx context.Context) error {
				if h.sourceRepo == nil || cmd.Fingerprint == nil {
					return nil
				}
				return h.sourceRepo.Upsert(ctx, cmd.Fingerprint)
			},
			Compensate: func(ctx context.Context) error {
				if h.sourceRepo == nil {
					return nil
				}
				return h.sourceRepo.FullCleanup(ctx, cmd.Name)
			},
		},
	)
	if err := runner.Run(ctx); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(resp)
	return body, nil
}
```

---

## 3. PART 2: Demo Luồng Click Snapshot Trong `snapshot_runner_handler.go` (`centralized-data-service`)

Trong file `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go` (tại dòng 198):

```go
	if isSFTP {
		// A. Resolve connection code
		var connRow struct {
			ConnectionCode string `gorm:"column:connection_code"`
		}
		err = r.db.Raw("SELECT connection_code FROM cdc_system.connection_registry WHERE id = ?", so.SourceConnectionID).Scan(&connRow).Error
		if err != nil {
			return fmt.Errorf("failed to query connection registry: %w", err)
		}

		if connRow.ConnectionCode == "" {
			r.logger.Warn("sftp connection code not found for source object", zap.Int64("id", so.ID))
			return fmt.Errorf("connection code not found for source object")
		}
		connectionCode = connRow.ConnectionCode
		targetTable = so.ObjectCode

		// B. Resolve connector config from cdc_system.sources
		var rawConfig map[string]string
		var rawConfigSanitized string
		err = r.db.Raw("SELECT raw_config_sanitized::text FROM cdc_system.sources WHERE connector_name = ?", connRow.ConnectionCode).Scan(&rawConfigSanitized).Error
		if err != nil {
			return fmt.Errorf("failed to query sources configuration: %w", err)
		}

		if rawConfigSanitized != "" {
			_ = json.Unmarshal([]byte(rawConfigSanitized), &rawConfig)
		}

		topic := ""
		if len(rawConfig) > 0 {
			topic = rawConfig["topic"]
			bootstrap := rawConfig["signal.kafka.bootstrap.servers"]
			if bootstrap == "" && len(r.kafkaBrokers) > 0 {
				bootstrap = strings.Join(r.kafkaBrokers, ",")
			}
			if topic != "" {
				autoCreateKafkaTopic(ctx, bootstrap, topic, r.logger)
			}
		}

		// =========================================================================
		// LAZY SFTP CONNECTOR CREATION ON SNAPSHOT RUNNER
		// =========================================================================
		if connectionCode != "" {
			baseURL := strings.TrimRight(r.kafkaConnectURL, "/")
			if baseURL == "" {
				baseURL = "http://localhost:8084"
			}

			// 1. Kiểm tra trạng thái connector trên Kafka Connect REST API
			statusURL := baseURL + "/connectors/" + connectionCode + "/status"
			var statusResp struct {
				Name string `json:"name"`
			}
			errStatus := kafkaConnectDo(ctx, "GET", statusURL, nil, &statusResp)

			if errStatus != nil || statusResp.Name != connectionCode {
				// CONNECTOR CHƯA TỒN TẠI (Đang ở dạng Lazy "configured" từ DB):
				// Thực hiện POST /connectors khởi tạo Connector trên Kafka Connect ngay lúc này!
				r.logger.Info("sftp snapshot: connector does not exist on Kafka Connect, creating lazily from DB config...",
					zap.String("connector", connectionCode))

				createReq := map[string]interface{}{
					"name":   connectionCode,
					"config": rawConfig,
				}
				createBody, _ := json.Marshal(createReq)
				if errCreate := kafkaConnectDo(ctx, "POST", baseURL+"/connectors", bytes.NewReader(createBody), nil); errCreate != nil {
					r.logger.Error("sftp snapshot: lazy create connector failed",
						zap.String("connector", connectionCode),
						zap.Error(errCreate))
					return fmt.Errorf("lazy_create_sftp_connector_failed: %w", errCreate)
				}
				r.logger.Info("sftp snapshot: connector created dynamically on Kafka Connect!",
					zap.String("connector", connectionCode))

				// Cập nhật trạng thái status trong DB cdc_system.sources thành 'active'
				_ = r.db.Exec("UPDATE cdc_system.sources SET status = 'active' WHERE connector_name = ?", connectionCode).Error

			} else {
				// CONNECTOR ĐÃ TỒN TẠI: Giữ lại code cũ (comment preserving) hoặc restart task
				/*
				   // LEGACY RESUME/RESTART FLOW - PRESERVED FOR REFERENCE
				   if errResume := kafkaConnectDo(ctx, "PUT", baseURL+"/connectors/"+connectionCode+"/resume", nil, nil); errResume != nil {
				   		r.logger.Warn("sftp snapshot: resume connector failed", zap.Error(errResume))
				   }
				*/
				// Restart task để kích hoạt đọc file CSV mới nếu cần
				_ = kafkaConnectDo(ctx, "POST", baseURL+"/connectors/"+connectionCode+"/restart", nil, nil)
			}
		}

		if topic == "" {
			objName := so.SourceObjectName
			if objName == "" {
				objName = targetTable
			}
			topic = fmt.Sprintf("cdc.sftplocal.%s.%s", connectionCode, objName)
		}

		var bindingArg any
		if p.ShadowBindingID > 0 {
			bindingArg = p.ShadowBindingID
		}

		err = r.db.Exec(`
			INSERT INTO cdc_system.snapshot_progress
				(source_object_id, shadow_binding_id, status, trace_id, started_at, updated_at, finished_at, rows_processed)
			VALUES (?, ?, 'done', ?, NOW(), NOW(), NOW(), 0)
		`, p.SourceObjectID, bindingArg, p.TraceID).Error
		if err != nil {
			r.logger.Error("failed to insert sftp snapshot progress into database", zap.Error(err))
		}

		return nil
	}
```
