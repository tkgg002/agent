# Workspace Context - PostgreSQL Schema Support in Registry

Hỗ trợ PostgreSQL schema trong luồng "Register New Source Object -> Shadow Object" theo quy ước:
- `source_db` (Mongo DB name) tương đương với Postgres Schema.
- `source_table` (Mongo Collection name) tương đương với Postgres Table.
- `database name` thực tế của Postgres được lấy từ cấu hình connector liên kết (Connection Registry).

## Repositories
- Frontend: `cdc-cms-web`
- Backend: `cdc-cms-service`
