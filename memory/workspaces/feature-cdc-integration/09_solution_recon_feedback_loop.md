# Solution: Reconciliation → Data Source Status Feedback Loop

> Date: 2026-04-16
> Phase: data-integrity
> Problem: Reconciliation chạy 30 phút/lần nhưng KHÔNG cập nhật trạng thái data source

---

## 1. Vấn đề

Khi reconcile chạy, nó phát hiện:
- `source collection not found` → status=error
- `source=2, dest=1000002` → status=drift

Nhưng kết quả **chỉ nằm trong `cdc_reconciliation_report`** — KHÔNG feedback ngược lại:
- `cdc_table_registry.is_active` vẫn = true cho tables source không tồn tại
- `cdc_table_registry.sync_status` không cập nhật
- FE Registry page vẫn hiện "Active" cho tables thực tế đã chết

## 2. Solution: Recon → Registry Feedback

Sau mỗi `CheckAll()`, cập nhật `cdc_table_registry`:

| Recon Status | Registry Update |
|:-------------|:----------------|
| `ok` | `sync_status = 'healthy'`, `last_recon_at = now` |
| `drift` | `sync_status = 'drift'`, `last_recon_at = now`, `recon_drift = diff` |
| `error` (source not found) | `sync_status = 'source_error'`, `last_recon_at = now` |

### Fields cần thêm vào `cdc_table_registry`:
- `sync_status` VARCHAR(50) — healthy / drift / source_error / unknown
- `last_recon_at` TIMESTAMP — lần check cuối
- `recon_drift` INT — số lệch (source - dest)

## 3. Implementation

### Worker: `recon_core.go` — sau CheckAll
```go
for _, report := range reports {
    updates := map[string]interface{}{
        "last_recon_at": time.Now(),
    }
    switch report.Status {
    case "ok":
        updates["sync_status"] = "healthy"
        updates["recon_drift"] = 0
    case "drift":
        updates["sync_status"] = "drift"
        updates["recon_drift"] = report.Diff
    case "error":
        updates["sync_status"] = "source_error"
    }
    db.Model(&model.TableRegistry{}).
        Where("target_table = ?", report.TargetTable).
        Updates(updates)
}
```

### Model: `table_registry.go` — thêm fields
### FE: `TableRegistry.tsx` — hiện sync_status badge

## 4. Definition of Done
- [x] Registry model có sync_status, last_recon_at, recon_drift
- [x] CheckAll() cập nhật registry sau mỗi lần chạy
- [x] FE TableRegistry hiện sync_status badge (healthy/drift/source_error)
- [x] Runtime verify: trigger recon → registry updated (healthy/source_error/drift)

## 5. Runtime Proof (2026-04-17)
```
export_jobs            | healthy      | recon_drift=0
identitycounters       | source_error | (collection not found)
payment_bills          | drift        | recon_drift=-1000000
refund_requests        | drift        | recon_drift=-1710
```
