# 08 Tasks — Add "Data Type Current" Column to Mapping Fields Page

## Phase 1: Backend Implementation (cdc-cms-service)
- [x] **Task 1.1:** Mở rộng port `ShadowSchemaReader` interface thêm method `GetColumnsWithTypes(ctx, schema, table) (map[string]string, error)` trong `internal/app/ports/repository.go`.
- [x] **Task 1.2:** Implement `GetColumnsWithTypes` trong `internal/infra/persistence/shadow/shadow_schema_reader_gorm.go` query `information_schema.columns` và chuẩn hóa tên kiểu dữ liệu qua `normalizePostgresType()`.
- [x] **Task 1.3:** Thêm HTTP handler `ShadowColumnsWithTypes` trong `internal/api/system/introspection_handler.go`.
- [x] **Task 1.4:** Đăng ký dual route `GET /introspection/shadow-columns-with-types/:table` trong `internal/router/router.go`.

## Phase 2: Frontend Implementation (cdc-cms-web)
- [x] **Task 2.1:** Thêm state `shadowColumnTypes: Record<string, string>` trong `MappingFieldsPage.tsx`.
- [x] **Task 2.2:** Cập nhật `fetchShadowColumns` gọi endpoint `shadow-columns-with-types` và populate đồng thời cả `shadowColumns` (Set) lẫn `shadowColumnTypes` (Map).
- [x] **Task 2.3:** Thêm column "Data Type Current" vào `Table` columns sau "Data Type Target" với drift detection logic.

## Phase 3: Adversarial Audit & Quality Control
- [x] **Task 3.1:** Adversarial self-review phát hiện bug ANSI SQL naming (`character varying` vs `VARCHAR`).
- [x] **Task 3.2:** Khắc phục triệt để normalization và map key casing.
- [x] **Task 3.3:** Chạy QC build verify backend (`go build`, `go test`) & frontend (`tsc`).
- [x] **Task 3.4:** Ghi nhận bài học vào `lessons.md` và kiểm tra chỉ số `governance_metrics.sh`.
