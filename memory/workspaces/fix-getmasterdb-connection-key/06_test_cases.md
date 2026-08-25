# 06_test_cases.md — Validation & Test Cases

## I. MA TRẬN KIỂM THỬ (TEST MATRIX)

| ID | Kịch bản kiểm thử | Dữ liệu đầu vào | Kết quả kỳ vọng | Phương thức verify |
|---|---|---|---|---|
| **TC-01** | RunNow với schema cụ thể | Schedule ID của `master_bidv_connector_service.bank_requests` | NATS payload chứa `master_table: "master_bidv_connector_service.bank_requests"`, Worker bốc đúng binding | Trace log & NATS payload inspection |
| **TC-02** | Scheduler tick với binding có `master_schema` | Binding id=X, schema=`master_bidv_connector_service`, table=`bank_requests` | `masterFQN` = `"master_bidv_connector_service.bank_requests"` | SQL execution check |
| **TC-03** | Scheduler tick với binding có `master_schema` là `NULL` | Binding id=Y, schema=NULL, table=`orders` | `masterFQN` = `"public.orders"` (không bị NULL) | SQL execution check |
| **TC-04** | API POST `/api/v1/schedules` tạo lịch mới | `{master_schema: "master_bidv", master_table: "bank_requests", mode: "cron", ...}` | HTTP 201 Created, schedule được gắn đúng `master_binding_id` | API test |
| **TC-05** | API POST `/api/v1/schedules` tạo lịch không truyền schema | `{master_table: "orders", mode: "cron", ...}` | HTTP 201 Created, tìm đúng binding public | API test |
| **TC-06** | Shadow fanout trigger Transmute | Event shadow table `bank_requests` | `ListMasterTablesByShadowIdentity` trả về FQN chuẩn | Unit test repository |

## II. LỆNH CHẠY KIỂM THỬ AUTOMATED

```bash
# Build verification cả 2 services
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./internal/... ./cmd/...
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./internal/... ./cmd/...
```
