# 09_tasks_solution_recon_source_agent_refactor

Hồ sơ giải pháp chi tiết cho việc tái cấu trúc `recon_source_agent.go`.

## 1. File: `recon_models.go`
```go
package recon

import (
	"errors"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"go.mongodb.org/mongo-driver/mongo"
)

// ChunkHash — kept for backward compatibility with CMS callers that
// still reference the legacy Merkle-style result shape. New v3 code
// paths use WindowResult / BucketHashResult directly.
type ChunkHash struct {
	StartID string `json:"start_id"`
	EndID   string `json:"end_id"`
	Count   int    `json:"count"`
	Hash    string `json:"hash"`
}

// WindowResult is the fixed 16-byte streaming reconciliation unit.
// 8 bytes count + 8 bytes XOR hash. No ID set retained.
type WindowResult struct {
	Count     int64  `json:"count"`
	XorHash   uint64 `json:"xor_hash"`
	Err       error  `json:"-"`
	ErrorCode string `json:"error_code,omitempty"`
}

// Recon error codes — ADR v4 §2.3. Kept as const strings so callers can
// compare/switch without importing a separate enum package.
const (
	ErrCodeSrcTimeout      = "SRC_TIMEOUT"
	ErrCodeSrcConnection   = "SRC_CONNECTION"
	ErrCodeSrcFieldMissing = "SRC_FIELD_MISSING"
	ErrCodeSrcEmpty        = "SRC_EMPTY"
	ErrCodeDstTimeout      = "DST_TIMEOUT"
	ErrCodeDstMissingCol   = "DST_MISSING_COLUMN"
	ErrCodeCircuitOpen     = "CIRCUIT_OPEN"
	ErrCodeAuthError       = "AUTH_ERROR"
	ErrCodeUnknown         = "UNKNOWN"
)

// ClassifyMongoErrorForTest exposes classifyMongoError for tests in the external test/ tree.
func ClassifyMongoErrorForTest(err error) string { return classifyMongoError(err) }

// classifyMongoError maps a raw Mongo/driver error to a structured recon
// error code. Order matters: check most-specific patterns first (timeout,
// auth) before falling into the generic transient bucket.
func classifyMongoError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "circuit breaker is open") ||
		strings.Contains(lower, "breaker") && strings.Contains(lower, "open"):
		return ErrCodeCircuitOpen
	case strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "i/o timeout"):
		return ErrCodeSrcTimeout
	case strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "authentication failed") ||
		strings.Contains(lower, "auth fail"):
		return ErrCodeAuthError
	case strings.Contains(lower, "no field") ||
		strings.Contains(lower, "field does not exist") ||
		strings.Contains(lower, "missing field"):
		return ErrCodeSrcFieldMissing
	case isMongoTransient(err):
		return ErrCodeSrcConnection
	default:
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) {
			switch cmdErr.Code {
			case 13, 18:
				return ErrCodeAuthError
			case 50:
				return ErrCodeSrcTimeout
			}
		}
		return ErrCodeUnknown
	}
}

// BucketHashResult is a fixed-size 256-bucket XOR fingerprint for
// whole-table drift detection (Tier 3). 256 * 8 bytes = 2 KiB total.
type BucketHashResult struct {
	Buckets [256]uint64 `json:"buckets"`
	Total   int64       `json:"total"`
}

// ReconSourceAgentConfig holds tunables for MongoDB-side recon reads.
// All fields optional — sensible defaults supplied when zero-valued.
type ReconSourceAgentConfig struct {
	MaxDocsPerSec    int           // rate limit, default 5000
	QueryTimeout     time.Duration // per-query ctx deadline, default 30s
	BatchSize        int32         // Mongo cursor batchSize, default 1000
	BreakerTimeout   time.Duration // open-circuit cool-off, default 60s
	BreakerThreshold uint32        // consecutive failures before open, default 5
}

func (c *ReconSourceAgentConfig) applyDefaults() {
	if c.MaxDocsPerSec <= 0 {
		c.MaxDocsPerSec = 5000
	}
	if c.QueryTimeout <= 0 {
		c.QueryTimeout = 30 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1000
	}
	if c.BreakerTimeout <= 0 {
		c.BreakerTimeout = 60 * time.Second
	}
	if c.BreakerThreshold == 0 {
		c.BreakerThreshold = 5
	}
}
```

## 2. File: `recon_hash.go`
```go
package recon

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HashWindow streams docs whose updated_at ∈ [tLo, tHi) and builds a
// 16-byte fingerprint = (count, XOR of per-doc xxhash). The cursor is
// read in 1000-doc batches, per-doc the limiter waits so we never
// exceed MaxDocsPerSec.
func (sa *ReconSourceAgent) HashWindow(ctx context.Context, sourceURL, database, collection, timestampField string, tLo, tHi time.Time) (*WindowResult, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	coll := sa.secondaryColl(client, database, collection)

	tsField := resolveTimestampField(timestampField)
	filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
	opts := sa.selectOpts(bson.M{"_id": 1, tsField: 1})

	var out *WindowResult
	err = sa.queryWithRetry(ctx, "HashWindow", func() error {
		result, innerErr := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
			cursor, err := coll.Find(ctx, filter, opts)
			if err != nil {
				return nil, err
			}
			defer cursor.Close(ctx)

			var (
				xorAcc uint64
				count  int64
			)
			for cursor.Next(ctx) {
				if err := sa.limiter.Wait(ctx); err != nil {
					return nil, fmt.Errorf("rate limiter: %w", err)
				}
				var raw bson.M
				if err := cursor.Decode(&raw); err != nil {
					return nil, fmt.Errorf("decode: %w", err)
				}
				idStr := extractMongoID(raw["_id"])
				ts := extractTimestampMs(raw, tsField, idStr)
				xorAcc ^= hashIDPlusTsMs(idStr, ts)
				count++
			}
			if err := cursor.Err(); err != nil {
				return nil, err
			}
			return &WindowResult{Count: count, XorHash: xorAcc}, nil
		})
		if innerErr != nil {
			return innerErr
		}
		out = result.(*WindowResult)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BucketHash streams the entire collection once and distributes docs
// into 256 buckets keyed by the FIRST BYTE of xxhash(_id). Each bucket
// accumulates XOR of xxhash64(id + "|" + ts_ms) so source and
// destination fingerprints are directly comparable bucket-by-bucket.
func (sa *ReconSourceAgent) BucketHash(ctx context.Context, sourceURL, database, collection, timestampField string) (*BucketHashResult, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	coll := sa.secondaryColl(client, database, collection)

	tsField := resolveTimestampField(timestampField)
	opts := sa.selectOpts(bson.M{"_id": 1, tsField: 1})
	var out *BucketHashResult
	err = sa.queryWithRetry(ctx, "BucketHash", func() error {
		result, innerErr := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
			cursor, err := coll.Find(ctx, bson.M{}, opts)
			if err != nil {
				return nil, err
			}
			defer cursor.Close(ctx)

			var bh BucketHashResult
			for cursor.Next(ctx) {
				if err := sa.limiter.Wait(ctx); err != nil {
					return nil, fmt.Errorf("rate limiter: %w", err)
				}
				var raw bson.M
				if err := cursor.Decode(&raw); err != nil {
					return nil, fmt.Errorf("decode: %w", err)
				}
				idStr := extractMongoID(raw["_id"])
				ts := extractTimestampMs(raw, tsField, idStr)
				if ts == 0 {
					continue
				}
				bucket := bucketIndex(idStr)
				bh.Buckets[bucket] ^= hashIDPlusTsMs(idStr, ts)
				bh.Total++
			}
			if err := cursor.Err(); err != nil {
				return nil, err
			}
			return &bh, nil
		})
		if innerErr != nil {
			return innerErr
		}
		out = result.(*BucketHashResult)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HashIDPlusTsForTest, HashIDPlusTsMsForTest, BucketIndexForTest expose
// the recon hash primitives for tests in the external test/ tree.
func HashIDPlusTsForTest(idStr string, ts time.Time) uint64 { return hashIDPlusTs(idStr, ts) }

// HashIDPlusTsMsForTest exposes hashIDPlusTsMs for tests.
func HashIDPlusTsMsForTest(idStr string, tsMs int64) uint64 { return hashIDPlusTsMs(idStr, tsMs) }

// BucketIndexForTest exposes bucketIndex for tests.
func BucketIndexForTest(idStr string) uint8 { return bucketIndex(idStr) }

func hashIDPlusTs(idStr string, ts time.Time) uint64 {
	var b strings.Builder
	b.Grow(len(idStr) + 1 + 32)
	b.WriteString(idStr)
	b.WriteByte('|')
	b.WriteString(ts.UTC().Format(time.RFC3339Nano))
	return xxhash.Sum64String(b.String())
}

func hashIDPlusTsMs(idStr string, tsMs int64) uint64 {
	var buf [32]byte
	num := strconv.AppendInt(buf[:0], tsMs, 10)
	var b strings.Builder
	b.Grow(len(idStr) + 1 + len(num))
	b.WriteString(idStr)
	b.WriteByte('|')
	b.Write(num)
	return xxhash.Sum64String(b.String())
}

func bucketIndex(idStr string) uint8 {
	h := xxhash.Sum64String(idStr)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], h)
	return buf[0]
}

func extractMongoID(v interface{}) string {
	if oid, ok := v.(primitive.ObjectID); ok {
		return oid.Hex()
	}
	return fmt.Sprintf("%v", v)
}

func extractTimestampMs(raw bson.M, tsField, idHex string) int64 {
	if v, ok := raw[tsField]; ok && v != nil {
		switch t := v.(type) {
		case primitive.DateTime:
			return int64(t)
		case time.Time:
			if !t.IsZero() {
				return t.UnixMilli()
			}
		case int64:
			return t
		case int32:
			return int64(t)
		case float64:
			return int64(t)
		}
	}
	if oid, err := primitive.ObjectIDFromHex(idHex); err == nil {
		return oid.Timestamp().UnixMilli()
	}
	return 0
}
```

## 3. File: `recon_query.go`
```go
package recon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"centralized-data-service/internal/service/governance"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func resolveTimestampField(tsField string) string {
	const def = "updated_at"
	if tsField == "" {
		return def
	}
	if len(tsField) > 64 {
		return def
	}
	for i, r := range tsField {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return def
	}
	return tsField
}

// CountDocuments — Tier 1 legacy helper. Verifies the collection exists
// before counting (Mongo silently returns 0 on missing coll which would
// mask drift). Uses secondary read.
func (sa *ReconSourceAgent) CountDocuments(ctx context.Context, sourceURL, database, collection string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return 0, err
	}

	db := client.Database(database)
	names, err := db.ListCollectionNames(ctx, bson.M{"name": collection})
	if err != nil {
		return 0, fmt.Errorf("list collections failed on %s: %w", database, err)
	}
	if len(names) == 0 {
		return 0, fmt.Errorf("source collection not found: %s.%s", database, collection)
	}

	coll := sa.secondaryColl(client, database, collection)
	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		return coll.CountDocuments(ctx, bson.M{})
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// EstimatedCount — V5 Tier-0: đếm O(1) từ collection metadata (KHÔNG collscan
// như CountDocuments). Sai số nhỏ sau crash/resync — Tier-0 dùng tolerance,
// phán quyết drift cuối cùng thuộc bucket-aggregate (đếm thật).
func (sa *ReconSourceAgent) EstimatedCount(ctx context.Context, sourceURL, database, collection string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()
	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return 0, err
	}
	coll := sa.secondaryColl(client, database, collection)
	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		return coll.EstimatedDocumentCount(ctx)
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// BucketCounts — V5 Tier-1: 1 AGGREGATE server-side trả count theo bucket-giờ
// trong [tLo, tHi) — thay 672 round-trip CountInWindow (thủ phạm 328s/bảng @
// remote RTT). Key = epoch-ms đầu giờ (floor ts/3600000*3600000) — KHỚP công
// thức bucket của ReconDestAgent.BucketCounts để so trực tiếp.
func (sa *ReconSourceAgent) BucketCounts(ctx context.Context, sourceURL, database, collection, timestampField string, tLo, tHi time.Time) (map[int64]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()
	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	tsField := resolveTimestampField(timestampField)
	coll := sa.secondaryColl(client, database, collection)

	msExpr := bson.M{"$toLong": bson.M{"$toDate": "$" + tsField}}
	pipeline := []bson.M{
		{"$match": bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}},
		{"$group": bson.M{
			"_id": bson.M{"$subtract": []interface{}{msExpr, bson.M{"$mod": []interface{}{msExpr, int64(3600000)}}}},
			"n":   bson.M{"$sum": 1},
		}},
	}

	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		cur, err := coll.Aggregate(ctx, pipeline)
		if err != nil {
			return nil, err
		}
		defer cur.Close(ctx)
		out := make(map[int64]int64)
		for cur.Next(ctx) {
			var row struct {
				ID int64 `bson:"_id"`
				N  int64 `bson:"n"`
			}
			if err := cur.Decode(&row); err != nil {
				return nil, err
			}
			out[row.ID] = row.N
		}
		return out, cur.Err()
	})
	if err != nil {
		return nil, err
	}
	return result.(map[int64]int64), nil
}

// CountInWindow counts documents with <timestampField> ∈ [tLo, tHi)
func (sa *ReconSourceAgent) CountInWindow(ctx context.Context, sourceURL, database, collection, timestampField string, tLo, tHi time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return 0, err
	}
	coll := sa.secondaryColl(client, database, collection)

	tsField := resolveTimestampField(timestampField)
	filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
	var out int64
	err = sa.queryWithRetry(ctx, "CountInWindow", func() error {
		result, innerErr := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
			return coll.CountDocuments(ctx, filter)
		})
		if innerErr != nil {
			return innerErr
		}
		out = result.(int64)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return out, nil
}

// CountInWindowWithFallback first queries with `primaryField`. When the
// primary returns 0 AND no error, it probes each fallback candidate.
func (sa *ReconSourceAgent) CountInWindowWithFallback(
	ctx context.Context,
	sourceURL, database, collection, primaryField string,
	tLo, tHi time.Time,
	fallbackFields []string,
) (count int64, fieldUsed string, err error) {
	primary := resolveTimestampField(primaryField)
	count, err = sa.CountInWindow(ctx, sourceURL, database, collection, primary, tLo, tHi)
	if err != nil || count > 0 {
		return count, primary, err
	}
	for _, cand := range fallbackFields {
		if cand == "" {
			continue
		}
		if !governance.CandidateNameRE.MatchString(cand) {
			continue
		}
		if cand == primary {
			continue
		}
		c, e := sa.CountInWindow(ctx, sourceURL, database, collection, cand, tLo, tHi)
		if e != nil {
			continue
		}
		if c > 0 {
			if sa.logger != nil {
				sa.logger.Warn("primary timestamp field returned 0, fallback used",
					zap.String("collection", collection),
					zap.String("primary", primary),
					zap.String("fallback", cand),
					zap.Int64("count", c),
				)
			}
			return c, cand, nil
		}
	}
	return count, primary, nil
}

// MaxWindowTs returns the highest updated_at on the source, used by the
// Core to pick the upper watermark for Tier 1 / Tier 2 window scan.
func (sa *ReconSourceAgent) MaxWindowTs(ctx context.Context, sourceURL, database, collection, timestampField string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return time.Time{}, err
	}
	coll := sa.secondaryColl(client, database, collection)

	tsField := resolveTimestampField(timestampField)
	opts := options.FindOne().
		SetProjection(bson.M{tsField: 1}).
		SetSort(bson.M{tsField: -1})

	var out time.Time
	err = sa.queryWithRetry(ctx, "MaxWindowTs", func() error {
		result, innerErr := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
			var raw bson.M
			err := coll.FindOne(ctx, bson.M{}, opts).Decode(&raw)
			if err == mongo.ErrNoDocuments {
				return time.Time{}, nil
			}
			if err != nil {
				return time.Time{}, err
			}
			idStr := extractMongoID(raw["_id"])
			ms := extractTimestampMs(raw, tsField, idStr)
			if ms == 0 {
				return time.Time{}, nil
			}
			return time.UnixMilli(ms), nil
		})
		if innerErr != nil {
			return innerErr
		}
		out = result.(time.Time)
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return out, nil
}

func (sa *ReconSourceAgent) queryWithRetry(ctx context.Context, op string, fn func() error) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			if sa.logger != nil {
				sa.logger.Warn("recon mongo transient error, retrying",
					zap.String("op", op),
					zap.Int("attempt", attempt+1),
					zap.Error(lastErr),
				)
			}
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isMongoTransient(lastErr) {
			return lastErr
		}
	}
	return fmt.Errorf("%s: after %d attempts: %w", op, maxAttempts, lastErr)
}

func isMongoTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	s := err.Error()
	if strings.Contains(s, "incomplete read of message header") ||
		strings.Contains(s, "incomplete read") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "unexpected EOF") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "i/o timeout") {
		return true
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		switch cmdErr.Code {
		case 6, 7, 89, 91, 189, 262, 318, 9001:
			return true
		}
	}
	if strings.Contains(s, "server selection error") ||
		strings.Contains(s, "no reachable servers") {
		return true
	}
	return false
}
```

## 4. File: `recon_stream.go`
```go
package recon

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/options"
)

// ListIDsInWindow returns the concrete IDs inside a drifted window.
func (sa *ReconSourceAgent) ListIDsInWindow(ctx context.Context, sourceURL, database, collection, timestampField string, tLo, tHi time.Time) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	coll := sa.secondaryColl(client, database, collection)

	tsField := resolveTimestampField(timestampField)
	filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
	opts := sa.selectOpts(bson.M{"_id": 1})

	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		cursor, err := coll.Find(ctx, filter, opts)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		var ids []string
		for cursor.Next(ctx) {
			if err := sa.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
			var doc struct {
				ID interface{} `bson:"_id"`
			}
			if err := cursor.Decode(&doc); err != nil {
				return nil, fmt.Errorf("decode: %w", err)
			}
			ids = append(ids, extractMongoID(doc.ID))
		}
		if err := cursor.Err(); err != nil {
			return nil, err
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}

// Deprecated: dùng StreamAllIDs thay thế. ListAllIDs load toàn bộ ID vào RAM.
func (sa *ReconSourceAgent) ListAllIDs(ctx context.Context, sourceURL, database, collection string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, sa.cfg.QueryTimeout)
	defer cancel()

	client, err := sa.getClient(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	coll := sa.secondaryColl(client, database, collection)
	opts := sa.selectOpts(bson.M{"_id": 1})

	result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
		cursor, err := coll.Find(ctx, bson.M{}, opts)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)
		var ids []string
		for cursor.Next(ctx) {
			if err := sa.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
			var doc struct {
				ID interface{} `bson:"_id"`
			}
			if err := cursor.Decode(&doc); err != nil {
				return nil, fmt.Errorf("decode: %w", err)
			}
			ids = append(ids, extractMongoID(doc.ID))
		}
		if err := cursor.Err(); err != nil {
			return nil, err
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}

// StreamAllIDs thay thế ListAllIDs. Sử dụng Keyset Pagination ($gt trên _id) để tránh
// Cursor Timeout (giới hạn 10 phút Mongo) và trả về Channel thay vì Slice để giữ
// memory footprint O(1), chống OOM cho collection hàng trăm triệu records.
func (sa *ReconSourceAgent) StreamAllIDs(ctx context.Context, sourceURL, database, collection string) (<-chan string, <-chan error) {
	idChan := make(chan string, 1000)
	errChan := make(chan error, 1)

	go func() {
		defer close(idChan)
		defer close(errChan)

		client, err := sa.getClient(ctx, sourceURL)
		if err != nil {
			errChan <- fmt.Errorf("get client: %w", err)
			return
		}
		coll := sa.secondaryColl(client, database, collection)

		batchSize := int64(sa.cfg.BatchSize)
		if batchSize <= 0 {
			batchSize = 1000
		}

		var lastID interface{} = nil

		for {
			if err := ctx.Err(); err != nil {
				errChan <- err
				return
			}

			filter := bson.M{}
			if lastID != nil {
				filter["_id"] = bson.M{"$gt": lastID}
			}

			opts := options.Find().
				SetProjection(bson.M{"_id": 1}).
				SetSort(bson.D{{Key: "_id", Value: 1}}).
				SetLimit(batchSize)

			var batchCount int64

			result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
				return coll.Find(ctx, filter, opts)
			})
			if err != nil {
				errChan <- fmt.Errorf("find batch (after _id=%v): %w", lastID, err)
				return
			}

			cursor := result.(*mongo.Cursor)
			for cursor.Next(ctx) {
				if err := sa.limiter.Wait(ctx); err != nil {
					cursor.Close(ctx)
					errChan <- fmt.Errorf("rate limiter: %w", err)
					return
				}

				var doc struct {
					ID interface{} `bson:"_id"`
				}
				if err := cursor.Decode(&doc); err != nil {
					cursor.Close(ctx)
					errChan <- fmt.Errorf("decode _id: %w", err)
					return
				}

				lastID = doc.ID

				select {
				case <-ctx.Done():
					cursor.Close(ctx)
					errChan <- ctx.Err()
					return
				case idChan <- extractMongoID(doc.ID):
				}
				batchCount++
			}

			if err := cursor.Err(); err != nil {
				cursor.Close(ctx)
				errChan <- fmt.Errorf("cursor error: %w", err)
				return
			}
			cursor.Close(ctx)

			if batchCount < batchSize {
				break
			}
		}
	}()

	return idChan, errChan
}
```

## 5. File: `recon_legacy.go`
```go
package recon

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
)

// GetChunkHashes — legacy API. Under v3 we delegate to BucketHash and
// surface the 256 buckets as ChunkHash entries.
func (sa *ReconSourceAgent) GetChunkHashes(ctx context.Context, sourceURL, database, collection string, chunkSize int) ([]ChunkHash, error) {
	bh, err := sa.BucketHash(ctx, sourceURL, database, collection, "")
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

func redactURL(u string) string {
	if u == "" {
		return ""
	}
	for _, sep := range []string{"://"} {
		if idx := strings.Index(u, sep); idx > 0 {
			return u[:idx+len(sep)] + "<redacted>"
		}
	}
	return "<redacted>"
}

func buildLegacyChunkHash(ids []string) ChunkHash {
	sort.Strings(ids)
	concat := strings.Join(ids, "|")
	hash := fmt.Sprintf("%x", md5.Sum([]byte(concat)))
	return ChunkHash{
		StartID: ids[0],
		EndID:   ids[len(ids)-1],
		Count:   len(ids),
		Hash:    hash,
	}
}
```

## 6. File: `recon_source_agent.go` (đã rút gọn)
```go
package recon

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"centralized-data-service/pkgs/mongodb"

	"github.com/sony/gobreaker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// ReconSourceAgent connects to MongoDB for reconciliation.
type ReconSourceAgent struct {
	defaultClient *mongo.Client
	clients       map[string]*mongo.Client
	breakers      map[string]*gobreaker.CircuitBreaker
	mu            sync.RWMutex
	cfg           ReconSourceAgentConfig
	limiter       *rate.Limiter
	logger        *zap.Logger
}

// NewReconSourceAgent constructs an agent with default v3 tunables.
func NewReconSourceAgent(defaultClient *mongo.Client, logger *zap.Logger) *ReconSourceAgent {
	return NewReconSourceAgentWithConfig(defaultClient, ReconSourceAgentConfig{}, logger)
}

// NewReconSourceAgentWithConfig allows callers to override rate limit /
// timeout / breaker settings.
func NewReconSourceAgentWithConfig(defaultClient *mongo.Client, cfg ReconSourceAgentConfig, logger *zap.Logger) *ReconSourceAgent {
	cfg.applyDefaults()
	return &ReconSourceAgent{
		defaultClient: defaultClient,
		clients:       make(map[string]*mongo.Client),
		breakers:      make(map[string]*gobreaker.CircuitBreaker),
		cfg:           cfg,
		limiter:       rate.NewLimiter(rate.Limit(cfg.MaxDocsPerSec), cfg.MaxDocsPerSec),
		logger:        logger,
	}
}

func (sa *ReconSourceAgent) getClient(ctx context.Context, sourceURL string) (*mongo.Client, error) {
	if sourceURL == "" {
		if sa.defaultClient == nil {
			return nil, fmt.Errorf("recon: no mongo client available — entry.SourceURL is empty AND default client not configured. Verify the source_object_registry row's connection_code resolves to a valid Mongo URI in cdc_system.connection_registry, or set cfg.MongoDB.URL as a legacy default")
		}
		return sa.defaultClient, nil
	}

	sa.mu.RLock()
	if c, ok := sa.clients[sourceURL]; ok {
		sa.mu.RUnlock()
		return c, nil
	}
	sa.mu.RUnlock()

	sa.mu.Lock()
	defer sa.mu.Unlock()
	if c, ok := sa.clients[sourceURL]; ok {
		return c, nil
	}

	urlWithTimeout := sourceURL
	if !strings.Contains(urlWithTimeout, "serverSelectionTimeoutMS") {
		sep := "?"
		if strings.Contains(urlWithTimeout, "?") {
			sep = "&"
		}
		urlWithTimeout += sep + "serverSelectionTimeoutMS=5000&connectTimeoutMS=5000"
	}
	client, err := mongodb.NewClient(ctx, mongodb.MongoConfig{URL: urlWithTimeout}, sa.logger)
	if err != nil {
		return nil, fmt.Errorf("connect to source %s: %w", redactURL(sourceURL), err)
	}
	sa.clients[sourceURL] = client
	sa.logger.Info("new MongoDB source connected (recon)", zap.String("url", redactURL(sourceURL)))
	return client, nil
}

func (sa *ReconSourceAgent) getBreaker(sourceURL string) *gobreaker.CircuitBreaker {
	key := sourceURL
	if key == "" {
		key = "_default"
	}
	sa.mu.RLock()
	if b, ok := sa.breakers[key]; ok {
		sa.mu.RUnlock()
		return b
	}
	sa.mu.RUnlock()

	sa.mu.Lock()
	defer sa.mu.Unlock()
	if b, ok := sa.breakers[key]; ok {
		return b
	}
	b := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "recon-source-" + redactURL(key),
		Timeout: sa.cfg.BreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= sa.cfg.BreakerThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			sa.logger.Warn("recon source breaker state change",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	})
	sa.breakers[key] = b
	return b
}

func (sa *ReconSourceAgent) selectOpts(projection bson.M) *options.FindOptions {
	return options.Find().
		SetProjection(projection).
		SetBatchSize(sa.cfg.BatchSize)
}

func (sa *ReconSourceAgent) secondaryColl(client *mongo.Client, database, collection string) *mongo.Collection {
	return client.Database(database).Collection(
		collection,
		options.Collection().SetReadPreference(readpref.Secondary()),
	)
}
```
