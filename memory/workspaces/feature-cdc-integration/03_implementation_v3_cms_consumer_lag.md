# 03 — Implementation v3 (CMS) — Consumer Lag wiring vào System Health

> **Fix #4** trong `07_status_NOT_DELIVERED.md` (session 2026-04-17).
> **Muscle**: claude-opus-4-7[1m]
> **Scope**: chỉ CMS service, không touch Worker/FE.

---

## 0. Root cause

- kafka-exporter sidecar emit `kafka_consumergroup_lag{consumergroup,topic,partition}` trên `:9308/metrics`.
- AlertManager rule `HighConsumerLag` đã code sẵn trong `internal/service/system_health_alerts.go` (expect `snap.CDCPipeline["consumer_lag"].total_lag` với threshold > 10k warning, > 100k critical).
- **Gap**: `Collector.collectAndCache()` không populate field `consumer_lag` → rule không bao giờ fire.

---

## 1. Files touched

| File | Δ |
|---|---|
| `cdc-cms-service/config/config.go` | Thêm `SystemConfig.KafkaExporterURL` (mapstructure `kafkaExporterUrl`) |
| `cdc-cms-service/config/config-local.yml` | Thêm `system.kafkaExporterUrl: http://localhost:9308/metrics` |
| `cdc-cms-service/internal/service/system_health_collector.go` | Thêm probe `probeKafkaLag`, import expfmt+prommodel, register vào errgroup, thêm `CollectorConfig.KafkaExporterURL` + `LagTopicPrefix` (default `cdc.goopay.`) |
| `cdc-cms-service/internal/server/server.go` | Pass `cfg.System.KafkaExporterURL` vào `CollectorConfig` |
| `cdc-cms-service/internal/service/system_health_collector_test.go` | Thêm `TestDetectConditions_HighConsumerLag` 6 sub-case (zero, below, warning boundary, critical boundary, float, missing) |

---

## 2. Implementation details

### 2.1 probeKafkaLag

- `GET <KafkaExporterURL>` với timeout = `cfg.ProbeTimeout` (2s default).
- Parse Prometheus text format qua `expfmt.NewTextParser(prommodel.UTF8Validation)` (giống prom_client.go để tránh panic `Invalid name validation scheme`).
- Metric family: `kafka_consumergroup_lag`. Nếu family absent (kafka-exporter up nhưng no consumers) → status=ok, total_lag=0.
- Mỗi gauge sample:
  - Bỏ qua negative values (kafka-exporter báo -1 khi rebalancing).
  - Cộng vào `totalLag`.
  - Filter topic theo `LagTopicPrefix` để populate `per_topic`.
- Shape output:
  ```json
  {
    "status": "ok" | "down" | "unknown",
    "source": "kafka_exporter",
    "total_lag": 0,
    "per_topic": {"cdc.goopay.xxx": 0},
    "latency_ms": 52,
    "error": "..."   // khi fail
  }
  ```
- Fail-soft: tất cả lỗi (HTTP, parse, missing family) đều trả section với status `unknown` hoặc `down` — không block các probe khác (convention của Collector errgroup).

### 2.2 Alert rule (đã có sẵn)

`system_health_alerts.go:detectConditions()` đã xử lý đúng:

- Đọc `snap.CDCPipeline["consumer_lag"].(map[string]any)["total_lag"]`.
- Coerce qua `toFloat64` (nhận int/int64/float32/float64; default 0 nếu không match).
- `> 100_000` → critical, `> 10_000` → warning.

**Không cần sửa rule**. Probe viết `total_lag` dưới dạng `int64` → `toFloat64` nhận đúng.

### 2.3 Safety

- Timeout-bound (`ProbeTimeout` 2s).
- Error string qua `sanitizeErr()` (redact credentials trong URL).
- Khi `KafkaExporterURL=""` → status `unknown`, error message explicit, không crash.
- `toFloat64` default 0 giữ backward-compat với snapshots cũ (không fire spurious alert).

---

## 3. Verify

### 3.1 Build + unit test

```bash
cd cdc-cms-service && go build ./...              # OK
go test ./internal/service/... -v | grep -E "PASS|FAIL"
# TestDetectConditions_HighConsumerLag PASS (6/6 sub-cases)
# All existing tests PASS
go test ./...                                     # all packages OK
```

### 3.2 Runtime

```bash
curl -s http://localhost:8083/api/system/health | jq '.cdc_pipeline.consumer_lag'
```

Output:
```json
{
  "latency_ms": 52,
  "per_topic": {
    "cdc.goopay.centralized-export-service.export-jobs": 0,
    "cdc.goopay.payment-bill-service.refund-requests": 0
  },
  "source": "kafka_exporter",
  "status": "ok",
  "total_lag": 0
}
```

- Status `ok`, total_lag type = int (Go int64 → JSON number).
- Per-topic filter `cdc.goopay.*` hoạt động (2 topics từ kafka-exporter thật).
- Startup log clean (zero error liên quan consumer_lag).

### 3.3 Alert fire test

Thay vì inject real lag (rủi ro cao), cover thresholds qua unit test trực tiếp vào `detectConditions`:

| total_lag | Expected | Actual |
|---|---|---|
| 0 | no fire | no fire |
| 9,999 | no fire | no fire |
| 10,001 | HighConsumerLag warning | HighConsumerLag warning |
| 100,001 | HighConsumerLag critical | HighConsumerLag critical |
| 50,000.0 (float) | HighConsumerLag warning | HighConsumerLag warning |
| missing field | no fire | no fire |

Tất cả pass.

---

## 4. Remaining gaps (out-of-scope fix #4)

- Prometheus scrape config cho kafka-exporter chưa verify (nằm ngoài scope CMS).
- AlertManager bên ngoài (khi có Prometheus + Alertmanager thật chạy) sẽ đánh cùng threshold trên `cdc_pipeline_consumer_lag_total` — chưa implement metric export path từ CMS (task tương lai).

---

## 5. Lesson

- **Pattern**: Khi alert rule code-path tồn tại mà collector không populate input field → silent no-op.
- **Correct flow**: Với mỗi alert rule dựa vào snapshot field, PHẢI kiểm tra ngược lại collector đã set field đó chưa. Trong Go structure map-based `map[string]any`, thiếu field = `nil` = 0 after coerce = never fires.
- **Global pattern**: `[Component A emits metric M via exporter E, rule R reads snapshot field F] → thiếu populate F ở aggregator layer thì rule R im lặng forever`. Đúng flow: aggregator builder TEST-COVER mọi field rule consumer đọc.
