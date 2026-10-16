# PHÂN TÍCH GỐC RỄ: TOPIC `cdc.goopaylocal` PUSH NHẦM VÀO `cdc_transaction_history`

## Timestamp: 2026-09-17T14:40:00+07:00
## Workspace: FixKafkaConsumerSourceRouteMultiConn20260916

---

## 1. Hiện tượng (Symptom)
Topic `cdc.goopaylocal.core-trans-proxy-history-service.trans-his` đang push dữ liệu vào bảng Master `public.cdc_transaction_history` (của Binding #55, connection `traitestmongodevct`), trong khi topic này thuộc về Kafka connector `goopaylocal` — CHƯA có connection riêng trong `connection_registry` nên KHÔNG CÓ shadow binding riêng.

## 2. Dữ liệu DB thật (`cdc_dw`)
```
 id  | source_object_name |         source_database          | source_conn        | sb_id | shadow_schema             | shadow_table | mb_id | master_schema                           | master_table            | mb_active
-----+--------------------+----------------------------------+--------------------+-------+---------------------------+--------------+-------+-----------------------------------------+-------------------------+----------
 181 | trans-his          | core-trans-proxy-history-service | traitestctphs      |   181 | shadow_traitestctphs      | trans_his    |    45 | master_core_trans_proxy_history_service | trans_his               | t
 204 | trans-his          | core-trans-proxy-history-service | traitestmongodevct |   204 | shadow_traitestmongodevct | trans_his    |    55 | public                                  | cdc_transaction_history | t
```

Connection registry: KHÔNG có `goopaylocal` — chỉ có `traitestctphs` (ID 296) và `traitestmongodevct` (ID 323).

## 3. Trace Root Cause — Luồng xử lý Kafka Message

### Bước 1: `kafka_consumer.go` trích xuất `sourceConnCode`
- **Debezium `source.name`** = tên connector prefix (thường = `cdc.goopaylocal`)
- Vòng lặp strip prefix (dòng 671-681):
  - Config `topicPrefix` chứa `cdc.goopaylocal`
  - Strip `cdc.goopaylocal.` từ `sourceConnCode` → kết quả rỗng `""`
- Hoặc nếu `source.name` = `goopaylocal` → strip `goopaylocal` → `""`

### Bước 2: Fallback parse topic (dòng 684-691)
- Topic: `cdc.goopaylocal.core-trans-proxy-history-service.trans-his`
- `parts = ["cdc", "goopaylocal", "core-trans-proxy-history-service", "trans-his"]` → 4 parts
- `len(parts) == 4 && parts[0] == "cdc"` → TRUE
- `!IsGenericConnectionCode(parts[1])` → `IsGenericConnectionCode("goopaylocal")` = **TRUE** (vì `cdc.goopaylocal` trong `topicPrefix` → `RegisterGenericPrefixes` đã đăng ký `goopaylocal` làm generic prefix)
- Nên: **KHÔNG gán** `sourceConnCode = parts[1]`

### Bước 3: Final check (dòng 695)
- `sourceConnCode` vẫn `""` → giữ rỗng

### Bước 4: `event_handler.go` processEvent (dòng 267)
- Gọi `ResolveSourceRoutes("core-trans-proxy-history-service", "trans-his", "")`
- `sourceConn` = nil (đã bị strip/reset)
- `buildRouteLookupKeys` tạo: `["core-trans-proxy-history-service|trans-his", ..., "trans-his"]`
- `routeCache["core-trans-proxy-history-service|trans-his"]` chứa **CẢ 2 routes**:
  - Route #1: source_conn=`traitestctphs` → shadow `shadow_traitestctphs.trans_his` → master `master_core_trans_proxy_history_service.trans_his`
  - Route #2: source_conn=`traitestmongodevct` → shadow `shadow_traitestmongodevct.trans_his` → master `public.cdc_transaction_history`
- **Không có filter theo connection** (vì `sourceConn` = `""`)
- → Fan-out ghi vào **CẢ 2 shadow tables** → transmute push sang **CẢ 2 master tables** bao gồm `cdc_transaction_history`

## 4. BẢN CHẤT VẤN ĐỀ

Connector `goopaylocal` trên Kafka **CHƯA ĐƯỢC đăng ký** là connection riêng trong `connection_registry` (DB chỉ có `traitestctphs` và `traitestmongodevct`). Khi topic `cdc.goopaylocal.*` đi vào:

1. `goopaylocal` bị nhận diện là generic prefix → sourceConnCode reset về rỗng
2. Route resolution không có connection filter → bốc **TẤT CẢ routes** cho `trans-his`
3. Fan-out ghi dữ liệu vào cả 2 shadow tables → transmute push sang cả `cdc_transaction_history` (sai) lẫn `master_core_trans_proxy_history_service.trans_his` (cũng sai — vì data này thuộc `goopaylocal`, không thuộc `traitestctphs` hay `traitestmongodevct`)

## 5. GIẢI PHÁP CẦN XÁC NHẬN TỪ USER

**Câu hỏi cốt lõi:** Connector `goopaylocal` đại diện cho nguồn dữ liệu gì?

- **Nếu `goopaylocal` = connector dev/test tổng quát (generic) chưa map vào connection cụ thể nào:**
  → Cần tạo `connection_registry` + `source_object_registry` + `shadow_binding` riêng cho `goopaylocal` để phân lập routing.
  → Hoặc: Khi `sourceConn` rỗng và có nhiều hơn 1 route cho cùng `sourceDB + sourceTable`, hệ thống cần LOG CẢNH BÁO và KHÔNG fan-out mù quáng.

- **Nếu `goopaylocal` chính là `traitestmongodevct` (alias dev):**
  → Cần tạo mapping alias `goopaylocal → traitestmongodevct` trong `connection_registry` hoặc trong config.

- **Nếu `goopaylocal` là connector cũ không còn cần thiết:**
  → Cần loại bỏ topic khỏi subscription hoặc thêm logic reject khi không tìm thấy connection match.
