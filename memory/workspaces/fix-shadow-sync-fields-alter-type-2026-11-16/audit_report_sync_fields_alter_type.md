# Audit Report (Tier T2 Consolidated Validation + Report + S5 Audit) — `sync_fields_alter_type`

- **Workspace:** `fix-shadow-sync-fields-alter-type-2026-11-16`
- **Tier:** `T2` | **Mode:** `standard` (Pure Forward Core — Lesson `L097`, không can thiệp hay khôi phục dữ liệu lịch sử cũ)
- **Ngày hoàn tất:** `2026-11-17`

---

## 0. Bảng Tổng Hợp Bằng Chứng Thực Thi (`06_evidence/`)

| Biên lai (`EV-...`) | Phân loại (`kind`) | Repo | Lệnh thực thi | Kết quả |
| :--- | :--- | :--- | :--- | :--- |
| `EV-20261117T103553-620b` | `runtime` | `centralized-data-service` | `go build ./internal/... ./cmd/...` | `PASS` (`EV-20261117T103553-620b`) |
| `EV-20261117T103602-f951` | `unit` | `centralized-data-service` | `go vet ./internal/... ./cmd/...` | `PASS` (`EV-20261117T103602-f951`) |
| `EV-20261117T103604-321f` | `integration` | `centralized-data-service` | `go test -v -count=1 -run TestAlterColumn_TextToJSONB_ForwardCore_RealPostgres ./test/internal/service/...` | `PASS` chạy thật trên PostgreSQL 15 `gpay-postgres-shadow` (`EV-20261117T103604-321f`) |
| `EV-20261117T103610-d6b0` | `unit` | `centralized-data-service` | `go test -v -count=1 -run TestCoerceAndConvertType_MapAndSliceOnTextColumn ./test/internal/service/...` | `PASS` (`EV-20261117T103610-d6b0`) |
| `EV-20261117T103636-fd13` | `unit` | `centralized-data-service` | `go test -v -count=1 ./internal/service/shadow/... ./internal/handler/shadow/...` | `PASS` (`EV-20261117T103636-fd13`) |
| `EV-20261117T103642-2093` | `unit` | `centralized-data-service` | `go test -v -count=1 -run TestSchemaAdapter ./test/internal/service/...` | `PASS` toàn bộ 7 test hồi quy `TestSchemaAdapter_*` (`EV-20261117T103642-2093`) |
| `EV-20261117T103646-1fd5` | `runtime` | `cdc-cms-service` | `go build ./internal/... ./cmd/...` | `PASS` (`EV-20261117T103646-1fd5`) |
| `EV-20261117T103650-4cb8` | `unit` | `cdc-cms-service` | `go vet ./internal/... ./cmd/...` | `PASS` (`EV-20261117T103650-4cb8`) |
| `EV-20261117T103652-4ce1` | `unit` | `cdc-cms-service` | `go test -v -count=1 ./internal/app/commands/source/... ./internal/api/source/...` | `PASS` (`EV-20261117T103652-4ce1`) |
| `EV-20261117T103656-434c` | `runtime` | `cdc-cms-web` | `npm run build` | `PASS` (`EV-20261117T103656-434c`) |

---

## 1. Phản biện từng dòng git diff & từng task (Line-by-Line Critique)

Đã chạy `git diff` và `wc -l` sau lần sửa cuối cùng trên cả 3 repository, đảm bảo không có God-file mới và không chạm các file dirty ngoài phạm vi:

1. **`centralized-data-service/internal/service/shadow/dynamic_mapper.go` (`431` dòng, sửa dòng `233-241` — Task `F1`):**
   - Trong `convertType` nhánh `default` (`TEXT`, `VARCHAR`, ...), bổ sung `switch val.(type)` cho `map[string]interface{}` và `[]interface{}` để gọi `json.Marshal(val)` trả chuỗi JSON chuẩn (`{"..."}` hoặc `[...]`) trước khi fallback về `fmt.Sprintf("%v", val)`.
   - Kiểm chứng đơn vị đạt `PASS` (`EV-20261117T103610-d6b0`) và hồi quy `TestSchemaAdapter_*` đạt `PASS` (`EV-20261117T103642-2093`).
2. **`centralized-data-service/internal/service/shadow/schema_adapter_coerce.go` (`397` dòng, sửa dòng `31-43` — Task `F1`):**
   - Trong `CoerceValue` khi cột đích hiện tại trên bảng Shadow là `character varying`, `text`, hoặc `varchar`, bổ sung nhánh `map[string]interface{}, []interface{}` gọi `json.Marshal(val)` thay vì `fmt.Sprintf("%v", val)`. Nhờ đó mọi nested object hoặc array từ Debezium/Mongo ghi tạm vào cột `TEXT` đều là JSON hợp lệ, chấm dứt tận gốc chuỗi `"map[...]"` làm gãy `ALTER COLUMN TYPE JSONB` về sau.
3. **`centralized-data-service/internal/service/shadow/schema_adapter.go` (`1163` dòng, sửa dòng `822-862` — Task `F2`):**
   - Giữ nguyên chữ ký 6 tham số của `AlterColumn`, kiểm tra chặt `sqlutil.IsSafeIdent(schemaName|tableName|columnName)` và `IsTypeWhitelisted(cleanType)`, định danh qua `sqlutil.QualifiedTable` + `sqlutil.QuoteIdent` (chống SQL Injection).
   - Khi `act == "alter_type"` sang `JSONB` hoặc `JSON`, dùng biểu thức chuẩn PostgreSQL `NULLIF(btrim(%s::text), '')::%s` để chuỗi rỗng `""` chuyển thành `NULL` hợp lệ, còn dữ liệu phi-JSON sẽ fail-fast trung thực. Thực thi gói trong `Transaction` có `SET LOCAL lock_timeout = '10s'`. Tuân thủ tuyệt đối `L097`: `0` dòng truy vấn `_raw_data`, `0` regex vá `"map["`.
   - Đã kiểm chứng tích hợp trên PostgreSQL 15 thật đạt `PASS` (`EV-20261117T103604-321f`).
4. **`centralized-data-service/internal/handler/shadow/schema_ddl_handler.go` (`739` dòng, sửa dòng `275-358`, `453-466`, `711-714` — Task `F3`):**
   - Trong `HandleCreateDefaultColumns`, gom đầy đủ lỗi từ `ListActiveBySourceObject*` và `AlterColumn` (`alter_type` lẫn `add`) vào `ddlErrors`.
   - Khi `columnsAdded > 0 || columnsAlteredType > 0`, gọi `h.schemaAdapter.InvalidateCacheInSchema(schemaName, payload.TargetTable)` và bắn NATS `"schema.config.reload"` ngay tại bước 6 (và đồng bộ tương tự ở `HandleAlterColumn` dòng `711-714`), không còn bị phụ thuộc điều kiện `TimestampField != nil` ở bước 9.
   - Gộp `columnsAlteredType` vào `RowsAffected: columnsAdded + columnsAlteredType + count`; khi `len(ddlErrors) > 0` trả `Status: "error"` kèm `Error: strings.Join(ddlErrors, "; ")` qua `PublishResult` (đã qua `SanitizeAdminError` ở `base_handler.go:97`).
5. **`cdc-cms-service` (`sync_shadow_columns.go` `70` dòng, `sync_shadow_columns_test.go` `64` dòng, `server.go` dòng `301`, `source_object_actions_handler.go` dòng `242-282` — Task `F4`):**
   - Bổ sung CQRS command `SyncShadowColumnsCommand` (`source.sync-shadow-columns`) gọi đồng bộ `h.rpc.Request(ctx, "cdc.cmd.create-default-columns", replySubj, payload, 30*time.Second)` qua `ports.RPCClient`.
   - Khi `?wait=true` hoặc `wait=1`, `CreateDefaultColumnsV2` thực thi đồng bộ: nếu worker trả `status == "error"` thì trả HTTP `422 Unprocessable Entity` kèm lỗi thật; nếu quá hạn `30s` (`ErrWorkerSyncTimeout`) thì trả HTTP `504 Gateway Timeout`; nếu thành công trả HTTP `200` kèm `rows_affected`. Các caller không truyền `wait` vẫn giữ nguyên `202 Accepted` bất đồng bộ.
   - Đã kiểm chứng build đạt `PASS` (`EV-20261117T103646-1fd5`), `go vet` đạt `PASS` (`EV-20261117T103650-4cb8`), và unit test command + API đạt `PASS` (`EV-20261117T103652-4ce1`).
6. **`cdc-cms-web/src/pages/MappingFieldsPage.tsx` (`834` dòng, sửa dòng `166-190`, `484-495` — Task `F4`):**
   - `handleSyncFields` gọi `POST /api/v1/source-objects/:id/create-default-columns?wait=true(&binding_id=...)`, chờ kết quả worker xong rồi `await Promise.all([fetchRules(), fetchShadowColumns()])` để cập nhật tức thì Tag kiểu cột sang trạng thái khớp (`color="green"`) hoặc hiển thị toast lỗi thật qua `humanizeApiError` (`EV-20261117T103656-434c`).
   - Tooltip cột `Shadow Type` phân biệt rõ trường hợp rule chưa `approved` + `is_active` (cần Duyệt trước khi bấm `Sync Fields`). Build frontend Vite đạt `PASS` (`EV-20261117T103656-434c`).
7. **Self-Healing tại S5 (`shadow_alter_column_jsonb_test.go` `149` dòng, dòng `99-103`):**
   - Phát hiện trong quá trình rà soát S5: dùng `t.Cleanup` để `DROP TABLE` chạy sau `defer connMgr.Registry().Close()` theo thứ tự teardown của `testing.T`, gây đóng DB pool trước khi drop bảng tạm. Đã sửa sang `defer func()` đặt sau `defer connMgr.Registry().Close()` (LIFO) kèm `t.Errorf` kiểm tra lỗi drop, chạy lại trên PostgreSQL 15 đạt `PASS` sạch tuyệt đối (`EV-20261117T103604-321f`).

---

## 2. Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

- **Không gắn closure giả hay mock vào struct production:** Không có bất kỳ trường `*Fn` giả lập nào được thêm vào `SchemaAdapter`, `SchemaDDLHandler`, hay `SyncShadowColumnsHandler`; `cdc-cms-service` tái sử dụng interface chuẩn `ports.RPCClient` có sẵn của dự án (`nats_publisher.go:39-66`).
- **Đối chiếu khớp giao thức NATS Request-Reply thực tế:**
  - `natsPublisher.Request` (`cdc-cms-service/internal/infra/messaging/nats_publisher.go:43-65`) đăng ký `SubscribeSync(replySubject)` và gán `msg.Reply = replySubject`.
  - `BaseHandler.NatsPublish` (`centralized-data-service/internal/handler/base/base_handler.go:67-70`) kiểm tra `if msg.Reply != "" { _ = msg.Respond(data); return }`, gửi thẳng JSON `CommandResult` về đúng `replySubject` mà `SyncShadowColumnsHandler` đang chờ.
- **Kiểm thử tích hợp trên PostgreSQL 15 vật lý (`gpay-postgres-shadow`):** `TestAlterColumn_TextToJSONB_ForwardCore_RealPostgres` tạo bảng tạm trên `shadow_devcoretrans`, nạp object lồng nhau + mảng + chuỗi rỗng `""` + `NULL` qua `CoerceValue` vào cột `TEXT`, chạy `sa.AlterColumn(..., "embedData", "JSONB", "alter_type")` thật trên PostgreSQL 15, truy vấn toán tử JSONB `"embedData"->'promotionInfo'->>'code' = 'SAVE50'`, kiểm tra chuỗi rỗng chuyển thành `IS NULL = true`, kiểm tra cột chứa chuỗi thường `'not_valid_json'` bị PostgreSQL chặn bằng `SQLSTATE 22P02` và giữ nguyên kiểu `text`, rồi dọn bảng tạm sạch sẽ (`EV-20261117T103604-321f`).

---

## 3. Kiểm tra Concurrency, Multi-Worker, Timeout-Heartbeat & Watermark Continuity

1. **Multi-Worker & Schema Cache Consistency (`schema.config.reload`):**
   - `cdc.cmd.create-default-columns` dùng `QueueSubscribe(..., "cdc-worker-group", ...)` (`server_setup.go:293`) nên chỉ 1 worker thực thi DDL tại một thời điểm (tránh 2 worker cùng chạy `ALTER TABLE`).
   - Ngay sau khi `columnsAdded > 0 || columnsAlteredType > 0`, worker thực thi gọi `InvalidateCacheInSchema` cục bộ đồng thời broadcast `NatsConn.Publish("schema.config.reload", ...)` (`schema_ddl_handler.go:343-348`). Toàn bộ các replica worker đang subscribe thường (`natsClient.Conn.Subscribe("schema.config.reload", ...)` tại `server_setup.go:221-225`) đều nhận tín hiệu và đồng loạt xóa Redis `"schema:*"` + gọi `flushBatchBufferSchemaCaches(cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)`. Nhờ đó các batch CDC kế tiếp nhìn thấy ngay kiểu `jsonb` mới mà không tiếp tục ép kiểu theo cache `text` cũ.
2. **Lock Timeout vs NATS RPC Timeout (`10s` < `30s`):**
   - `AlterColumn` đặt `SET LOCAL lock_timeout = '10s'` bên trong transaction (`schema_adapter.go:857`), nhỏ hơn ngưỡng `30s` của `SyncShadowColumnsHandler` (`sync_shadow_columns.go:50`). Nếu bảng Shadow đang bị giữ lock dài bởi batch lớn, PostgreSQL chủ động hủy lệnh `ALTER` sau `10s` với lỗi `lock timeout` rõ ràng và trả về UI trong vòng `< 11s`, không làm treo kết nối HTTP hay rò rỉ goroutine chờ NATS.
3. **Batch Debouncer & Watermark / Kafka Offset Continuity:**
   - Bộ test gói `internal/handler/shadow` (gồm `TestFlushTableWithContext_PreventsAdvanceCommitBeyondPendingRecords`, `TestCalculateSafeCommitOffsets_MiddleFailurePreventsDataLoss`, `TestStatefulWatermarkTracker_ContinuousWindow_AcrossBatches`) đều chạy lại sau thay đổi và đạt `PASS` (`EV-20261117T103636-fd13`), bảo toàn tính liên tục của Kafka offset và watermark khi flush cache schema.
4. **Kiểm định Bảo mật (`/security-agent`):**
   - Mọi tên schema, bảng, cột trong `AlterColumn` đều qua `sqlutil.IsSafeIdent` + `sqlutil.QuoteIdent`, kiểu dữ liệu qua `IsTypeWhitelisted`; lỗi trả từ worker về CMS đều qua `base.SanitizeAdminError` che các mẫu `secret|password|token|api_key|email|phone`. Không có secret hay PII nào trong mã nguồn và tài liệu workspace.

---

## 4. Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty)

- Mọi khẳng định kết quả build và test trong báo cáo này đều gắn trực tiếp 10 biên lai vật lý trong `06_evidence/` có `src_fingerprint` khớp trạng thái mã nguồn cuối cùng (`EV-20261117T103553-620b`, `EV-20261117T103604-321f`, `EV-20261117T103646-1fd5`, `EV-20261117T103656-434c`).
- Không tự ý đụng vào hay xóa dữ liệu các bảng cũ trên DB của người dùng (`L072`, `L097`).
- Minh bạch về hiệu chuẩn Decision Gate: ở chặng S2.5/S3 trước đó, fallback Laya `:8200` cho `trusted_rounds=[]` (`UNVALIDATED`); sang chặng `DG3` cuối phiên (`06_laya_summary_dg3_20261117T103932.json`), vòng `r3` đã vượt qua đầy đủ các câu control (`trusted_rounds=["r3"]`) và xác nhận `ACCEPT` toàn bộ 7 claim kiểm toán.

---

## 5. Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- **Bài học mới đã ghi nhận trong phiên (`L097`):** Khi người dùng phản hồi không muốn giữ logic vá/khôi phục dữ liệu cũ (`_raw_data` backfill & regex `map[...]`), đã dừng ngay lập tức theo Rule #5, đúc kết bài học tổng quát `L097` (`No Legacy Data Salvage Band-Aids & Pure Forward Core Fix`) vào `memory/global/lessons.md`, đồng bộ `build_lessons_index.py`, và thiết kế lại bản giải pháp Pure Forward Core gọn sạch trước khi nhận `APPROVE`.
- **Các bài học cốt lõi đã tuân thủ xuyên suốt:** `L005` (không hardcode tên bảng cụ thể), `L008` & Core Tripwire `G5` (sửa thứ tự `defer` LIFO để drop bảng test trước khi đóng pool DB), `L041` (gọi CLI `evidence_run.py` và `laya_gate.py` bằng tham số đơn sạch), `L072` & `L073` (không chạy lệnh ghi DB trực tiếp ngoài test cô lập), `L074` (chỉ chạy DDL ở luồng admin command có `lock_timeout = '10s'`, tuyệt đối không chạy DDL trong hot-path consumer). Không vi phạm bất kỳ lesson nào ở cuối phiên.

---

## 6. Kết quả QC & nhận xét từ /laya_router (laya_gate DG3)

- **Lệnh kiểm định:** `laya_gate.py run --stage dg3 --url http://127.0.0.1:8200/v1/predict --workspace fix-shadow-sync-fields-alter-type-2026-11-16 --claims claims_dg3.json`
- **File kết quả:** `06_laya_summary_dg3_20261117T103932.json` (đối chiếu đồng bộ cùng bộ biên lai thực thi `EV-20261117T103604-321f`, `EV-20261117T103652-4ce1`, `EV-20261117T103656-434c`)

```text
stage: dg3 | gate: PASS | trusted_rounds: ["r3"]
A-F1   (schema_adapter_coerce.go:31-43)       -> ACCEPT (r3: YES, conf=0.9014)
A-F1B  (dynamic_mapper.go:233-241)            -> ACCEPT (r3: YES, conf=0.8373)
A-F2   (schema_adapter.go:822-862)            -> ACCEPT (r3: YES, conf=0.8517)
A-F3   (schema_ddl_handler.go:275-358)        -> ACCEPT (r3: YES, conf=0.6407)
A-F4   (sync_shadow_columns.go:37-70)         -> ACCEPT (r3: YES, conf=0.8419)
A-F5   (MappingFieldsPage.tsx:166-190)        -> ACCEPT (r3: YES, conf=0.8501)
A-TEST (shadow_alter_column_jsonb_test.go:71) -> ACCEPT (r3: YES, conf=0.6478)
```

---

## 7. Giới hạn & Phần chưa kiểm trên tiến trình bên ngoài (Limitations)

- Hai tiến trình dịch vụ đang chạy sẵn ngoài phiên làm việc của anh trên máy local (`cdc-cms-service` tại cổng `:8083`, `centralized-data-service` tại cổng `:8082`) được khởi chạy từ trước bằng `go run` nên vẫn đang giữ binary cũ trên RAM; em không tự ý `kill` tiến trình của anh.
- Để áp dụng bộ code mới lên môi trường dev đang mở:
  1. Khởi động lại `centralized-data-service` (`:8082`) và `cdc-cms-service` (`:8083`), đồng thời reload dev server / bản build của `cdc-cms-web`.
  2. Với các bảng Shadow test cũ từng bị dính chuỗi `"map[...]"` trước đây (như `shadow_traitestpbs.payment_bills`), đúng như phương án anh đã chốt, anh có thể xoá tạo lại bảng đó; từ nay mọi dữ liệu object/array đi vào cột `TEXT` đều được serialize JSON chuẩn và khi bấm **"Sync Fields to Shadow"** sẽ `ALTER COLUMN TYPE JSONB` đồng bộ ngay lập tức.
