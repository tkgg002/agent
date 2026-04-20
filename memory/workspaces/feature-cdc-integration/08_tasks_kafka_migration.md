# Tasks: Kafka Migration

> Date: 2026-04-15
> Phase: kafka_migration

## T1: Docker Compose — Kafka stack
- [ ] T1.1: Kafka KRaft broker (confluentinc/cp-kafka:7.6, KAFKA_KRAFT_MODE)
- [ ] T1.2: Schema Registry (cp-schema-registry:7.6)
- [ ] T1.3: Kafka Connect + Debezium plugin (cp-kafka-connect + debezium-connector-mongodb)
- [ ] T1.4: Redpanda Console (redpandadata/console:latest)
- [ ] T1.5: Verify tất cả containers UP

## T2: Debezium Kafka Connector
- [ ] T2.1: MongoDB connector config JSON (Avro + Schema Registry)
- [ ] T2.2: Deploy connector via Kafka Connect REST API
- [ ] T2.3: Verify topics created trong Redpanda Console

## T3: Worker Kafka Consumer
- [ ] T3.1: Go dependency (kafka library)
- [ ] T3.2: Kafka consumer implementation (consumer group, Avro deserialize)
- [ ] T3.3: Integrate với EventHandler (giữ nguyên DynamicMapper)
- [ ] T3.4: Config: kafka broker URL, consumer group, topics pattern
- [ ] T3.5: Build OK

## T4: NATS giữ commands
- [ ] T4.1: Verify NATS vẫn hoạt động cho cdc.cmd.*, schema.config.reload
- [ ] T4.2: Bỏ NATS JetStream streams CDC (CDC_EVENTS)

## T5: Redpanda Console
- [ ] T5.1: Config connect Kafka + Schema Registry
- [ ] T5.2: Verify UI: topics, messages, consumer groups, lag

## T6: E2E Test
- [ ] T6.1: MongoDB insert → Kafka topic visible
- [ ] T6.2: Worker consume → PostgreSQL insert
- [ ] T6.3: Redpanda Console: consumer lag = 0
- [ ] T6.4: Schema Registry: Avro schema visible

## T7: Cleanup
- [ ] T7.1: Remove Debezium Server container
- [ ] T7.2: Remove NATS CDC streams
- [ ] T7.3: Update documentation
