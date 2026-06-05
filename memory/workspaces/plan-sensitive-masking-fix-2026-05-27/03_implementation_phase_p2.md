# 03_implementation_phase_p2 — Admin API + UI + Backfill

## M-10 — API CRUD mask-config

### File NEW: `cdc-cms-service/internal/api/dto/mask_config_dto.go`

```go
package dto

type MaskConfigRequest struct {
    MaskStrategy   string         `json:"mask_strategy" validate:"required,oneof=NONE DROP HASH_HMAC PARTIAL"`
    MaskOptions    map[string]any `json:"mask_options"`
    MaskKeyVersion int16          `json:"mask_key_version" validate:"min=1"`
}

type MaskConfigResponse struct {
    MappingRuleID  int64          `json:"mapping_rule_id"`
    TargetTable    string         `json:"target_table"`
    TargetColumn   string         `json:"target_column"`
    MaskStrategy   string         `json:"mask_strategy"`
    MaskOptions    map[string]any `json:"mask_options"`
    MaskKeyVersion int16          `json:"mask_key_version"`
    UpdatedAt      time.Time      `json:"updated_at"`
}

type MaskConfigAuditResponse struct {
    Items []MaskAuditItem `json:"items"`
    Total int64           `json:"total"`
}

type MaskAuditItem struct {
    Actor        string         `json:"actor"`
    Action       string         `json:"action"`
    OldStrategy  string         `json:"old_strategy"`
    NewStrategy  string         `json:"new_strategy"`
    OldOptions   map[string]any `json:"old_options"`
    NewOptions   map[string]any `json:"new_options"`
    ChangedAt    time.Time      `json:"changed_at"`
}
```

### File NEW: `cdc-cms-service/internal/api/mask_config_handler.go`

```go
package api

import (
    "github.com/gofiber/fiber/v2"
    "cdc-cms-service/internal/app/commands"
    "cdc-cms-service/internal/app/queries"
    "cdc-cms-service/internal/api/dto"
)

type MaskConfigHandler struct {
    getCfg     queries.GetMaskConfigHandler
    listAudit  queries.ListMaskAuditHandler
    updateCfg  commands.UpdateMaskConfigHandler
}

// GET /api/v1/admin/mapping-rules/:id/mask-config
func (h *MaskConfigHandler) Get(c *fiber.Ctx) error {
    id, _ := c.ParamsInt("id")
    resp, err := h.getCfg.Handle(c.Context(), queries.GetMaskConfigQuery{MappingRuleID: int64(id)})
    if err != nil { return err }
    return c.JSON(resp)
}

// PUT /api/v1/admin/mapping-rules/:id/mask-config
func (h *MaskConfigHandler) Update(c *fiber.Ctx) error {
    id, _ := c.ParamsInt("id")
    var req dto.MaskConfigRequest
    if err := c.BodyParser(&req); err != nil { return err }
    if err := validate.Struct(req); err != nil { return err }

    actor := c.Locals("user_email").(string) // từ middleware auth
    err := h.updateCfg.Handle(c.Context(), commands.UpdateMaskConfigCommand{
        MappingRuleID: int64(id),
        Strategy:      req.MaskStrategy,
        Options:       req.MaskOptions,
        KeyVersion:    req.MaskKeyVersion,
        Actor:         actor,
    })
    if err != nil { return err }
    return c.SendStatus(204)
}

// GET /api/v1/admin/mapping-rules/:id/mask-config/audit
func (h *MaskConfigHandler) Audit(c *fiber.Ctx) error {
    id, _ := c.ParamsInt("id")
    page := c.QueryInt("page", 1)
    pageSize := c.QueryInt("page_size", 20)
    resp, err := h.listAudit.Handle(c.Context(), queries.ListMaskAuditQuery{
        MappingRuleID: int64(id), Page: page, PageSize: pageSize,
    })
    if err != nil { return err }
    return c.JSON(resp)
}
```

### File NEW: `cdc-cms-service/internal/app/commands/update_mask_config.go`

```go
package commands

import (
    "context"
    "encoding/json"
    "fmt"

    "gorm.io/gorm"
)

type UpdateMaskConfigCommand struct {
    MappingRuleID int64
    Strategy      string
    Options       map[string]any
    KeyVersion    int16
    Actor         string
}

type UpdateMaskConfigHandler struct {
    db *gorm.DB
}

func (h *UpdateMaskConfigHandler) Handle(ctx context.Context, cmd UpdateMaskConfigCommand) error {
    return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var current struct {
            MaskStrategy string
            MaskOptions  json.RawMessage
        }
        if err := tx.Table("cdc_system.cdc_mapping_rules").
            Select("mask_strategy, mask_options").
            Where("id = ?", cmd.MappingRuleID).
            Scan(&current).Error; err != nil {
            return err
        }
        newOptsBytes, _ := json.Marshal(cmd.Options)

        // Update mapping_rule
        if err := tx.Table("cdc_system.cdc_mapping_rules").
            Where("id = ?", cmd.MappingRuleID).
            Updates(map[string]any{
                "mask_strategy":     cmd.Strategy,
                "mask_options":      newOptsBytes,
                "mask_key_version":  cmd.KeyVersion,
                "updated_at":        gorm.Expr("now()"),
            }).Error; err != nil {
            return err
        }

        // Insert audit
        return tx.Exec(`INSERT INTO cdc_system.mask_config_audit
            (mapping_rule_id, actor, action, old_strategy, new_strategy, old_options, new_options)
            VALUES (?, ?, 'UPDATE', ?, ?, ?, ?)`,
            cmd.MappingRuleID, cmd.Actor,
            current.MaskStrategy, cmd.Strategy,
            current.MaskOptions, newOptsBytes,
        ).Error
    })
}
```

### Router wire (`internal/router/router.go`)
```go
adminGroup := app.Group("/api/v1/admin", middleware.RequireRole("admin"))
mapRules := adminGroup.Group("/mapping-rules/:id")
mapRules.Get("/mask-config", h.MaskConfig.Get)
mapRules.Put("/mask-config", h.MaskConfig.Update)
mapRules.Get("/mask-config/audit", h.MaskConfig.Audit)
```

### Verify
- `curl -X PUT :8080/api/v1/admin/mapping-rules/42/mask-config -H "Authorization: Bearer $ADMIN" -d '{"mask_strategy":"HASH_HMAC","mask_options":{"key_ref":"v1"},"mask_key_version":1}'` → 204.
- `psql -c "SELECT COUNT(*) FROM cdc_system.mask_config_audit WHERE mapping_rule_id=42"` ≥ 1.

---

## M-11 — Admin UI tab Sensitive Masking

### File NEW: `cdc-cms-web/src/types/masking.ts`

```ts
export type MaskStrategy = 'NONE' | 'DROP' | 'HASH_HMAC' | 'PARTIAL';

export interface MaskConfig {
  mappingRuleId: number;
  targetTable: string;
  targetColumn: string;
  maskStrategy: MaskStrategy;
  maskOptions: Record<string, unknown>;
  maskKeyVersion: number;
  updatedAt: string;
}

export interface MaskAuditItem {
  actor: string;
  action: 'CREATE' | 'UPDATE' | 'DELETE';
  oldStrategy: MaskStrategy;
  newStrategy: MaskStrategy;
  oldOptions: Record<string, unknown>;
  newOptions: Record<string, unknown>;
  changedAt: string;
}
```

### File NEW: `cdc-cms-web/src/hooks/useMaskConfig.ts`

```ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { MaskConfig, MaskAuditItem } from '@/types/masking';

export function useMaskConfig(mappingRuleId: number) {
  return useQuery({
    queryKey: ['mask-config', mappingRuleId],
    queryFn: async () => {
      const { data } = await api.get<MaskConfig>(
        `/admin/mapping-rules/${mappingRuleId}/mask-config`,
      );
      return data;
    },
  });
}

export function useUpdateMaskConfig(mappingRuleId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (payload: Partial<MaskConfig>) => {
      await api.put(`/admin/mapping-rules/${mappingRuleId}/mask-config`, payload);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['mask-config', mappingRuleId] }),
  });
}

export function useMaskAudit(mappingRuleId: number, page = 1) {
  return useQuery({
    queryKey: ['mask-config-audit', mappingRuleId, page],
    queryFn: async () => {
      const { data } = await api.get<{ items: MaskAuditItem[]; total: number }>(
        `/admin/mapping-rules/${mappingRuleId}/mask-config/audit?page=${page}`,
      );
      return data;
    },
  });
}
```

### File NEW: `cdc-cms-web/src/components/masking/StrategySelector.tsx`

```tsx
import { Select, Form, Input, InputNumber, Space, Card, Tag, Alert } from 'antd';
import type { MaskStrategy } from '@/types/masking';

const HELP: Record<MaskStrategy, { color: string; legal: string; usage: string }> = {
  NONE:      { color: 'default', legal: 'Cho field không nhạy cảm.', usage: 'trans_id, created_at...' },
  DROP:      { color: 'red',     legal: 'NĐ 356 — Loại bỏ dữ liệu không cần thiết.', usage: 'password, OTP, PIN, CVV' },
  HASH_HMAC: { color: 'blue',    legal: 'Luật 91/2025 Điều "De-identification". Giữ tính đối soát.', usage: 'CCCD, card_number, account_number' },
  PARTIAL:   { color: 'orange',  legal: 'VBHN 25 — Hiển thị một phần cho audit.', usage: 'phone (display), email (display)' },
};

export function StrategySelector({ value, onChange }: { value: MaskStrategy; onChange: (v: MaskStrategy) => void }) {
  const help = HELP[value];
  return (
    <Card title="Chiến lược Masking">
      <Form.Item label="Strategy" required>
        <Select value={value} onChange={onChange} style={{ width: 240 }}>
          <Select.Option value="NONE">NONE — giữ nguyên</Select.Option>
          <Select.Option value="DROP">DROP — set NULL</Select.Option>
          <Select.Option value="HASH_HMAC">HASH_HMAC — băm 1 chiều</Select.Option>
          <Select.Option value="PARTIAL">PARTIAL — mask một phần</Select.Option>
        </Select>
      </Form.Item>
      <Alert
        type="info"
        message={<Space><Tag color={help.color}>{value}</Tag>{help.legal}</Space>}
        description={`Use case: ${help.usage}`}
      />
    </Card>
  );
}
```

### File NEW: `cdc-cms-web/src/components/masking/MaskPreview.tsx`

```tsx
import { Card, Input, Form, Tag } from 'antd';
import { useState } from 'react';
import type { MaskStrategy } from '@/types/masking';

// Client-side preview, không gọi BE. Mục đích: cho admin thấy trước output trước khi save.
export function MaskPreview({ strategy, options }: { strategy: MaskStrategy; options: Record<string, any> }) {
  const [sample, setSample] = useState('');

  function applyClient(): string {
    if (!sample) return '';
    switch (strategy) {
      case 'NONE': return sample;
      case 'DROP': return '(null)';
      case 'HASH_HMAC':
        // Preview hash bằng SubtleCrypto (NOT secret key — chỉ visualize)
        return '<hash 64 chars sẽ tính ở server>';
      case 'PARTIAL': {
        const p = options.prefix ?? 0, s = options.suffix ?? 4, ph = options.placeholder ?? '*';
        if (sample.length <= p + s) return ph.repeat(sample.length);
        return sample.slice(0, p) + ph.repeat(sample.length - p - s) + sample.slice(-s);
      }
    }
  }

  return (
    <Card title="Preview" size="small">
      <Form layout="vertical">
        <Form.Item label="Sample input"><Input value={sample} onChange={e => setSample(e.target.value)} placeholder="Nhập sample để xem output" /></Form.Item>
        <Form.Item label="Output">
          <Tag color="processing" style={{ fontSize: 16, padding: '4px 8px' }}>{applyClient() || '—'}</Tag>
        </Form.Item>
      </Form>
    </Card>
  );
}
```

### File sửa: `cdc-cms-web/src/pages/MappingRuleEditPage.tsx`

```tsx
// Thêm tab "Sensitive Masking" vào Tabs hiện có.
<Tabs>
  <Tabs.TabPane key="basic" tab="Basic">{/* existing */}</Tabs.TabPane>
  <Tabs.TabPane key="transform" tab="Transform">{/* existing */}</Tabs.TabPane>
  <Tabs.TabPane key="masking" tab={<><LockOutlined /> Sensitive Masking</>}>
    <MaskingTab mappingRuleId={id} />
  </Tabs.TabPane>
</Tabs>
```

### Verify FE
- `pnpm lint && pnpm typecheck && pnpm build` PASS.
- Navigate `/mapping-rules/42`, click tab "Sensitive Masking", chọn HASH_HMAC, Save → audit history hiện row mới.

---

## M-12 — Backfill script

### File NEW: `centralized-data-service/scripts/backfill_mask.go`

```go
//go:build backfill

package main

import (
    "context"
    "flag"
    "fmt"
    "log"

    "centralized-data-service/internal/service"
    "centralized-data-service/internal/service/masking"
    "centralized-data-service/pkgs/database"
    "centralized-data-service/pkgs/vault"
)

func main() {
    var (
        dsn       = flag.String("dsn", "", "Postgres DSN shadow")
        table     = flag.String("table", "", "Table to backfill")
        batchSize = flag.Int("batch", 1000, "Batch size")
        dryRun    = flag.Bool("dry-run", true, "Print only, no UPDATE")
    )
    flag.Parse()

    db := database.MustOpen(*dsn)
    svc := service.NewMaskingService(
        buildRegistry(), buildRuleRepo(db), make(chan masking.AuditRecord, 1000),
        log.Default(),
    )

    ctx := context.Background()
    offset := 0
    for {
        rows, err := db.WithContext(ctx).
            Table(*table).
            Select("id, _raw_data").
            Where(`_raw_data::text LIKE '%"***"%'`). // chỉ row còn dấu vết
            Limit(*batchSize).Offset(offset).
            Rows()
        if err != nil { log.Fatal(err) }

        count := 0
        for rows.Next() {
            var id int64
            var raw []byte
            _ = rows.Scan(&id, &raw)

            var data map[string]any
            _ = json.Unmarshal(raw, &data)
            // Re-mask theo strategy mới.
            masked, err := svc.MaskTableData(ctx, fmt.Sprintf("backfill-%d", id), "backfill", *table, data)
            if err != nil { log.Printf("skip id=%d err=%v", id, err); continue }
            newRaw, _ := json.Marshal(masked)

            if *dryRun {
                fmt.Printf("[DRY] id=%d old=%s new=%s\n", id, raw, newRaw)
            } else {
                db.Exec(fmt.Sprintf(`UPDATE %s SET _raw_data = ? WHERE id = ?`, *table), newRaw, id)
            }
            count++
        }
        rows.Close()
        if count < *batchSize { break }
        offset += *batchSize
    }
}
```

### Verify
- Dry-run trên staging: `go run -tags=backfill scripts/backfill_mask.go -dsn=... -table=shadow.users -dry-run=true` → print sample.
- Thực tế: `... -dry-run=false` → assert `SELECT COUNT(*) WHERE _raw_data::text LIKE '%"***"%'` = 0 sau khi xong.

---

## M-13 — Compliance evidence doc

### File NEW: `docs/compliance/sensitive-masking-vn-law.md`

```markdown
# Sensitive Masking — Tuân thủ Pháp lý Việt Nam

## Law mapping

| Văn bản | Điều/Điểm | Yêu cầu | Control trong hệ thống |
|---|---|---|---|
| Luật 91/2025/QH15 | Điều 13 | Tính chính xác + quyền chỉnh sửa của chủ thể | Strategy HASH_HMAC giữ tính đối soát; audit log chứng minh history. |
| Luật 91/2025/QH15 | Điều "De-identification" | Khử định danh giảm rủi ro | Strategy HASH_HMAC + DROP — chuỗi hash không phải dữ liệu cá nhân trực tiếp. |
| NĐ 356/2025/NĐ-CP | Biện pháp kỹ thuật | Mã hóa + kiểm soát truy cập | HMAC-SHA256 + RBAC `RequireRole("admin")` cho API masking config. |
| VBHN 25/VBHN-NHNN | Phân vùng + audit | Lưu trữ + thanh tra | `mask_audit_log` + `mask_config_audit` tables. |

## Strategy decision matrix

| Field type | Strategy | Lý do |
|---|---|---|
| password, OTP, PIN, CVV | DROP | Không có giá trị đối soát ở shadow + giảm bề mặt tấn công. |
| CCCD, card_number, account_number | HASH_HMAC | Cần đếm distinct, đối soát join, nhưng không cần plaintext. |
| phone, email (display) | PARTIAL | Cho phép support team xác nhận đúng KH mà không lộ full. |
| trans_id, created_at | NONE | Không nhạy cảm. |

## Audit trail evidence
- Mỗi config change → `mask_config_audit` (UPDATE/CREATE/DELETE) với actor + diff.
- Mỗi event được mask (sample 1%) → `mask_audit_log` chứng minh control hoạt động.

## Key rotation procedure
1. Generate key mới: `openssl rand -hex 32`.
2. Set env `MASKING_HMAC_KEY_V2`.
3. UPDATE `cdc_mapping_rules SET mask_key_version=2` trên field cần rotate (idle window).
4. Chạy backfill nếu cần re-hash.
```

---

## Composite impact P2
- Admin self-service config masking, không cần SQL migration mỗi lần.
- Backfill loại bỏ hoàn toàn `"***"` còn sót lại trong shadow.
- Compliance evidence doc sẵn sàng cho audit.
