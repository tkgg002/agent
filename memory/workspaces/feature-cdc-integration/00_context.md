# Workspace Context: CDC Integration (Hybrid Approach)

## Scope

Triển khai giải pháp CDC (Change Data Capture)하이brid kết hợp Debezium và Airbyte cho hệ thống GooPay, tập trung vào phần Developer (Logic & Integration).

### In Scope (Developer Tasks)
1. **Go CDC Worker (NATS Stream)**: Xây dựng service Go xử lý real-time data từ Debezium qua NATS với **Dynamic Mapping Engine**
2. **PostgreSQL Schema Mapping**: Thiết kế schema với **JSONB Landing Zone** nhận dữ liệu từ cả Airbyte và Go Worker
3. **Event Bridge (Postgres → NATS)**: Phát NATS events từ dữ liệu Airbyte ghi vào Postgres
4. **Data Reconciliation**: Script đối soát dữ liệu giữa luồng nhanh (Debezium) và luồng chậm (Airbyte)
5. **Schema Drift Detection Module**: Go service tự động phát hiện schema changes và trigger approval workflow
6. **CMS Integration**: Xây dựng approval workflow cho schema changes qua CMS
7. **Migration Automation**: CI/CD integration với Airbyte API cho automated schema evolution

### Out of Scope (DevOps Tasks)
- Cấu hình Debezium Server infrastructure
- Triển khai Airbyte pipelines
- Resource isolation và monitoring dashboard

## Business Context

GooPay có ~60 microservices với 2 loại dữ liệu cần đồng bộ:
- **Critical Real-time Data**: Orders, Payments, Wallet Transactions → cần độ trễ thấp (ms)
- **Batch Analytics Data**: Logs, User Activity, Reports → chấp nhận độ trễ cao hơn (phút)

## Technical Strategy

**Hybrid Approach**:
- **Debezium + NATS**: Real-time CDC cho bảng quan trọng → Go Worker xử lý → PostgreSQL
- **Airbyte**: Batch sync cho bảng phân tích → Direct write PostgreSQL

**Target Database**: PostgreSQL (Single Source of Truth)

## Key Components

| Component | Technology | Purpose |
|-----------|-----------|---------|
| CDC Worker | Go (Generic Processor) | Consume NATS events, dynamic mapping theo CMS rules, write to Postgres |
| Event Bridge | Go + Postgres Trigger | Phát NATS events từ Postgres cho Moleculer services |
| Schema Inspector | Go Module | Detect schema drift, compare JSON payload với DB schema |
| CMS Service | Go/Node.js + React UI | Approval workflow cho schema changes, manage mapping rules |
| Dynamic Mapper | Go Engine | Map source fields → target columns dựa trên `cdc_mapping_rules` |
| JSONB Landing Zone | PostgreSQL Column | Cột `_raw_data` lưu toàn bộ JSON thô để zero data loss |
| Migration Orchestrator | CI/CD Pipeline | Tự động ALTER TABLE và trigger Airbyte API |
| Reconciliation Job | Go Script/Cron | Kiểm tra consistency giữa 2 luồng |

## Dependencies

- **Infrastructure**: Debezium Server, NATS JetStream, Airbyte, PostgreSQL cluster
- **Existing Services**: Moleculer services cần subscribe NATS events
- **Database Access**: Read access vào MongoDB/MySQL sources, Write access vào PostgreSQL target

## Success Criteria

- [ ] CDC Worker xử lý được events từ NATS với latency < 100ms
- [ ] PostgreSQL schema không conflict giữa Airbyte và Go Worker writes
- [ ] Event Bridge phát events đúng format cho Moleculer services
- [ ] Reconciliation script phát hiện data drift trong vòng 5 phút
- [ ] **Zero data loss** trong quá trình migration (JSONB Landing Zone)
- [ ] Schema changes được detect tự động trong vòng < 1 phút
- [ ] CMS approval workflow hoạt động end-to-end
- [ ] Dynamic mapping cho phép thêm field mới **không cần restart** Go Worker
- [ ] Migration CI/CD tự động ALTER TABLE trước khi Airbyte sync
- [ ] Config reload qua NATS event `RELOAD_CONFIG` trong < 5 giây
