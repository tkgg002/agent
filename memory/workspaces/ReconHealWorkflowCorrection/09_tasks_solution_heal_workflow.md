# Hồ sơ giải pháp kỹ thuật - Sửa đổi Luồng và Trạng thái Chữa lành Đối soát

Hồ sơ này định nghĩa các thay đổi mã nguồn chính xác cần thực hiện trên hệ thống.

---

## 1. Centralized Data Service (`centralized-data-service`)

### [MODIFY] [reconciliation_report_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go)

Sửa đổi hàm `ReleaseHealClaim` để an toàn hơn khi giải phóng claim gặp lỗi, tránh bị kẹt trạng thái:

```go
// ReleaseHealClaim reverts a report status when heal fails mid-processing,
// allowing it to be retried later.
func (r *ReconciliationReportRepo) ReleaseHealClaim(ctx context.Context, id uint64, prevStatus string) error {
	if prevStatus == "" || prevStatus == "healing" {
		prevStatus = "drift"
	}
	return r.db.WithContext(ctx).
		Model(&modelrecon.ReconciliationReport{}).
		Where("id = ? AND status = 'healing'", id).
		Update("status", prevStatus).Error
}
```

---

## 2. CDC CMS Service (`cdc-cms-service`)

### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/handler/recon/recon_execute_heal_handler.go)

Sửa hàm `finalizeReport` để chỉ set `healed_at` khi toàn bộ lỗi được xử lý:

```go
func (h *ExecuteHealHandler) finalizeReport(ctx context.Context, rpt *modelrecon.ReconciliationReport) {
	now := time.Now().UTC()
	
	// Check if all detected issues are fully healed/resolved
	isFullyHealed := rpt.HealedMissingDestCount >= rpt.MissingCount &&
		rpt.HealedMismatchedCount >= rpt.StaleCount &&
		rpt.PrunedMissingSrcCount >= rpt.OrphanCount

	updates := map[string]any{
		"healed_mismatched_count":         rpt.HealedMismatchedCount,
		"healed_mismatched_duration_ms":   rpt.HealedMismatchedDurationMs,
		"healed_missing_dest_count":       rpt.HealedMissingDestCount,
		"healed_missing_dest_duration_ms": rpt.HealedMissingDestDurationMs,
		"pruned_missing_src_count":        rpt.PrunedMissingSrcCount,
		"pruned_missing_src_duration_ms":  rpt.PrunedMissingSrcDurationMs,
	}

	if isFullyHealed {
		updates["healed_at"] = now
		updates["status"] = "healed"
	} else {
		updates["healed_at"] = nil
		updates["status"] = "partially_healed"
	}

	_ = h.reportRepo.UpdateByID(ctx, rpt.ID, updates)
}
```

---

## 3. CDC CMS Web (`cdc-cms-web`)

### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)

Cập nhật các cột đếm lỗi và vô hiệu hóa các checkbox chữa lành đã hoàn tất:

#### 3.1. Cập nhật columns `Thiếu`, `Lệch`, `Thừa` trong bảng báo cáo:
```typescript
    {
      title: 'Thiếu', dataIndex: 'missing_count', width: 70,
      render: (v: number, record: any) => {
        const healed = record.healed_missing_dest_count || 0;
        const remaining = Math.max(0, v - healed);
        if (healed > 0) {
          return (
            <span style={{ fontWeight: 600 }}>
              <Text type={remaining > 0 ? "danger" : "secondary"}>{remaining}</Text>
              <Text type="secondary" style={{ fontSize: 11, fontWeight: 400 }}>/{v}</Text>
            </span>
          );
        }
        return v > 0 ? <Text type="danger" style={{ fontWeight: 600 }}>{v}</Text> : <Text type="secondary">0</Text>;
      }
    },
    {
      title: 'Lệch', dataIndex: 'stale_count', width: 70,
      render: (v: number, record: any) => {
        const healed = record.healed_mismatched_count || 0;
        const remaining = Math.max(0, v - healed);
        if (healed > 0) {
          return (
            <span style={{ fontWeight: 600 }}>
              <Text type={remaining > 0 ? "warning" : "secondary"}>{remaining}</Text>
              <Text type="secondary" style={{ fontSize: 11, fontWeight: 400 }}>/{v}</Text>
            </span>
          );
        }
        return v > 0 ? <Text type="warning" style={{ fontWeight: 600 }}>{v}</Text> : <Text type="secondary">0</Text>;
      }
    },
    {
      title: 'Thừa', dataIndex: 'orphan_count', width: 70,
      render: (v: number, record: any) => {
        const healed = record.pruned_missing_src_count || 0;
        const remaining = Math.max(0, v - healed);
        if (healed > 0) {
          return (
            <span style={{ fontWeight: 600 }}>
              <Text style={{ color: remaining > 0 ? '#ff4d4f' : '#8c8c8c' }}>{remaining}</Text>
              <Text type="secondary" style={{ fontSize: 11, fontWeight: 400 }}>/{v}</Text>
            </span>
          );
        }
        return v > 0 ? <Text style={{ color: '#ff4d4f', fontWeight: 600 }}>{v}</Text> : <Text type="secondary">0</Text>;
      }
    },
```

#### 3.2. Cập nhật disable logic cho các checkbox và useEffect:
```typescript
  // Reset state when modal opens or reports data is loaded
  useEffect(() => {
    if (open && reports.length > 0) {
      setReason('');
      setSubmitting(false);
      
      const hasRemainingMismatched = reports.some(
        (r) => r.stale_count - (r.healed_mismatched_count || 0) > 0
      );
      const hasRemainingMissingDest = reports.some(
        (r) => r.missing_count - (r.healed_missing_dest_count || 0) > 0
      );

      setHealMismatched(hasRemainingMismatched);
      setHealMissingDest(hasRemainingMissingDest);
      setPruneMissingSrc(false);
    }
  }, [open, reports]);
```

#### 3.3. Checkbox render:
```typescript
          <Space direction="vertical" size={8}>
            <Checkbox 
              checked={healMismatched} 
              disabled={!reports.some(r => r.stale_count - (r.healed_mismatched_count || 0) > 0)}
              onChange={(e) => setHealMismatched(e.target.checked)}
            >
              <span>Chữa lành dữ liệu <strong>lệch</strong> (mismatched) — ghi đè bản ghi sai từ nguồn</span>
            </Checkbox>
            <Checkbox 
              checked={healMissingDest} 
              disabled={!reports.some(r => r.missing_count - (r.healed_missing_dest_count || 0) > 0)}
              onChange={(e) => setHealMissingDest(e.target.checked)}
            >
              <span>Bổ dung dữ liệu <strong>thiếu ở đích</strong> (missing from dest) — đồng bộ bản ghi mới</span>
            </Checkbox>
            <Checkbox 
              checked={pruneMissingSrc} 
              disabled={!reports.some(r => r.orphan_count - (r.pruned_missing_src_count || 0) > 0)}
              onChange={(e) => setPruneMissingSrc(e.target.checked)}
            >
              <span style={{ color: '#ff4d4f' }}>Xoá bản ghi <strong>thừa ở đích</strong> (orphan) — soft-delete dữ liệu không còn ở nguồn</span>
            </Checkbox>
          </Space>
```
