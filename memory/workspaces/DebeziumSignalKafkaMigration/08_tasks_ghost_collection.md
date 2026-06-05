# 08 Tasks — Ghost Collection

| # | Task | Owner | DoD | Status |
|---|---|---|---|---|
| G1 | Revert docker-compose plugins 2.7.4 → 2.5.4 | Muscle | git diff confirm 3 dòng → 2.5.4 | DONE |
| G2 | Recreate gpay-kafka-connect | Muscle | container Up + 3 plugin install OK (REST `/connector-plugins` list ≥ 3 debezium plugin) | TODO |
| G3 | Tạo Mongo `cdc_system.debezium_watermarks` (empty) | Muscle | `mongosh ... getCollectionNames()` chứa `debezium_watermarks`; `countDocuments({})` = 0 | TODO |
| G4 | PATCH `goopay-local` config thêm `signal.data.collection` | Muscle | REST GET config response chứa key đó | TODO |
| G5 | Restart `goopay-local` | Muscle | state RUNNING, task[0] state RUNNING | TODO |
| G6 | Capture shadow PG count BEFORE | Muscle | số được lưu vào report | TODO |
| G7 | Capture source Mongo count | Muscle | số được lưu vào report | TODO |
| G8 | NATS publish `cdc.cmd.debezium-snapshot` | Muscle | worker log `debezium signal published` | TODO |
| G9 | Wait 60s + capture Connect log | Muscle | có `Requested INCREMENTAL`, zero `NullPointerException` trong window | TODO |
| G10 | Capture shadow PG count AFTER | Muscle | delta > 0 | TODO |
| G11 | Viết `report_2026-05-20_snapshot-ghost-collection.md` | Muscle | file vật lý + số liệu thật | TODO |
| G12 | APPEND `05_progress.md` (entry phase ghost-collection) | Muscle | file grew, không overwrite | TODO |
| G13 | APPEND lesson Global Pattern vào `lessons.md` | Muscle | có heading mới, không xóa cũ | TODO |
