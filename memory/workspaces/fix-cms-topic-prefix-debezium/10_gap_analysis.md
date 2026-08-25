# 10 - Architectural Gap Analysis: Callchain Topic Dependency Map (Brain)

## 1. Bản đồ 7 Điểm chạm Topic trong Hệ thống (The 7 Touchpoints)

| # | Thành phần (Component) | File / Vị trí | Trách nhiệm | Nguy cơ nếu không đồng bộ |
|---|---|---|---|---|
| **1** | **CMS Web Generator** | `cdc-cms-web/src/pages/SourceConnectors.tsx` | Tạo config `topic.prefix` gửi Kafka Connect | Gây duplicate segment hoặc topic collision |
| **2** | **Kafka Connect Runtime** | Debezium MongoDB / PG / MySQL / SFTP | Sinh topic vật lý trên Kafka Broker | Topic không theo chuẩn naming convention |
| **3** | **CDS Topic Discovery** | `internal/handler/shadow/topic_helper.go` (`filterMatchingTopics`) | Lọc topic theo whitelist `configuredPrefixes` & `debeziumTables` | **Bẫy Drop Topic:** Nếu prefix không khớp whitelist, consumer skip không đọc |
| **4** | **CDS Event Router** | `internal/handler/shadow/event_handler.go` (`HandleRaw`, `extractSourceAndTable`) | Trích xuất connection/db/table & resolve route | **Bẫy Nhầm Shadow:** Ghi nhầm dữ liệu sang cluster khác |
| **5** | **CDS Metadata Registry** | `internal/service/source/metadata_registry_utils.go` (`buildRouteLookupKeys`) | Tạo lookup keys map từ DB sang Route | Route cache không phân tách được 2 cluster trùng DB/Table |
| **6** | **CDS Observability** | `pkgs/observability/trace_helpers.go` (`ParseDebeziumTopic`) | Tách topic thành Spans, OTel tags | Gắn nhầm tag engine, schema, db trên Grafana/SigNoz |
| **7** | **CDS Sink Worker / DLQ** | `internal/sinkworker/utils.go` & `internal/handler/dlq/dlq_handler.go` | Fallback shadow target & DLQ routing | `extractShadowTarget` lấy sai schema name |

---

## 2. Chi tiết Lỗ hổng Kỹ thuật (Technical Gaps)

### Gap 1: `topic_helper.go` Whitelist Filtering
- Code hiện tại:
  ```go
  if strings.Contains(matched, "gpay") || strings.Contains(matched, "goopay") || strings.Contains(matched, "mariadb")
  ```
- Nếu prefix đổi thành `cdc.mongo_core`, `matched` sẽ không chứa `"goopay"` -> Topic bị bỏ qua khỏi danh sách consume nếu không chuẩn hóa logic filter!

### Gap 2: `event_handler.go` Fallback Hardcode Index
- Code hiện tại:
  ```go
  parts := strings.Split(subject, ".")
  if len(parts) >= 4 {
      db = parts[2]
      table = parts[3]
  }
  ```
- Nếu PostgreSQL có 5 segments (`cdc.pg_main.db.public.table`), `parts[3]` là schema `public`, KHÔNG phải table! Cần parse đúng theo `dbKind` hoặc dùng regex chuẩn.

### Gap 3: `buildRouteLookupKeys` thiếu Connection Scoping
- Khi có 2 connection cùng chứa `payment-services|payments`, `ResolveSourceRoutes` chỉ tìm theo `db|table` sẽ lấy danh sách route chung. BẮT BUỘC phải ưu tiên key dạng `connection_code:db|table`.
