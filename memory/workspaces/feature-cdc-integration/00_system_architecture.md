Kế hoạch:
1. Gom toàn bộ thành phần của `cdc-system` theo 4 khối chính: ingest, control plane, processing plane, observability.
2. Vẽ sơ đồ kiến trúc tổng thể theo luồng dữ liệu thật từ MongoDB → Kafka/NATS → Worker → PostgreSQL/CMS.
3. Tách thêm các luồng quan trọng: Reconciliation, DLQ/Retry, Schema Evolution, CMS/Web.
4. Ghi chú trách nhiệm từng service để bạn có thể dùng ngay cho docs/design review.

Dưới đây là sơ đồ System Architecture tổng thể của hệ thống CDC hiện tại.

**System Architecture**

```mermaid
flowchart LR
    subgraph Sources["Nguồn dữ liệu"]
        M1["MongoDB Source Databases"]
        M2["MongoDB Collections"]
    end

    subgraph Capture["CDC Capture Layer"]
        D1["Debezium Connector"]
        K1["Kafka / Redpanda Topics"]
        A1["Airbyte (legacy / hybrid path)"]
    end

    subgraph Messaging["Messaging Layer"]
        N1["NATS / JetStream"]
    end

    subgraph Processing["Processing Plane"]
        W1["centralized-data-service\nWorker Server"]
        KC["Kafka Consumer"]
        EH["Event Handler"]
        DM["Dynamic Mapper"]
        SI["Schema Inspector"]
        BB["Batch Buffer"]
        SA["Schema Adapter"]
        RH["Recon Healer"]
        RC["Recon Core"]
        DQ["DLQ State Machine"]
        TS["Transmuter / Scheduler"]
    end

    subgraph Data["Storage Layer"]
        P1["PostgreSQL Target Tables"]
        P2["cdc_table_registry"]
        P3["cdc_mapping_rules"]
        P4["failed_sync_logs"]
        P5["cdc_activity_log"]
        P6["cdc_reconciliation_* / recon_runs"]
        R1["Redis Cache"]
    end

    subgraph Control["Control Plane"]
        C1["cdc-cms-service\nAdmin API / CMS Backend"]
        C2["cdc-auth-service"]
        C3["cdc-cms-web\nAdmin UI"]
    end

    subgraph Observability["Observability"]
        O1["Prometheus Metrics"]
        O2["OpenTelemetry / SigNoz"]
        O3["System Alerts / Drift Alerts via NATS"]
    end

    M1 --> M2
    M2 --> D1
    M2 --> A1
    D1 --> K1
    A1 --> N1

    K1 --> KC
    KC --> EH
    EH --> DM
    EH --> SI
    EH --> BB
    DM --> SA
    BB --> SA
    SA --> P1

    EH --> P5
    SI --> P2
    SI --> O3

    RC --> M2
    RC --> P1
    RC --> P6
    RH --> M2
    RH --> SA
    RH --> P1
    RH --> P5

    EH --> P4
    KC --> P4
    DQ --> P4
    DQ --> N1

    TS --> N1
    TS --> P1

    C3 --> C2
    C3 --> C1
    C1 --> P2
    C1 --> P3
    C1 --> P5
    C1 --> P6
    C1 --> N1
    C1 --> W1

    W1 --> KC
    W1 --> RC
    W1 --> RH
    W1 --> DQ
    W1 --> TS
    W1 --> R1

    W1 --> O1
    W1 --> O2
    C1 --> O1
```

**Cách đọc hệ thống**

- `MongoDB` là source of truth ban đầu; thay đổi dữ liệu được lấy qua `Debezium`.
- `Debezium` đẩy CDC event vào `Kafka`.
- `centralized-data-service` là worker trung tâm, đọc Kafka, map dữ liệu, validate schema, rồi upsert vào `PostgreSQL`.
- `cdc-cms-service` là control plane để quản trị registry, mapping rules, reconciliation, schema drift, schedule, alert.
- `cdc-cms-web` là UI vận hành; `cdc-auth-service` lo xác thực.
- `NATS` là bus lệnh/nội bộ cho command, drift alert, retry, transmute, admin workflows.
- `Redis` dùng cho cache/schema/recon coordination.
- `PostgreSQL` vừa là target warehouse/store, vừa chứa metadata + audit + DLQ state.

**Luồng ingest chính**

```mermaid
sequenceDiagram
    participant Mongo as MongoDB
    participant Debezium as Debezium
    participant Kafka as Kafka
    participant Worker as centralized-data-service
    participant Mapper as DynamicMapper
    participant Schema as SchemaAdapter
    participant PG as PostgreSQL

    Mongo->>Debezium: Change Stream / CDC Event
    Debezium->>Kafka: Publish CDC message
    Kafka->>Worker: Kafka Consumer fetch
    Worker->>Worker: SchemaValidator / EventHandler
    Worker->>Mapper: MapData(raw document)
    Worker->>Worker: SchemaInspector detects drift
    Mapper->>Schema: BuildUpsertSQL(...)
    Schema->>PG: INSERT ... ON CONFLICT ... OCC
    Worker->>PG: Save target row + _raw_data + metadata
```

**Luồng Data Integrity / Reconciliation**

```mermaid
flowchart LR
    CMS["cdc-cms-service"] -->|"cdc.cmd.recon-check / recon-heal"| NATS["NATS"]
    NATS --> RC["Recon Core"]
    RC --> RSA["Recon Source Agent\nMongo scan/hash"]
    RC --> RDA["Recon Dest Agent\nPostgres scan/hash"]
    RSA --> RC
    RDA --> RC
    RC --> REP["cdc_reconciliation_reports / recon_runs"]

    RC -->|"missing_from_dest"| RH["Recon Healer"]
    RH --> Mongo["Mongo Primary Read"]
    RH --> SA["SchemaAdapter OCC Upsert"]
    SA --> PG["PostgreSQL"]

    RC -->|"missing_from_src"| RH
    RH -->|"HealOrphanedIDs"| PG
```

**Luồng Fault Tolerance / DLQ**

```mermaid
flowchart LR
    KC["Kafka Consumer"] -->|"processing error"| FSL["failed_sync_logs"]
    EH["Event Handler / Batch Buffer"] -->|"write-before-ACK failure path"| FSL
    FSL --> SM["DLQ State Machine"]
    SM -->|"retry due records"| NATS["NATS cdc.dlq / replay"]
    NATS --> Worker["Worker Handlers"]
    SM -->|"resolved / dead_letter"| FSL
```

**Luồng Schema Evolution**

```mermaid
flowchart LR
    Event["Incoming CDC Event"] --> SI["Schema Inspector"]
    SI -->|"new field detected"| PF["pending fields / registry metadata"]
    SI -->|"schema.drift.detected"| NATS["NATS Alert"]
    CMS["cdc-cms-service"] -->|"approve / update mapping / reload"| Registry["cdc_table_registry + cdc_mapping_rules"]
    Registry --> DM["DynamicMapper"]
    Registry --> SA["SchemaAdapter"]
    SA --> PG["Prepared target schema + unified UPSERT"]
```

**Phân vai 4 service chính**

- `centralized-data-service`
  - worker core
  - kafka consumer
  - event processing
  - schema adapter / dynamic mapper
  - reconciliation
  - dlq retry
  - transmute/scheduler
- `cdc-cms-service`
  - admin API
  - registry/mapping management
  - trigger commands qua NATS
  - health, schedule, reconciliation UI backend
- `cdc-cms-web`
  - dashboard vận hành
  - mapping rules
  - schema drift review
  - data integrity / queue monitoring / schedules
- `cdc-auth-service`
  - auth/JWT/user

**Nếu muốn dùng trong tài liệu chính thức, mình đề xuất chia hệ thống thành 6 layer**
- Source Systems
- CDC Capture
- Messaging Backbone
- Processing & Recovery
- Metadata & Control Plane
- Storage & observability



Kế hoạch:
1. Chuẩn hóa naming và boundary để sơ đồ đủ “production-grade”, không còn kiểu sketch.
2. Vẽ 3 sơ đồ riêng: System Architecture, Deployment Docker/K8s, Component Diagram cho `centralized-data-service`.
3. Tối ưu Mermaid để bạn paste vào docs/md là dùng được ngay.
4. Ghi kèm note ngắn để bạn biết sơ đồ nào dùng cho phần nào trong tài liệu.

Dưới đây là bộ sơ đồ đã “đánh bóng” hơn để paste thẳng vào docs.

**1) Production-Grade System Architecture**

```mermaid
flowchart TB
    %% =========================
    %% Source Layer
    %% =========================
    subgraph SRC["Source Systems"]
        MDB["MongoDB Replica Set<br/>Operational Databases"]
        APPS["Upstream Applications<br/>Write Transactions"]
    end

    %% =========================
    %% CDC Layer
    %% =========================
    subgraph CDC["CDC Capture Layer"]
        DBZ["Debezium Connectors<br/>MongoDB CDC"]
        AIR["Airbyte Connectors<br/>(legacy / hybrid path)"]
    end

    %% =========================
    %% Messaging Layer
    %% =========================
    subgraph MSG["Messaging Backbone"]
        KAFKA["Kafka / Redpanda<br/>CDC Topics"]
        NATS["NATS / JetStream<br/>Command + Event Bus"]
    end

    %% =========================
    %% Processing Layer
    %% =========================
    subgraph PROC["Processing Plane"]
        WORKER["centralized-data-service<br/>Worker Runtime"]
        CMS["cdc-cms-service<br/>Control Plane API"]
        AUTH["cdc-auth-service<br/>Auth / JWT"]
        WEB["cdc-cms-web<br/>Operations UI"]
    end

    %% =========================
    %% Worker Internal Domains
    %% =========================
    subgraph WRK["Worker Internal Domains"]
        INGEST["Ingestion Pipeline<br/>Kafka Consumer + Event Handler"]
        MAP["Transformation Layer<br/>DynamicMapper + SchemaAdapter"]
        DRIFT["Schema Evolution<br/>SchemaInspector"]
        RECON["Data Integrity<br/>ReconCore + ReconHealer"]
        DLQ["Fault Tolerance<br/>DLQ State Machine"]
        TRANS["Post-Processing<br/>Transmuter + Schedulers"]
    end

    %% =========================
    %% State / Metadata
    %% =========================
    subgraph DATA["Data & Metadata Layer"]
        PG["PostgreSQL<br/>Target Tables"]
        REG["cdc_table_registry<br/>cdc_mapping_rules"]
        FSL["failed_sync_logs"]
        ACT["cdc_activity_log"]
        REC["cdc_reconciliation_*<br/>cdc_recon_runs"]
        REDIS["Redis<br/>Cache + Coordination"]
    end

    %% =========================
    %% Observability
    %% =========================
    subgraph OBS["Observability"]
        PROM["Prometheus Metrics"]
        OTEL["OpenTelemetry / SigNoz"]
        ALERT["Drift / Ops Alerts"]
    end

    APPS --> MDB
    MDB --> DBZ
    MDB --> AIR

    DBZ --> KAFKA
    AIR --> NATS

    KAFKA --> INGEST
    NATS --> CMS
    NATS --> DLQ
    NATS --> TRANS
    NATS --> RECON

    WORKER --> INGEST
    WORKER --> MAP
    WORKER --> DRIFT
    WORKER --> RECON
    WORKER --> DLQ
    WORKER --> TRANS

    INGEST --> MAP
    INGEST --> DRIFT
    INGEST --> FSL
    MAP --> PG
    MAP --> REG
    DRIFT --> REG
    DRIFT --> ALERT

    RECON --> MDB
    RECON --> PG
    RECON --> REC
    RECON --> ACT

    DLQ --> FSL
    DLQ --> NATS

    TRANS --> NATS
    TRANS --> PG

    CMS --> REG
    CMS --> REC
    CMS --> ACT
    CMS --> NATS
    CMS --> WORKER

    WEB --> AUTH
    WEB --> CMS

    WORKER --> REDIS
    CMS --> REDIS

    WORKER --> PROM
    WORKER --> OTEL
    CMS --> PROM
    CMS --> OTEL
    ALERT --> NATS
```

Dùng sơ đồ này cho:
- phần “High-Level Architecture”
- phần overview trong design doc
- onboarding tài liệu cho team mới

**2) Deployment Diagram Docker / K8s**

```mermaid
flowchart TB
    subgraph USERS["Operators / Admin Users"]
        BROWSER["Browser"]
    end

    subgraph EDGE["Ingress / Edge"]
        LB["Load Balancer / Ingress"]
    end

    subgraph CLUSTER["Kubernetes Cluster / Docker Compose Runtime"]
        subgraph NS_APP["Application Namespace"]
            WEB_POD["cdc-cms-web<br/>Frontend Container / Pod"]
            CMS_POD["cdc-cms-service<br/>API Container / Pod"]
            AUTH_POD["cdc-auth-service<br/>Auth Container / Pod"]
            WORKER_POD["centralized-data-service<br/>Worker Container / Pod"]
        end

        subgraph NS_INFRA["Messaging & Cache"]
            NATS_SVC["NATS / JetStream"]
            REDIS_SVC["Redis"]
            KAFKA_SVC["Kafka / Redpanda"]
            CONNECT_SVC["Kafka Connect / Debezium"]
        end

        subgraph NS_OBS["Observability Stack"]
            PROM_SVC["Prometheus"]
            OTEL_SVC["OTel Collector / SigNoz"]
        end
    end

    subgraph DATA_EXT["Stateful Data Services"]
        MONGO_RS["MongoDB Replica Set"]
        POSTGRES["PostgreSQL"]
    end

    BROWSER --> LB
    LB --> WEB_POD
    LB --> CMS_POD
    LB --> AUTH_POD

    WEB_POD --> AUTH_POD
    WEB_POD --> CMS_POD

    CMS_POD --> POSTGRES
    CMS_POD --> REDIS_SVC
    CMS_POD --> NATS_SVC

    AUTH_POD --> POSTGRES

    WORKER_POD --> KAFKA_SVC
    WORKER_POD --> NATS_SVC
    WORKER_POD --> REDIS_SVC
    WORKER_POD --> POSTGRES
    WORKER_POD --> MONGO_RS

    CONNECT_SVC --> MONGO_RS
    CONNECT_SVC --> KAFKA_SVC

    WORKER_POD --> PROM_SVC
    CMS_POD --> PROM_SVC
    WORKER_POD --> OTEL_SVC
    CMS_POD --> OTEL_SVC

    %% Optional deployment relationships
    KAFKA_SVC --> POSTGRES
```

Nếu muốn ghi chú deployment dưới sơ đồ, bạn có thể thêm:
- `cdc-cms-web`: stateless frontend
- `cdc-cms-service`, `cdc-auth-service`, `centralized-data-service`: scale horizontally
- `PostgreSQL`, `MongoDB`, `Kafka`, `Redis`, `NATS`: stateful services
- `Debezium/Kafka Connect`: bridge CDC từ MongoDB sang Kafka

**3) Component Diagram chi tiết cho `centralized-data-service`**

```mermaid
flowchart LR
    subgraph EXT["External Interfaces"]
        KAFKA["Kafka Topics"]
        NATS["NATS / JetStream"]
        MONGO["MongoDB"]
        PG["PostgreSQL"]
        REDIS["Redis"]
    end

    subgraph CORE["centralized-data-service"]
        SERVER["WorkerServer"]

        subgraph ING["Ingestion"]
            KC["KafkaConsumer"]
            CP["ConsumerPool"]
            EH["EventHandler"]
            SV["SchemaValidator"]
        end

        subgraph XFORM["Transformation / Persistence"]
            DM["DynamicMapper"]
            SA["SchemaAdapter"]
            BB["BatchBuffer"]
            REGSVC["RegistryService"]
            MASK["MaskingService"]
        end

        subgraph SCHEMA["Schema Evolution"]
            SI["SchemaInspector"]
            TSDET["TimestampDetector"]
        end

        subgraph RECON["Data Integrity"]
            RCA["ReconCore"]
            RSA["ReconSourceAgent"]
            RDA["ReconDestAgent"]
            RH["ReconHealer"]
            SIG["DebeziumSignalClient"]
            FCA["FullCountAggregator"]
        end

        subgraph FT["Fault Tolerance"]
            DLQH["DLQHandler"]
            DLQSM["DLQStateMachine"]
        end

        subgraph OPS["Ops / Scheduling"]
            CMD["CommandHandler"]
            TRANS["TransmuteHandler / Module"]
            TSCH["TransmuteScheduler"]
            PART["PartitionDropper"]
            ACT["ActivityLogger"]
        end
    end

    SERVER --> KC
    SERVER --> CP
    SERVER --> EH
    SERVER --> SV
    SERVER --> DM
    SERVER --> SA
    SERVER --> BB
    SERVER --> REGSVC
    SERVER --> MASK
    SERVER --> SI
    SERVER --> TSDET
    SERVER --> RCA
    SERVER --> RH
    SERVER --> DLQH
    SERVER --> DLQSM
    SERVER --> CMD
    SERVER --> TRANS
    SERVER --> TSCH
    SERVER --> PART
    SERVER --> ACT
    SERVER --> FCA

    KAFKA --> KC
    NATS --> CP
    NATS --> CMD
    NATS --> DLQSM
    MONGO --> RSA
    MONGO --> RH
    MONGO --> TSDET
    PG --> SA
    PG --> BB
    PG --> RCA
    PG --> RH
    PG --> DLQSM
    PG --> REGSVC
    REDIS --> SI
    REDIS --> RCA

    KC --> EH
    CP --> EH
    EH --> SV
    EH --> DM
    EH --> SI
    EH --> BB

    DM --> REGSVC
    DM --> SA
    DM --> MASK

    SI --> MASK
    SI --> REGSVC
    SI --> NATS
    SI --> PG

    BB --> SA
    BB --> PG

    RCA --> RSA
    RCA --> RDA
    RCA --> RH
    RCA --> SIG
    RDA --> PG
    RSA --> MONGO
    RH --> SA
    RH --> MASK
    RH --> PG
    RH --> MONGO

    DLQH --> MASK
    DLQH --> PG
    DLQH --> NATS

    DLQSM --> PG
    DLQSM --> NATS

    CMD --> NATS
    CMD --> PG
    TRANS --> NATS
    TRANS --> PG
    TSCH --> NATS
    PART --> PG
    FCA --> PG
    FCA --> MONGO
```

**4) Một bản sequence ngắn cho docs “critical path”**

```mermaid
sequenceDiagram
    participant Mongo as MongoDB
    participant Debezium as Debezium
    participant Kafka as Kafka
    participant Worker as centralized-data-service
    participant Mapper as DynamicMapper
    participant Schema as SchemaInspector
    participant PG as PostgreSQL
    participant DLQ as DLQ State Machine

    Mongo->>Debezium: Change event
    Debezium->>Kafka: Publish CDC message
    Kafka->>Worker: Consume message
    Worker->>Schema: Validate / inspect drift
    Worker->>Mapper: Map + mask raw data
    Mapper->>PG: Upsert via SchemaAdapter
    alt Processing error
        Worker->>PG: Insert failed_sync_logs
        DLQ->>PG: Poll due retries
        DLQ->>Worker: Replay message
    end
```

**Gợi ý cách đặt vào docs**
- `01-overview.md`: dùng sơ đồ số 1
- `02-deployment.md`: dùng sơ đồ số 2
- `03-worker-components.md`: dùng sơ đồ số 3
- `04-critical-paths.md`: dùng sequence ở cuối
