# 03_implementation_sftp_internal_worker.md — Thiết kế Kỹ thuật Chi tiết

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## A. Cấu trúc `SFTPPollingWorker`

```go
// internal/handler/shadow/sftp_worker.go
package shadow

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
```

---

## B. Flow `pollOnce(ctx context.Context)`

```
1. Dial SSH → pkg/sftp.NewClient(sshClient)
2. client.ReadDir(cfg.InputPath)
3. Lọc file theo FilePattern regexp
4. Với từng file:
   a. Đọc nội dung: client.Open(path) → io.ReadAll
   b. Parse CSV: csv.NewReader → Read header → loop rows
   c. Mỗi row: build flat JSON map[string]string
   d. Push Kafka: kafkaWriter.WriteMessages(ctx, kafka.Message{Value: rowJSON})
   e. Move file → ProcessedPath (client.Rename)
   f. Nếu lỗi ở bất kỳ bước nào → Move file → ErrorPath, log error
5. Close sftp client + ssh client
```

---

## C. Kafka Producer Pattern

```go
// Dùng segmentio/kafka-go (đã có trong go.mod)
writer := &kafka.Writer{
    Addr:         kafka.TCP(brokers...),
    Topic:        cfg.TopicPrefix,
    Balancer:     &kafka.LeastBytes{},
    WriteTimeout: 10 * time.Second,
}
defer writer.Close()

// Push từng row
err := writer.WriteMessages(ctx, kafka.Message{
    Value: rowJSON, // flat JSON bytes
})
```

---

## D. SSH Connection Pattern (Disable StrictHostKey cho local dev)

```go
sshCfg := &ssh.ClientConfig{
    User: cfg.Username,
    Auth: []ssh.AuthMethod{ssh.Password(cfg.Password)},
    // Local dev: disable strict host key check
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    Timeout:         10 * time.Second,
}
addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
sshClient, err := ssh.Dial("tcp", addr, sshCfg)
```

> ⚠️ `InsecureIgnoreHostKey()` chỉ dùng cho local dev. Production cần `ssh.FixedHostKey()` hoặc `knownhosts`.

---

## E. Graceful Shutdown

```go
func (w *SFTPPollingWorker) Start(ctx context.Context) {
    ctx, w.cancel = context.WithCancel(ctx)
    go func() {
        ticker := time.NewTicker(w.cfg.PollInterval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                w.logger.Info("SFTP polling worker stopped")
                return
            case <-ticker.C:
                if err := w.pollOnce(ctx); err != nil {
                    w.logger.Error("SFTP poll error", zap.Error(err))
                }
            }
        }
    }()
    w.logger.Info("SFTP polling worker started", zap.String("host", w.cfg.Host), zap.Int("port", w.cfg.Port))
}

func (w *SFTPPollingWorker) Stop() {
    if w.cancel != nil {
        w.cancel()
    }
}
```

---

## F. CSV → Kafka Topic Mapping

- Topic name: `cfg.TopicPrefix` (ví dụ: `sftp.reconcile.final`)
- Message key: filename (để partition theo file nếu cần)
- Message value: flat JSON từng row, ví dụ:
  ```json
  {"transaction_id":"TX1001","amount":"150000.00","status":"SUCCESS","partner_code":"MOMO","created_at":"2026-08-11T08:00:00Z"}
  ```
- `EventHandler.HandleRaw()` đã nhận đúng topic prefix `sftp.` → gọi `SFTPEventAdapter.ConvertToCDCEvent()`.

---

## G. Config Integration

```go
// config/config.go — thêm vào AppConfig struct
SFTPWorker SFTPWorkerConfig `mapstructure:"sftpWorker"`
```

```yaml
# config/config-local.yml
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
