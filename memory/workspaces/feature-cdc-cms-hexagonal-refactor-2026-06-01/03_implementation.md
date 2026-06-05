# 03_implementation.md — Code Demo Go (Reference Implementation)

> Mục đích: minh hoạ chi tiết cho Muscle. KHÔNG phải code production — Muscle khi thực thi phải đọc file thật và adapt.
> 3 demo: (1) Port Split — Master BC, (2) Composition Root pure-func, (3) Refactor 1 command từ raw gorm sang port.

---

## DEMO 1 — Port Split cho Master BC (Phase 1)

### 1.1 Hiện trạng (BEFORE) — God Interface

**File: `internal/app/ports/repository.go` (151 LOC — trích đoạn liên quan Master)**
```go
package ports

import (
    "context"
    "cdc-cms-service/internal/domain/master"
)

// God Interface — chứa quá nhiều method, mọi caller phụ thuộc toàn bộ
type MasterRepo interface {
    // Reader
    List(ctx context.Context, f master.Filter) ([]master.Binding, error)
    GetByName(ctx context.Context, name string) (*master.Binding, error)
    Exists(ctx context.Context, name string) (bool, error)

    // Writer
    Save(ctx context.Context, b *master.Binding) error
    Delete(ctx context.Context, name string) error

    // Approver
    UpdateSchemaStatus(ctx context.Context, name string, status master.SchemaStatus) error
    ApproveSchemaDrift(ctx context.Context, name string, approverID string) error

    // Swapper (đặc thù migration master)
    SwapActive(ctx context.Context, oldName, newName string) error
    LockForSwap(ctx context.Context, name string) error
}
```

**Vấn đề:**
- Handler `list_master.go` chỉ cần `List()` nhưng phải mock cả 9 method khi viết test.
- Thay đổi `SwapActive` signature → tất cả mock break dù không liên quan.
- Vi phạm Interface Segregation Principle.

### 1.2 Sau refactor (AFTER) — Port hẹp ngữ nghĩa

**File mới: `internal/app/ports/master_port.go` (~ 45 LOC)**
```go
package ports

import (
    "context"
    "cdc-cms-service/internal/domain/master"
)

// MasterReader — read-only query side
type MasterReader interface {
    List(ctx context.Context, f master.Filter) ([]master.Binding, error)
    GetByName(ctx context.Context, name string) (*master.Binding, error)
    Exists(ctx context.Context, name string) (bool, error)
}

// MasterWriter — mutate state
type MasterWriter interface {
    Save(ctx context.Context, b *master.Binding) error
    Delete(ctx context.Context, name string) error
}

// MasterApprover — schema drift approval workflow
type MasterApprover interface {
    UpdateSchemaStatus(ctx context.Context, name string, status master.SchemaStatus) error
    ApproveSchemaDrift(ctx context.Context, name string, approverID string) error
}

// MasterSwapper — atomic swap operation (rare, separated due to lock semantics)
type MasterSwapper interface {
    SwapActive(ctx context.Context, oldName, newName string) error
    LockForSwap(ctx context.Context, name string) error
}
```

### 1.3 Caller migrate

**File: `internal/app/queries/list_master.go` (BEFORE)**
```go
type ListMasterHandler struct {
    repo ports.MasterRepo  // ❌ depends on full god interface
}

func (h *ListMasterHandler) Handle(ctx context.Context, q ListMasterQuery) ([]master.Binding, error) {
    return h.repo.List(ctx, q.Filter)
}
```

**File: `internal/app/queries/list_master.go` (AFTER)**
```go
type ListMasterHandler struct {
    reader ports.MasterReader  // ✅ chỉ cần Reader
}

func (h *ListMasterHandler) Handle(ctx context.Context, q ListMasterQuery) ([]master.Binding, error) {
    return h.reader.List(ctx, q.Filter)
}
```

**File: `internal/app/commands/swap_master.go` (AFTER)**
```go
type SwapMasterHandler struct {
    swapper  ports.MasterSwapper
    approver ports.MasterApprover  // cần update status sau swap
    reader   ports.MasterReader    // cần check exists trước swap
}
```

### 1.4 Implementation port (KHÔNG đổi)

**File: `internal/infra/persistence/master_repo_gorm.go` (EXISTING — không sửa)**
```go
type masterRepoGorm struct {
    db *gorm.DB
}

// satisfy MasterReader
func (r *masterRepoGorm) List(ctx context.Context, f master.Filter) ([]master.Binding, error) { ... }
func (r *masterRepoGorm) GetByName(ctx context.Context, name string) (*master.Binding, error) { ... }
func (r *masterRepoGorm) Exists(ctx context.Context, name string) (bool, error) { ... }

// satisfy MasterWriter
func (r *masterRepoGorm) Save(ctx context.Context, b *master.Binding) error { ... }
func (r *masterRepoGorm) Delete(ctx context.Context, name string) error { ... }

// satisfy MasterApprover
func (r *masterRepoGorm) UpdateSchemaStatus(ctx context.Context, name string, s master.SchemaStatus) error { ... }
func (r *masterRepoGorm) ApproveSchemaDrift(ctx context.Context, name string, approverID string) error { ... }

// satisfy MasterSwapper
func (r *masterRepoGorm) SwapActive(ctx context.Context, oldName, newName string) error { ... }
func (r *masterRepoGorm) LockForSwap(ctx context.Context, name string) error { ... }

// 1 struct satisfy 4 interface → KHÔNG cần đổi implementation
func NewMasterRepo(db *gorm.DB) *masterRepoGorm {
    return &masterRepoGorm{db: db}
}
```

### 1.5 Wiring (composition root)

**File: `internal/server/server.go` (AFTER Phase 1, before Phase 2)**
```go
masterRepo := persistence.NewMasterRepo(db)

// Inject từng port hẹp
listHandler := queries.NewListMasterHandler(masterRepo)        // MasterReader
swapHandler := commands.NewSwapMasterHandler(masterRepo, masterRepo, masterRepo) // Swapper+Approver+Reader
```

> 💡 Go duck-typing → 1 struct satisfy nhiều interface tự động, không cần tag hay register.

---

## DEMO 2 — Composition Root Pure-Func (Phase 2)

### 2.1 BEFORE — `server.go` 333 LOC monolithic

**Anti-pattern hiện tại (trích):**
```go
type Server struct {
    cfg    *config.AppConfig
    logger *zap.Logger
    db     *gorm.DB
    nats   *natsconn.NatsClient
    redis  *rediscache.RedisCache
    bus    *bus.CommandBus
    app    *fiber.App
}

func New(cfg *config.AppConfig, logger *zap.Logger) (*Server, error) {
    s := &Server{cfg: cfg, logger: logger}

    if err := s.initDB(); err != nil { return nil, err }      // mutate s.db
    if err := s.initNATS(); err != nil { return nil, err }    // mutate s.nats
    if err := s.initRedis(); err != nil { return nil, err }   // mutate s.redis
    s.initRepos()                                              // mutate (lots of fields hidden)
    s.initBus()                                                // mutate s.bus
    s.registerHandlers()                                       // 100 dòng RegisterSubject/RegisterSync
    s.initRoutes()                                             // mutate s.app
    s.initWorkers()                                            // start goroutines (hidden side effect)

    return s, nil
}
```

**Vấn đề:**
- Receiver-state mutation: khó test sub-step.
- Order matter ngầm: nếu đổi thứ tự gọi → nil pointer panic.
- 333 LOC trong 1 file → khó tìm wiring cụ thể.
- `s.initWorkers()` start goroutine ẩn → khó stop khi test.

### 2.2 AFTER — Pure function, explicit input/output

#### 2.2.1 `internal/server/infra.go` (~ 60 LOC)
```go
package server

import (
    "fmt"

    "cdc-cms-service/internal/config"
    "cdc-cms-service/internal/infra/database"
    "cdc-cms-service/internal/infra/natsconn"
    "cdc-cms-service/internal/infra/rediscache"

    "go.uber.org/zap"
    "gorm.io/gorm"
)

// Infra — bundle các adapter hạ tầng đã connect thành công
type Infra struct {
    DB       *gorm.DB
    ShadowDB *gorm.DB
    NATS     *natsconn.NatsClient
    Redis    *rediscache.RedisCache
    Logger   *zap.Logger
}

// buildInfra — pure function: input = cfg+logger, output = Infra hoặc error
// KHÔNG mutate state ngoài, không start goroutine
func buildInfra(cfg *config.AppConfig, logger *zap.Logger) (*Infra, error) {
    db, err := database.NewPostgresConnection(cfg.DB)
    if err != nil {
        return nil, fmt.Errorf("infra: postgres: %w", err)
    }

    var shadowDB *gorm.DB
    if cfg.ShadowDB.Enabled {
        shadowDB, err = database.NewPostgresConnection(cfg.ShadowDB.Connection)
        if err != nil {
            return nil, fmt.Errorf("infra: shadow db: %w", err)
        }
    }

    nats, err := natsconn.NewClient(cfg.NATS)
    if err != nil {
        return nil, fmt.Errorf("infra: nats: %w", err)
    }

    redis, err := rediscache.New(cfg.Redis)
    if err != nil {
        return nil, fmt.Errorf("infra: redis: %w", err)
    }

    return &Infra{
        DB: db, ShadowDB: shadowDB, NATS: nats, Redis: redis, Logger: logger,
    }, nil
}

// Close — graceful shutdown trong reverse order
func (i *Infra) Close() error {
    var errs []error
    if i.Redis != nil { errs = append(errs, i.Redis.Close()) }
    if i.NATS != nil { i.NATS.Close() }
    if sqlDB, _ := i.DB.DB(); sqlDB != nil { errs = append(errs, sqlDB.Close()) }
    // ... combine errs
    return nil
}
```

#### 2.2.2 `internal/server/repos.go` (~ 50 LOC)
```go
package server

import (
    "cdc-cms-service/internal/app/ports"
    "cdc-cms-service/internal/infra/persistence"
)

// Repos — typed struct (KHÔNG dùng map[string]any)
// Mỗi field là port hẹp được satisfy bởi concrete impl
type Repos struct {
    // Master BC
    MasterReader   ports.MasterReader
    MasterWriter   ports.MasterWriter
    MasterApprover ports.MasterApprover
    MasterSwapper  ports.MasterSwapper

    // Mapping BC
    MappingReader   ports.MappingRuleReader
    MappingWriter   ports.MappingRuleWriter
    MappingApprover ports.MappingRuleApprover

    // Source BC
    SourceReader    ports.SourceReader
    SourceWriter    ports.SourceWriter
    SourceLifecycle ports.SourceLifecycleWriter

    // ... các BC khác
}

// buildRepos — pure: input infra, output Repos
// 1 concrete struct (master_repo_gorm) satisfy nhiều port → đỡ duplicate wiring
func buildRepos(infra *Infra) Repos {
    masterRepo := persistence.NewMasterRepo(infra.DB)
    mappingRepo := persistence.NewMappingRuleRepo(infra.DB)
    sourceRepo := persistence.NewSourceRepo(infra.DB)
    // ... khởi tạo các repo khác

    return Repos{
        MasterReader: masterRepo, MasterWriter: masterRepo,
        MasterApprover: masterRepo, MasterSwapper: masterRepo,

        MappingReader: mappingRepo, MappingWriter: mappingRepo,
        MappingApprover: mappingRepo,

        SourceReader: sourceRepo, SourceWriter: sourceRepo,
        SourceLifecycle: sourceRepo,
    }
}
```

#### 2.2.3 `internal/server/bus.go` (~ 120 LOC)
```go
package server

import (
    "cdc-cms-service/internal/app/commands"
    "cdc-cms-service/internal/app/queries"
    "cdc-cms-service/internal/platform/bus"
)

// buildCommandBus — pure: input repos+infra, output (*bus.CommandBus, error)
func buildCommandBus(infra *Infra, repos Repos) (*bus.CommandBus, error) {
    cb, err := bus.NewCommandBus(infra.NATS, infra.Logger)
    if err != nil {
        return nil, err
    }

    if err := registerSyncHandlers(cb, repos, infra); err != nil {
        return nil, err
    }
    if err := registerAsyncSubjects(cb, repos, infra); err != nil {
        return nil, err
    }
    return cb, nil
}

// registerSyncHandlers — tách khối 100 dòng RegisterSync ra hàm riêng
func registerSyncHandlers(cb *bus.CommandBus, repos Repos, infra *Infra) error {
    // Master BC
    if err := bus.RegisterSync(cb, commands.NewSwapMasterHandler(
        repos.MasterSwapper, repos.MasterApprover, repos.MasterReader,
    )); err != nil { return err }

    // Mapping BC
    if err := bus.RegisterSync(cb, commands.NewApproveMappingHandler(
        repos.MappingApprover, repos.MasterReader,
    )); err != nil { return err }

    // ... 30+ sync handler khác
    return nil
}

// registerAsyncSubjects — NATS subject registration
func registerAsyncSubjects(cb *bus.CommandBus, repos Repos, infra *Infra) error {
    if err := bus.RegisterSubject(cb, "cdc.cms.source.register",
        commands.NewRegisterSourceHandler(repos.SourceWriter, repos.SourceReader),
    ); err != nil { return err }

    // ... 25 subject khác
    return nil
}
```

#### 2.2.4 `internal/server/routes.go` (~ 80 LOC)
```go
package server

import (
    "cdc-cms-service/internal/api"
    "cdc-cms-service/internal/platform/middleware"
    "github.com/gofiber/fiber/v2"
)

// buildRouter — pure: input handlers struct + middleware, output *fiber.App
func buildRouter(h *api.Handlers, mw *middleware.Bundle) *fiber.App {
    app := fiber.New(fiber.Config{
        ErrorHandler: middleware.ErrorHandler,
        ReadTimeout:  cfg.HTTPReadTimeout,
    })

    app.Use(mw.RequestID, mw.Logger, mw.Recover, mw.Metrics)

    v1 := app.Group("/api/v1")
    api.RegisterSourceRoutes(v1, h)
    api.RegisterMasterRoutes(v1, h)
    api.RegisterMappingRoutes(v1, h)
    // ... 8 BC routes
    return app
}
```

#### 2.2.5 `internal/server/workers.go` (~ 70 LOC)
```go
package server

// Worker — interface chung cho mọi background task
type Worker interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Name() string
}

// buildWorkers — pure: input infra+bus, output list of workers (chưa start)
func buildWorkers(infra *Infra, cb *bus.CommandBus) []Worker {
    return []Worker{
        worker.NewReconciliationCron(infra.DB, cb),
        worker.NewSchemaDriftDetector(infra.DB, infra.NATS),
        worker.NewBackfillScheduler(infra.DB),
    }
}
```

#### 2.2.6 `internal/server/server.go` (≤ 80 LOC — orchestrate only)
```go
package server

import (
    "context"
    "fmt"

    "cdc-cms-service/internal/config"
    "go.uber.org/zap"
)

type Server struct {
    infra   *Infra
    app     *fiber.App
    workers []Worker
}

// New — orchestrate tuần tự, mỗi step là pure function
func New(cfg *config.AppConfig, logger *zap.Logger) (*Server, error) {
    infra, err := buildInfra(cfg, logger)
    if err != nil {
        return nil, fmt.Errorf("server: infra: %w", err)
    }

    repos := buildRepos(infra)

    cb, err := buildCommandBus(infra, repos)
    if err != nil {
        return nil, fmt.Errorf("server: bus: %w", err)
    }

    handlers := buildHandlers(repos, cb)
    mw := buildMiddleware(infra)
    app := buildRouter(handlers, mw)
    workers := buildWorkers(infra, cb)

    return &Server{infra: infra, app: app, workers: workers}, nil
}

// Run — start HTTP + workers, block until ctx done
func (s *Server) Run(ctx context.Context) error {
    for _, w := range s.workers {
        if err := w.Start(ctx); err != nil {
            return fmt.Errorf("server: worker %s: %w", w.Name(), err)
        }
    }
    return s.app.ListenWithContext(ctx, cfg.HTTPAddr)
}

// Close — graceful shutdown reverse order
func (s *Server) Close(ctx context.Context) error {
    for _, w := range s.workers {
        _ = w.Stop(ctx)
    }
    _ = s.app.ShutdownWithContext(ctx)
    return s.infra.Close()
}
```

**Kết quả:**
- `server.go` từ 333 → ~ 60 LOC (orchestrate)
- 5 file pure-func độc lập, test riêng dễ
- `cmd/server/main.go` không cần đổi (vẫn gọi `server.New(cfg, logger)`)

---

## DEMO 3 — Refactor 1 Command từ Raw GORM sang Port (Phase 3)

### 3.1 BEFORE — `internal/app/commands/swap_master.go` (raw gorm)
```go
package commands

import (
    "context"
    "fmt"

    "gorm.io/gorm"  // ❌ command import gorm trực tiếp
)

type SwapMasterCommand struct {
    bus.SyncCommandMixin
    OldName string
    NewName string
}

type SwapMasterHandler struct {
    db *gorm.DB  // ❌ depends on raw DB
}

func NewSwapMasterHandler(db *gorm.DB) *SwapMasterHandler {
    return &SwapMasterHandler{db: db}
}

func (h *SwapMasterHandler) Handle(ctx context.Context, cmd SwapMasterCommand) (any, error) {
    // ❌ business logic ăn raw SQL trực tiếp
    return nil, h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // Lock old master
        if err := tx.Exec(
            "SELECT 1 FROM cdc_cms.master_binding WHERE name = ? FOR UPDATE",
            cmd.OldName,
        ).Error; err != nil {
            return fmt.Errorf("lock old: %w", err)
        }

        // Check new exists
        var count int64
        if err := tx.Table("cdc_cms.master_binding").
            Where("name = ?", cmd.NewName).Count(&count).Error; err != nil {
            return err
        }
        if count == 0 {
            return fmt.Errorf("new master %q not found", cmd.NewName)
        }

        // Swap
        if err := tx.Exec(
            "UPDATE cdc_cms.master_binding SET is_active = false WHERE name = ?",
            cmd.OldName,
        ).Error; err != nil {
            return err
        }
        return tx.Exec(
            "UPDATE cdc_cms.master_binding SET is_active = true WHERE name = ?",
            cmd.NewName,
        ).Error
    })
}
```

### 3.2 AFTER — Port hẹp + implementation tách riêng

**File: `internal/app/ports/master_port.go` (thêm method vào port có sẵn)**
```go
type MasterSwapper interface {
    SwapActive(ctx context.Context, oldName, newName string) error
    LockForSwap(ctx context.Context, name string) error
}

type MasterReader interface {
    // ... existing
    Exists(ctx context.Context, name string) (bool, error)  // đã có
}
```

**File: `internal/infra/persistence/master_repo_gorm.go` (wrap SQL thành method)**
```go
// Method mới — wrap nguyên block SQL từ command cũ
func (r *masterRepoGorm) SwapActive(ctx context.Context, oldName, newName string) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Exec(
            "SELECT 1 FROM cdc_cms.master_binding WHERE name = ? FOR UPDATE",
            oldName,
        ).Error; err != nil {
            return fmt.Errorf("lock old: %w", err)
        }
        if err := tx.Exec(
            "UPDATE cdc_cms.master_binding SET is_active = false WHERE name = ?",
            oldName,
        ).Error; err != nil {
            return err
        }
        return tx.Exec(
            "UPDATE cdc_cms.master_binding SET is_active = true WHERE name = ?",
            newName,
        ).Error
    })
}

func (r *masterRepoGorm) LockForSwap(ctx context.Context, name string) error {
    return r.db.WithContext(ctx).Exec(
        "SELECT 1 FROM cdc_cms.master_binding WHERE name = ? FOR UPDATE",
        name,
    ).Error
}
```

**File: `internal/app/commands/swap_master.go` (AFTER — KHÔNG còn gorm)**
```go
package commands

import (
    "context"
    "fmt"

    "cdc-cms-service/internal/app/ports"
    "cdc-cms-service/internal/platform/bus"
)

type SwapMasterCommand struct {
    bus.SyncCommandMixin
    OldName string
    NewName string
}

type SwapMasterHandler struct {
    swapper ports.MasterSwapper
    reader  ports.MasterReader
}

func NewSwapMasterHandler(
    swapper ports.MasterSwapper,
    reader ports.MasterReader,
) *SwapMasterHandler {
    return &SwapMasterHandler{swapper: swapper, reader: reader}
}

func (h *SwapMasterHandler) Handle(ctx context.Context, cmd SwapMasterCommand) (any, error) {
    // Business validation rõ ràng (không lẫn SQL)
    exists, err := h.reader.Exists(ctx, cmd.NewName)
    if err != nil {
        return nil, fmt.Errorf("check new: %w", err)
    }
    if !exists {
        return nil, fmt.Errorf("new master %q not found", cmd.NewName)
    }

    // Delegate atomic operation cho infra
    if err := h.swapper.SwapActive(ctx, cmd.OldName, cmd.NewName); err != nil {
        return nil, fmt.Errorf("swap: %w", err)
    }
    return nil, nil
}
```

### 3.3 Test (Phase 3 DoD)

**File: `test/app/commands/swap_master_test.go`**
```go
package commands_test

import (
    "context"
    "errors"
    "testing"

    "cdc-cms-service/internal/app/commands"
    "github.com/stretchr/testify/assert"
)

type fakeSwapper struct {
    called   bool
    oldName  string
    newName  string
    swapErr  error
}

func (f *fakeSwapper) SwapActive(ctx context.Context, old, new string) error {
    f.called = true
    f.oldName = old
    f.newName = new
    return f.swapErr
}
func (f *fakeSwapper) LockForSwap(ctx context.Context, name string) error { return nil }

type fakeReader struct {
    existsResult bool
    existsErr    error
}

func (f *fakeReader) Exists(ctx context.Context, name string) (bool, error) {
    return f.existsResult, f.existsErr
}
// ... other Reader methods returning zero

func TestSwapMaster_Success(t *testing.T) {
    swapper := &fakeSwapper{}
    reader := &fakeReader{existsResult: true}
    h := commands.NewSwapMasterHandler(swapper, reader)

    _, err := h.Handle(context.Background(), commands.SwapMasterCommand{
        OldName: "v1", NewName: "v2",
    })

    assert.NoError(t, err)
    assert.True(t, swapper.called)
    assert.Equal(t, "v1", swapper.oldName)
    assert.Equal(t, "v2", swapper.newName)
}

func TestSwapMaster_NewNotExists(t *testing.T) {
    h := commands.NewSwapMasterHandler(&fakeSwapper{}, &fakeReader{existsResult: false})
    _, err := h.Handle(context.Background(), commands.SwapMasterCommand{
        OldName: "v1", NewName: "v2",
    })
    assert.ErrorContains(t, err, "not found")
}

func TestSwapMaster_SwapperFails(t *testing.T) {
    swapper := &fakeSwapper{swapErr: errors.New("db conflict")}
    h := commands.NewSwapMasterHandler(swapper, &fakeReader{existsResult: true})
    _, err := h.Handle(context.Background(), commands.SwapMasterCommand{
        OldName: "v1", NewName: "v2",
    })
    assert.ErrorContains(t, err, "db conflict")
}
```

**Lợi ích test BEFORE/AFTER:**
- BEFORE: cần testcontainer Postgres để test command → chậm, flaky.
- AFTER: in-memory fake → < 1ms / test, 100% deterministic.

### 3.4 Wiring update trong `server/bus.go`
```go
// BEFORE
bus.RegisterSync(cb, commands.NewSwapMasterHandler(infra.DB))

// AFTER
bus.RegisterSync(cb, commands.NewSwapMasterHandler(
    repos.MasterSwapper,  // = persistence.masterRepoGorm
    repos.MasterReader,   // = persistence.masterRepoGorm (same struct, different interface)
))
```

---

## DEMO 4 — Linter Config (Phase 0)

### 4.1 `tools/lint/arch-lint.yml`
```yaml
version: 3
workdir: .

allow:
  depOnAnyVendor: false

vendors:
  gorm:
    in: gorm.io/gorm

components:
  domain:
    in: internal/domain/**
  app:
    in: internal/app/**
  api:
    in: internal/api/**
  infra:
    in: internal/infra/**
  server:
    in: internal/server/**
  platform:
    in: internal/platform/**
  bootstrap:
    in: internal/bootstrap/**

deps:
  # Domain — pure, không depend gì ngoài stdlib
  domain:
    mayDependOn: []

  # App — depend domain + ports, KHÔNG infra
  app:
    mayDependOn: [domain]
    cannotDependOn: [infra, api, server, bootstrap]
    vendors: { cannot: [gorm] }  # ❌ command/query không được import gorm

  # API — depend app + domain (cho DTO mapping)
  api:
    mayDependOn: [app, domain, platform]
    cannotDependOn: [infra, server, bootstrap]

  # Infra — implement port, được phép gorm
  infra:
    mayDependOn: [domain, app, platform]
    vendors: { mayDependOn: [gorm] }

  # Server — composition root, depend tất cả
  server:
    mayDependOn: [domain, app, api, infra, platform, bootstrap]

  # Platform — cross-cutting, KHÔNG depend domain
  platform:
    mayDependOn: []
    cannotDependOn: [domain, app, api, infra, server, bootstrap]
```

### 4.2 Phase 4 extension — enforce BC isolation
```yaml
components:
  bc_source:        { in: internal/bc/source/** }
  bc_master:        { in: internal/bc/master/** }
  bc_mapping:       { in: internal/bc/mapping/** }
  bc_transform:     { in: internal/bc/transform/** }
  bc_reconciliation:{ in: internal/bc/reconciliation/** }
  bc_wizard:        { in: internal/bc/wizard/** }
  bc_system_control:{ in: internal/bc/system_control/** }
  bc_observability: { in: internal/bc/observability/** }

deps:
  bc_source:
    mayDependOn: [platform, shared]
    cannotDependOn: [bc_master, bc_mapping, bc_transform, bc_reconciliation,
                     bc_wizard, bc_system_control, bc_observability]

  bc_master:
    mayDependOn: [platform, shared]
    cannotDependOn: [bc_source, bc_mapping, ..., bc_observability]

  # ... mỗi BC tự khép kín

  # Wizard — exception: được cross-BC vì là Saga orchestrator
  bc_wizard:
    mayDependOn: [platform, shared,
                  bc_source, bc_mapping, bc_master, bc_transform]
    cannotDependOn: [bc_reconciliation, bc_system_control, bc_observability]
```

### 4.3 Makefile add
```makefile
arch-lint:
	go-arch-lint check --arch-file tools/lint/arch-lint.yml
```

CI add: `make arch-lint` PASS trước khi merge.

---

## DEMO 5 — Smoke test 53 routes (NFR-2 + AC-9)

### 5.1 `scripts/smoke_routes.sh`
```bash
#!/usr/bin/env bash
set -euo pipefail

BASE=${BASE:-http://localhost:8080}
TOKEN=${TOKEN:?CMS_API_TOKEN required}

declare -a ROUTES=(
  "GET /api/v1/sources"
  "GET /api/v1/sources/:id"
  "POST /api/v1/sources"
  # ... 53 routes từ router.go
)

PASS=0
FAIL=0
for r in "${ROUTES[@]}"; do
  method=$(echo "$r" | awk '{print $1}')
  path=$(echo "$r" | awk '{print $2}' | sed 's/:id/00000000-0000-0000-0000-000000000000/g')
  code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" \
    -H "Authorization: Bearer $TOKEN" "$BASE$path" || echo "000")
  if [[ "$code" =~ ^(2|4)[0-9]{2}$ ]]; then
    echo "PASS $method $path → $code"
    PASS=$((PASS+1))
  else
    echo "FAIL $method $path → $code"
    FAIL=$((FAIL+1))
  fi
done

echo "==== $PASS pass / $FAIL fail / 53 total ===="
[[ $FAIL -eq 0 ]] || exit 1
```

---

## 6. Tổng kết Implementation Notes

| Pattern | Áp dụng phase | Lý do |
|---|---|---|
| Port-per-aggregate (Interface Segregation) | 1 | Caller chỉ depend cái mình dùng |
| Pure function composition root | 2 | Explicit dep, test sub-step dễ |
| 1 struct satisfy N interface (duck typing) | 1, 2 | Go idiom, KHÔNG cần factory phức tạp |
| Repository wrap raw SQL thành method ngữ nghĩa | 3 | Command không lẫn SQL, test = fake interface |
| Saga orchestrator được phép cross-BC | 4 | Wizard điều phối nhiều BC — bản chất phải biết các BC |
| Technical primitives ≠ Shared Kernel | 4 | `internal/shared/` cho naming/pg/id — KHÔNG domain enum |
| Linter enforce thay vì code review | 0, 4 | Convention as code |

---

## 7. KHÔNG được làm (recap rule)

- ❌ KHÔNG đổi public API (route path, NATS subject, DB schema)
- ❌ KHÔNG đổi business logic (Phase 3 chỉ wrap SQL, không sửa rule)
- ❌ KHÔNG dùng DI framework (Wire/Fx) — pure func đủ
- ❌ KHÔNG tạo `internal/domain/shared/` (Shared Kernel — xem ADR-04)
- ❌ KHÔNG sửa migration / seed
- ❌ Brain KHÔNG sửa source code — chỉ document
