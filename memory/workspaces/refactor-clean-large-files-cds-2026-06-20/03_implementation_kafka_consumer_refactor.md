# Technical Design - Phân rã file kafka_consumer.go

Tài liệu này cung cấp thiết kế kỹ thuật chi tiết cùng giải pháp cụ thể (phân chia code) cho việc tái cấu trúc file `kafka_consumer.go` thành các file nhỏ hơn thuộc cùng package `shadow`.

## Nguyên tắc thiết kế

1. **Simplicity First & Minimal Impact**: Không thay đổi bất kỳ signature public nào của `KafkaConsumer`, giữ nguyên cơ chế hoạt động của toàn bộ hệ thống.
2. **Không thay đổi Package**: Tất cả các file mới được tạo ra đều thuộc `package shadow`.
3. **Phân rã theo trách nhiệm**:
   - Quản lý Batch & Adaptive Batcher -> `adaptive_batcher.go`
   - Giải mã Avro & Schema Registry -> `avro_helper.go`
   - Xử lý Dead Letter Queue (DLQ) -> `dlq_helper.go`
   - Khảo sát & Làm mới Topic -> `topic_helper.go`
   - Hàm tiện ích (Tracing, Error Handling) -> `utils.go`
   - Logic chính của Consumer -> `kafka_consumer.go`

---

## Chi tiết các File sau khi phân tách

### 1. File `adaptive_batcher.go` (Logic Batching & Adaptive)
File này sẽ chứa struct `adaptiveBatcher`, `batchStats` và các phương thức liên quan của `KafkaConsumer`:

```go
package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"centralized-data-service/internal/activity"
	"centralized-data-service/internal/model/system"
	"centralized-data-service/pkgs/metrics"

	"go.uber.org/zap"
)

// batchStats tracks per-topic batch metrics for Activity Log
type batchStats struct {
	topic        string
	processed    int
	success      int
	failed       int
	rowsAffected int
	startTime    time.Time
}

type adaptiveBatcher struct {
	baseBatchSize int
	currentSize   int
	lastAdjust    time.Time
	lagThreshold  int64 // events
	maxMultiplier int
	mu            sync.RWMutex
	destHealth    func() bool
}

func (ab *adaptiveBatcher) adjust(currentLag int64) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.destHealth != nil && !ab.destHealth() {
		ab.currentSize = ab.baseBatchSize
		metrics.BurstModeActive.Set(0)
		metrics.DestThrottledTotal.WithLabelValues("dest_unhealthy").Inc()
		ab.lastAdjust = time.Now()
		return
	}

	if time.Since(ab.lastAdjust) < 30*time.Second {
		return
	}
	if currentLag > ab.lagThreshold {
		mult := int(currentLag / ab.lagThreshold)
		if mult > ab.maxMultiplier {
			mult = ab.maxMultiplier
		}
		if mult < 2 {
			mult = 2
		}
		ab.currentSize = ab.baseBatchSize * mult
		metrics.BurstModeActive.Set(1)
	} else {
		ab.currentSize = ab.baseBatchSize
		metrics.BurstModeActive.Set(0)
	}
	ab.lastAdjust = time.Now()
}

func (ab *adaptiveBatcher) getSize() int {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	if ab.currentSize <= 0 {
		return ab.baseBatchSize
	}
	return ab.currentSize
}

// getOrCreateBatch returns the batch stats for a topic, creating if needed
func (kc *KafkaConsumer) getOrCreateBatch(topic string) *batchStats {
	if b, ok := kc.batches[topic]; ok {
		return b
	}
	b := &batchStats{topic: topic, startTime: time.Now()}
	kc.batches[topic] = b
	return b
}

// flushBatch writes batch Activity Log for a single topic and resets
func (kc *KafkaConsumer) flushBatch(ctx context.Context, topic string) {
	b, ok := kc.batches[topic]
	if !ok || b.processed == 0 {
		return
	}

	durationMs := int(time.Since(b.startTime).Milliseconds())
	details, _ := json.Marshal(map[string]interface{}{
		"topic":         b.topic,
		"processed":     b.processed,
		"success":       b.success,
		"failed":        b.failed,
		"rows_affected": b.rowsAffected,
	})

	now := time.Now()
	entry := &system.ActivityLog{
		Operation:    activity.OperationKafkaConsumeBatch.String(),
		TargetTable:  b.topic,
		Status:       "success",
		RowsAffected: int64(b.rowsAffected),
		DurationMs:   &durationMs,
		Details:      details,
		TriggeredBy:  activity.TriggeredByKafkaConsumer.String(),
		StartedAt:    b.startTime,
		CompletedAt:  &now,
	}

	kc.runPostConsumeAction(ctx, b, entry, now)

	delete(kc.batches, topic)
}

// flushAllBatches flushes batch stats for all topics
func (kc *KafkaConsumer) flushAllBatches(ctx context.Context) {
	for topic := range kc.batches {
		kc.flushBatch(ctx, topic)
	}
	if kc.eventHandler != nil {
		kc.eventHandler.FlushDrift(ctx)
	}
}

func (kc *KafkaConsumer) runPostConsumeAction(ctx context.Context, b *batchStats, entry *system.ActivityLog, completedAt time.Time) {
	if kc.postConsumeAction == nil {
		return
	}
	actionName := kc.postConsumeActionName
	if actionName == "" {
		actionName = activity.OperationKafkaPostConsumeAction.String()
	}
	event := KafkaPostConsumeEvent{
		Topic:        b.topic,
		Processed:    b.processed,
		Success:      b.success,
		Failed:       b.failed,
		TriggeredBy:  activity.TriggeredByKafkaConsumer,
		ActivityID:   entry.ID,
		ActivityRows: entry.RowsAffected,
		StartedAt:    b.startTime,
		CompletedAt:  completedAt,
	}
	actionStart := time.Now()
	batchDurationMs := completedAt.Sub(b.startTime).Milliseconds()
	if err := kc.postConsumeAction(ctx, event); err != nil {
		actionMs := time.Since(actionStart).Milliseconds()
		errType := classifyKafkaErr(err)
		kc.logger.Warn(fmt.Sprintf("kafka post-consume action failed component=kafka_consumer op=post_consume_action action=%s topic=%s processed=%d success=%d failed=%d batch_duration_ms=%d action_duration_ms=%d err_type=%s err=%s",
			actionName, event.Topic, event.Processed, event.Success, event.Failed, batchDurationMs, actionMs, errType, err.Error()),
			zap.String("component", "kafka_consumer"),
			zap.String("op", "post_consume_action"),
			zap.String("triggered_by", event.TriggeredBy.String()),
			zap.String("operation", activity.OperationKafkaConsumeBatch.String()),
			zap.String("action", actionName),
			zap.String("topic", event.Topic),
			zap.Int("processed", event.Processed),
			zap.Int("success", event.Success),
			zap.Int("failed", event.Failed),
			zap.Int64("batch_duration_ms", batchDurationMs),
			zap.Int64("action_duration_ms", actionMs),
			zap.String("err_type", errType),
			zap.Error(err),
		)
		return
	}
	actionMs := time.Since(actionStart).Milliseconds()
	throughput := 0.0
	if batchDurationMs > 0 {
		throughput = float64(event.Processed) / (float64(batchDurationMs) / 1000.0)
	}
	kc.logger.Info(fmt.Sprintf("kafka post-consume action completed component=kafka_consumer op=post_consume_action action=%s topic=%s processed=%d success=%d failed=%d batch_duration_ms=%d action_duration_ms=%d throughput_msg_per_sec=%.2f",
		actionName, event.Topic, event.Processed, event.Success, event.Failed, batchDurationMs, actionMs, throughput),
		zap.String("component", "kafka_consumer"),
		zap.String("op", "post_consume_action"),
		zap.String("triggered_by", event.TriggeredBy.String()),
		zap.String("operation", activity.OperationKafkaConsumeBatch.String()),
		zap.String("action", actionName),
		zap.String("topic", event.Topic),
		zap.Int("processed", event.Processed),
		zap.Int("success", event.Success),
		zap.Int("failed", event.Failed),
		zap.Int64("batch_duration_ms", batchDurationMs),
		zap.Int64("action_duration_ms", actionMs),
		zap.Float64("throughput_msg_per_sec", throughput),
	)
}
```

### 2. File `avro_helper.go` (Avro Codec & Unwrap)
Chứa các hàm liên quan đến Schema Registry và Avro:

```go
package shadow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkedin/goavro/v2"
	"go.uber.org/zap"
)

// avro schema cache
var schemaCache = make(map[int32]*goavro.Codec)

// getAvroCodec fetches Avro schema from Schema Registry by ID and caches it
func (kc *KafkaConsumer) getAvroCodec(schemaID int32) (*goavro.Codec, error) {
	if codec, ok := schemaCache[schemaID]; ok {
		return codec, nil
	}

	url := fmt.Sprintf("%s/schemas/ids/%d", kc.config.SchemaRegistryURL, schemaID)
	client := &http.Client{Timeout: 5 * time.Second}
	fetchStart := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch schema: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("schema registry %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse schema response: %w", err)
	}

	sanitizedSchema := sanitizeAvroSchemaNames(result.Schema)

	codec, err := goavro.NewCodec(sanitizedSchema)
	if err != nil {
		return nil, fmt.Errorf("create avro codec: %w", err)
	}

	schemaCache[schemaID] = codec
	fetchMs := time.Since(fetchStart).Milliseconds()
	kc.logger.Info(fmt.Sprintf("avro schema cached component=kafka_consumer op=fetch_schema schema_id=%d registry_url=%s fetch_duration_ms=%d schema_bytes=%d cache_size=%d",
		schemaID, kc.config.SchemaRegistryURL, fetchMs, len(result.Schema), len(schemaCache)),
		zap.String("component", "kafka_consumer"),
		zap.String("op", "fetch_schema"),
		zap.Int32("schema_id", schemaID),
		zap.String("registry_url", kc.config.SchemaRegistryURL),
		zap.Int64("fetch_duration_ms", fetchMs),
		zap.Int("schema_bytes", len(result.Schema)),
		zap.Int("cache_size", len(schemaCache)))
	return codec, nil
}

// unwrapAvroUnion extracts value from goavro union type.
func unwrapAvroUnion(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok && len(m) == 1 {
		for _, val := range m {
			return val
		}
	}
	return v
}

// unwrapAvroUnionMap applies unwrapAvroUnion to every top-level value in a map.
func unwrapAvroUnionMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = unwrapAvroUnion(v)
	}
	return out
}

// sanitizeAvroSchemaNames replaces invalid chars in Avro name/namespace fields
func sanitizeAvroSchemaNames(schema string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		return strings.ReplaceAll(schema, "-", "_")
	}
	fixNames(parsed)
	fixed, _ := json.Marshal(parsed)
	return string(fixed)
}

func fixNames(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, v2 := range val {
			if k == "name" || k == "namespace" {
				if s, ok := v2.(string); ok {
					val[k] = strings.ReplaceAll(s, "-", "_")
				}
			}
			fixNames(v2)
		}
	case []interface{}:
		for _, item := range val {
			fixNames(item)
		}
	}
}

// UnwrapAvroUnionForTest exposes unwrapAvroUnion for the test suite.
func UnwrapAvroUnionForTest(v interface{}) interface{} { return unwrapAvroUnion(v) }

// UnwrapAvroUnionMapForTest exposes unwrapAvroUnionMap for the test suite.
func UnwrapAvroUnionMapForTest(m map[string]interface{}) map[string]interface{} {
	return unwrapAvroUnionMap(m)
}
```

### 3. File `dlq_helper.go` (Dead Letter Queue & Diagnostics)
Chứa các hàm liên quan đến failed sync logs (DLQ):

```go
package shadow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"centralized-data-service/internal/model/shadow"
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/pkgs/metrics"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// writeDLQ persists a failed Kafka message into failed_sync_logs.
func (kc *KafkaConsumer) writeDLQ(ctx context.Context, msg kafka.Message, procErr error) error {
	var targetTable string
	parts := strings.Split(msg.Topic, ".")
	if len(parts) >= 4 {
		targetTable = parts[3]
	}

	recordID, operation, rawJSON := extractDLQMetadata(msg)
	rawJSON = kc.sanitizeDLQRawJSON(targetTable, rawJSON)

	errorType := "processing"
	status := "failed"
	errText := procErr.Error()
	sanitizedErrText := governance.SanitizeFreeformText(errText, 2000)
	if strings.Contains(errText, "schema_drift") {
		errorType = "schema_drift"
		status = "pending"
	} else if strings.Contains(errText, "missing_required_field") {
		errorType = "missing_required"
		status = "pending"
	}

	partition := msg.Partition
	offset := msg.Offset
	row := shadow.FailedSyncLog{
		TargetTable:    targetTable,
		SourceTable:    targetTable,
		RecordID:       recordID,
		Operation:      operation,
		RawJSON:        rawJSON,
		ErrorMessage:   sanitizedErrText,
		ErrorType:      errorType,
		KafkaTopic:     msg.Topic,
		KafkaPartition: &partition,
		KafkaOffset:    &offset,
		RetryCount:     0,
		MaxRetries:     5,
		Status:         status,
	}
	return kc.db.WithContext(ctx).Create(&row).Error
}

// extractDLQMetadata pulls (recordID, operation, rawJSON) out of a Kafka message.
func extractDLQMetadata(msg kafka.Message) (string, string, []byte) {
	var recordID string
	if len(msg.Key) > 0 {
		var keyMap map[string]interface{}
		if err := json.Unmarshal(msg.Key, &keyMap); err == nil {
			if v, ok := keyMap["id"]; ok {
				recordID = fmt.Sprintf("%v", v)
			}
		}
		if recordID == "" {
			safeKey := bytes.ReplaceAll(msg.Key, []byte{0}, []byte{})
			recordID = strings.Trim(string(safeKey), `"`)
		}
	}

	var operation string
	if len(msg.Value) > 0 {
		var v map[string]interface{}
		if err := json.Unmarshal(msg.Value, &v); err == nil {
			if op, ok := v["op"].(string); ok {
				operation = op
			}
			if recordID == "" {
				if after, ok := v["after"].(map[string]interface{}); ok {
					if id, ok := after["_id"]; ok {
						recordID = fmt.Sprintf("%v", id)
					}
				}
				if recordID == "" {
					if after, ok := v["after"].(string); ok {
						var afterMap map[string]interface{}
						if err := json.Unmarshal([]byte(after), &afterMap); err == nil {
							if id, ok := afterMap["_id"]; ok {
								recordID = fmt.Sprintf("%v", id)
							}
						}
					}
				}
			}
		}
	}

	raw := msg.Value
	if !json.Valid(raw) || !utf8.Valid(raw) {
		encoded := base64.StdEncoding.EncodeToString(raw)
		wrapped, _ := json.Marshal(map[string]string{
			"raw_base64": encoded,
			"encoding":   "base64",
			"note":       "original payload contained binary or invalid utf8",
		})
		raw = wrapped
	}
	return recordID, operation, raw
}

func (kc *KafkaConsumer) sanitizeDLQRawJSON(table string, raw []byte) json.RawMessage {
	if kc.masking != nil {
		return kc.masking.MaskJSONPayload(table, raw)
	}
	if json.Valid(raw) {
		return json.RawMessage(raw)
	}
	safeRaw := bytes.ReplaceAll(raw, []byte{0}, []byte{})
	wrapped, _ := json.Marshal(map[string]string{"raw": string(safeRaw)})
	return json.RawMessage(wrapped)
}

// ExtractDLQMetadataForTest exposes extractDLQMetadata for the test suite.
func ExtractDLQMetadataForTest(msg kafka.Message) (string, string, []byte) {
	return extractDLQMetadata(msg)
}

// SanitizeDLQRawJSONForTest exposes KafkaConsumer.sanitizeDLQRawJSON for the test suite.
func (kc *KafkaConsumer) SanitizeDLQRawJSONForTest(table string, raw []byte) json.RawMessage {
	return kc.sanitizeDLQRawJSON(table, raw)
}

// WriteDLQForTest exposes KafkaConsumer.writeDLQ for integration tests.
func (kc *KafkaConsumer) WriteDLQForTest(ctx context.Context, msg kafka.Message, procErr error) error {
	return kc.writeDLQ(ctx, msg, procErr)
}
```

### 4. File `topic_helper.go` (Topic Discovery & Filtering)
Chứa logic discovery:

```go
package shadow

import (
	"context"
	"fmt"
	"strings"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// RefreshTopics re-discovers Kafka topics and recreates the reader if the topic set has changed.
func (kc *KafkaConsumer) RefreshTopics(ctx context.Context) error {
	if reloader, ok := kc.registrySvc.(interface{ ReloadAll(context.Context) error }); ok {
		if err := reloader.ReloadAll(ctx); err != nil {
			kc.logger.Warn(fmt.Sprintf("registry reload failed during topic refresh component=kafka_consumer op=registry_reload err_type=%s err=%s",
				classifyKafkaErr(err), err.Error()),
				zap.String("component", "kafka_consumer"),
				zap.String("op", "registry_reload"),
				zap.String("err_type", classifyKafkaErr(err)),
				zap.Error(err))
		}
	}

	newTopics, err := kc.discoverTopics(ctx)
	if err != nil {
		return fmt.Errorf("discover topics: %w", err)
	}

	kc.refreshMu.Lock()
	defer kc.refreshMu.Unlock()

	if topicSetEqual(kc.currentTopics, newTopics) {
		kc.logger.Debug("topic set unchanged, skipping refresh",
			zap.Int("count", len(newTopics)))
		return nil
	}

	kc.logger.Info(fmt.Sprintf("topic set changed recreating reader component=kafka_consumer op=refresh_topics old_count=%d new_count=%d delta=%d",
		len(kc.currentTopics), len(newTopics), len(newTopics)-len(kc.currentTopics)),
		zap.String("component", "kafka_consumer"),
		zap.String("op", "refresh_topics"),
		zap.Int("old_count", len(kc.currentTopics)),
		zap.Int("new_count", len(newTopics)),
		zap.Int("delta", len(newTopics)-len(kc.currentTopics)),
		zap.Strings("old", kc.currentTopics),
		zap.Strings("new", newTopics))

	kc.flushAllBatches(ctx)

	for _, r := range kc.readers {
		r := r
		done := make(chan struct{})
		go func() {
			defer close(done)
			if cerr := r.Close(); cerr != nil {
				kc.logger.Warn(fmt.Sprintf("close old reader component=kafka_consumer op=close_reader err_type=%s err=%s",
					classifyKafkaErr(cerr), cerr.Error()),
					zap.String("component", "kafka_consumer"),
					zap.String("op", "close_reader"),
					zap.String("err_type", classifyKafkaErr(cerr)),
					zap.Error(cerr))
			}
		}()
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		select {
		case <-done:
		case <-closeCtx.Done():
			kc.logger.Warn("old reader close timed out continuing component=kafka_consumer op=close_reader phase=timeout timeout_ms=5000",
				zap.String("component", "kafka_consumer"),
				zap.String("op", "close_reader"),
				zap.String("phase", "timeout"),
				zap.Int("timeout_ms", 5000))
		}
		cancel()
	}
	kc.readers = nil
	kc.currentTopics = nil

	if len(newTopics) == 0 {
		kc.logger.Warn("no active topics consumer entering idle state component=kafka_consumer op=refresh_topics phase=idle",
			zap.String("component", "kafka_consumer"),
			zap.String("op", "refresh_topics"),
			zap.String("phase", "idle"))
		return nil
	}

	newReader := kc.buildReader(newTopics)
	kc.readers = append(kc.readers, newReader)
	kc.currentTopics = append([]string(nil), newTopics...)

	return nil
}

func topicSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, t := range a {
		set[t] = struct{}{}
	}
	for _, t := range b {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

func (kc *KafkaConsumer) discoverTopics(ctx context.Context) ([]string, error) {
	if kc.discoverFunc != nil {
		return kc.discoverFunc(ctx)
	}
	conn, err := kafka.DialContext(ctx, "tcp", kc.config.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("read partitions: %w", err)
	}

	debeziumTables := make(map[string]bool)
	if kc.registrySvc != nil {
		for _, t := range kc.registrySvc.GetDebeziumTables() {
			debeziumTables[t] = true
		}
	}

	topicNames := make([]string, 0, len(partitions))
	for _, p := range partitions {
		topicNames = append(topicNames, p.Topic)
	}

	topics, perPrefix, prefixes := filterMatchingTopics(topicNames, kc.config.TopicPrefix, debeziumTables)
	if len(prefixes) == 0 {
		kc.logger.Warn("kafka topic discovery no prefixes configured component=kafka_consumer op=discover_topics phase=config_missing",
			zap.String("component", "kafka_consumer"),
			zap.String("op", "discover_topics"),
			zap.String("phase", "config_missing"))
		return nil, nil
	}

	fields := []zap.Field{
		zap.String("component", "kafka_consumer"),
		zap.String("op", "discover_topics"),
		zap.Strings("prefixes", prefixes),
		zap.Strings("topics", topics),
		zap.Int("topic_count", len(topics)),
		zap.Int("broker_count", len(kc.config.Brokers)),
		zap.Int("debezium_tables", len(debeziumTables)),
		zap.Int("raw_topic_count", len(topicNames)),
	}
	perPrefixSummary := make([]string, 0, len(prefixes))
	for _, pre := range prefixes {
		fields = append(fields, zap.Int("count_"+pre, perPrefix[pre]))
		perPrefixSummary = append(perPrefixSummary, fmt.Sprintf("%s=%d", pre, perPrefix[pre]))
	}
	kc.logger.Info(fmt.Sprintf("discovered kafka topics component=kafka_consumer op=discover_topics topic_count=%d raw_topic_count=%d debezium_tables=%d broker_count=%d %s",
		len(topics), len(topicNames), len(debeziumTables), len(kc.config.Brokers), strings.Join(perPrefixSummary, " ")),
		fields...)
	return topics, nil
}

func filterMatchingTopics(topicNames, configuredPrefixes []string, debeziumTables map[string]bool) ([]string, map[string]int, []string) {
	prefixes := make([]string, 0, len(configuredPrefixes))
	for _, p := range configuredPrefixes {
		p = strings.TrimSpace(p)
		if p != "" {
			prefixes = append(prefixes, p)
		}
	}
	if len(prefixes) == 0 {
		return nil, nil, nil
	}

	perPrefix := make(map[string]int, len(prefixes))
	topicSet := make(map[string]bool)
	out := make([]string, 0)

	for _, topic := range topicNames {
		if strings.HasPrefix(topic, "_") {
			continue
		}
		var matched string
		for _, pre := range prefixes {
			if strings.HasPrefix(topic, pre) {
				matched = pre
				break
			}
		}
		if matched == "" {
			continue
		}
		parts := strings.Split(topic, ".")
		var tableName string
		if len(parts) >= 4 {
			tableName = parts[len(parts)-1]
		}
		if len(debeziumTables) > 0 && !debeziumTables[tableName] {
			continue
		}
		if !topicSet[topic] {
			topicSet[topic] = true
			perPrefix[matched]++
			out = append(out, topic)
		}
	}
	return out, perPrefix, prefixes
}

// FilterMatchingTopicsForTest exposes filterMatchingTopics for the test suite.
func FilterMatchingTopicsForTest(topicNames, configuredPrefixes []string, debeziumTables map[string]bool) ([]string, map[string]int, []string) {
	return filterMatchingTopics(topicNames, configuredPrefixes, debeziumTables)
}
```

### 5. File `utils.go` (Tracing, Errors & Time Helpers)
Chứa các function phụ trợ, độc lập:

```go
package shadow

import (
	"context"
	"errors"
	"io"
	"encoding/json"
	"strconv"
	"strings"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func otelTraceSpanFromContext(ctx context.Context) oteltrace.Span {
	sp := oteltrace.SpanFromContext(ctx)
	if !sp.SpanContext().IsValid() {
		return nil
	}
	return sp
}

func isKafkaTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Not Leader For Partition") ||
		strings.Contains(msg, "Broker Not Available") ||
		strings.Contains(msg, "request timed out") ||
		strings.Contains(msg, "network resume") ||
		strings.Contains(msg, "connection reset by peer")
}

func classifyKafkaErr(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "ctx_deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "ctx_canceled"
	}
	if errors.Is(err, io.EOF) {
		return "io_eof"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "Not Leader For Partition"):
		return "kafka_not_leader"
	case strings.Contains(msg, "Broker Not Available"):
		return "kafka_broker_unavailable"
	case strings.Contains(msg, "request timed out"):
		return "kafka_request_timeout"
	case strings.Contains(msg, "connection reset by peer"):
		return "net_conn_reset"
	case strings.Contains(msg, "connection refused"):
		return "net_conn_refused"
	case strings.Contains(msg, "reader closed"):
		return "kafka_reader_closed"
	case strings.Contains(msg, "rebalance"):
		return "kafka_rebalance"
	case strings.Contains(msg, "schema") && strings.Contains(msg, "not found"):
		return "schema_not_found"
	case strings.Contains(msg, "timeout"):
		return "timeout"
	default:
		return "unknown"
	}
}

func extractSourceTsMs(source interface{}) int64 {
	m, ok := source.(map[string]interface{})
	if !ok {
		return 0
	}
	raw := m["ts_ms"]
	if mm, ok := raw.(map[string]interface{}); ok && len(mm) == 1 {
		for _, v := range mm {
			raw = v
			break
		}
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}
```
