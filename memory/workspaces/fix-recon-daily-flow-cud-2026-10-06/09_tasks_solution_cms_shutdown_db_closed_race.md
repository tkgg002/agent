# HỒ SƠ PHÂN TÍCH GỐC RỄ & GIẢI PHÁP: LỖI `sql: database is closed` / `redis: client is closed` KHI START `cdc-cms-service`

- **Ngày lập:** 2026-10-15
- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **File liên quan:**
  1. [`cdc-cms-service/internal/server/server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go) (dòng 393–446)
  2. [`cdc-cms-service/internal/infra/observability/system_health_collector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector.go) (dòng 144–259)

---

## 1. Phân tích Nguyên nhân Gốc rễ (100% Thực chứng)

### 1.1. Nguyên nhân trực tiếp (Tại sao `cdc-cms-service` không start được lúc `15:52:12`)
Kết quả kiểm tra `lsof -nP -iTCP:8082 -iTCP:8083` và `ps aux`:
- **Cổng `:8083` (`cdc-cms-service`):** Hiện đang bị chiếm giữ bởi một tiến trình `go run cmd/server/main.go` cũ chạy từ **`14:30`** trên terminal `ttys021`:
  - `PID 37116` (`go run cmd/server/main.go`) $\rightarrow$ child binary `main` **`PID 37127`** đang giữ `TCP *:8083 (LISTEN)`.
- **Cổng `:8082` (`centralized-data-service` / `cdc-worker`):** Đang chạy tốt với `PID 42649` / `main PID 42655` (khởi động lúc `15:52` trên `ttys001`).

Khi anh chạy thêm lệnh khởi động `cdc-cms-service` lúc `15:52:12`, do cổng `:8083` đã bị `PID 37127` giữ chặt nên `s.app.Listen(":8083")` trả về lỗi `bind: address already in use` và lập tức kích hoạt `srv.Shutdown()`.

---

### 1.2. Nguyên nhân trong Code khiến Terminal văng hàng loạt lỗi `sql: database is closed` và `redis: client is closed`
Tại [`cdc-cms-service/internal/server/server.go` dòng 393–446](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L393-L446):

```go
func (s *Server) Start() error {
	s.logger.Info("CMS Service started", zap.String("port", s.cfg.Server.Port))

	if s.healthCollector != nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.collectorCancel = cancel
		go s.healthCollector.Run(ctx) // <--- (1) Bắn goroutine query DB ngay TRƯỚC khi Listen port!
		s.logger.Info("system health collector started")
	}
	...
	return s.app.Listen(s.cfg.Server.Port) // <--- (2) Bị lỗi bind :8083 ngay lập tức (sau <1ms)
}

func (s *Server) Shutdown() {
	s.logger.Info("shutting down CMS Service...")
	if s.collectorCancel != nil {
		s.collectorCancel() // <--- (3) Chỉ phát tín hiệu cancel, KHÔNG đợi healthCollector dừng!
	}
	...
	s.app.Shutdown()
	s.nats.Close()
	s.redis.Close() // <--- (4) Đóng thẳng Redis khi healthCollector vẫn đang chạy!
	sqlDB, _ := s.db.DB()
	sqlDB.Close()   // <--- (5) Đóng thẳng PostgreSQL khi 4 câu query của healthCollector đang chạy!
	s.logger.Info("CMS Service stopped")
}
```

**Chuỗi sự kiện (Race Condition) diễn ra trong 8ms lúc `15:52:12.692` – `15:52:12.700`:**
1. `go s.healthCollector.Run(ctx)` vừa khởi chạy đã gọi ngay `c.collectAndCache(ctx)` ([`system_health_collector.go:149`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector.go#L149)), bắn 4 goroutine truy vấn PostgreSQL (`system_health_queries.go:106`, `:126`, `:55`, `probes/postgres.go:23`).
2. Cùng lúc đó, `s.app.Listen(":8083")` thất bại do trùng port với `PID 37127` $\rightarrow$ `main.go` gọi `srv.Shutdown()` $\rightarrow$ đóng thẳng `s.redis.Close()` và `sqlDB.Close()`.
3. 4 câu query SQL của `healthCollector` đập vào `sqlDB` vừa bị đóng $\rightarrow$ GORM in ra 4 lỗi `sql: database is closed`.
4. Tiếp đó, tại [`system_health_collector.go:238-256`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector.go#L238-L256), sau khi `g.Wait()` xong, hàm `collectAndCache` **không kiểm tra `if parent.Err() != nil { return parent.Err() }`** mà vẫn chạy tiếp:
   - Gọi `c.evaluateAlerts(parent, snap)` $\rightarrow$ thấy probe Postgres trả `down` (vì DB vừa bị `sqlDB.Close()`) nên cố ghi cảnh báo `InfrastructureDown` vào bảng `cdc_alerts` $\rightarrow$ nổ tiếp lỗi `alert_manager.go:337 sql: database is closed`!
   - Gọi `c.redis.Set(...)` $\rightarrow$ nổ tiếp lỗi `redis SET snapshot failed: redis: client is closed`!

---

## 2. Giải pháp Khắc phục Triệt để

### 2.1. Giải phóng cổng `:8083` bị tiến trình cũ (`PID 37116` / `37127`) chiếm giữ
- Dừng tiến trình `cdc-cms-service` cũ từ lúc `14:30` (`kill 37116 37127`) để giải phóng cổng `:8083`.

### 2.2. Vá lỗi Race Condition trong `cdc-cms-service` (Đề xuất Propose-Only chờ `APPROVE`)
1. **Trong [`system_health_collector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_collector.go#L232-L258):**
   Sau `g.Wait()`, kiểm tra ngay `if parent.Err() != nil { return parent.Err() }` để khi context đã bị hủy lúc shutdown thì thoát ngay, tuyệt đối không gọi `evaluateAlerts` (tránh báo giả `InfrastructureDown`) và không gọi `c.redis.Set` vào Redis đã đóng:
   ```go
   _ = g.Wait()
   if parent.Err() != nil {
       return parent.Err()
   }
   ```
2. **Trong [`cdc-cms-service/internal/server/server.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L393-L446):**
   Bổ sung `bgWg sync.WaitGroup` vào struct `Server` để `Shutdown()` đợi (`s.bgWg.Wait()`) toàn bộ các goroutine nền (`healthCollector`, `auditLogger`, `alertResolver`, `stuckJobReaper`) dừng hoàn toàn **trước khi** gọi `s.redis.Close()` và `sqlDB.Close()`.
