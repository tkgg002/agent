# Technical Solution - Core kafka_consumer.go after refactoring

Dưới đây là mã nguồn chi tiết của file `kafka_consumer.go` mới (sau khi đã di chuyển các hàm helper và các struct phụ trợ sang các file tương ứng). 

File này nằm tại: `internal/handler/shadow/kafka_consumer.go`.

```go
package shadow

import (
	"centralized-data-service/internal/model/shadow"
	"centralized-data-service/internal/model/system"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"centralized-data-service/internal/activity"
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/pkgs/metrics"
	"centralized-data-service/pkgs/observability"

	"github.com/linkedin/goavro/v2"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
	
	"centralized-data-service/pkgs/database"
)

// KafkaConsumerConfig holds Kafka consumer configuration.
type KafkaConsumerConfig struct {
	Brokers           []string `mapstructure:"brokers"`
	GroupID           string   `mapstructure:"groupId"`
	TopicPrefix       []string `mapstructure:"topicPrefix"`
	SchemaRegistryURL string   `mapstructure:"schemaRegistryUrl"`
}

type KafkaPostConsumeEvent struct {
	Topic        string
	Processed    int
	Success      int
	Failed       int
	TriggeredBy  activity.TriggeredBy
	ActivityID   uint64
	ActivityRows int64
	StartedAt    time.Time
	CompletedAt  time.Time
}

type KafkaPostConsumeAction func(context.Context, KafkaPostConsumeEvent) error

// KafkaConsumer consumes CDC events from Kafka (Debezium → Kafka → Worker)
type KafkaConsumer struct {
	config       KafkaConsumerConfig
	eventHandler *EventHandler
	registrySvc  interface{ GetDebeziumTables() []string }
	validator    *governance.SchemaValidator
	masking      *governance.MaskingService
	logger       *zap.Logger
	readers       []*kafka.Reader
	db            *gorm.DB
	perSourcePool *database.PerSourcePool
	batches       map[string]*batchStats // per-topic batch accumulator
	batchFlushSize int
	flushIntervalSeconds int
	refreshMu     sync.Mutex
	currentTopics []string
	discoverFunc func(ctx context.Context) ([]string, error)
	postConsumeAction     KafkaPostConsumeAction
	postConsumeActionName string
	dlqCircuitBreaker     *DLQCircuitBreaker
	batcher               *adaptiveBatcher
	adaptiveEnabled       bool
}

func NewKafkaConsumer(cfg KafkaConsumerConfig, handler *EventHandler, registrySvc interface{ GetDebeziumTables() []string }, db *gorm.DB, perSourcePool *database.PerSourcePool, logger *zap.Logger) *KafkaConsumer {
	return &KafkaConsumer{
		config:        cfg,
		eventHandler:  handler,
		registrySvc:   registrySvc,
		db:            db,
		perSourcePool: perSourcePool,
		logger:        logger,
		batches:       make(map[string]*batchStats),
	}
}

func (kc *KafkaConsumer) SetSchemaValidator(v *governance.SchemaValidator) {
	kc.validator = v
}

func (kc *KafkaConsumer) SetMaskingService(masking *governance.MaskingService) {
	kc.masking = masking
}

func (kc *KafkaConsumer) SetBatchFlushSize(n int) {
	kc.batchFlushSize = n
}

func (kc *KafkaConsumer) SetFlushInterval(seconds int) {
	kc.flushIntervalSeconds = seconds
}

func (kc *KafkaConsumer) SetPostConsumeAction(name string, action KafkaPostConsumeAction) {
	kc.postConsumeActionName = strings.TrimSpace(name)
	kc.postConsumeAction = action
}

func (kc *KafkaConsumer) SetDLQCircuitBreaker(cb *DLQCircuitBreaker) {
	kc.dlqCircuitBreaker = cb
}

func (kc *KafkaConsumer) SetDestHealthCheck(f func() bool) {
	if kc.batcher == nil {
		return
	}
	kc.batcher.mu.Lock()
	kc.batcher.destHealth = f
	kc.batcher.mu.Unlock()
}

func (kc *KafkaConsumer) SetAdaptiveBatchConfig(enabled bool, lagThreshold int64, maxMultiplier int) {
	kc.adaptiveEnabled = enabled
	if enabled {
		baseSize := kc.batchFlushSize
		if baseSize <= 0 {
			baseSize = 100
		}
		if lagThreshold <= 0 {
			lagThreshold = 50000
		}
		if maxMultiplier <= 0 {
			maxMultiplier = 4
		}
		kc.batcher = &adaptiveBatcher{
			baseBatchSize: baseSize,
			currentSize:   baseSize,
			lagThreshold:  lagThreshold,
			maxMultiplier: maxMultiplier,
		}
	}
}

func (kc *KafkaConsumer) buildReader(topics []string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:          kc.config.Brokers,
		GroupID:          kc.config.GroupID,
		GroupTopics:      topics,
		MinBytes:         10e3, // 10KB
		MaxBytes:         10e6, // 10MB
		CommitInterval:   time.Second,
		SessionTimeout:   30 * time.Second,
		RebalanceTimeout: 30 * time.Second,
		StartOffset:      kafka.FirstOffset,
		Logger:           nil,
	})
}

func (kc *KafkaConsumer) Start(ctx context.Context) {
	const component = "kafka_consumer"
	kc.logger.Info(fmt.Sprintf("kafka consumer starting component=%s brokers=%v broker_count=%d group=%s prefixes=%v schema_registry=%s adaptive_enabled=%t",
		component, kc.config.Brokers, len(kc.config.Brokers), kc.config.GroupID, kc.config.TopicPrefix,
		kc.config.SchemaRegistryURL, kc.adaptiveEnabled),
		zap.String("component", component),
		zap.Strings("brokers", kc.config.Brokers),
		zap.Int("broker_count", len(kc.config.Brokers)),
		zap.String("group", kc.config.GroupID),
		zap.Strings("prefixes", kc.config.TopicPrefix),
		zap.String("schema_registry", kc.config.SchemaRegistryURL),
		zap.Bool("adaptive_enabled", kc.adaptiveEnabled),
	)

	time.Sleep(10 * time.Second)

	discoverStart := time.Now()
	topics, err := kc.discoverTopics(ctx)
	discoverDurMs := time.Since(discoverStart).Milliseconds()
	if err != nil {
		kc.logger.Warn(fmt.Sprintf("failed to discover kafka topics retrying in 30s component=%s op=discover_topics phase=initial discover_duration_ms=%d broker_count=%d err=%s",
			component, discoverDurMs, len(kc.config.Brokers), err.Error()),
			zap.String("component", component),
			zap.String("op", "discover_topics"),
			zap.String("phase", "initial"),
			zap.Int64("discover_duration_ms", discoverDurMs),
			zap.Int("broker_count", len(kc.config.Brokers)),
			zap.Error(err))
		time.Sleep(30 * time.Second)
		retryStart := time.Now()
		topics, err = kc.discoverTopics(ctx)
		retryDurMs := time.Since(retryStart).Milliseconds()
		if err != nil {
			kc.logger.Warn(fmt.Sprintf("kafka topic discovery failed at startup starting in idle state component=%s op=discover_topics phase=retry discover_duration_ms=%d broker_count=%d err=%s",
				component, retryDurMs, len(kc.config.Brokers), err.Error()),
				zap.String("component", component),
				zap.String("op", "discover_topics"),
				zap.String("phase", "retry"),
				zap.Int64("discover_duration_ms", retryDurMs),
				zap.Int("broker_count", len(kc.config.Brokers)),
				zap.Error(err))
			topics = nil
		}
	}

	kc.refreshMu.Lock()
	if len(topics) > 0 {
		reader := kc.buildReader(topics)
		kc.readers = append(kc.readers, reader)
		kc.currentTopics = append([]string(nil), topics...)
		kc.logger.Info(fmt.Sprintf("kafka consumer started component=%s topic_count=%d group=%s broker_count=%d start_offset=first session_timeout=30s rebalance_timeout=30s",
			component, len(topics), kc.config.GroupID, len(kc.config.Brokers)),
			zap.String("component", component),
			zap.Strings("topics", topics),
			zap.Int("topic_count", len(topics)),
			zap.String("group", kc.config.GroupID),
			zap.Int("broker_count", len(kc.config.Brokers)),
		)
	} else {
		kc.readers = nil
		kc.currentTopics = nil
		kc.logger.Warn(fmt.Sprintf("no kafka topics found at startup starting in idle state component=%s prefixes=%v broker_count=%d",
			component, kc.config.TopicPrefix, len(kc.config.Brokers)),
			zap.String("component", component),
			zap.Strings("prefixes", kc.config.TopicPrefix),
			zap.Int("broker_count", len(kc.config.Brokers)))
	}
	kc.refreshMu.Unlock()

	flushInterval := kc.flushIntervalSeconds
	if flushInterval <= 0 {
		flushInterval = 5
	}
	flushTicker := time.NewTicker(time.Duration(flushInterval) * time.Second)
	defer flushTicker.Stop()
	kc.logger.Info(fmt.Sprintf("kafka consumer flush ticker configured interval=%ds batch_flush_size=%d",
		flushInterval, kc.batchFlushSize),
		zap.Int("interval_seconds", flushInterval),
		zap.Int("batch_flush_size", kc.batchFlushSize),
	)

	refreshTicker := time.NewTicker(60 * time.Second)
	defer refreshTicker.Stop()

	metricsTicker := time.NewTicker(15 * time.Second)
	defer metricsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			kc.flushAllBatches(ctx)
			kc.Stop()
			return
		case <-flushTicker.C:
			kc.flushAllBatches(ctx)
		case <-refreshTicker.C:
			refreshStart := time.Now()
			if err := kc.RefreshTopics(ctx); err != nil {
				kc.logger.Warn(fmt.Sprintf("auto refresh topics failed component=kafka_consumer op=refresh_topics phase=auto refresh_duration_ms=%d err_type=%s err=%s",
					time.Since(refreshStart).Milliseconds(), classifyKafkaErr(err), err.Error()),
					zap.String("component", "kafka_consumer"),
					zap.String("op", "refresh_topics"),
					zap.String("phase", "auto"),
					zap.Int64("refresh_duration_ms", time.Since(refreshStart).Milliseconds()),
					zap.String("err_type", classifyKafkaErr(err)),
					zap.Error(err))
			}
		case <-metricsTicker.C:
			kc.refreshMu.Lock()
			if len(kc.readers) > 0 {
				stats := kc.readers[0].Stats()
				metrics.ConsumerLag.
					WithLabelValues(stats.Topic, stats.Partition).
					Set(float64(stats.Lag))
				metrics.ConsumerOffset.
					WithLabelValues(stats.Topic, stats.Partition).
					Set(float64(stats.Offset))

				if kc.adaptiveEnabled && kc.batcher != nil {
					kc.batcher.adjust(stats.Lag)
				}
			}
			kc.refreshMu.Unlock()
		default:
			kc.refreshMu.Lock()
			if len(kc.readers) == 0 {
				kc.refreshMu.Unlock()
				time.Sleep(100 * time.Millisecond)
				continue
			}
			currentReader := kc.readers[0]
			kc.refreshMu.Unlock()

			fetchStart := time.Now()
			fetchCtx, fetchCancel := context.WithTimeout(ctx, 200*time.Millisecond)
			msg, err := currentReader.FetchMessage(fetchCtx)
			fetchCancel()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				if ctx.Err() != nil {
					return
				}
				fetchMs := time.Since(fetchStart).Milliseconds()
				readerLag := currentReader.Stats().Lag
				errType := classifyKafkaErr(err)
				if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "reader closed") {
					kc.logger.Debug(fmt.Sprintf("kafka reader closed during refresh component=kafka_consumer op=fetch_message phase=transient fetch_duration_ms=%d reader_lag=%d err_type=%s err=%s",
						fetchMs, readerLag, errType, err.Error()),
						zap.String("component", "kafka_consumer"),
						zap.String("op", "fetch_message"),
						zap.String("phase", "reader_closed"),
						zap.Int64("fetch_duration_ms", fetchMs),
						zap.Int64("reader_lag", readerLag),
						zap.String("err_type", errType),
						zap.Error(err))
					time.Sleep(200 * time.Millisecond)
					continue
				}
				if isKafkaTransientError(err) {
					kc.logger.Warn(fmt.Sprintf("kafka fetch transient error retrying component=kafka_consumer op=fetch_message phase=transient fetch_duration_ms=%d reader_lag=%d err_type=%s err=%s",
						fetchMs, readerLag, errType, err.Error()),
						zap.String("component", "kafka_consumer"),
						zap.String("op", "fetch_message"),
						zap.String("phase", "transient"),
						zap.Int64("fetch_duration_ms", fetchMs),
						zap.Int64("reader_lag", readerLag),
						zap.String("err_type", errType),
						zap.Error(err))
					time.Sleep(200 * time.Millisecond)
					continue
				}
				kc.logger.Error(fmt.Sprintf("kafka fetch error component=kafka_consumer op=fetch_message phase=fatal fetch_duration_ms=%d reader_lag=%d err_type=%s err=%s",
					fetchMs, readerLag, errType, err.Error()),
					zap.String("component", "kafka_consumer"),
					zap.String("op", "fetch_message"),
					zap.String("phase", "fatal"),
					zap.Int64("fetch_duration_ms", fetchMs),
					zap.Int64("reader_lag", readerLag),
					zap.String("err_type", errType),
					zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			start := time.Now()
			carrier := propagation.MapCarrier{}
			for _, h := range msg.Headers {
				carrier[h.Key] = string(h.Value)
			}
			parentCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)

			engine, sourceDB, _, _ := observability.ParseDebeziumTopic(msg.Topic)
			spanCtx, span := observability.StartSpan(parentCtx, "kafka.consume",
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.destination", msg.Topic),
				attribute.String("messaging.operation", "receive"),
				attribute.Int("messaging.kafka.partition", msg.Partition),
				attribute.Int64("messaging.kafka.offset", msg.Offset),
				attribute.Int64("messaging.kafka.message.timestamp_ms", msg.Time.UnixMilli()),
				observability.SourceTableAttr(msg.Topic),
				attribute.String("cdc.engine", engine),
				attribute.String("cdc.source_db", sourceDB),
			)

			if !msg.Time.IsZero() {
				e2eLatency := time.Since(msg.Time)
				metrics.E2ELatency.Observe(e2eLatency.Seconds())
				span.SetAttributes(attribute.Float64("e2e_latency_seconds", e2eLatency.Seconds()))
			}

			batch := kc.getOrCreateBatch(msg.Topic)

			var rows int
			var procErr error

			if kc.perSourcePool != nil {
				release, err := kc.perSourcePool.Acquire(spanCtx, sourceDB)
				if err != nil {
					procErr = fmt.Errorf("failed to acquire source pool semaphore: %w", err)
				} else {
					rows, procErr = kc.processMessage(spanCtx, msg)
					release()
				}
			} else {
				rows, procErr = kc.processMessage(spanCtx, msg)
			}

			if procErr != nil {
				procMs := time.Since(start).Milliseconds()
				errType := classifyKafkaErr(procErr)
				observability.Ctx(spanCtx, kc.logger).Error(fmt.Sprintf("kafka message processing failed component=kafka_consumer op=process_message topic=%s partition=%d offset=%d payload_bytes=%d process_duration_ms=%d err_type=%s err=%s",
					msg.Topic, msg.Partition, msg.Offset, len(msg.Value), procMs, errType, procErr.Error()),
					observability.ErrorField(procErr),
					observability.Attrs(
						zap.String("component", "kafka_consumer"),
						zap.String("op", "process_message"),
						zap.String("topic", msg.Topic),
						zap.Int("partition", msg.Partition),
						zap.Int64("offset", msg.Offset),
						zap.Int("payload_bytes", len(msg.Value)),
						zap.Int64("process_duration_ms", procMs),
						zap.String("err_type", errType),
					),
				)
				span.RecordError(procErr)
				span.SetStatus(codes.Error, procErr.Error())
				metrics.EventsProcessed.WithLabelValues("error", "", msg.Topic, "error").Inc()
				batch.failed++
			} else {
				duration := time.Since(start)
				metrics.EventsProcessed.WithLabelValues("kafka", "", msg.Topic, "success").Inc()
				metrics.ProcessingDuration.WithLabelValues("kafka", "", msg.Topic).Observe(duration.Seconds())
				span.SetAttributes(
					attribute.Float64("duration_seconds", duration.Seconds()),
					attribute.Int("cdc.rows_affected", rows),
				)
				batch.success++
				batch.rowsAffected += rows
			}
			batch.processed++
			span.End()

			var flushAt int
			if kc.adaptiveEnabled && kc.batcher != nil {
				flushAt = kc.batcher.getSize()
			} else {
				flushAt = kc.batchFlushSize
				if flushAt <= 0 {
					flushAt = 100
				}
			}
			if batch.processed >= flushAt {
				kc.flushBatch(ctx, msg.Topic)
			}

			if procErr != nil {
				if dlqErr := kc.writeDLQ(ctx, msg, procErr); dlqErr != nil {
					metrics.DLQWriteFail.Inc()
					if kc.dlqCircuitBreaker != nil {
						kc.dlqCircuitBreaker.RecordDLQWrite(ctx)
					}
					dlqErrType := classifyKafkaErr(dlqErr)
					observability.Ctx(spanCtx, kc.logger).Error(fmt.Sprintf("kafka DLQ write failed skipping offset commit component=kafka_consumer op=write_dlq topic=%s partition=%d offset=%d payload_bytes=%d err_type=%s err=%s",
						msg.Topic, msg.Partition, msg.Offset, len(msg.Value), dlqErrType, dlqErr.Error()),
						observability.ErrorField(dlqErr),
						observability.Attrs(
							zap.String("component", "kafka_consumer"),
							zap.String("op", "write_dlq"),
							zap.String("topic", msg.Topic),
							zap.Int("partition", msg.Partition),
							zap.Int64("offset", msg.Offset),
							zap.Int("payload_bytes", len(msg.Value)),
							zap.String("err_type", dlqErrType),
						),
					)
					continue
				}
			}

			if kc.dlqCircuitBreaker != nil && kc.dlqCircuitBreaker.IsPaused() {
				kc.logger.Warn(fmt.Sprintf("pipeline paused skip offset commit component=kafka_consumer op=commit phase=paused topic=%s partition=%d offset=%d",
					msg.Topic, msg.Partition, msg.Offset),
					zap.String("component", "kafka_consumer"),
					zap.String("op", "commit"),
					zap.String("phase", "paused"),
					zap.String("topic", msg.Topic),
					zap.Int("partition", msg.Partition),
					zap.Int64("offset", msg.Offset))
				continue
			}

			commitStart := time.Now()
			if err := currentReader.CommitMessages(ctx, msg); err != nil {
				commitMs := time.Since(commitStart).Milliseconds()
				errType := classifyKafkaErr(err)
				observability.Ctx(spanCtx, kc.logger).Error(fmt.Sprintf("kafka commit failed component=kafka_consumer op=commit topic=%s partition=%d offset=%d commit_duration_ms=%d err_type=%s err=%s",
					msg.Topic, msg.Partition, msg.Offset, commitMs, errType, err.Error()),
					observability.ErrorField(err),
					observability.Attrs(
						zap.String("component", "kafka_consumer"),
						zap.String("op", "commit"),
						zap.String("topic", msg.Topic),
						zap.Int("partition", msg.Partition),
						zap.Int64("offset", msg.Offset),
						zap.Int64("commit_duration_ms", commitMs),
						zap.String("err_type", errType),
					))
			}
		}
	}
}

func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) (rows int, err error) {
	ctx, span := observability.ChildSpan(ctx, "cdc.process_message",
		attribute.Int("cdc.value_size_bytes", len(msg.Value)),
	)
	defer observability.EndSpan(span, &err)

	value := msg.Value
	if len(value) == 0 {
		return 0, nil
	}

	var event map[string]interface{}
	if len(value) > 5 && value[0] == 0 {
		schemaID := int32(binary.BigEndian.Uint32(value[1:5]))
		avroData := value[5:]

		codec, err := kc.getAvroCodec(schemaID)
		if err != nil {
			return 0, fmt.Errorf("get avro schema %d: %w", schemaID, err)
		}

		native, _, err := codec.NativeFromBinary(avroData)
		if err != nil {
			return 0, fmt.Errorf("avro decode (schema %d): %w", schemaID, err)
		}

		var ok bool
		event, ok = native.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("avro decoded to %T, expected map", native)
		}
	} else {
		if err := json.Unmarshal(value, &event); err != nil {
			return 0, fmt.Errorf("parse kafka message as JSON: %w", err)
		}
	}

	op := unwrapAvroUnion(event["op"])
	afterRaw := unwrapAvroUnion(event["after"])
	beforeRaw := unwrapAvroUnion(event["before"])
	sourceRaw := event["source"]

	var afterData map[string]interface{}
	switch v := afterRaw.(type) {
	case map[string]interface{}:
		afterData = v
	case string:
		json.Unmarshal([]byte(v), &afterData)
	}

	var beforeData map[string]interface{}
	switch v := beforeRaw.(type) {
	case map[string]interface{}:
		beforeData = v
	case string:
		json.Unmarshal([]byte(v), &beforeData)
	}

	if afterData != nil {
		afterData = unwrapAvroUnionMap(afterData)
	}
	if beforeData != nil {
		beforeData = unwrapAvroUnionMap(beforeData)
	}

	sourceTsMs := extractSourceTsMs(sourceRaw)

	if span := otelTraceSpanFromContext(ctx); span != nil {
		span.SetAttributes(
			attribute.Int64("source.ts_ms", sourceTsMs),
		)
	}

	opStr := fmt.Sprintf("%v", op)
	if afterData == nil && opStr != "d" {
		kc.logger.Debug("kafka message has no 'after' data, skipping",
			zap.String("topic", msg.Topic),
			zap.String("op", opStr),
		)
		return 0, nil
	}

	if kc.validator != nil && afterData != nil {
		parts := strings.Split(msg.Topic, ".")
		if len(parts) >= 4 {
			tbl := parts[3]
			if err := kc.validator.ValidatePayloadWithCase(tbl, afterData); err != nil {
				return 0, fmt.Errorf("schema_validator: %w", err)
			}
		}
	}

	var beforeField interface{}
	if len(beforeData) > 0 {
		beforeField = beforeData
	}
	cdcEvent := map[string]interface{}{
		"source": "debezium",
		"data": map[string]interface{}{
			"op":           opStr,
			"before":       beforeField,
			"after":        afterData,
			"source_ts_ms": sourceTsMs,
		},
	}

	cdcJSON, _ := json.Marshal(cdcEvent)
	subject := msg.Topic

	kc.logger.Debug("kafka CDC event",
		zap.String("topic", msg.Topic),
		zap.String("op", opStr),
		zap.Int("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
		zap.Int("after_fields", len(afterData)),
		zap.Int64("source_ts_ms", sourceTsMs),
	)

	return kc.eventHandler.HandleRaw(ctx, subject, cdcJSON)
}

func (kc *KafkaConsumer) Stop() {
	for _, r := range kc.readers {
		r.Close()
	}
	kc.logger.Info(fmt.Sprintf("kafka consumer stopped component=kafka_consumer op=stop reader_count=%d", len(kc.readers)),
		zap.String("component", "kafka_consumer"),
		zap.String("op", "stop"),
		zap.Int("reader_count", len(kc.readers)))
}
```
