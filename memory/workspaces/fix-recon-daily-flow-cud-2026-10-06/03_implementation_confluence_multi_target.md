# Thiết Kế Chi Tiết: Bảng Cấu Hình Multi-Target Confluence Report & Pipeline Whitelist (Dynamic Date Column)

- **Ngày khởi tạo:** 2026-10-12 (Cập nhật 16:45)
- **Tác giả:** Brain (Architect)

---

## 1. Cấu Trúc Dữ Liệu `pipelines` (JSONB) Kèm `date_column` Động

Mỗi pipeline được cấu hình rõ ràng:
- `shadow_schema`, `shadow_table`, `master_table`
- `date_column`: Tên cột ngày tạo riêng biệt của bảng (`createdAt`, `created_at`, `create_time`, `trans_time`...).

```json
[
  {
    "shadow_schema": "goopay",
    "shadow_table": "trans_his",
    "master_table": "trans_his",
    "source_database": "goopay_prod",
    "date_column": "createdAt",
    "display_name": "Core Proxy Transactions"
  },
  {
    "shadow_schema": "goopay",
    "shadow_table": "payment_bills",
    "master_table": "payment_bills",
    "source_database": "goopay_prod",
    "date_column": "created_at",
    "display_name": "Core Bills Transaction"
  },
  {
    "shadow_schema": "goopay",
    "shadow_table": "orders",
    "master_table": "orders",
    "source_database": "goopay_prod",
    "date_column": "create_time",
    "display_name": "Order Service"
  }
]
```

---

## 2. Go Struct

```go
package reconmodel

import (
	"encoding/json"
	"time"
)

type PipelineTarget struct {
	ShadowSchema   string `json:"shadow_schema"`
	ShadowTable    string `json:"shadow_table"`
	MasterTable    string `json:"master_table"`
	SourceDatabase string `json:"source_database,omitempty"`
	DateColumn     string `json:"date_column"` // Tên cột ngày tạo riêng: createdAt, created_at, create_time...
	DisplayName    string `json:"display_name,omitempty"`
}

type ConfluenceReportConfig struct {
	ID            int64           `gorm:"primaryKey" json:"id"`
	Name          string          `gorm:"column:name;unique;not null" json:"name"`
	DisplayTitle  string          `gorm:"column:display_title;not null" json:"display_title"`
	TitleTemplate string          `gorm:"column:title_template;not null" json:"title_template"`
	BaseURL       string          `gorm:"column:base_url;not null" json:"base_url"`
	SpaceKey      string          `gorm:"column:space_key;not null" json:"space_key"`
	ParentPageID  string          `gorm:"column:parent_page_id;not null" json:"parent_page_id"`
	UserEmail     string          `gorm:"column:user_email;not null" json:"user_email"`
	APIToken      string          `gorm:"column:api_token;not null" json:"api_token"`
	Pipelines     json.RawMessage `gorm:"column:pipelines;type:jsonb" json:"pipelines"`
	CronExpr      string          `gorm:"column:cron_expr;not null" json:"cron_expr"`
	IsEnabled     bool            `gorm:"column:is_enabled;default:true" json:"is_enabled"`
	LastRunAt     *time.Time      `gorm:"column:last_run_at" json:"last_run_at"`
	LastStatus    string          `gorm:"column:last_status" json:"last_status"`
	LastPageURL   string          `gorm:"column:last_page_url" json:"last_page_url"`
	LastError     string          `gorm:"column:last_error" json:"last_error"`
	CreatedAt     time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at" json:"updated_at"`
}

func (ConfluenceReportConfig) TableName() string {
	return "cdc_system.confluence_report_configs"
}

func (c *ConfluenceReportConfig) ParsePipelines() ([]PipelineTarget, error) {
	if len(c.Pipelines) == 0 {
		return nil, nil
	}
	var targets []PipelineTarget
	err := json.Unmarshal(c.Pipelines, &targets)
	return targets, err
}
```
