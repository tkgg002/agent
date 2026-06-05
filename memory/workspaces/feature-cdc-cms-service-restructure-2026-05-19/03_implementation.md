# 03_implementation.md — Code demo module `mapping/`

> **Đây là DEMO trong workspace, KHÔNG phải source code thực tế.**
>
> Mục đích: minh họa pattern Vertical Slice cho module `mapping/` (theo Phase 3 trong `02_plan.md`). Khi Muscle thực thi, dùng làm reference.
>
> Brain CODE PROHIBITION §12: KHÔNG ghi code Go vào `cdc-cms-service/`. Chỉ doc.

## Tổng quan file cần tạo

```
internal/modules/mapping/
├── domain.go          ← Rule entity + behavior (Validate, CanApply, Transition)
├── repo.go            ← GORM struct + Repository + RepoPort interface
├── dto.go             ← Request/response payload + mapper
├── commands.go        ← Create / Update / Delete logic (write)
├── queries.go         ← List / Get logic (read)
├── handler.go         ← Fiber HTTP handlers
├── routes.go          ← RegisterRoutes(app, m)
├── module.go          ← New(deps) *Module constructor
├── handler_test.go    ← Demo unit test
└── domain_test.go     ← Demo domain test (Validate)
```

---

## 1. `domain.go` — entity + behavior (chấm dứt anemic)

```go
// Package mapping is the bounded context of CDC mapping-rule capability.
// Owns Rule entity + write/read logic + HTTP boundary.
package mapping

import (
	"errors"
	"strings"
	"time"
)

// Status is the workflow state of a Rule.
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

// RuleType discriminates how the rule was authored.
type RuleType string

const (
	RuleTypeSystem     RuleType = "system"
	RuleTypeDiscovered RuleType = "discovered"
	RuleTypeMapping    RuleType = "mapping"
)

// Rule is the domain entity (di chuyển từ internal/domain/mapping/rule.go).
type Rule struct {
	ID                 int64
	SourceObjectID     int64
	MasterBindingID    *int64
	SourceDatabase     *string
	SourceSchema       *string
	SourceNamespace    *string
	SourceTable        string
	ShadowSchema       *string
	ShadowTable        *string
	SourceField        string
	SourcePath         *string
	TargetTable        string
	TargetColumn       string
	DataType           string
	SourceFormat       string
	TransformFn        *string
	IsActive           bool
	IsEnriched         bool
	IsNullable         bool
	DefaultValue       *string
	EnrichmentFunction *string
	Status             Status
	RuleType           RuleType
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          *string
	UpdatedBy          *string
	Notes              *string
}

// Filter narrows a List query.
type Filter struct {
	Status         Status
	RuleType       RuleType
	TargetTable    string
	SourceObjectID int64
	SourceDatabase string
	SourceTable    string
	ShadowSchema   string
	ShadowTable    string
	IsActive       *bool
}

// ─── Domain errors ───────────────────────────────────────────────
var (
	ErrRequiredField     = errors.New("mapping: source_field, target_column, data_type required")
	ErrInvalidStatus     = errors.New("mapping: invalid status")
	ErrInvalidTransition = errors.New("mapping: invalid status transition")
	ErrNotFound          = errors.New("mapping: rule not found")
)

// ─── Domain behavior — entity giờ có method (hết anemic) ─────────

// Validate enforces minimum invariant của Rule.
// Gọi mỗi khi nhận input từ command (Create/Update).
func (r *Rule) Validate() error {
	if strings.TrimSpace(r.SourceField) == "" ||
		strings.TrimSpace(r.TargetColumn) == "" ||
		strings.TrimSpace(r.DataType) == "" {
		return ErrRequiredField
	}
	switch r.Status {
	case StatusPending, StatusApproved, StatusRejected, "":
		// ok
	default:
		return ErrInvalidStatus
	}
	return nil
}

// CanTransitionTo trả về true nếu chuyển status hợp lệ.
// pending → approved | rejected. approved → rejected. rejected → pending (re-review).
func (r *Rule) CanTransitionTo(next Status) bool {
	switch r.Status {
	case StatusPending:
		return next == StatusApproved || next == StatusRejected
	case StatusApproved:
		return next == StatusRejected
	case StatusRejected:
		return next == StatusPending
	default:
		return false
	}
}

// Apply sets next status if transition is valid.
func (r *Rule) Apply(next Status, by string, at time.Time) error {
	if !r.CanTransitionTo(next) {
		return ErrInvalidTransition
	}
	r.Status = next
	r.UpdatedBy = &by
	r.UpdatedAt = at
	return nil
}

// IsActiveRule trả về true khi rule đang active và đã approved.
// Helper cho query layer khi cần filter "rule sẵn sàng chạy".
func (r *Rule) IsActiveRule() bool {
	return r.IsActive && r.Status == StatusApproved
}
```

---

## 2. `repo.go` — GORM persistence + RepoPort

```go
package mapping

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ─── GORM struct (di chuyển từ infra/persistence/mapping_rule_repo_gorm.go) ──

// ruleRow là GORM model cho bảng mapping_rule_v2.
// KHÔNG export — chỉ dùng trong package này (cross-module isolation).
type ruleRow struct {
	ID                 int64     `gorm:"primaryKey;column:id"`
	SourceObjectID     int64     `gorm:"column:source_object_id"`
	MasterBindingID    *int64    `gorm:"column:master_binding_id"`
	SourceDatabase     *string   `gorm:"column:source_database"`
	SourceSchema       *string   `gorm:"column:source_schema"`
	SourceNamespace    *string   `gorm:"column:source_namespace"`
	SourceTable        string    `gorm:"column:source_table"`
	ShadowSchema       *string   `gorm:"column:shadow_schema"`
	ShadowTable        *string   `gorm:"column:shadow_table"`
	SourceField        string    `gorm:"column:source_field"`
	SourcePath         *string   `gorm:"column:source_path"`
	TargetTable        string    `gorm:"column:target_table"`
	TargetColumn       string    `gorm:"column:target_column"`
	DataType           string    `gorm:"column:data_type"`
	SourceFormat       string    `gorm:"column:source_format"`
	TransformFn        *string   `gorm:"column:transform_fn"`
	IsActive           bool      `gorm:"column:is_active"`
	IsEnriched         bool      `gorm:"column:is_enriched"`
	IsNullable         bool      `gorm:"column:is_nullable"`
	DefaultValue       *string   `gorm:"column:default_value"`
	EnrichmentFunction *string   `gorm:"column:enrichment_function"`
	Status             string    `gorm:"column:status"`
	RuleType           string    `gorm:"column:rule_type"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
	CreatedBy          *string   `gorm:"column:created_by"`
	UpdatedBy          *string   `gorm:"column:updated_by"`
	Notes              *string   `gorm:"column:notes"`
}

func (ruleRow) TableName() string { return "mapping_rule_v2" }

// toDomain converts persistence row sang domain entity.
func (r ruleRow) toDomain() Rule {
	return Rule{
		ID:                 r.ID,
		SourceObjectID:     r.SourceObjectID,
		MasterBindingID:    r.MasterBindingID,
		SourceDatabase:     r.SourceDatabase,
		SourceSchema:       r.SourceSchema,
		SourceNamespace:    r.SourceNamespace,
		SourceTable:        r.SourceTable,
		ShadowSchema:       r.ShadowSchema,
		ShadowTable:        r.ShadowTable,
		SourceField:        r.SourceField,
		SourcePath:         r.SourcePath,
		TargetTable:        r.TargetTable,
		TargetColumn:       r.TargetColumn,
		DataType:           r.DataType,
		SourceFormat:       r.SourceFormat,
		TransformFn:        r.TransformFn,
		IsActive:           r.IsActive,
		IsEnriched:         r.IsEnriched,
		IsNullable:         r.IsNullable,
		DefaultValue:       r.DefaultValue,
		EnrichmentFunction: r.EnrichmentFunction,
		Status:             Status(r.Status),
		RuleType:           RuleType(r.RuleType),
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
		CreatedBy:          r.CreatedBy,
		UpdatedBy:          r.UpdatedBy,
		Notes:              r.Notes,
	}
}

func fromDomain(d Rule) ruleRow {
	return ruleRow{
		ID:              d.ID,
		SourceObjectID:  d.SourceObjectID,
		MasterBindingID: d.MasterBindingID,
		SourceDatabase:  d.SourceDatabase,
		// ... (đối xứng — bỏ qua cho gọn doc)
		Status:    string(d.Status),
		RuleType:  string(d.RuleType),
		UpdatedAt: d.UpdatedAt,
		UpdatedBy: d.UpdatedBy,
	}
}

// ─── Port interface (internal to module) ──────────────────────────

// RepoPort exposes the persistence operations this module needs.
// Command/Query handlers depend on RepoPort, not *Repository directly,
// để test inject mock dễ.
type RepoPort interface {
	Create(ctx context.Context, r *Rule) error
	GetByID(ctx context.Context, id int64) (Rule, error)
	List(ctx context.Context, f Filter, limit, offset int) ([]Rule, int64, error)
	Update(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, id int64) error
}

// ─── Repository implementation ────────────────────────────────────

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (rp *Repository) Create(ctx context.Context, r *Rule) error {
	row := fromDomain(*r)
	row.CreatedAt = time.Now().UTC()
	row.UpdatedAt = row.CreatedAt
	if err := rp.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	r.ID = row.ID
	r.CreatedAt = row.CreatedAt
	r.UpdatedAt = row.UpdatedAt
	return nil
}

func (rp *Repository) GetByID(ctx context.Context, id int64) (Rule, error) {
	var row ruleRow
	err := rp.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Rule{}, ErrNotFound
	}
	if err != nil {
		return Rule{}, err
	}
	return row.toDomain(), nil
}

func (rp *Repository) List(ctx context.Context, f Filter, limit, offset int) ([]Rule, int64, error) {
	q := rp.db.WithContext(ctx).Model(&ruleRow{})
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	if f.RuleType != "" {
		q = q.Where("rule_type = ?", string(f.RuleType))
	}
	if f.TargetTable != "" {
		q = q.Where("target_table = ?", f.TargetTable)
	}
	if f.SourceObjectID != 0 {
		q = q.Where("source_object_id = ?", f.SourceObjectID)
	}
	if f.IsActive != nil {
		q = q.Where("is_active = ?", *f.IsActive)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []ruleRow
	if err := q.Limit(limit).Offset(offset).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]Rule, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out, total, nil
}

func (rp *Repository) Update(ctx context.Context, r *Rule) error {
	row := fromDomain(*r)
	row.UpdatedAt = time.Now().UTC()
	res := rp.db.WithContext(ctx).Save(&row)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	r.UpdatedAt = row.UpdatedAt
	return nil
}

func (rp *Repository) Delete(ctx context.Context, id int64) error {
	res := rp.db.WithContext(ctx).Where("id = ?", id).Delete(&ruleRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
```

---

## 3. `dto.go` — request/response

```go
package mapping

import (
	"time"
)

// MappingRuleRow là JSON shape công khai (giữ y hệt response cũ để backward-compat).
type MappingRuleRow struct {
	ID              int64   `json:"id"`
	SourceObjectID  int64   `json:"source_object_id"`
	MasterBindingID *int64  `json:"master_binding_id,omitempty"`
	SourceDatabase  *string `json:"source_database,omitempty"`
	SourceSchema    *string `json:"source_schema,omitempty"`
	SourceNamespace *string `json:"source_namespace,omitempty"`
	SourceTable     string  `json:"source_table"`
	ShadowSchema    *string `json:"shadow_schema,omitempty"`
	ShadowTable     *string `json:"shadow_table,omitempty"`
	SourceField     string  `json:"source_field"`
	SourcePath      *string `json:"source_path,omitempty"`
	TargetColumn    string  `json:"target_column"`
	DataType        string  `json:"data_type"`
	SourceFormat    string  `json:"source_format"`
	TransformFn     *string `json:"transform_fn,omitempty"`
	IsNullable      bool    `json:"is_nullable"`
	IsActive        bool    `json:"is_active"`
	Status          string  `json:"status"`
	Notes           *string `json:"notes,omitempty"`
	CreatedBy       *string `json:"created_by,omitempty"`
	UpdatedBy       *string `json:"updated_by,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	RuleType        string  `json:"rule_type"`
	IsEnriched      bool    `json:"is_enriched"`
}

type CreateRequest struct {
	SourceObjectID  *int64  `json:"source_object_id"`
	MasterBindingID *int64  `json:"master_binding_id"`
	SourceDatabase  *string `json:"source_database"`
	SourceSchema    *string `json:"source_schema"`
	SourceNamespace *string `json:"source_namespace"`
	SourceTable     string  `json:"source_table"`
	ShadowSchema    *string `json:"shadow_schema"`
	ShadowTable     *string `json:"shadow_table"`
	SourceField     string  `json:"source_field"`
	SourcePath      *string `json:"source_path"`
	TargetColumn    string  `json:"target_column"`
	DataType        string  `json:"data_type"`
	SourceFormat    string  `json:"source_format"`
	TransformFn     *string `json:"transform_fn"`
	IsNullable      *bool   `json:"is_nullable"`
	IsActive        *bool   `json:"is_active"`
	Status          string  `json:"status"`
	Notes           *string `json:"notes"`
}

type BatchUpdateRequest struct {
	IDs          []uint `json:"ids"`
	Status       string `json:"status"`
	AutoBackfill bool   `json:"auto_backfill"`
}

// formatPgOF reproduces Postgres TO_CHAR YYYY-MM-DD"T"HH24:MI:SSOF.
func formatPgOF(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05") + "+00"
}

// ruleToRow converts domain Rule → public JSON row.
// Unexported — internal to module.
func ruleToRow(r Rule) MappingRuleRow {
	return MappingRuleRow{
		ID:              r.ID,
		SourceObjectID:  r.SourceObjectID,
		MasterBindingID: r.MasterBindingID,
		SourceDatabase:  r.SourceDatabase,
		SourceSchema:    r.SourceSchema,
		SourceNamespace: r.SourceNamespace,
		SourceTable:     r.SourceTable,
		ShadowSchema:    r.ShadowSchema,
		ShadowTable:     r.ShadowTable,
		SourceField:     r.SourceField,
		SourcePath:      r.SourcePath,
		TargetColumn:    r.TargetColumn,
		DataType:        r.DataType,
		SourceFormat:    r.SourceFormat,
		TransformFn:     r.TransformFn,
		IsNullable:      r.IsNullable,
		IsActive:        r.IsActive,
		Status:          string(r.Status),
		Notes:           r.Notes,
		CreatedBy:       r.CreatedBy,
		UpdatedBy:       r.UpdatedBy,
		CreatedAt:       formatPgOF(r.CreatedAt),
		UpdatedAt:       formatPgOF(r.UpdatedAt),
		RuleType:        string(r.RuleType),
		IsEnriched:      r.IsEnriched,
	}
}

func ptrBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
```

---

## 4. `commands.go` — write logic

```go
package mapping

import (
	"context"
	"time"

	"cdc-cms-service/internal/platform/bus"
)

type CommandHandlers struct {
	repo RepoPort
	bus  bus.CommandBus // dùng để dispatch follow-up (vd: restart debezium nếu cần)
}

func NewCommandHandlers(repo RepoPort, b bus.CommandBus) *CommandHandlers {
	return &CommandHandlers{repo: repo, bus: b}
}

// Create persists 1 mapping rule + side-effects (if any).
// Logic "create" tự đảm bảo domain.Validate() pass.
func (h *CommandHandlers) Create(ctx context.Context, req CreateRequest, actor string) (Rule, error) {
	r := Rule{
		SourceObjectID:  ptrInt64(req.SourceObjectID),
		MasterBindingID: req.MasterBindingID,
		SourceDatabase:  req.SourceDatabase,
		SourceSchema:    req.SourceSchema,
		SourceNamespace: req.SourceNamespace,
		SourceTable:     req.SourceTable,
		ShadowSchema:    req.ShadowSchema,
		ShadowTable:     req.ShadowTable,
		SourceField:     req.SourceField,
		SourcePath:      req.SourcePath,
		TargetColumn:    req.TargetColumn,
		DataType:        req.DataType,
		SourceFormat:    req.SourceFormat,
		TransformFn:     req.TransformFn,
		IsNullable:      ptrBool(req.IsNullable, true),
		IsActive:        ptrBool(req.IsActive, true),
		Status:          Status(defaultStatus(req.Status)),
		RuleType:        RuleTypeMapping,
		Notes:           req.Notes,
		CreatedBy:       &actor,
		UpdatedBy:       &actor,
	}
	if err := r.Validate(); err != nil {
		return Rule{}, err
	}
	if err := h.repo.Create(ctx, &r); err != nil {
		return Rule{}, err
	}
	return r, nil
}

// Update applies partial change + status transition.
func (h *CommandHandlers) Update(ctx context.Context, id int64, req CreateRequest, actor string) (Rule, error) {
	r, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return Rule{}, err
	}
	// apply patch
	if req.SourceField != "" {
		r.SourceField = req.SourceField
	}
	if req.TargetColumn != "" {
		r.TargetColumn = req.TargetColumn
	}
	if req.DataType != "" {
		r.DataType = req.DataType
	}
	if req.IsActive != nil {
		r.IsActive = *req.IsActive
	}
	r.UpdatedBy = &actor
	r.UpdatedAt = time.Now().UTC()

	if req.Status != "" && Status(req.Status) != r.Status {
		// Sử dụng domain behavior — KHÔNG sửa status thô.
		if err := r.Apply(Status(req.Status), actor, r.UpdatedAt); err != nil {
			return Rule{}, err
		}
	}
	if err := r.Validate(); err != nil {
		return Rule{}, err
	}
	if err := h.repo.Update(ctx, &r); err != nil {
		return Rule{}, err
	}
	return r, nil
}

// Delete xóa rule theo id (chỉ với rule đã không active).
func (h *CommandHandlers) Delete(ctx context.Context, id int64) error {
	r, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if r.IsActive {
		return ErrInvalidTransition
	}
	return h.repo.Delete(ctx, id)
}

// ─── helpers ───────────────────────────────────────
func ptrInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func defaultStatus(s string) string {
	if s == "" {
		return string(StatusPending)
	}
	return s
}
```

---

## 5. `queries.go` — read logic

```go
package mapping

import "context"

type QueryHandlers struct {
	repo RepoPort
}

func NewQueryHandlers(repo RepoPort) *QueryHandlers {
	return &QueryHandlers{repo: repo}
}

type ListResult struct {
	Items []MappingRuleRow `json:"items"`
	Total int64            `json:"total"`
	Limit int              `json:"limit"`
	Offset int             `json:"offset"`
}

func (q *QueryHandlers) List(ctx context.Context, f Filter, limit, offset int) (ListResult, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rules, total, err := q.repo.List(ctx, f, limit, offset)
	if err != nil {
		return ListResult{}, err
	}
	out := make([]MappingRuleRow, len(rules))
	for i, r := range rules {
		out[i] = ruleToRow(r)
	}
	return ListResult{
		Items: out, Total: total, Limit: limit, Offset: offset,
	}, nil
}

func (q *QueryHandlers) Get(ctx context.Context, id int64) (MappingRuleRow, error) {
	r, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return MappingRuleRow{}, err
	}
	return ruleToRow(r), nil
}
```

---

## 6. `handler.go` — Fiber HTTP boundary

```go
package mapping

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type HTTPHandlers struct {
	cmd *CommandHandlers
	qry *QueryHandlers
}

func NewHTTPHandlers(cmd *CommandHandlers, qry *QueryHandlers) *HTTPHandlers {
	return &HTTPHandlers{cmd: cmd, qry: qry}
}

// @Summary Create a mapping rule
// @Tags    Mapping Rules
// @Router  /api/v1/mapping-rules [post]
func (h *HTTPHandlers) Create(c *fiber.Ctx) error {
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	actor, _ := c.Locals("username").(string)
	r, err := h.cmd.Create(c.UserContext(), req, actor)
	if err != nil {
		return mapErrorStatus(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(ruleToRow(r))
}

func (h *HTTPHandlers) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	actor, _ := c.Locals("username").(string)
	r, err := h.cmd.Update(c.UserContext(), id, req, actor)
	if err != nil {
		return mapErrorStatus(c, err)
	}
	return c.JSON(ruleToRow(r))
}

func (h *HTTPHandlers) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.cmd.Delete(c.UserContext(), id); err != nil {
		return mapErrorStatus(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *HTTPHandlers) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	row, err := h.qry.Get(c.UserContext(), id)
	if err != nil {
		return mapErrorStatus(c, err)
	}
	return c.JSON(row)
}

func (h *HTTPHandlers) List(c *fiber.Ctx) error {
	f := Filter{
		Status:         Status(c.Query("status")),
		RuleType:       RuleType(c.Query("rule_type")),
		TargetTable:    c.Query("target_table"),
		SourceTable:    c.Query("source_table"),
		SourceDatabase: c.Query("source_database"),
		ShadowSchema:   c.Query("shadow_schema"),
		ShadowTable:    c.Query("shadow_table"),
	}
	if v := c.Query("source_object_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.SourceObjectID = id
		}
	}
	if v := c.Query("is_active"); v == "true" {
		t := true
		f.IsActive = &t
	} else if v == "false" {
		fa := false
		f.IsActive = &fa
	}
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	res, err := h.qry.List(c.UserContext(), f, limit, offset)
	if err != nil {
		return mapErrorStatus(c, err)
	}
	return c.JSON(res)
}

func mapErrorStatus(c *fiber.Ctx, err error) error {
	switch err {
	case ErrRequiredField, ErrInvalidStatus, ErrInvalidTransition:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case ErrNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
}
```

---

## 7. `routes.go` — mount

```go
package mapping

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the mapping module subgroup on the Fiber app.
// Called from internal/router/router.go.
func (m *Module) RegisterRoutes(app *fiber.App) {
	g := app.Group("/api/v1/mapping-rules")
	g.Get("/", m.h.List)
	g.Post("/", m.h.Create)
	g.Get("/:id", m.h.Get)
	g.Put("/:id", m.h.Update)
	g.Delete("/:id", m.h.Delete)
}
```

---

## 8. `module.go` — constructor + DI

```go
package mapping

import (
	"cdc-cms-service/internal/platform/bus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps là contract DI khi wire vào server/wire.go.
type Deps struct {
	DB     *gorm.DB
	Bus    bus.CommandBus
	Logger *zap.Logger
}

// Module gói gọn 1 bounded context.
type Module struct {
	repo *Repository
	cmd  *CommandHandlers
	qry  *QueryHandlers
	h    *HTTPHandlers
	log  *zap.Logger
}

// New assembles module — gọi 1 lần ở server/wire.go.
func New(deps Deps) *Module {
	repo := NewRepository(deps.DB)
	cmd := NewCommandHandlers(repo, deps.Bus)
	qry := NewQueryHandlers(repo)
	h := NewHTTPHandlers(cmd, qry)
	return &Module{repo: repo, cmd: cmd, qry: qry, h: h, log: deps.Logger}
}
```

---

## 9. `domain_test.go` — demo unit test domain behavior

```go
package mapping_test

import (
	"testing"
	"time"

	"cdc-cms-service/internal/modules/mapping"
)

func TestRule_Validate_RequiresCoreFields(t *testing.T) {
	r := mapping.Rule{}
	if err := r.Validate(); err != mapping.ErrRequiredField {
		t.Fatalf("expected ErrRequiredField, got %v", err)
	}
}

func TestRule_Validate_AcceptsValid(t *testing.T) {
	r := mapping.Rule{
		SourceField:  "name",
		TargetColumn: "name",
		DataType:     "text",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRule_CanTransitionTo(t *testing.T) {
	cases := []struct {
		from, to mapping.Status
		want     bool
	}{
		{mapping.StatusPending, mapping.StatusApproved, true},
		{mapping.StatusPending, mapping.StatusRejected, true},
		{mapping.StatusApproved, mapping.StatusRejected, true},
		{mapping.StatusApproved, mapping.StatusPending, false},
		{mapping.StatusRejected, mapping.StatusPending, true},
	}
	for _, tc := range cases {
		r := mapping.Rule{Status: tc.from}
		if got := r.CanTransitionTo(tc.to); got != tc.want {
			t.Errorf("%s → %s: want %v, got %v", tc.from, tc.to, tc.want, got)
		}
	}
}

func TestRule_Apply_RejectsInvalidTransition(t *testing.T) {
	r := mapping.Rule{Status: mapping.StatusApproved}
	err := r.Apply(mapping.StatusPending, "alice", time.Now())
	if err != mapping.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}
```

---

## 10. Wiring tại `server/wire.go`

```go
package server

import (
	"cdc-cms-service/internal/modules/mapping"
	"cdc-cms-service/internal/modules/health"
	"cdc-cms-service/internal/modules/registry"
	// ... các module khác
	"cdc-cms-service/internal/platform/bus"
	"cdc-cms-service/internal/platform/db"
	"go.uber.org/zap"
	"github.com/gofiber/fiber/v2"
)

type App struct {
	Fiber  *fiber.App
	Logger *zap.Logger
}

// Wire assembles all modules + platform → 1 App ready to start.
func Wire(cfg Config) (*App, error) {
	logger := initLogger(cfg.LogLevel)
	gormDB, err := db.Open(cfg.DB)
	if err != nil {
		return nil, err
	}
	natsBus, err := bus.NewCommandBus(cfg.NATS, logger)
	if err != nil {
		return nil, err
	}

	app := fiber.New()

	// Module deps
	mappingMod := mapping.New(mapping.Deps{DB: gormDB, Bus: natsBus, Logger: logger})
	healthMod := health.New(health.Deps{DB: gormDB, Logger: logger})
	registryMod := registry.New(registry.Deps{DB: gormDB, Bus: natsBus, Logger: logger})
	// ...

	// Mount routes
	mappingMod.RegisterRoutes(app)
	healthMod.RegisterRoutes(app)
	registryMod.RegisterRoutes(app)
	// ...

	return &App{Fiber: app, Logger: logger}, nil
}
```

---

## 11. Router thin

```go
// internal/router/router.go (≤ 50 LOC)
package router

import (
	"cdc-cms-service/internal/platform/middleware"
	"cdc-cms-service/internal/server"
)

// MountGlobal applies global middleware (JWT, RBAC, recover, CORS).
// Module-specific routes được mount qua module.RegisterRoutes() trong server.Wire.
func MountGlobal(app *server.App, cfg Config) {
	app.Fiber.Use(middleware.Recover(app.Logger))
	app.Fiber.Use(middleware.CORS(cfg.CORS))
	// JWT/RBAC apply theo group trong từng module.RegisterRoutes()
}
```

---

## 12. Tham chiếu thay đổi import path (cheatsheet cho Muscle)

| Old import | New import |
|------------|-----------|
| `cdc-cms-service/internal/domain/mapping` | `cdc-cms-service/internal/modules/mapping` |
| `cdc-cms-service/internal/api/dto` (mapping_rule_dto) | `cdc-cms-service/internal/modules/mapping` |
| `cdc-cms-service/internal/app/commands` (CreateMappingRuleCommand) | `cdc-cms-service/internal/modules/mapping` (CommandHandlers.Create) |
| `cdc-cms-service/internal/app/queries` (ListMappingRulesQuery) | `cdc-cms-service/internal/modules/mapping` (QueryHandlers.List) |
| `cdc-cms-service/internal/infra/persistence` (mapping_rule_repo_gorm) | `cdc-cms-service/internal/modules/mapping` (Repository) |
| `cdc-cms-service/internal/infra/messaging` | `cdc-cms-service/internal/platform/bus` |
| `cdc-cms-service/internal/infra/observability` | `cdc-cms-service/internal/platform/observability` |
| `cdc-cms-service/internal/middleware` | `cdc-cms-service/internal/platform/middleware` |
| `cdc-cms-service/internal/infra/http` | `cdc-cms-service/internal/platform/http` |

---

## Ghi chú

- File `module.go` cố tình **gói gọn export**: bên ngoài chỉ thấy `mapping.New()`, `Module.RegisterRoutes()`. `Repository`/`CommandHandlers` không export — không module khác đụng được.
- Test domain CHẠY ĐƯỢC trong RAM, không cần DB → tốc độ cao.
- Test repo dùng `gorm.io/driver/sqlite` in-memory cho integration light. Không thêm file mới — chỉ minh họa.
- Migration GORM tag KHÔNG đụng — chỉ di chuyển struct. Bảng vẫn là `mapping_rule_v2` (xem `TableName()` ở line 35 repo.go).

**Pattern này lặp lại y hệt cho 9 module còn lại** — chỉ thay tên + payload.
