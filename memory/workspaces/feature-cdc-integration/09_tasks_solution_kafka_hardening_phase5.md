# Kafka Hardening Playbook — Phase 5 (DevOps coord)

> **Date**: 2026-04-17
> **Author**: Brain (claude-opus-4-7)
> **Purpose**: Script + runbook cho DevOps thực hiện Phase 5 (Kafka config hardening + lag monitoring). Phase này cần change infra (topic config, có thể deploy sidecar) → cần DevOps approval + maintenance window.
> **Reference**: `02_plan_data_integrity_v3.md` §8, `02_plan_observability_v3.md` §8.

---

## 1. Overview

**Mục đích**: Thay `cleanup.policy=compact` hiện tại (v2 plan sai) → `cleanup.policy=delete` với long retention. Deploy `kafka_exporter` sidecar cho consumer lag metrics.

**Risk**: Change topic config → log cleaner re-compute → có thể tăng I/O tạm thời. Phải áp dụng off-peak.

**Prerequisite**:
- [ ] DevOps approval.
- [ ] Maintenance window 30-60 phút off-peak (2-5 AM).
- [ ] Backup Kafka log dirs (snapshot if possible).
- [ ] Verify disk có đủ capacity cho 14-day retention × traffic estimate (~2000 events/s × 14d × avg 2KB = ~5 TB worst case — DevOps sizing needed).

---

## 2. Task Breakdown

### P5-1: Inventory CDC topics
```bash
# List tất cả topics bắt đầu cdc.goopay.*
docker exec gpay-kafka kafka-topics \
  --bootstrap-server localhost:9092 \
  --list | grep '^cdc\.goopay\.' | tee /tmp/cdc_topics.txt

# Alternative: từ cdc_table_registry
docker exec gpay-postgres psql -U postgres -d goopay_dw -c \
  "SELECT DISTINCT 'cdc.goopay.' || source_service || '.' || table_name AS topic FROM cdc_table_registry WHERE is_active = true"
```

**Deliverable**: `/tmp/cdc_topics.txt` list.

### P5-2: Describe current config (pre-change snapshot)
```bash
for topic in $(cat /tmp/cdc_topics.txt); do
  echo "=== $topic ==="
  docker exec gpay-kafka kafka-configs \
    --bootstrap-server localhost:9092 \
    --entity-type topics \
    --entity-name "$topic" \
    --describe
done | tee /tmp/kafka_config_before.log
```

**Deliverable**: `/tmp/kafka_config_before.log`.

### P5-3: Apply new config (ĐÂY LÀ STEP CHÍNH)

#### Dry run (verify syntax trước)
```bash
TOPIC="cdc.goopay.payment-service.transactions"  # 1 topic test trước
docker exec gpay-kafka kafka-configs \
  --bootstrap-server localhost:9092 \
  --alter \
  --entity-type topics \
  --entity-name "$TOPIC" \
  --add-config "cleanup.policy=delete,retention.ms=1209600000,retention.bytes=107374182400,segment.ms=86400000,min.insync.replicas=1"
```

- `cleanup.policy=delete` — thay compact
- `retention.ms=1209600000` — 14 days
- `retention.bytes=107374182400` — 100 GB
- `segment.ms=86400000` — 1 day segments (help cleanup + recovery)
- `min.insync.replicas=1` — single broker; tăng = 2 khi có multi-broker

**VERIFY**: Producer vẫn publish được, Consumer vẫn consume:
```bash
# Producer test
docker exec gpay-kafka kafka-console-producer --broker-list localhost:9092 --topic "$TOPIC" <<< '{"test":1}'

# Consumer peek
docker exec gpay-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic "$TOPIC" --from-beginning --max-messages 1 --timeout-ms 5000
```

#### Bulk apply (sau khi dry run OK)
```bash
for topic in $(cat /tmp/cdc_topics.txt); do
  echo "Applying to $topic..."
  docker exec gpay-kafka kafka-configs \
    --bootstrap-server localhost:9092 \
    --alter \
    --entity-type topics \
    --entity-name "$topic" \
    --add-config "cleanup.policy=delete,retention.ms=1209600000,retention.bytes=107374182400,segment.ms=86400000,min.insync.replicas=1"
  sleep 2  # throttle
done | tee /tmp/kafka_config_apply.log
```

**Watch during**: broker CPU/IO metrics — nếu spike > 80% → pause, apply slower.

### P5-4: Describe post-change (verify)
```bash
for topic in $(cat /tmp/cdc_topics.txt); do
  echo "=== $topic ==="
  docker exec gpay-kafka kafka-configs \
    --bootstrap-server localhost:9092 \
    --entity-type topics \
    --entity-name "$topic" \
    --describe | grep -E "cleanup.policy|retention"
done | tee /tmp/kafka_config_after.log

# Diff to confirm all updated
diff /tmp/kafka_config_before.log /tmp/kafka_config_after.log
```

**AC**: Tất cả topics hiện `cleanup.policy=delete, retention.ms=1209600000, retention.bytes=107374182400`.

### P5-5: Schema History Topic — KEEP compact (quan trọng)
```bash
# Debezium schema history topic (thường tên `_schemas` hoặc `dbserver1.debezium_history`)
SCHEMA_HISTORY_TOPIC="_schemas"  # xác minh tên qua `kafka-topics --list`

docker exec gpay-kafka kafka-configs \
  --bootstrap-server localhost:9092 \
  --alter \
  --entity-type topics \
  --entity-name "$SCHEMA_HISTORY_TOPIC" \
  --add-config "cleanup.policy=compact,retention.ms=-1"  # unlimited retention
```

Lý do: schema history là state-store, KHÔNG phải event stream → compact đúng.

### P5-6: Deploy `kafka_exporter` sidecar

#### Add to `docker-compose.yml`
```yaml
  kafka-exporter:
    image: danielqsj/kafka-exporter:v1.7.0
    container_name: gpay-kafka-exporter
    command:
      - '--kafka.server=kafka:9092'
      - '--web.listen-address=:9308'
      - '--topic.filter=^cdc\..*'
      - '--group.filter=.*'
    ports:
      - "9308:9308"
    networks:
      - cdc-network
    depends_on:
      - kafka
    restart: unless-stopped
```

```bash
docker compose up -d kafka-exporter
```

#### Verify metrics
```bash
curl -s localhost:9308/metrics | grep kafka_consumergroup_lag | head -20
```

Expected metrics:
- `kafka_consumergroup_lag{consumergroup, topic, partition}`
- `kafka_consumergroup_current_offset{...}`
- `kafka_topic_partition_current_offset{...}`

### P5-7: Prometheus scrape config
```yaml
# infra/prometheus/prometheus.yml (add)
scrape_configs:
  - job_name: kafka-exporter
    static_configs:
      - targets: ['kafka-exporter:9308']
    scrape_interval: 15s
```

### P5-8: Apply alert rules (from SLO doc)
```yaml
# infra/prometheus/alert_rules.yml
groups:
  - name: cdc_kafka_retention
    rules:
      - alert: KafkaConsumerLagApproachingRetention
        expr: |
          (
            kafka_consumergroup_lag 
            * on(topic) group_left() kafka_topic_partition_current_offset
          ) / kafka_topic_partition_oldest_offset > 0.7
        for: 5m
        labels: { severity: warning, slo: "5" }
        annotations:
          summary: "Consumer {{ $labels.consumergroup }} lag tiến gần retention (70%)"
      
      - alert: KafkaConsumerLagCritical  
        expr: |
          (
            kafka_consumergroup_lag 
            * on(topic) group_left() kafka_topic_partition_current_offset
          ) / kafka_topic_partition_oldest_offset > 0.9
        for: 5m
        labels: { severity: critical, slo: "5" }
```

Reload Prometheus:
```bash
curl -X POST http://prometheus:9090/-/reload
```

### P5-9: Prometheus recording rules (optimization)
```yaml
# infra/prometheus/recording_rules.yml
groups:
  - name: cdc_kafka_precompute
    interval: 30s
    rules:
      - record: cdc:kafka_lag_seconds:by_topic
        expr: |
          sum by (topic) (
            kafka_consumergroup_lag 
            / on(topic) group_left() 
            (rate(kafka_topic_partition_current_offset[5m]))
          )
```

---

## 3. Rollback plan

### Nếu broker unstable sau change
```bash
# Revert retention config
for topic in $(cat /tmp/cdc_topics.txt); do
  docker exec gpay-kafka kafka-configs \
    --bootstrap-server localhost:9092 \
    --alter \
    --entity-type topics \
    --entity-name "$topic" \
    --delete-config "cleanup.policy,retention.ms,retention.bytes,segment.ms"
done
```

### Remove kafka_exporter
```bash
docker compose stop kafka-exporter
docker compose rm -f kafka-exporter
# Remove block from docker-compose.yml
```

---

## 4. Validation Checklist

- [ ] P5-1: `/tmp/cdc_topics.txt` có danh sách đúng.
- [ ] P5-2: Pre-change snapshot captured.
- [ ] P5-3: All CDC topics applied new config (verified P5-4 diff).
- [ ] P5-4: All topics `cleanup.policy=delete`.
- [ ] P5-5: Schema history topic KEEP compact.
- [ ] P5-6: kafka-exporter container running, metrics exposed.
- [ ] P5-7: Prometheus scraping kafka-exporter successfully.
- [ ] P5-8: Alert rules loaded (`/api/v1/rules` confirm).
- [ ] P5-9: Recording rules generating data.
- [ ] Runtime 1 giờ sau: Worker consume tốc độ bình thường, broker CPU stable, no data loss events in `failed_sync_logs`.

---

## 5. Post-deploy monitoring (24h)

Watch:
- Broker disk usage (retention tăng → có thể tăng disk usage).
- Log cleaner CPU (compact bỏ → giảm).
- Consumer lag trong normal range.
- Alert fire count (expect 0 false positives với threshold 70/90%).

Nếu alert fire false positive > 2× trong 24h → điều chỉnh threshold trong SLO doc.

---

## 6. Effort estimate

| Task | Effort | Owner |
|:-----|:-------|:------|
| P5-1..4 config change | 1h | DevOps |
| P5-5 schema history keep | 10m | DevOps |
| P5-6 kafka_exporter deploy | 30m | DevOps |
| P5-7..9 Prom config + rules | 1h | DevOps |
| Validation | 1h | DevOps + Muscle |
| **Total** | **4h** | |
| Post-deploy watch | 24h | Oncall |

---

## 7. Communication

**Who to notify**:
- Before: CDC team channel + DevOps + downstream consumers (FYI).
- During: status update every 30 phút trong maintenance window.
- After: retrospective — disk usage, lag baseline, any incident.

---

## 8. Known unknowns

- Topic `cdc.goopay.*` pattern có cover hết không? Validate P5-1 output vs `cdc_table_registry`.
- `min.insync.replicas` hiện prod = mấy broker? 1 broker = dùng 1; 3 broker = có thể tăng 2 cho durability.
- Disk headroom prod? Need DevOps confirm.
- `delete.retention.ms` cho schema history — default 24h — enough? Consider tăng 7 ngày cho safety.
