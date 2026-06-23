# 04_decisions.md — Architecture Decision Records

> **Rule**: Append-only. Không xóa quyết định cũ.

---

## ADR-001: Package Naming Convention (2026-06-19)

**Quyết định**: Đổi `package service` / `package handler` → đặt tên theo sub-folder chứa nó.

**Ví dụ**:
```go
// internal/service/master/transmuter.go
package master

// internal/service/shadow/schema_adapter.go  
package shadow

// internal/handler/recon/recon_handler.go
package recon
```

**Vấn đề Import Collision**: Khi `handler/master` và `service/master` đều là `package master` → dùng **Named Imports** tại caller:
```go
import (
    handlermaster "github.com/.../internal/handler/master"
    servicemaster "github.com/.../internal/service/master"
)
```

**Lý do**: Chuẩn Go Idiom — tên thư mục và tên package phải đồng nhất. Anti-pattern nếu file `package service` nằm trong `internal/handler/master/`.

---

## ADR-002: Execution Strategy — Strangler Fig Pattern (2026-06-19)

**Quyết định**: TUYỆT ĐỐI thực hiện từng Phase nhỏ (không batch).

**Lý do**: ~200 files + God Object (command_handler.go >3000 dòng) → Big Bang Rewrite có tỷ lệ thất bại ~100%.

**Quy trình mỗi iteration**:
1. Tạo thư mục mới
2. Move nhóm chức năng nhỏ và ít rủi ro nhất (bắt đầu từ `model/system/`, `service/governance/`)
3. Chạy `go mod tidy && go build ./... && go test ./...`
4. `git commit` (ví dụ: `refactor: move model/system to sub-folder structure`)
5. Lặp lại với nhóm phức tạp hơn

**Thứ tự ưu tiên** (ít rủi ro → nhiều rủi ro):
```
model/system/ → model/source/ → model/shadow/ → model/master/
repository/source/ → repository/shadow/ → repository/master/ → repository/recon/
service/governance/ → service/source/ → service/shadow/ → service/master/ → service/recon/
handler/shadow/ → handler/recon/ → handler/source/ → handler/master/ → handler/orchestration/
server/ (DI wiring update)
```

---

## ADR-003: Không có prefix `internal/domain/` (2026-06-19)

**Quyết định**: KHÔNG có thư mục `domain/` nào. Cấu trúc flat là:
```
internal/handler/<subdomain>/
internal/service/<subdomain>/
internal/repository/<subdomain>/
internal/model/<subdomain>/
```

Không phải:
```
internal/domain/<subdomain>/handler/  ← KHÔNG
```

---

## ADR-004: Service/Handler Sub-packages — Deferred (2026-06-19)

**Quyết định**: HOÃN việc tạo sub-packages cho `service/` và `handler/` layers.

**Lý do kỹ thuật** (phát hiện trong quá trình thực hiện):

Go package visibility barrier — các file trong cùng package chia sẻ private functions/types. Khi move ra sub-package:
1. `MetadataRegistry` interface (định nghĩa trong `service/`) bị dependency cycle khi `service/governance/` cần import ngược `service/`
2. `text_sanitizer.go` chứa private helpers (`SanitizeFreeformText`) được ~8 files trong `service/` gọi — export toàn bộ sẽ phình public API
3. Handler files chia sẻ `CommandHandler` struct + private helpers cross-file

**Attempted**: Di chuyển 5 governance files → `service/governance/` → compile failed (MetadataRegistry import cycle).

**Scope ảnh hưởng**: Phase 4 (service governance+source), Phase 5 (service shadow+master+recon), Phase 6 (handler shadow+recon).

**Giải pháp thay thế đã chọn**:
- **Model + Repository**: Sub-packages ✅ (không có private function sharing)
- **Service + Handler**: Giữ flat root package, tách God Objects bằng file-split thay vì package-split

**Điều kiện mở lại**: Khi refactor sâu hơn — extract `MetadataRegistry` interface ra `internal/ports/` package riêng + export shared helpers ra `internal/pkgs/` → unblock sub-package migration.

---

## ADR-005: Handler God Object Split — File-Split Strategy (2026-06-19)

**Quyết định**: Tách `command_handler.go` (3441L) bằng **file-split** (cùng package, cùng struct) thay vì **struct-split** (mỗi domain 1 struct mới).

**Plan gốc** (02_plan.md Phase 7):
```
SyncHandler, SchemaDDLHandler, BatchTransformHandler,
DiscoverHandler, ScanHandler, MongoDiscoverHandler
→ 6 struct mới trong 3 sub-packages
```

**Thực tế** (after Phase 8b wiring):
```
command_handler.go       (506L) — struct + setup + shared helpers
command_handler_ddl.go   (767L) — DDL operations
command_handler_discover.go (899L) — Discover operations
command_handler_scan.go  (836L) — Scan operations
command_handler_transform.go (340L) — Transform operations
command_handler_sync.go  (181L) — Sync operations
→ 5 files mới, cùng handler/ package, cùng CommandHandler struct
```

**Lý do**:
1. Tạo struct mới → cần duplicate DI wiring cho mỗi struct (db, logger, repos, metadata, natsConn...)
2. Shared helpers (sanitizeAdmin*, resolveTarget*, publishResult) được dùng cross-domain → extract ra shared package trước mới tách struct được
3. File-split đã giảm 85% LOC trong file chính — navigation và maintainability đã cải thiện đáng kể

**Trade-off**: Không đạt isolation hoàn toàn giữa domains, nhưng risk-free và ship được ngay.

**Điều kiện nâng cấp lên struct-split**: Sau khi hoàn thành Phase 8 (extract shared helpers) → mỗi domain handler có thể tạo struct riêng mà không cần duplicate code.
