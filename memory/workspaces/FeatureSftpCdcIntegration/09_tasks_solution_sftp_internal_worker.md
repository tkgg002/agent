# 09_tasks_solution_sftp_internal_worker.md — Hồ sơ Giải pháp Kỹ thuật

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## File 1: `internal/handler/shadow/sftp_worker.go` [NEW]

```go
package shadow

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"time"

	"github.com/pkg/sftp"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type SFTPWorkerConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Host          string        `mapstructure:"host"`
	Port          int           `mapstructure:"port"`
	Username      string        `mapstructure:"username"`
	Password      string        `mapstructure:"password"`
	InputPath     string        `mapstructure:"inputPath"`
	FilePattern   string        `mapstructure:"filePattern"`
	ProcessedPath string        `mapstructure:"processedPath"`
	ErrorPath     string        `mapstructure:"errorPath"`
	TopicPrefix   string        `mapstructure:"topicPrefix"`
	PollInterval  time.Duration `mapstructure:"pollInterval"`
}

type SFTPPollingWorker struct {
	cfg     SFTPWorkerConfig
	brokers []string
	logger  *zap.Logger
	cancel  context.CancelFunc
	pattern *regexp.Regexp
}

func NewSFTPPollingWorker(cfg SFTPWorkerConfig, brokers []string, logger *zap.Logger) *SFTPPollingWorker {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	var pattern *regexp.Regexp
	if cfg.FilePattern != "" {
		pattern, _ = regexp.Compile(cfg.FilePattern)
	}
	return &SFTPPollingWorker{cfg: cfg, brokers: brokers, logger: logger, pattern: pattern}
}

func (w *SFTPPollingWorker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(w.cfg.PollInterval)
		defer ticker.Stop()
		// Poll ngay lần đầu khi start
		if err := w.pollOnce(ctx); err != nil {
			w.logger.Error("sftp_worker: initial poll error", zap.Error(err))
		}
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("sftp_worker: stopped")
				return
			case <-ticker.C:
				if err := w.pollOnce(ctx); err != nil {
					w.logger.Error("sftp_worker: poll error", zap.Error(err))
				}
			}
		}
	}()
	w.logger.Info("sftp_worker: started",
		zap.String("host", w.cfg.Host),
		zap.Int("port", w.cfg.Port),
		zap.String("input_path", w.cfg.InputPath),
		zap.Duration("poll_interval", w.cfg.PollInterval),
	)
}

func (w *SFTPPollingWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *SFTPPollingWorker) pollOnce(ctx context.Context) error {
	sshClient, sftpClient, err := w.connect()
	if err != nil {
		return fmt.Errorf("sftp_worker: connect failed: %w", err)
	}
	defer sshClient.Close()
	defer sftpClient.Close()

	entries, err := sftpClient.ReadDir(w.cfg.InputPath)
	if err != nil {
		return fmt.Errorf("sftp_worker: readdir %s: %w", w.cfg.InputPath, err)
	}

	found := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if w.pattern != nil && !w.pattern.MatchString(name) {
			continue
		}
		found++
		filePath := path.Join(w.cfg.InputPath, name)
		if processErr := w.processFile(ctx, sftpClient, filePath, name); processErr != nil {
			w.logger.Error("sftp_worker: process file error",
				zap.String("file", name), zap.Error(processErr))
			_ = sftpClient.Rename(filePath, path.Join(w.cfg.ErrorPath, name))
		}
	}

	if found > 0 {
		w.logger.Info("sftp_worker: poll cycle done", zap.Int("files_found", found))
	}
	return nil
}

func (w *SFTPPollingWorker) connect() (*ssh.Client, *sftp.Client, error) {
	sshCfg := &ssh.ClientConfig{
		User:            w.cfg.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(w.cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // local dev only
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", w.cfg.Host, w.cfg.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, nil, err
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, nil, err
	}
	return sshClient, sftpClient, nil
}

func (w *SFTPPollingWorker) processFile(ctx context.Context, client *sftp.Client, filePath, fileName string) error {
	f, err := client.Open(filePath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	rows, err := w.parseCSVRows(f)
	if err != nil {
		return fmt.Errorf("parse csv: %w", err)
	}
	if len(rows) == 0 {
		w.logger.Warn("sftp_worker: CSV file has no data rows", zap.String("file", fileName))
		_ = client.Rename(filePath, path.Join(w.cfg.ProcessedPath, fileName))
		return nil
	}

	if err := w.publishRows(ctx, fileName, rows); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	// Move to processed
	if renErr := client.Rename(filePath, path.Join(w.cfg.ProcessedPath, fileName)); renErr != nil {
		w.logger.Warn("sftp_worker: move to processed failed", zap.String("file", fileName), zap.Error(renErr))
	}
	w.logger.Info("sftp_worker: file processed",
		zap.String("file", fileName), zap.Int("rows", len(rows)))
	return nil
}

func (w *SFTPPollingWorker) parseCSVRows(r io.Reader) ([]map[string]string, error) {
	csvReader := csv.NewReader(r)
	header, err := csvReader.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []map[string]string
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			w.logger.Warn("sftp_worker: skip malformed CSV row", zap.Error(err))
			continue
		}
		row := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(record) {
				row[h] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (w *SFTPPollingWorker) publishRows(ctx context.Context, fileName string, rows []map[string]string) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      w.brokers,
		Topic:        w.cfg.TopicPrefix,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
	})
	defer writer.Close()

	msgs := make([]kafka.Message, 0, len(rows))
	for _, row := range rows {
		b, err := json.Marshal(row)
		if err != nil {
			w.logger.Warn("sftp_worker: marshal row error", zap.Error(err))
			continue
		}
		msgs = append(msgs, kafka.Message{
			Key:   []byte(fileName),
			Value: b,
		})
	}
	if len(msgs) == 0 {
		return nil
	}
	if err := writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}
	w.logger.Info("sftp_worker: pushed rows to Kafka",
		zap.String("topic", w.cfg.TopicPrefix),
		zap.Int("rows", len(msgs)),
		zap.String("file", fileName),
	)
	return nil
}
```

---

## File 2: `config/config.go` [MODIFY] — Thêm SFTPWorkerConfig

Thêm vào struct `AppConfig`:
```go
SFTPWorker SFTPWorkerConfig `mapstructure:"sftpWorker"`
```

Thêm struct `SFTPWorkerConfig` (import từ `handlershadow` hoặc khai báo trực tiếp trong config):
```go
type SFTPWorkerConfig struct {
    Enabled       bool          `mapstructure:"enabled"`
    Host          string        `mapstructure:"host"`
    Port          int           `mapstructure:"port"`
    Username      string        `mapstructure:"username"`
    Password      string        `mapstructure:"password"`
    InputPath     string        `mapstructure:"inputPath"`
    FilePattern   string        `mapstructure:"filePattern"`
    ProcessedPath string        `mapstructure:"processedPath"`
    ErrorPath     string        `mapstructure:"errorPath"`
    TopicPrefix   string        `mapstructure:"topicPrefix"`
    PollInterval  time.Duration `mapstructure:"pollInterval"`
}
```

---

## File 3: `config/config-local.yml` [MODIFY] — Thêm sftpWorker block

```yaml
sftpWorker:
  enabled: true
  host: localhost
  port: 2022
  username: gp-reconcile-admin
  password: sftp_password
  inputPath: /goopay/reconcile_final
  filePattern: "^reconcile_final_.*\\.csv$"
  processedPath: /goopay/reconcile_final/processed
  errorPath: /goopay/reconcile_final/error
  topicPrefix: sftp.reconcile.final
  pollInterval: 30s
```

---

## File 4: `internal/server/server_setup.go` [MODIFY] — Wire SFTPPollingWorker

Thêm sau block kafkaConsumer (dòng ~L474):
```go
if cfg.SFTPWorker.Enabled {
    sftpWorker := handlershadow.NewSFTPPollingWorker(
        handlershadow.SFTPWorkerConfig(cfg.SFTPWorker),
        cfg.Kafka.Brokers,
        logger,
    )
    ws.RegisterOnStart(func() { sftpWorker.Start(context.Background()) })
    ws.RegisterOnStop(func() { sftpWorker.Stop() })
}
```
