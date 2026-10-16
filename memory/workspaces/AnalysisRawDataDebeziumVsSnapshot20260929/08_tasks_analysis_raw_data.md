# 08 Tasks: Phân tích so sánh _raw_data Debezium vs Snapshot:v2

- [x] Task 1: Đối soát chi tiết 2 cấu trúc JSON payload (các trường `_id`, `createdAt`, `lastUpdatedAt`, v.v.).
- [x] Task 2: Điều tra mã nguồn sinh ra 2 định dạng:
  - Debezium: `MongoDbConnector` + Avro/JSON converter -> `kafka_consumer.go`.
  - Snapshot:v2: Go mongo-driver `bson.MarshalExtJSON(doc, false, false)` -> `snapshot_runner_handler.go`.
- [x] Task 3: Điều tra khả năng thích ứng và cơ chế xử lý trong toàn bộ pipeline CDC:
  - `dynamic_mapper.go`: `unwrapMongoTypes` cho typed columns.
  - `mapping_utils.go`: `BuildCastExpr` cho SQL shadow/views.
  - `transmuter_utils.go` & `flatten.go`: `unwrapMongoExtJSON` cho Master upsert.
- [x] Task 4: Viết tài liệu phân tích chuyên sâu `13_analysis_raw_data_debezium_vs_snapshot.md` và giải đáp toàn diện cho User.
