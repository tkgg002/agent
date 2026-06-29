# Test & Validation: Fallback Default Schema từ Connection Registry

## 1. Kết quả kiểm thử tự động (Unit Tests)

### Lệnh chạy test:
```bash
go test -v ./internal/service/source/... ./internal/handler/orchestration/...
```

### Output kết quả:
```text
=== RUN   TestMetadataRegistryService_MaskingIsolation
--- PASS: TestMetadataRegistryService_MaskingIsolation (0.00s)
=== RUN   TestBuildDSNFromFieldsPatched
=== RUN   TestBuildDSNFromFieldsPatched/postgres_default_schema_is_nil
=== RUN   TestBuildDSNFromFieldsPatched/postgres_default_schema_is_populated
--- PASS: TestBuildDSNFromFieldsPatched (0.00s)
    --- PASS: TestBuildDSNFromFieldsPatched/postgres_default_schema_is_nil (0.00s)
    --- PASS: TestBuildDSNFromFieldsPatched/postgres_default_schema_is_populated (0.00s)
=== RUN   TestHelpers_TopicNameFor
--- PASS: TestHelpers_TopicNameFor (0.00s)
=== RUN   TestHelpers_ShadowSchemaFor
--- PASS: TestHelpers_ShadowSchemaFor (0.00s)
=== RUN   TestExtendDatabaseList_NewValue
--- PASS: TestExtendDatabaseList_NewValue (0.00s)
=== RUN   TestExtendDatabaseList_AlreadyPresent
--- PASS: TestExtendDatabaseList_AlreadyPresent (0.00s)
=== RUN   TestExtendDatabaseList_EmptyConfig
--- PASS: TestExtendDatabaseList_EmptyConfig (0.00s)
=== RUN   TestExtendDebeziumInclude_Mongo_BothTiers
--- PASS: TestExtendDebeziumInclude_Mongo_BothTiers (0.00s)
=== RUN   TestExtendDebeziumInclude_Mongo_DBExistsCollNew
--- PASS: TestExtendDebeziumInclude_Mongo_DBExistsCollNew (0.00s)
=== RUN   TestExtendDebeziumInclude_PG_DBLockMismatch
--- PASS: TestExtendDebeziumInclude_PG_DBLockMismatch (0.00s)
PASS
ok  	centralized-data-service/internal/service/source	0.800s

=== RUN   TestMarkProgressDone_UpdatesStatusDone
=== RUN   TestMarkProgressDone_UpdatesStatusDone/complete
=== RUN   TestMarkProgressDone_UpdatesStatusDone/partial_persisted_dedup
=== RUN   TestMarkProgressDone_UpdatesStatusDone/no_baseline
--- PASS: TestMarkProgressDone_UpdatesStatusDone (0.00s)
    --- PASS: TestMarkProgressDone_UpdatesStatusDone/complete (0.00s)
    --- PASS: TestMarkProgressDone_UpdatesStatusDone/partial_persisted_dedup (0.00s)
    --- PASS: TestMarkProgressDone_UpdatesStatusDone/no_baseline (0.00s)
=== RUN   TestCursorEarlyExit_NoPrematureBreak
--- PASS: TestCursorEarlyExit_NoPrematureBreak (0.00s)
=== RUN   TestPause_NoFallThroughToDone
--- PASS: TestPause_NoFallThroughToDone (0.00s)
=== RUN   TestPostgreSQLSnapshotRunner
--- PASS: TestPostgreSQLSnapshotRunner (0.00s)
PASS
ok  	centralized-data-service/internal/handler/orchestration	1.294s
```

## 2. Kết quả kiểm thử thực tế (Manual Verification)
- **Object 52 (pg_dev - public)**: Khởi chạy snapshot và đồng bộ thành công 5 dòng dữ liệu vào shadow table.
- **Object 55 (pg_dev2 - cdc_schema_testing)**: Khởi chạy snapshot thành công, DSN tự động append `search_path=cdc_schema_testing` và kết nối thành công, không gặp lỗi SASL Auth.
