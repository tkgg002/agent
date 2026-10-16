# Yêu cầu chi tiết (Specs)

- Bổ sung chế độ tạo JSON subset type (source_field_types) cho 18 fields cốt lõi từ MongoDB sang PostgreSQL.
- Contract chạy tại `scripts/migrate-mongo-to-postgres.cjs` -> `buildCorePayloadSourceFieldTypes()`.
- Chế độ mới chạy song song/opt-in bên cạnh chế độ JSON subset value hiện tại.
- Giữ BSON wrapper, không dùng số JavaScript.
- _id, transId missing/null hoặc BSON undefined phải bị reject.
- Không tự đổi các giá trị empty/falsy thành null/missing.
