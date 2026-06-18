# 01 — Requirements: Saga Pattern & OTel Tracing

## Yêu cầu chức năng

### REQ-01: Saga Pattern
- Các luồng multi-step có side-effects đa hệ thống (DB + NATS + External API) phải có compensation khi step giữa thất bại
- Compensation phải chạy theo thứ tự ngược (reverse order), best-effort
- Khi compensation thất bại → log ERROR rõ ràng để operator can thiệp thủ công
- Không được dùng distributed saga coordinator (quá phức tạp) — dùng choreography local

### REQ-02: OTel Tracing
- Mọi HTTP request phải extract W3C `traceparent` header → inject vào context
- CommandBus `Execute` và `Dispatch` phải tạo span
- Saga Runner phải tạo parent span + per-step child span
- Logs phải inject `trace_id` / `span_id` khi OTel enabled
- Không break backward compatibility khi OTel disabled (noop spans)

### REQ-03: Báo cáo
- Phải có `report_saga_tracing_2026-06-18.md` với danh sách files thay đổi và LOC delta thực tế
- Không được báo Done nếu chưa có `go build` + `go vet` + `go test` EXIT=0

## Yêu cầu phi chức năng

- **Zero overhead khi OTel disabled**: OTel SDK trả về noop span → không ảnh hưởng performance
- **Thread-safe**: Saga Runner dùng local slice (không global state)
- **Idiomatic Go**: Named return + defer pattern cho span lifecycle

## Saga Priority (theo risk)

| ID | Luồng | Risk nếu thiếu saga |
|----|-------|---------------------|
| S1 | `registry.register` (Source) | Registry row tồn tại, Shadow DDL không có → CDC broken |
| S2 | `approve_schema_proposal` (Governance) | DDL partial apply → schema drift giữa shadow ↔ master |
| S3 | `master.create` (Master) | Master binding orphan, mapping rules bị mất |
| S4 | `master.approve` (Governance) | Schema approved nhưng worker chưa materialize |
| S5 | `connector.create/delete` (Source) | DB ↔ KafkaConnect state desync |

## Không trong scope

- Saga cho AsyncCommand (worker tự retry)
- Saga cho single-step atomic DB writes
- Distributed saga coordinator (overkill)
- Handler-level span cho từng SyncHandler cụ thể (defer Phase 2)
