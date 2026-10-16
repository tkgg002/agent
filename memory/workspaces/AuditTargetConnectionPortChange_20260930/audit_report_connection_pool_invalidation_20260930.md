# 🔬 Audit Report: Connection Pool Invalidation Implementation
**Task:** `AuditTargetConnectionPortChange_20260930`  
**Timestamp:** 2026-09-30T13:35:00+07:00  
**Agent:** Gemini (QA Role — Adversarial Self-Review)

---

## 0. Phương pháp Audit

- Đọc lại từng file đã thay đổi, từng dòng code, đối chiếu với plan đã approve
- Kiểm tra tất cả caller/consumer của `GetMasterDB`, `masterDBPool`, `InvalidatePool`
- Phân tích race condition, edge case, missing invalidation
- Kiểm tra hệ thống có cache tầng khác (SchemaAdapter, BatchBuffer adapters) giữ reference pool cũ
- Cross-check NATS subject consistency giữa publisher và subscriber
- Build verify thật (không mock)

---

## 1. Đối chiếu Plan vs Implementation (Từng file)

### 1.1. `master_registry_handler.go` — ✅ PASS

| Hạng mục | Kết quả | Nhận định |
|:--|:--|:--|
| Thêm field `publisher ports.Publisher` | ✅ Đúng plan | Đặt cuối struct, tuân thủ convention Go |
| Cập nhật constructor | ✅ Đúng plan | Param mới cuối cùng, backward compatible nếu truyền `nil` |
| Import `ports` | ✅ Đã có sẵn | Line 8: `"cdc-cms-service/internal/app/ports"` |
| **Phản biện:** Constructor breaking change? | ⚠️ **QUAN TRỌNG** | Thêm param thứ 5 = breaking all callers. Đã kiểm tra: chỉ 2 caller — `server.go:346` (đã fix) và `test:82` (đã fix). **Không còn caller nào khác.** |

### 1.2. `server.go:346` — ✅ PASS

```go
// TRƯỚC:
h.Master.Registry = apimaster.NewMasterRegistryHandler(masterRepo, logger, listMastersH, cmdBus)
// SAU:
h.Master.Registry = apimaster.NewMasterRegistryHandler(masterRepo, logger, listMastersH, cmdBus, natsPublisher)
```

| Hạng mục | Kết quả | Nhận định |
|:--|:--|:--|
| `natsPublisher` đã khai báo? | ✅ Đúng | Line 180: `natsPublisher := messaging.NewNatsPublisher(natsClient.Conn)` |
| `natsPublisher` là `ports.Publisher`? | ✅ Đúng | `NewNatsPublisher` return `ports.Publisher` interface |
| Truyền sai vị trí? | ✅ Không | Param thứ 5, đúng slot |

### 1.3. `master_registry_handler_connections.go:356-362` — ✅ PASS (có 1 finding)

```go
// Notify workers to invalidate cached connection pool for this connection_code
if h.publisher != nil {
    if pubErr := h.publisher.Publish(c.Context(), "connection.config.changed", []byte(existing.ConnectionCode)); pubErr != nil {
        h.logger.Warn("failed to publish connection.config.changed",
            zap.String("connection_code", existing.ConnectionCode), zap.Error(pubErr))
    }
}
```

| Hạng mục | Kết quả | Nhận định |
|:--|:--|:--|
| Nil-safe cho publisher | ✅ | `if h.publisher != nil` — test truyền nil = không crash |
| Dùng `existing.ConnectionCode` | ✅ Đúng | `existing` query từ DB ở line 272, có sẵn ConnectionCode |
| NATS subject string | ✅ Khớp | `"connection.config.changed"` = cùng chuỗi với subscriber |
| Payload format | ✅ Đơn giản | `[]byte(existing.ConnectionCode)` — subscriber parse bằng `string(msg.Data)` |
| Error handling | ✅ Warn, không fail | Publish lỗi chỉ Warn, không chặn response — đúng: NATS signal là best-effort |
| **Finding F-1:** Chỉ publish khi `UpdateConnection`, chưa publish khi `CreateConnection` | ℹ️ Acceptable | `CreateConnection` tạo mới → worker chưa cache pool nào → không cần invalidate |
| **Finding F-2:** Không log `connection_code` trong success path | ℹ️ Acceptable | Không cần log success cho mỗi NATS publish |

### 1.4. `connection_manager.go:157-179` (InvalidatePool) — ✅ PASS (có findings)

```go
func (m *ConnectionManager) InvalidatePool(key string) {
    key = strings.TrimSpace(key)
    if key == "" || key == "default" {
        return
    }
    m.masterPoolMu.Lock()
    defer m.masterPoolMu.Unlock()
    if db, ok := m.masterDBPool[key]; ok && db != nil {
        if sqlDB, err := db.DB(); err == nil {
            _ = sqlDB.Close()
        }
        delete(m.masterDBPool, key)
        if m.logger != nil {
            m.logger.Info("invalidated master connection pool", zap.String("key", key))
        }
    }
}
```

| Hạng mục | Kết quả | Nhận định |
|:--|:--|:--|
| Thread-safe (mutex) | ✅ | Dùng `masterPoolMu.Lock()` — cùng mutex với `GetMasterDB` |
| Trim key | ✅ | `strings.TrimSpace(key)` — phòng thủ |
| Skip `"default"` | ✅ Đúng | Key `"default"` → dùng `Registry.GetDB(RoleDestination)` tĩnh, không nằm trong `masterDBPool` → invalidate vô nghĩa |
| Close pool SQL | ✅ | `sqlDB.Close()` giải phóng tài nguyên TCP |
| Delete from map | ✅ | `delete(m.masterDBPool, key)` → lần gọi `GetMasterDB` tiếp theo sẽ re-query DB |
| Nil logger safe | ✅ | `if m.logger != nil` |
| Pattern matching Close() | ✅ | Cùng pattern với `Close()` ở line 181-193 |

> **Finding F-3 (⚠️ Low-Medium): In-flight queries trên pool bị close đột ngột**
> 
> Khi `InvalidatePool` gọi `sqlDB.Close()`, mọi goroutine đang giữ `*gorm.DB` reference cũ (ví dụ: transmuter đang chạy `transmuteTable` với `masterDB` lấy từ `GetMasterDB` trước đó) sẽ nhận lỗi `sql: database is closed` hoặc `driver: bad connection` ở query tiếp theo.
>
> **Đánh giá Risk: THẤP-TRUNG BÌNH**
> - Hệ thống đã có `isRetryableDBError()` bắt `"conn closed"`, `"driver: bad connection"` → batch_buffer sẽ retry
> - Transmuter cũng có retry loop (`maxRetries`, `isRetryableDBError`)
> - Thao tác edit port thường diễn ra lúc ít tải (admin thao tác), ít khi trùng thời điểm transmute/batch
> - Goroutine retry sẽ gọi lại `GetMasterDB` → map đã bị xóa → re-query connection_registry → pool mới ✅
>
> **Kết luận:** Hành vi này chấp nhận được vì retry mechanism hiện có đã cover.

> **Finding F-4 (ℹ️ Info): `ConnectionOverrides` (config YAML) bypass connection_registry**
> 
> Trong `GetMasterDB` line 79-87: Nếu connection key có trong `cfg.ConnectionOverrides` (hardcode DSN trong config YAML), worker sẽ không query `connection_registry` → edit port trên CMS không có tác dụng cho key đó.
>
> **Đánh giá:** Đây là by design — config override luôn thắng DB. Nhưng cần ghi nhận.

### 1.5. `server_setup.go:222-230` (NATS Subscription) — ✅ PASS

```go
natsClient.Conn.Subscribe("connection.config.changed", func(msg *nats.Msg) {
    connKey := strings.TrimSpace(string(msg.Data))
    if connKey != "" {
        connectionManager.InvalidatePool(connKey)
        logger.Info("connection pool invalidated via NATS signal",
            zap.String("connection_code", connKey))
    }
})
```

| Hạng mục | Kết quả | Nhận định |
|:--|:--|:--|
| Subject khớp publisher | ✅ | `"connection.config.changed"` = cùng chuỗi |
| Parse payload | ✅ | `string(msg.Data)` → `strings.TrimSpace` → phòng thủ |
| Guard empty key | ✅ | `if connKey != ""` |
| `connectionManager` capture | ✅ | Biến `connectionManager` khai báo ở line 146, closure capture đúng |
| `strings` import | ✅ | Line 7 |
| Duplicate subscribe? | ✅ Không | Không có subscription nào khác trên subject này |
| Multi-worker instance | ✅ | NATS default = broadcast → tất cả worker instance đều nhận → tất cả invalidate → đúng |

### 1.6. `master_connection_handler_test.go:82` — ✅ PASS

| Hạng mục | Kết quả |
|:--|:--|
| Thêm nil cho publisher | ✅ Compile pass, nil-safe đã verify |
| Có caller test khác? | ✅ Chỉ 1 caller test duy nhất |

---

## 2. Phân tích sâu: Cache tầng khác bị bỏ sót?

> **Finding F-5 (⚠️ Medium — QUAN TRỌNG NHẤT): `BatchBuffer.adapters` (SchemaAdapter cache) giữ reference `*gorm.DB` cũ**

**Luồng dữ liệu:**
1. `BatchBuffer.resolveDB(record)` → gọi `connMgr.GetMasterDB(key)` → trả pool (cũ hoặc mới)
2. `BatchBuffer.resolveSchemaAdapter(record, db)` → cache adapter vào `bb.adapters` theo key `"master:key"`
3. `SchemaAdapter` bên trong giữ reference `*gorm.DB` truyền vào lúc khởi tạo

**Vấn đề:** Khi `InvalidatePool(key)` close pool cũ → `bb.adapters["master:key"]` vẫn giữ `SchemaAdapter` với `*gorm.DB` cũ (đã closed). Adapter này không bị flush khi connection invalidate.

**Tuy nhiên:** `resolveDB()` mỗi lần gọi đều gọi lại `connMgr.GetMasterDB(key)` để lấy `*gorm.DB` mới nhất → `*gorm.DB` truyền vào `batchUpsert` là pool mới. SchemaAdapter chỉ dùng để introspect column metadata (schema cache) — nếu pool cũ bị closed, introspect sẽ fail tại lần gọi tiếp theo, nhưng `FlushSchemaCache()` sẽ không được gọi tự động.

**Risk:** THẤP — SchemaAdapter chỉ cache column names/types, không cache data. Nếu port thay đổi nhưng schema giữ nguyên (trường hợp phổ biến nhất khi edit port), adapter cũ vẫn valid cho pool mới. Chỉ fail nếu DB mới có schema khác.

**Recommend fix:** Thêm `batchBuffer.FlushSchemaCache()` vào handler NATS `connection.config.changed`.

---

## 3. Build Verification (Thật, không mock)

| Service | Command | Exit Code | Bằng chứng |
|:--|:--|:--|:--|
| `cdc-cms-service` | `go build -o /dev/null ./cmd/...` | **0** ✅ | Chạy thật trên terminal |
| `centralized-data-service` | `go build -o /dev/null ./cmd/worker/...` | **0** ✅ | Chạy thật trên terminal |

**Lưu ý:** Build pass ≠ logic đúng. Build chỉ chứng minh không lỗi syntax/import/type.

---

## 4. Kiểm tra Suy diễn & Hallucination

| Câu hỏi phản biện | Trả lời | Verdict |
|:--|:--|:--|
| Có bịa hàm/file không tồn tại? | Không — `InvalidatePool` là hàm mới tự viết, tất cả hàm khác đã verify tồn tại | ✅ |
| Có báo "test pass" mà chưa chạy test? | Không — chỉ báo "build pass" kèm bằng chứng exit code | ✅ |
| NATS subject có khớp giữa pub/sub? | ✅ Grep verify: đúng 1 chuỗi `"connection.config.changed"` ở 2 file | ✅ |
| `connectionManager` trong closure có đúng biến? | ✅ Line 146 `server_setup.go` | ✅ |
| Có test thật trên live system? | **KHÔNG** — chưa chạy integration test | ⚠️ Ghi nhận |

---

## 5. Tổng hợp Findings & Recommendations

| ID | Mức độ | Mô tả | Action |
|:--|:--|:--|:--|
| **F-1** | ℹ️ Info | `CreateConnection` không publish — acceptable vì pool chưa tồn tại | Không cần fix |
| **F-2** | ℹ️ Info | Không log success NATS publish | Không cần fix |
| **F-3** | ⚠️ Low-Med | In-flight queries nhận `conn closed` khi pool bị invalidate | Accepted — retry mechanism cover |
| **F-4** | ℹ️ Info | `ConnectionOverrides` bypass DB — by design | Ghi nhận cho user |
| **F-5** | ⚠️ Medium | `BatchBuffer.adapters` giữ SchemaAdapter với `*gorm.DB` cũ | **Recommend fix**: Thêm `batchBuffer.FlushSchemaCache()` |

---

## 6. Đề xuất Fix cho F-5

**File:** `centralized-data-service/internal/server/server_setup.go`

```go
// Đề xuất bổ sung FlushSchemaCache:
natsClient.Conn.Subscribe("connection.config.changed", func(msg *nats.Msg) {
    connKey := strings.TrimSpace(string(msg.Data))
    if connKey != "" {
        connectionManager.InvalidatePool(connKey)
        batchBuffer.FlushSchemaCache()  // ← BỔ SUNG
        logger.Info("connection pool invalidated via NATS signal",
            zap.String("connection_code", connKey))
    }
})
```

---

## 7. Self-Improvement Loop

### Lesson phiên này:

**Pattern:** Khi invalidate connection pool [A] trong hệ thống có nhiều tầng cache [B] (ConnectionManager → BatchBuffer adapters → SchemaAdapter), Agent chỉ invalidate tầng chính mà bỏ sót tầng phụ [X] → các tầng phụ giữ reference cũ có thể gây lỗi âm thầm [Y].

**Correct:** Khi invalidate cache ở bất kỳ tầng nào, BẮT BUỘC truy vết toàn bộ consumer/caller của object bị invalidate và kiểm tra xem có cache tầng khác giữ reference cũ không.

### Lessons KHÔNG vi phạm:
- ✅ Không mock test rồi báo pass (#mock-testing-illusion)
- ✅ Không bịa hàm (#hallucinated-function)  
- ✅ Không suy diễn kết quả (#speculative-pass-declaration)
- ✅ Tuân thủ Simplicity First (#simplicity-first)
- ✅ Tuân thủ DNA pattern hệ thống (dùng `ports.Publisher`, cùng pattern NATS subscription)

---

## 8. Verdict tổng thể

| Tiêu chí | Kết quả |
|:--|:--|
| Code đúng plan? | ✅ 100% khớp |
| Build thật? | ✅ Cả 2 service |
| Có suy diễn/hallucination? | ✅ Không |
| Có lỗ hổng bị bỏ sót? | ⚠️ F-5 (SchemaAdapter cache) — recommend fix |
| Tuân thủ architecture? | ✅ |
| Tuân thủ pattern hệ thống? | ✅ |
| Breaking change có kiểm soát? | ✅ — Tất cả callers đã update |
| Cần integration test? | ⚠️ CÓ — chưa chạy test trên live stack |
