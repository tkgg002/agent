# Context: Bug Missing Debezium PG Plugin 2026-06-23

## Problem Description
Khi thực hiện tạo connector Postgres thông qua API của `cdc-cms-service` (`POST http://localhost:8083/api/v1/system/connectors`), Kafka Connect trả về lỗi 400:
`Failed to find any class that implements Connector and which name matches io.debezium.connector.postgresql.PostgresConnector`

## System Diagnostics
- `cdc-cms-service` (đang chạy local trên cổng `8083`) được cấu hình trỏ tới Kafka Connect URL từ xa: `http://10.200.186.203:8083` (trong `config-local.yml`).
- Kafka Connect instance chạy trên server `10.200.186.203:8083` chỉ chứa các plugin:
  - `com.mongodb.kafka.connect.MongoSinkConnector`
  - `com.mongodb.kafka.connect.MongoSourceConnector`
  - `io.debezium.connector.jdbc.JdbcSinkConnector`
  - `io.debezium.connector.mongodb.MongoDbConnector`
  - `io.debezium.connector.mongodb.MongoDbSinkConnector`
  - `org.apache.kafka.connect.mirror.*`
- Hoàn toàn thiếu plugin Debezium PostgreSQL Connector (`io.debezium.connector.postgresql.PostgresConnector`).

## Goal
Xác định cách thức triển khai Kafka Connect trên server `10.200.186.203` và bổ sung plugin `debezium-connector-postgresql` vào đó để cho phép tạo PostgreSQL connector thành công.
