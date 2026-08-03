# Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution Profile)

## Refactor Code Go: `internal/infra/persistence/source/source_object_read_repo_gorm.go`

Sửa khối `listBaseFromWhere`:
```sql
	LEFT JOIN LATERAL (
		SELECT
			rr.shadow_table AS target_table,
			rr.diff,
			rr.status,
			rr.checked_at
		FROM cdc_system.cdc_reconciliation_report rr
		WHERE rr.shadow_table = COALESCE(sb.shadow_table, tr.target_table)
		  AND rr.checked_at >= NOW() - INTERVAL '7 days'
		ORDER BY rr.checked_at DESC
		LIMIT 1
	) rr ON TRUE
```

Đảm bảo ngoặc đóng `) rr ON TRUE` và `LIMIT 1` có mặt đầy đủ trước `LEFT JOIN LATERAL (...) tj ON TRUE`.
