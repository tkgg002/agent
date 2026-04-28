# Requirements — Phase 12 Reconciliation Scope V2

## Mục tiêu

- Làm `DataIntegrity` và reconciliation APIs bớt phụ thuộc vào `:table` string legacy.
- Giữ đúng mô hình 2 luồng:
  - auto-flow là luồng chính chạy Debezium
  - cms-fe operator-flow dùng cho monitoring / backup / retry / reconcile
- Enrich contract reconciliation để operator nhìn và gửi đúng `source/shadow` scope.

## Yêu cầu chức năng

1. `GET /api/reconciliation/report` phải trả thêm source/shadow context nếu resolve được từ metadata V2.
2. `GET /api/failed-sync-logs` phải trả thêm source/shadow context nếu resolve được.
3. `POST /api/reconciliation/check` phải hỗ trợ gửi scope theo:
   - `source_database`
   - `source_table`
   - `shadow_schema`
   - `shadow_table`
   và chỉ fallback về `target_table` khi cần compatibility.
4. `POST /api/reconciliation/heal` phải hỗ trợ scope tương tự.
5. FE `DataIntegrity` phải gửi và hiển thị source/shadow scope thật, không chỉ render `target_table`.

## Yêu cầu phi chức năng

1. Không phá compatibility với route path legacy hiện có.
2. Nếu API bị thay đổi contract thì phải update swagger/comment trong cùng phase.
3. Phải verify bằng `go test ./...` cho `cdc-cms-service` và `npm run build` cho `cdc-cms-web`.
