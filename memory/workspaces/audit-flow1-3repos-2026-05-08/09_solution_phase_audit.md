# Solution Notes

## Findings summary

1. `cdc-cms-service` không phải blocker chính của lần audit này.
   - `go test ./...` pass.
   - Router hiện có mount introspection routes và provisioning routes cho Flow 1.

2. `cdc-cms-web` là blocker trực tiếp của Flow 1.
   - Build fail ở `src/pages/flow1/Flow1Layout.tsx` vì dùng `Steps.Step` / children pattern không còn hợp lệ với Ant Design v6.
   - `Step1_Connection.tsx` có lỗi type và dùng prop `small` không tồn tại trên `Typography.Text`.
   - `Step3_Shadow.tsx` có unused imports/destructuring làm fail strict TS build.

3. Flow 1 FE↔CMS contract đang lệch ở nhiều chỗ.
   - `Step1_Connection` gọi `GET /api/v1/introspection/mongo/databases`.
   - Router/code hiện có route này, nhưng log runtime ghi nhận 404 trong các lần dùng trước; nhiều khả năng binary đang chạy từng bị stale hoặc chưa restart sau thay đổi.
   - `Step1_Connection` kỳ vọng response `POST /api/v1/sources` có `default_database`, nhưng `SourcesHandler.Create` trả model `Source` với field `database_include_list`.
   - `Step3_Shadow` gửi payload:
     - `source_connection_id`
     - `source_database`
     - `source_object_name`
     - `source_object_type`
   - Trong khi `RegistryHandler.Register` parse vào `model.TableRegistry` với schema khác (`source_db`, `source_table`, `target_table`, `primary_key_type`, ...). Đây là mismatch contract nghiêm trọng.

4. Logic step 3 của FE chưa khớp với flow manual đã được document trước đó.
   - `Step3_Shadow` gọi `POST /api/v1/cms/sources/:id/provisioning/mode` với `mode: auto`.
   - Tài liệu Flow 1 cũ nhấn mạnh đường manual: set mode rồi advance rõ ràng.
   - Theo orchestrator hiện tại, `SetMode(auto)` chỉ auto-fanout khi thực sự đổi từ `manual -> auto`; nếu row đã là `auto` hoặc state không advanceable thì không bảo đảm chạy tiếp như FE đang ngầm kỳ vọng.

5. `centralized-data-service` có 3 tầng vấn đề khác nhau.
   - Build binary pass.
   - Test fail vì:
     - `scratch/` có nhiều file `main` làm vỡ `go test ./...`.
     - `internal/handler` có test regression thật ở DLQ raw wrapper.
     - `internal/service` có panic thật do nil logger trong `SchemaValidator`.
     - thêm một phần fail do sandbox chặn kết nối DB localhost.
   - Runtime logs còn lỗi dữ liệu thật ở transmuter numeric cast (`\"8999/100\"`, `\"31/2\"`, ...), không trực tiếp là blocker Flow 1 source→shadow nhưng là blocker chất lượng cho flow downstream shadow→master.
