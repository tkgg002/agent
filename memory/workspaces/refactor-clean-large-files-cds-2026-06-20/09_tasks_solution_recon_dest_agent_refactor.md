# Giải pháp mã nguồn chi tiết - Recon Dest Agent Refactor

Tài liệu này chứa mã nguồn chi tiết của các file mới và file `recon_dest_agent.go` sau khi phân rã.

## 1. `internal/service/recon/recon_dest_models.go` [NEW]
```go
package recon

import "time"

// ReconDestAgentConfig tunes Postgres-side recon reads.
// Defaults mirror ReconSourceAgentConfig.
type ReconDestAgentConfig struct {
	MaxRowsPerSec    int
	QueryTimeout     time.Duration
	BreakerTimeout   time.Duration
	BreakerThreshold uint32
	// ReadReplicaDSN — optional. When set the agent uses a dedicated
	// replica connection. Empty means reuse the primary connection
	// but wrap every transaction in SET TRANSACTION READ ONLY so a
	// misconfigured query still cannot mutate data.
	ReadReplicaDSN string
}

func (c *ReconDestAgentConfig) applyDefaults() {
	if c.MaxRowsPerSec <= 0 {
		c.MaxRowsPerSec = 5000
	}
	if c.QueryTimeout <= 0 {
		c.QueryTimeout = 30 * time.Second
	}
	if c.BreakerTimeout <= 0 {
		c.BreakerTimeout = 60 * time.Second
	}
	if c.BreakerThreshold == 0 {
		c.BreakerThreshold = 5
	}
}

// BucketStat — count + XOR fingerprint của 1 bucket-giờ (V5 Tier-1).
type BucketStat struct {
	Count int64
	Xor   int64
}

// IDTs — cặp (pk, _source_ts) cho diff stale đích danh (Recon V4 P4 L2-B+).
type IDTs struct {
	ID string
	Ts int64
}
```

## 2. `internal/service/recon/recon_dest_hash.go` [NEW]
```go
package recon

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// HashWindow streams rows whose `_source_ts` ∈ [tLo, tHi) and builds the
// same (count, XOR of per-row xxhash) fingerprint the source-side agent
// emits. Hash input PER ROW = xxhash64(id || "|" || _source_ts_ms).
func (da *ReconDestAgent) HashWindow(ctx context.Context, tableName, pkColumn string, tLo, tHi time.Time) (*WindowResult, error) {
	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	if err := validateIdent(pkColumn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	loMs, hiMs := tLo.UnixMilli(), tHi.UnixMilli()

	// Stream rows — projection is just (id, _source_ts). Same index
	// usage as the old aggregate (idx_<tbl>_source_ts). No ORDER BY:
	// XOR is commutative so we don't need ordering.
	sql := fmt.Sprintf(
		`SELECT %s::text AS id, "_source_ts" AS source_ts
		   FROM %s
		  WHERE "_source_ts" >= ? AND "_source_ts" < ?`,
		quoteIdent(pkColumn), quoteRelation(tableName),
	)

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()

		rows, err := tx.Raw(sql, loMs, hiMs).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var (
			xorAcc      uint64
			count       int64
			nullSkipped int64
		)
		for rows.Next() {
			if err := da.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
			var (
				id       string
				sourceTs *int64
			)
			if err := rows.Scan(&id, &sourceTs); err != nil {
				return nil, err
			}
			if sourceTs == nil {
				// Backfill pending — unknown ts can never match the
				// source-side representation, so skip rather than pollute
				// the XOR with a false negative.
				nullSkipped++
				continue
			}
			xorAcc ^= hashIDPlusTsMs(id, *sourceTs)
			count++
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}

		if nullSkipped > 0 && da.logger != nil {
			da.logger.Warn("recon dest HashWindow: rows with NULL _source_ts skipped",
				zap.String("table", tableName),
				zap.Int64("skipped", nullSkipped),
				zap.Int64("hashed", count),
			)
		}
		return &WindowResult{Count: count, XorHash: xorAcc}, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*WindowResult), nil
}

// BucketHash streams destination rows and computes the same 256-bucket
// xxhash64(id + "|" + ts_ms) fingerprint as the source-side agent.
// This makes Tier 3 bucket values directly comparable across stores.
func (da *ReconDestAgent) BucketHash(ctx context.Context, tableName, pkColumn string) (*BucketHashResult, error) {
	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	if err := validateIdent(pkColumn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	sql := fmt.Sprintf(
		`SELECT %s::text AS id, "_source_ts" AS source_ts
		   FROM %s`,
		quoteIdent(pkColumn), quoteRelation(tableName),
	)

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()

		rows, err := tx.Raw(sql).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var bh BucketHashResult
		for rows.Next() {
			if err := da.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
			var (
				id       string
				sourceTs *int64
			)
			if err := rows.Scan(&id, &sourceTs); err != nil {
				return nil, err
			}
			if sourceTs == nil {
				continue
			}
			idx := bucketIndex(id)
			bh.Buckets[idx] ^= hashIDPlusTsMs(id, *sourceTs)
			bh.Total++
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return &bh, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*BucketHashResult), nil
}
```

## 3. `internal/service/recon/recon_dest_query.go` [NEW]
```go
package recon

import (
	"context"
	"fmt"
	"time"
)

// CountRows — Tier 1 legacy helper. Kept for backward-compat with the
// CMS reporting API. Uses read-only transaction on the replica.
func (da *ReconDestAgent) CountRows(ctx context.Context, tableName, pkColumn string) (int64, error) {
	if err := validateIdent(tableName); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		var count int64
		sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteRelation(tableName))
		if err := tx.Raw(sql).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// CountInWindow counts rows whose `_source_ts` ∈ [tLo, tHi) in ms.
func (da *ReconDestAgent) CountInWindow(ctx context.Context, tableName string, tLo, tHi time.Time) (int64, error) {
	if err := validateIdent(tableName); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	loMs, hiMs := tLo.UnixMilli(), tHi.UnixMilli()

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		var count int64
		sql := fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE "_source_ts" >= ? AND "_source_ts" < ?`,
			quoteRelation(tableName),
		)
		if err := tx.Raw(sql, loMs, hiMs).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// BucketCounts — V5 Tier-1: 1 QUERY trả count + bit_xor fingerprint theo
// bucket-giờ trong [tLo, tHi) — thay 672 round-trip CountInWindow/HashWindow.
// Bucket key = epoch-ms đầu giờ (floor _source_ts/3600000*3600000) — KHỚP
// ReconSourceAgent.BucketCounts. bit_xor cần PG14+; hashtextextended built-in.
func (da *ReconDestAgent) BucketCounts(ctx context.Context, tableName, pkColumn string, tLo, tHi time.Time) (map[int64]BucketStat, error) {
	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	if err := validateIdent(pkColumn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	loMs, hiMs := tLo.UnixMilli(), tHi.UnixMilli()
	sql := fmt.Sprintf(`
		SELECT ("_source_ts" - ("_source_ts" %% 3600000))::bigint AS bucket,
		       COUNT(*) AS n,
		       COALESCE(bit_xor(hashtextextended(%s::text || '|' || "_source_ts"::text, 0)), 0) AS xorh
		  FROM %s
		 WHERE "_source_ts" >= ? AND "_source_ts" < ?
		 GROUP BY 1`,
		quoteIdent(pkColumn), quoteRelation(tableName),
	)

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		rows, err := tx.Raw(sql, loMs, hiMs).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := make(map[int64]BucketStat)
		for rows.Next() {
			var b, n, x int64
			if err := rows.Scan(&b, &n, &x); err != nil {
				return nil, err
			}
			out[b] = BucketStat{Count: n, Xor: x}
		}
		return out, rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return result.(map[int64]BucketStat), nil
}

// ListIDTsInWindow — như ListIDsInWindow nhưng kèm _source_ts để Segment B
// chỉ đích danh row STALE (id trùng 2 bên, ts lệch) thay vì chỉ đếm window.
func (da *ReconDestAgent) ListIDTsInWindow(ctx context.Context, tableName, pkColumn string, tLo, tHi time.Time) ([]IDTs, error) {
	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	if err := validateIdent(pkColumn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	loMs, hiMs := tLo.UnixMilli(), tHi.UnixMilli()
	sql := fmt.Sprintf(
		`SELECT %s::text AS id, "_source_ts" AS ts FROM %s
		 WHERE "_source_ts" >= ? AND "_source_ts" < ?`,
		quoteIdent(pkColumn), quoteRelation(tableName),
	)

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()

		rows, err := tx.Raw(sql, loMs, hiMs).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []IDTs
		for rows.Next() {
			if err := da.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
			var it IDTs
			if err := rows.Scan(&it.ID, &it.Ts); err != nil {
				return nil, err
			}
			out = append(out, it)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]IDTs), nil
}

// MaxWindowTs returns the highest `_source_ts` as a time.Time, used by
// Core to pick the Tier 1/2 upper watermark. Returns zero time if the
// table is empty or has no populated _source_ts yet.
func (da *ReconDestAgent) MaxWindowTs(ctx context.Context, tableName string) (time.Time, error) {
	if err := validateIdent(tableName); err != nil {
		return time.Time{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	sql := fmt.Sprintf(`SELECT COALESCE(MAX("_source_ts"), 0) FROM %s`, quoteRelation(tableName))

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		var maxMs int64
		if err := tx.Raw(sql).Scan(&maxMs).Error; err != nil {
			return time.Time{}, err
		}
		if maxMs == 0 {
			return time.Time{}, nil
		}
		return time.UnixMilli(maxMs), nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return result.(time.Time), nil
}
```

## 4. `internal/service/recon/recon_dest_stream.go` [NEW]
```go
package recon

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ListIDsInWindow returns IDs for a drifted window. Expected small —
// caller should only request it after HashWindow pinpointed drift.
// Rate-limited per row.
func (da *ReconDestAgent) ListIDsInWindow(ctx context.Context, tableName, pkColumn string, tLo, tHi time.Time) ([]string, error) {
	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	if err := validateIdent(pkColumn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	loMs, hiMs := tLo.UnixMilli(), tHi.UnixMilli()
	sql := fmt.Sprintf(
		`SELECT %s::text AS id FROM %s
		 WHERE "_source_ts" >= ? AND "_source_ts" < ?`,
		quoteIdent(pkColumn), quoteRelation(tableName),
	)

	result, err := da.breaker.Execute(func() (interface{}, error) {
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()

		rows, err := tx.Raw(sql, loMs, hiMs).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var ids []string
		for rows.Next() {
			if err := da.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}

// GetIDs — legacy API. v3 does not need full-table ID pagination; we
// keep the signature but return an empty slice when called outside the
// narrow window scope. Callers that still need a bounded list should
// move to `ListIDsInWindow`.
func (da *ReconDestAgent) GetIDs(ctx context.Context, tableName, pkColumn string, batchSize, offset int) ([]string, error) {
	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	if err := validateIdent(pkColumn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	sql := fmt.Sprintf(
		`SELECT %s::text AS id FROM %s ORDER BY %s LIMIT ? OFFSET ?`,
		quoteIdent(pkColumn), quoteRelation(tableName), quoteIdent(pkColumn),
	)

	tx := da.readOnlyDB(ctx)
	defer tx.Rollback()

	var ids []string
	if err := tx.Raw(sql, batchSize, offset).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// GetAllIDs — REMOVED in v3 runtime sense. The CMS report layer still
// references this symbol, so we keep a stub that returns an empty slice
// + a WARN log explaining why. Any caller that relied on the full set
// should migrate to window-scoped APIs. We DO NOT scan all IDs here —
// that was the bug this rewrite exists to kill.
func (da *ReconDestAgent) GetAllIDs(ctx context.Context, tableName, pkColumn string) ([]string, error) {
	if da.logger != nil {
		da.logger.Warn("ReconDestAgent.GetAllIDs called — v3 returns empty slice; migrate caller to ListIDsInWindow",
			zap.String("table", tableName),
		)
	}
	return nil, nil
}
```

## 5. `internal/service/recon/recon_dest_legacy.go` [NEW]
```go
package recon

import (
	"context"
	"fmt"
)

// GetChunkHashes — legacy API, delegates to BucketHash and repackages.
// chunkSize is accepted for signature compat but ignored (256 buckets
// are the new primitive).
func (da *ReconDestAgent) GetChunkHashes(ctx context.Context, tableName, pkColumn string, chunkSize int) ([]ChunkHash, error) {
	bh, err := da.BucketHash(ctx, tableName, pkColumn)
	if err != nil {
		return nil, err
	}
	out := make([]ChunkHash, 0, 256)
	for i, h := range bh.Buckets {
		if h == 0 {
			continue
		}
		out = append(out, ChunkHash{
			StartID: fmt.Sprintf("bucket:%03d", i),
			EndID:   fmt.Sprintf("bucket:%03d", i),
			Count:   0,
			Hash:    fmt.Sprintf("%016x", h),
		})
	}
	return out, nil
}
```

## 6. `internal/service/recon/recon_dest_safety.go` [NEW]
```go
package recon

import (
	"fmt"
	"strings"
)

// validateIdent rejects obviously-malicious identifiers BEFORE we
// embed them into SQL. Real safety is enforced by `quoteIdent` which
// wraps the identifier in double quotes and escapes embedded quotes,
// but a defense-in-depth check against nulls / control chars / DML
// keywords prevents human mistakes upstream.
func validateIdent(s string) error {
	if s == "" {
		return fmt.Errorf("identifier must not be empty")
	}
	if len(s) > 128 {
		return fmt.Errorf("identifier too long: %d chars", len(s))
	}
	for _, r := range s {
		if r == 0 || r == '\x00' || r == '\n' || r == '\r' {
			return fmt.Errorf("identifier contains control character")
		}
	}
	return nil
}

// quoteIdent returns `"<ident>"` with embedded `"` doubled — matches
// `pgx.Identifier{s}.Sanitize()` but without importing pgx just for
// this helper.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// quoteRelation quotes a (possibly schema-qualified) relation reference:
// "shadow_dev000.export_jobs" → "shadow_dev000"."export_jobs";
// bare "export_jobs" → "export_jobs" (V1 legacy, search_path resolve).
// Only the FIRST dot splits schema/table — shadow table names are
// slugified (no dots), so no ambiguity.
func quoteRelation(s string) string {
	if i := strings.IndexByte(s, '.'); i > 0 {
		return quoteIdent(s[:i]) + "." + quoteIdent(s[i+1:])
	}
	return quoteIdent(s)
}
```

## 7. `internal/service/recon/recon_dest_agent.go` [MODIFY]
```go
package recon

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// ReconDestAgent queries Postgres for reconciliation.
type ReconDestAgent struct {
	primary *gorm.DB // write-owner, kept for metadata queries
	replica *gorm.DB // read-only path (may == primary)
	breaker *gobreaker.CircuitBreaker
	limiter *rate.Limiter
	cfg     ReconDestAgentConfig
	logger  *zap.Logger
}

// NewReconDestAgent keeps the original 2-arg signature so existing
// worker_server wiring compiles.
func NewReconDestAgent(db *gorm.DB, logger *zap.Logger) *ReconDestAgent {
	return NewReconDestAgentWithConfig(db, nil, ReconDestAgentConfig{}, logger)
}

// NewReconDestAgentWithConfig constructs the agent with explicit replica
// DSN (when wired from config).
func NewReconDestAgentWithConfig(primary, replica *gorm.DB, cfg ReconDestAgentConfig, logger *zap.Logger) *ReconDestAgent {
	cfg.applyDefaults()
	if replica == nil {
		replica = primary
	}
	return &ReconDestAgent{
		primary: primary,
		replica: replica,
		cfg:     cfg,
		limiter: rate.NewLimiter(rate.Limit(cfg.MaxRowsPerSec), cfg.MaxRowsPerSec),
		breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "recon-dest",
			Timeout: cfg.BreakerTimeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= cfg.BreakerThreshold
			},
			OnStateChange: func(name string, from, to gobreaker.State) {
				if logger != nil {
					logger.Warn("recon dest breaker state change",
						zap.String("name", name),
						zap.String("from", from.String()),
						zap.String("to", to.String()),
					)
				}
			},
		}),
		logger: logger,
	}
}

// readOnlyDB starts a context-bound, read-only transaction handle.
// Every v3 read path goes through this helper.
func (da *ReconDestAgent) readOnlyDB(ctx context.Context) *gorm.DB {
	tx := da.replica.WithContext(ctx).Begin()
	tx.Exec("SET TRANSACTION READ ONLY")
	return tx
}
```
