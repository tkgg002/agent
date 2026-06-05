# 10_gap_analysis — Gap Residual Sau Re-Audit 2026-06-01

## Phân loại Gap Residual

### P0 — Blocker còn lại (0 item)
Tất cả 4 P0 gốc đã được fix (G-1, G-2, G-3, G-4 đều L3).

### P1 — Release-impact (5 item)

#### G1-RES (P1) — Missing `cdc_kafka_consumer_offset` metric
- **Mô tả**: `scripts/smoke_failover.sh` chỉ verify final row COUNT sau restart, không có metric đo offset position trước/sau failover.
- **Risk**: Nếu consumer resume từ sai offset (duplicate hoặc gap), không có cách phát hiện ngoài count mismatch.
- **Fix demo**:
  ```go
  // pkgs/metrics/prometheus.go (APPEND)
  ConsumerOffset = promauto.NewGaugeVec(prometheus.GaugeOpts{
      Name: "cdc_kafka_consumer_offset",
      Help: "Last committed offset per topic/partition",
  }, []string{"topic", "partition", "group"})
  
  // internal/handler/kafka_consumer.go (hot loop, sau CommitMessages)
  metrics.ConsumerOffset.WithLabelValues(stats.Topic, stats.Partition, kc.groupID).Set(float64(stats.Offset))
  ```
- **Effort**: 1h.

#### G2-RES (P1) — WAL auto snapshot resume vắng mặt
- **Mô tả**: `alerts/wal_slot.yml` cảnh báo operator nhưng KHÔNG có logic tự động trigger snapshot back khi slot drop / WAL expire.
- **Risk**: Khi consumer down quá lâu (replication slot bị PG dọn), pipeline cần re-snapshot thủ công — RTO/RPO không định lượng được.
- **Fix demo**:
  ```go
  // internal/service/wal_monitor.go (NEW)
  func (w *WalMonitor) OnSlotExpired(ctx, srcID) {
      log.Warn("WAL slot expired, triggering snapshot resume", "srcID", srcID)
      metrics.SnapshotResumeTriggered.WithLabelValues("wal_expire").Inc()
      return w.snapshotSvc.TriggerImmediate(ctx, srcID, SnapshotReasonWALExpire)
  }
  ```
- **Effort**: 4h (NEW component + integration + test).

#### G3-RES (P1) — G-NEW-24 test file FAKE
- **Mô tả**: `report_execute_remaining_gaps_2026-05-27.md` claim "metrics_callback_test.go 2 unit test PASS" nhưng `pkgs/database/metrics_callback_test.go` KHÔNG TỒN TẠI.
- **Risk**: 6 GORM verb callback (Query, Create, Update, Delete, Row, Raw) không có test coverage — silent regression nếu GORM API đổi.
- **Fix demo**:
  ```go
  // pkgs/database/metrics_callback_test.go (NEW)
  func TestRegisterQueryMetrics_RecordsHistogram(t *testing.T) {
      db := openTestDB(t)
      err := RegisterQueryMetrics(db, "test_role")
      require.NoError(t, err)
      db.Exec("SELECT 1")
      // Assert histogram has observation:
      count, _ := testutil.GatherAndCount(prometheus.DefaultGatherer, "cdc_source_query_duration_seconds")
      require.Greater(t, count, 0)
  }
  
  func TestRegisterQueryMetrics_RejectsBadInput(t *testing.T) {
      require.Error(t, RegisterQueryMetrics(nil, "role"))
      require.Error(t, RegisterQueryMetrics(openTestDB(t), ""))
  }
  ```
- **Effort**: 2h.

#### G4-RES (P1) — k6 load test sai target
- **Mô tả**: `scripts/load_test.js` chỉ HTTP GET `http://localhost:8080/metrics` thay vì generate CDC events vào source DB và đo end-to-end lag.
- **Risk**: Kết quả k6 không phản ánh production throughput (50vus scrape /metrics chỉ test Prometheus handler, không test ingest pipeline).
- **Fix demo**: viết k6 script mới:
  ```js
  // scripts/load_test_cdc.js
  import sql from 'k6/x/sql';
  
  export const options = {
    stages: [
      { duration: '2m', target: 100 },  // ramp insert TPS
      { duration: '5m', target: 100 },  // sustain
      { duration: '1m', target: 0 },
    ],
  };
  
  const db = sql.open('postgres', __ENV.SOURCE_DSN);
  
  export default function () {
    const id = randomString(16);
    db.exec(`INSERT INTO orders(id, amount) VALUES ($1, $2)`, [id, Math.random() * 1000]);
    // wait + verify in shadow:
    const shadow = db.query(`SELECT 1 FROM cdc_shadow.orders WHERE _gpay_source_id=$1`, [id]);
    check(shadow, { 'cdc replicated < 5s': (r) => r.length > 0 });
  }
  ```
- **Effort**: 3h (k6-sql extension + 2 DSN env + assert lag).

#### G5-RES (P1) — Cross-cut: 2 FAIL test cdc-cms-service mapping_rule
- **Mô tả**: Pre-existing regression — validation message đổi từ `'status is required'` → `'status or data_type is required'` nhưng 2 test chưa update.
- **Files**:
  - `cdc-cms-service/test/internal/api/mapping_rule_handler_test.go:90`
  - `cdc-cms-service/test/internal/app/commands/sync_metadata_test.go:40`
- **Fix**: 1-line edit per test, update assertion string.
- **Effort**: 15 min.
- **Note**: Không thuộc 16 gap nhưng block CI green.

### P2 — Backlog (3 item)

#### G6-RES (P2) — G-15 Chaos test không portable
- **Mô tả**: `scripts/chaos_network.sh` dùng `iptables DROP` cần sudo, không chạy trong container CI Alpine/Debian tiêu chuẩn không có `NET_ADMIN`.
- **Risk**: Test không thực sự chạy trong CI → false confidence.
- **Fix demo**: chuyển sang toxiproxy hoặc pumba:
  ```bash
  # scripts/chaos_network.sh (REWRITE)
  docker run --rm \
    --network cdc_default \
    -v /var/run/docker.sock:/var/run/docker.sock \
    gaiaadm/pumba netem --duration ${DURATION_SEC}s loss --percent 50 cdc-worker
  ```
- **Effort**: 2h (replace iptables + CI compose update).

#### G7-RES (P2) — G-12 Adaptive batch ngược chiều cho overload
- **Mô tả**: Adaptive batch hiện tăng size khi lag cao (burst-up). Nếu DB đang overloaded (lock contention, IOPS saturated), việc tăng batch sẽ làm tệ hơn.
- **Risk**: Cascading slowdown — không có circuit ngược chiều khi destination unhealthy.
- **Fix demo**: thêm health check before burst:
  ```go
  // internal/handler/kafka_consumer.go
  if ab.shouldBurst(currentLag) {
      if !ab.destHealthCheck.IsHealthy() {
          ab.currentSize = ab.baseBatchSize  // throttle instead of burst
          metrics.BurstThrottled.Inc()
          return
      }
      ab.currentSize = ab.baseBatchSize * mult
  }
  ```
- **Effort**: 3h.

#### G8-RES (P2) — G-9 Path documentation drift
- **Mô tả**: Report `report_execute_remaining_gaps_2026-05-27.md` claim path `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` nhưng file thực tại `cdc-cms-service/test/internal/app/commands/approve_schema_proposal_integration_test.go`.
- **Risk**: CI lookup theo path cũ sẽ miss test.
- **Fix**: APPEND correction note vào `05_progress.md` của workspace `plan-cdc-qa-gap-fix-2026-05-27` (do §11 không sửa entry cũ).
- **Effort**: 5 min.

## Lỗi Audit Gốc Đã Correction (không phải gap fix)
- **G-11** audit gốc claim "BatchesFlushed là NewCounter không label" → SAI. Re-verify đã LÀ NewCounterVec[sink,table] từ trước fix. Đã correction trong remaining gaps report Entry 14.
- **G-13** audit gốc claim "NewPerSourcePool dead code" → SAI. Re-verify đã wired đầy đủ. Đã correction.

## Pre-Existing Failures (Resolved)
- `TestSanitizeMongoDSN` 4 case FAIL → RESOLVED (test bị remove/rename).
- `internal/handler` kafka-go goleak FAIL → RESOLVED (test/internal/handler PASS với goleak.VerifyTestMain).

## Effort Tổng Cộng để đạt 87.5% Target

| Gap Residual | Effort |
|---|---|
| G1-RES ConsumerOffset metric | 1h |
| G2-RES WAL auto snapshot resume | 4h |
| G3-RES G-NEW-24 test file | 2h |
| G4-RES k6 CDC data path | 3h |
| G5-RES cms mapping_rule 2 FAIL | 0.25h |
| G6-RES chaos pumba | 2h |
| G7-RES adaptive throttle-down | 3h |
| G8-RES path doc drift | 0.1h |
| **TOTAL** | **~15.5h** |

Nếu thực hiện hết: score dự kiến **56/64 (87.5%)** — đạt target plan 2026-05-27.

## Verdict Tổng Quan

| Cluster | Status |
|---|---|
| 4 P0 gốc | ✅ tất cả FIXED |
| 5 P1 gốc | 2 FIXED + 3 PARTIAL |
| 7 P2 gốc | 4 FIXED + 3 PARTIAL |
| 3 G-NEW | 2 FIXED + 1 FAKE-PARTIAL (test missing) |
| Build/Vet | ✅ 3/3 service PASS |
| Test | ⚠ 1 regression mới ngoài scope (mapping_rule) |
