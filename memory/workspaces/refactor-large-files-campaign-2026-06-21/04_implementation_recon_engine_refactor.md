# Giải pháp kỹ thuật: Phân tách recon_engine.go (Gom nhóm theo Flow)

Để tránh băm nhỏ code thành quá nhiều file helper vụn vặt gây phân tán logic, chúng tôi đề xuất gom nhóm toàn bộ nội dung của `recon_engine.go` (730 dòng) thành đúng **3 file** mạch lạc theo flow logic:

---

## 1. `recon_engine.go` (MODIFY)
Chứa core struct `ReconCore`, configurations, constructors, and static/generic helper functions.

```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/model/system"
	"centralized-data-service/internal/service/metadata"
	"centralized-data-service/internal/service/shadow"
	reposource "centralized-data-service/internal/repository/source"
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"os"
	"strings"
	"time"

	"centralized-data-service/pkgs/rediscache"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ReconCoreConfig struct {
	WindowSize         time.Duration
	WindowLookback     time.Duration
	WindowFreezeMargin time.Duration
	CountDriftThreshold int64
	Tier3MaxDocsPerRun int64
	Tier3OffPeakStart  int
	Tier3OffPeakEnd    int
	JitterMaxSeconds int
	LeaderLockTTL   time.Duration
	LeaderHeartbeat time.Duration
	LeaderLockKey   string
	InstanceID string
}

func (c *ReconCoreConfig) applyDefaults() {
	if c.WindowSize <= 0 {
		c.WindowSize = 15 * time.Minute
	}
	if c.WindowLookback <= 0 {
		c.WindowLookback = 7 * 24 * time.Hour
	}
	if c.WindowFreezeMargin <= 0 {
		c.WindowFreezeMargin = 5 * time.Minute
	}
	if c.CountDriftThreshold <= 0 {
		c.CountDriftThreshold = 1
	}
	if c.Tier3MaxDocsPerRun <= 0 {
		c.Tier3MaxDocsPerRun = 10_000_000
	}
	if c.Tier3OffPeakStart == 0 && c.Tier3OffPeakEnd == 0 {
		c.Tier3OffPeakStart = 2
		c.Tier3OffPeakEnd = 5
	}
	if c.JitterMaxSeconds <= 0 {
		c.JitterMaxSeconds = 30
	}
	if c.LeaderLockTTL <= 0 {
		c.LeaderLockTTL = 60 * time.Second
	}
	if c.LeaderHeartbeat <= 0 {
		c.LeaderHeartbeat = 20 * time.Second
	}
	if c.LeaderLockKey == "" {
		c.LeaderLockKey = "recon:leader"
	}
	if c.InstanceID == "" {
		host, _ := os.Hostname()
		c.InstanceID = fmt.Sprintf("%s-%s", host, uuid.NewString()[:8])
	}
}

type ReconCore struct {
	sourceAgent *ReconSourceAgent
	destAgent   *ReconDestAgent
	masterAgent *ReconDestAgent
	shadowPlane   *gorm.DB
	masterPlane   *gorm.DB
	db            *gorm.DB
	mongoClient   *mongo.Client
	schemaAdapter *shadow.SchemaAdapter
	registryRepo  *reposource.RegistryRepo
	metadata metadata.MetadataRegistry
	redis         *rediscache.RedisCache
	cfg           ReconCoreConfig
	logger        *zap.Logger
}

func NewReconCore(
	sourceAgent *ReconSourceAgent,
	destAgent *ReconDestAgent,
	db *gorm.DB,
	mongoClient *mongo.Client,
	schemaAdapter *shadow.SchemaAdapter,
	registryRepo *reposource.RegistryRepo,
	logger *zap.Logger,
) *ReconCore {
	return NewReconCoreWithConfig(sourceAgent, destAgent, db, mongoClient, schemaAdapter, registryRepo, nil, ReconCoreConfig{}, logger)
}

func (rc *ReconCore) SetMetadataRegistry(metadata metadata.MetadataRegistry) {
	rc.metadata = metadata
}

func NewReconCoreWithConfig(
	sourceAgent *ReconSourceAgent,
	destAgent *ReconDestAgent,
	db *gorm.DB,
	mongoClient *mongo.Client,
	schemaAdapter *shadow.SchemaAdapter,
	registryRepo *reposource.RegistryRepo,
	redis *rediscache.RedisCache,
	cfg ReconCoreConfig,
	logger *zap.Logger,
) *ReconCore {
	cfg.applyDefaults()
	return &ReconCore{
		sourceAgent:   sourceAgent,
		destAgent:     destAgent,
		db:            db,
		mongoClient:   mongoClient,
		schemaAdapter: schemaAdapter,
		registryRepo:  registryRepo,
		redis:         redis,
		cfg:           cfg,
		logger:        logger,
	}
}

// --- Utilities ---

func TableGroupForTest(table string) string { return tableGroup(table) }
func DiffIDsForTest(a, b []string) ([]string, []string) { return diffIDs(a, b) }

func tableGroup(table string) string {
	idx := strings.Index(table, "_")
	if idx <= 0 {
		return "other"
	}
	return table[:idx]
}

type window struct {
	Lo time.Time
	Hi time.Time
}

func IsOffPeakForTest(start, end int, now time.Time) bool {
	rc := &ReconCore{cfg: ReconCoreConfig{Tier3OffPeakStart: start, Tier3OffPeakEnd: end}}
	return rc.isOffPeak(now)
}

func (rc *ReconCore) isOffPeak(now time.Time) bool {
	h := now.Hour()
	if rc.cfg.Tier3OffPeakStart <= rc.cfg.Tier3OffPeakEnd {
		return h >= rc.cfg.Tier3OffPeakStart && h < rc.cfg.Tier3OffPeakEnd
	}
	return h >= rc.cfg.Tier3OffPeakStart || h < rc.cfg.Tier3OffPeakEnd
}

func diffIDs(a, b []string) ([]string, []string) {
	setA := make(map[string]struct{}, len(a))
	for _, s := range a {
		setA[s] = struct{}{}
	}
	setB := make(map[string]struct{}, len(b))
	for _, s := range b {
		setB[s] = struct{}{}
	}
	var fromB, fromA []string
	for s := range setA {
		if _, ok := setB[s]; !ok {
			fromB = append(fromB, s)
		}
	}
	for s := range setB {
		if _, ok := setA[s]; !ok {
			fromA = append(fromA, s)
		}
	}
	return fromB, fromA
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func md5Hex(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func unwrapBSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case primitive.ObjectID:
		return val.Hex()
	case primitive.DateTime:
		return val.Time().Format(time.RFC3339Nano)
	case primitive.A:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = unwrapBSONValue(item)
		}
		return result
	case bson.M:
		result := make(map[string]interface{}, len(val))
		for k, item := range val {
			result[k] = unwrapBSONValue(item)
		}
		return result
	case bson.D:
		result := make(map[string]interface{}, len(val))
		for _, elem := range val {
			result[elem.Key] = unwrapBSONValue(elem.Value)
		}
		return result
	case int32:
		return int64(val)
	case primitive.Decimal128:
		return val.String()
	default:
		return v
	}
}

func fnvHash32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func advisoryLockKey(name string) int64 {
	return int64(crc32.ChecksumIEEE([]byte(name)))
}

var _ = fnvHash32
```

---

## 2. `recon_engine_run.go` (NEW)
Gom nhóm toàn bộ flow thực thi đối soát Segment A (Scheduled CheckAll, database run logging, and stale run cleanup daemon).

```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/model/system"
	"centralized-data-service/internal/service/governance"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"centralized-data-service/pkgs/metrics"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (rc *ReconCore) beginRun(ctx context.Context, table string, tier int) (*reconRunHandle, error) {
	h := &reconRunHandle{
		id:      uuid.NewString(),
		table:   table,
		tier:    tier,
		started: time.Now().UTC(),
	}
	insert := func() error {
		return rc.db.WithContext(ctx).Exec(
			`INSERT INTO cdc_system.recon_runs
				(id, table_name, tier, status, started_at, instance_id)
			 VALUES (?, ?, ?, 'running', ?, ?)`,
			h.id, table, tier, h.started, rc.cfg.InstanceID,
		).Error
	}
	err := insert()
	if err != nil && strings.Contains(err.Error(), "recon_runs_one_running") {
		res := rc.db.WithContext(ctx).Exec(
			`UPDATE cdc_system.recon_runs
			    SET status='cancelled', finished_at=NOW(),
			        error_message='stale running auto-cancelled by beginRun (worker restart?)'
			  WHERE table_name = ? AND status = 'running'
			    AND (started_at < NOW() - interval '5 minutes' OR instance_id <> ?)`,
			table, rc.cfg.InstanceID,
		)
		if res.Error == nil && res.RowsAffected > 0 {
			rc.logger.Warn("recon beginRun: auto-cancelled stale running run",
				zap.String("table", table), zap.Int64("cancelled", res.RowsAffected))
			err = insert()
		}
	}
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (rc *ReconCore) ReapStaleRuns(ctx context.Context) (int64, error) {
	res := rc.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.recon_runs
		    SET status='cancelled', finished_at=NOW(),
		        error_message='stale running reaped (worker restart / hung run)'
		  WHERE status='running'
		    AND started_at < NOW() - interval '15 minutes'`,
	)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		rc.logger.Warn("recon: reaped stale running recon_runs",
			zap.Int64("cancelled", res.RowsAffected))
	}
	return res.RowsAffected, nil
}

func (rc *ReconCore) ReapOrphanRunsFromDeadInstances(ctx context.Context) (int64, error) {
	res := rc.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.recon_runs
		    SET status='cancelled', finished_at=NOW(),
		        error_message='orphan from previous worker instance reaped at startup'
		  WHERE status='running'
		    AND instance_id IS DISTINCT FROM ?`,
		rc.cfg.InstanceID,
	)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		rc.logger.Warn("recon: reaped orphan running recon_runs from dead instances",
			zap.Int64("cancelled", res.RowsAffected),
			zap.String("current_instance", rc.cfg.InstanceID))
	}
	return res.RowsAffected, nil
}

func (rc *ReconCore) finishRun(ctx context.Context, h *reconRunHandle, status, errMsg string) {
	finished := time.Now().UTC()
	dur := finished.Sub(h.started).Seconds()
	errMsg = governance.SanitizeFreeformText(errMsg, 2000)

	updates := map[string]interface{}{
		"status":           status,
		"finished_at":      finished,
		"docs_scanned":     h.docsScanned,
		"windows_checked":  h.windowsCount,
		"mismatches_found": h.mismatches,
		"heal_actions":     h.healActions,
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if err := rc.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.recon_runs
		 SET status=?, finished_at=?, docs_scanned=?, windows_checked=?,
		     mismatches_found=?, heal_actions=?, error_message=?
		 WHERE id=?`,
		updates["status"], updates["finished_at"], updates["docs_scanned"],
		updates["windows_checked"], updates["mismatches_found"],
		updates["heal_actions"], errMsg, h.id,
	).Error; err != nil {
		rc.logger.Warn("finish recon_runs row failed", zap.Error(err))
	}

	group := tableGroup(h.table)
	metrics.ReconRunDuration.WithLabelValues(group, fmt.Sprintf("%d", h.tier)).Observe(dur)
	metrics.ReconMismatchCount.WithLabelValues(h.table, fmt.Sprintf("%d", h.tier)).Set(float64(h.mismatches))
	if status == "success" {
		metrics.ReconLastSuccessTs.WithLabelValues(h.table, fmt.Sprintf("%d", h.tier)).Set(float64(finished.Unix()))
	}
}

func (rc *ReconCore) CheckAll(ctx context.Context) []*system.ReconciliationReport {
	isLeader, release := rc.AcquireLeader(ctx)
	defer release()
	if !isLeader {
		rc.logger.Info("recon CheckAll — not leader, skipping")
		return nil
	}

	entries := rc.listActiveTableConfigs(ctx)
	if len(entries) == 0 {
		rc.logger.Error("recon CheckAll: registry load failed", zap.String("reason", "no active table configs"))
		return nil
	}

	runID := uuid.NewString()
	var (
		reports []*system.ReconciliationReport
		mu      sync.Mutex
		wg      sync.WaitGroup
	)
	now := time.Now()
	skippedNoSchema := 0

	const checkAllConcurrency = 8
	const perConnConcurrency = 2
	globalSem := make(chan struct{}, checkAllConcurrency)
	var connMu sync.Mutex
	connSems := make(map[string]chan struct{})
	connSem := func(url string) chan struct{} {
		connMu.Lock()
		defer connMu.Unlock()
		s, ok := connSems[url]
		if !ok {
			s = make(chan struct{}, perConnConcurrency)
			connSems[url] = s
		}
		return s
	}

	for _, entry := range entries {
		schemaName := entry.ShadowSchema
		if schemaName == "" {
			schemaName = "public"
		}
		if rc.schemaAdapter.GetSchemaInSchema(schemaName, entry.TargetTable) == nil {
			skippedNoSchema++
			rc.logger.Warn("recon CheckAll: skip table — schema introspect nil (not materialised?)",
				zap.String("schema", schemaName), zap.String("table", entry.TargetTable))
			continue
		}
		entry.RunID = runID
		e := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			globalSem <- struct{}{}
			defer func() { <-globalSem }()
			cs := connSem(e.SourceURL)
			cs <- struct{}{}
			defer func() { <-cs }()

			tableCtx, cancelTable := context.WithTimeout(ctx, 45*time.Second)
			defer cancelTable()
			report := rc.RunTier1(tableCtx, e)
			if report == nil {
				return
			}

			updates := map[string]interface{}{"last_recon_at": now}
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
			rc.db.Model(&source.TableRegistry{}).Where("target_table = ?", report.TargetTable).Updates(updates)

			mu.Lock()
			reports = append(reports, report)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(reports) == 0 {
		rc.logger.Warn("recon CheckAll: 0 tables checked — toàn bộ entries bị skip",
			zap.Int("entries", len(entries)),
			zap.Int("skipped_no_schema", skippedNoSchema),
			zap.String("fix_hint", "kiểm tra shadow_binding.shadow_schema + shadowDB wiring + bảng đã materialise chưa"))
	}
	return reports
}

func (rc *ReconCore) listActiveTableConfigs(ctx context.Context) []source.TableRegistry {
	if rc.metadata != nil {
		if items := rc.metadata.ListTableConfigs(); len(items) > 0 {
			return items
		}
	}
	if rc.registryRepo == nil {
		return nil
	}
	items, err := rc.registryRepo.GetAllActive(ctx)
	if err != nil {
		return nil
	}
	return items
}

func (rc *ReconCore) errorReport(entry source.TableRegistry, checkType string, tier int, err error) *system.ReconciliationReport {
	errMsg := governance.SanitizeFreeformText(err.Error(), 2000)
	code := classifyMongoError(err)
	if code == "" {
		code = ErrCodeUnknown
	}
	report := &system.ReconciliationReport{
		TargetTable:  entry.TargetTable,
		SourceDB:     entry.SourceDB,
		CheckType:    checkType,
		Status:       "error",
		Tier:         tier,
		ErrorMessage: &errMsg,
		ErrorCode:    code,
		CheckedAt:    time.Now().UTC(),
	}
	rc.stampA(report, entry)
	return report
}
```

---

## 3. `recon_engine_segment_b.go` (NEW)
Gom nhóm toàn bộ static config, refs, and setups cho transmute path đối soát (Segment B: Shadow ↔ Master).

```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/model/system"
	"context"

	"gorm.io/gorm"
)

const tierSegmentB = 5
const segmentShadowMaster = "shadow_master"
const segBPKColumn = "_gpay_id"
const segBDiffIDCap = 50_000

func (rc *ReconCore) SetMasterAgent(agent *ReconDestAgent) { rc.masterAgent = agent }

type MasterBindingRef struct {
	ID           int64  `gorm:"column:id"`
	MasterSchema string `gorm:"column:master_schema"`
	MasterTable  string `gorm:"column:master_table"`
	ShadowSchema string `gorm:"column:shadow_schema"`
	ShadowTable  string `gorm:"column:shadow_table"`
	RunID        string `gorm:"-"`
}

func (r MasterBindingRef) ShadowRel() string { return r.ShadowSchema + "." + r.ShadowTable }
func (r MasterBindingRef) MasterRel() string { return r.MasterSchema + "." + r.MasterTable }

func (rc *ReconCore) stampA(report *system.ReconciliationReport, entry source.TableRegistry) *system.ReconciliationReport {
	report.ShadowSchema, report.ShadowTable, report.RunID = entry.ShadowSchema, entry.TargetTable, entry.RunID
	rc.db.Create(report)
	return report
}

func (rc *ReconCore) stampB(report *system.ReconciliationReport, ref MasterBindingRef) *system.ReconciliationReport {
	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
	rc.db.Create(report)
	return report
}

func (r MasterBindingRef) runName() string { return r.MasterRel() }

func (rc *ReconCore) listActiveMasterBindings(ctx context.Context) []MasterBindingRef {
	var out []MasterBindingRef
	err := rc.db.WithContext(ctx).Raw(`
		SELECT mb.id, mb.master_schema, mb.master_table, sb.shadow_schema, sb.shadow_table
		  FROM cdc_system.master_binding mb
		  JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
		 WHERE mb.is_active = true AND mb.schema_status = 'approved'
		 ORDER BY mb.master_schema, mb.master_table`).Scan(&out).Error
	if err != nil {
		rc.logger.Error("recon segment B: list master bindings failed", zap.Error(err))
		return nil
	}
	return out
}

func (rc *ReconCore) SetPlaneDBs(shadowDB, masterDB *gorm.DB) {
	rc.shadowPlane = shadowDB
	rc.masterPlane = masterDB
}

type FieldDiff struct {
	GpayID   string `json:"gpay_id"`
	Column   string `json:"column"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

const rowDiffMaxIDs = 200
const rowDiffMaxEntries = 200
```
