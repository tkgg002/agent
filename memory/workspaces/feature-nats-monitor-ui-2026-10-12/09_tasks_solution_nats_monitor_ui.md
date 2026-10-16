# Hồ sơ Giải pháp Kỹ thuật: NATS Monitor UI

## 1. Thiết kế Chi tiết Backend (`cdc-cms-service`)

### File mới: `internal/api/system/nats_monitor_handler.go`
```go
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type NATSMonitorHandler struct {
	natsMonitorURL string
	httpClient     *http.Client
	logger         *zap.Logger
}

func NewNATSMonitorHandler(natsMonitorURL string, logger *zap.Logger) *NATSMonitorHandler {
	if natsMonitorURL == "" {
		natsMonitorURL = "http://localhost:18222"
	}
	return &NATSMonitorHandler{
		natsMonitorURL: strings.TrimRight(natsMonitorURL, "/"),
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
		logger: logger,
	}
}

// GetOverview aggregates /varz, /jsz (with full stream & consumer details), and /connz
func (h *NATSMonitorHandler) GetOverview(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	var varz, jsz, connz map[string]any

	errVarz := h.fetchJSON(ctx, "/varz", &varz)
	errJsz := h.fetchJSON(ctx, "/jsz?streams=true&config=true&consumers=true", &jsz)
	errConnz := h.fetchJSON(ctx, "/connz", &connz)

	if errVarz != nil && errJsz != nil && errConnz != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "down",
			"error":   fmt.Sprintf("cannot reach NATS monitor at %s", h.natsMonitorURL),
			"details": errVarz.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":      "ok",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server":      varz,
		"jetstream":   jsz,
		"connections": connz,
	})
}

func (h *NATSMonitorHandler) fetchJSON(ctx context.Context, endpoint string, target *map[string]any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.natsMonitorURL+endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("nats monitor returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
```

### Sửa đổi: `internal/router/router.go`
1. Bổ sung `NATSMonitor *apisystem.NATSMonitorHandler` vào `HandlerGroup.System`:
```go
	System struct {
		Health        *apisystem.HealthHandler
		SystemHealth  *apisystem.SystemHealthHandler
		NATSMonitor   *apisystem.NATSMonitorHandler
		Introspection *apisystem.IntrospectionHandler
		ActivityLog   *apisystem.ActivityLogHandler
		Alerts        *apisystem.AlertsHandler
	}
```
2. Trong hàm `SetupRoutes`:
```go
	// Public / Basic NATS Monitor endpoint
	app.Get("/api/system/nats/overview", h.System.NATSMonitor.GetOverview)
	app.Get("/api/v1/system/nats/overview", h.System.NATSMonitor.GetOverview)
```
Và trong nhóm `api` (sau JWT auth):
```go
	dual("GET", api, "/system/nats/overview", h.System.NATSMonitor.GetOverview)
```

### Sửa đổi: `internal/server/server.go`
Khởi tạo `NATSMonitorHandler`:
```go
	h.System.NATSMonitor = apisystem.NewNATSMonitorHandler(cfg.System.NatsMonitorURL, logger)
```

---

## 2. Thiết kế Chi tiết Frontend (`cdc-cms-web`)

### File mới: `src/pages/NatsMonitor.tsx`
Cung cấp UI toàn diện:
- Overview KPI cards (Server Uptime, Version, In/Out Throughput, JetStream Message count & Storage).
- Table JetStream Streams với cột: Tên Stream, Subjects, Messages, Storage (KB/MB), Consumers count, Last Seq, First Seq.
- Nút "Xem Consumers" mở Modal/Drawer hiển thị danh sách consumers của stream đó (Ack pending, Redelivered, Waiting).
- Table Active Connections: CID, Client Name / User, IP, Subscriptions, In/Out bytes, Uptime.
- Tự động làm mới (polling 5s, 10s, 30s hoặc Manual Refresh).

### Sửa đổi: `src/App.tsx`
1. Import lazy `const NatsMonitor = lazy(() => import('./pages/NatsMonitor'));`
2. Thêm menu item vào nhóm `operate` hoặc `advanced`:
```tsx
        {
          key: '/nats-monitor',
          icon: <CloudServerOutlined />,
          label: <Link to="/nats-monitor">NATS Monitor</Link>,
        },
```
3. Thêm route trong `<Routes>`:
```tsx
<Route path="/nats-monitor" element={<NatsMonitor />} />
```
