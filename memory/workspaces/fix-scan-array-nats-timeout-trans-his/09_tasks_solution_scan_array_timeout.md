# Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution): Khắc phục lỗi 503 Worker timeout (nats: timeout) scan-array/trans_his

## 1. Bản chất vấn đề (Root Cause Architecture)
Endpoint `POST /api/introspection/scan-array/:table`:
1. `cdc-cms-service` (`internal/api/system/introspection_handler.go:341`) gửi command NATS `cdc.cmd.scan-array` với `timeout = 15s`, trong đó hardcode `"mode": "full"`.
2. `centralized-data-service` (`internal/service/source/scan_service.go:327-340`) nhận command, thấy `mode == "full"` nên không đặt `LIMIT` và không dùng `TABLESAMPLE`.
3. Câu query Postgres thực thi trên bảng `trans_his` (Transaction History - thường có quy mô hàng triệu bản ghi):
   ```sql
   SELECT (_raw_data #> ?::text[])::text AS node_data
   FROM <schemaName>.trans_his
   WHERE _raw_data #> ?::text[] IS NOT NULL 
     AND jsonb_typeof(_raw_data #> ?::text[]) != 'null'
   ```
4. Vì không có index biểu thức trên `_raw_data #> ...` và không có `LIMIT`, PostgreSQL buộc phải quét toàn bộ bảng (Seq Scan) hàng triệu dòng, và Go worker phải fetch từng dòng qua socket và deserialize JSON trong vòng lặp `for rows.Next()`.
5. Quá trình này ngốn rất nhiều CPU/I/O và chắc chắn vượt quá 15 giây. Khi quá 15 giây, NATS client ở phía CMS ngắt kết nối với lỗi `nats: timeout` và trả về HTTP 503.
*(Lưu ý phụ: Trường hợp worker service chưa khởi động hoặc rớt kết nối NATS cũng sinh ra đúng `nats: timeout` sau 15s)*.

---

## 2. Giải pháp kỹ thuật tối ưu duy nhất (The Single Best Approach)
Khắc phục cả 2 đầu: **Đầu caller (CMS)** và **Đầu executor (Worker)** theo chuẩn phòng vệ sâu (Defense in Depth) & Kỷ luật hiệu năng (Simplicity & Core Systems First):

### A. Tầng Centralized Data Service (Worker) - Đặt trần an toàn & Query Timeout
Trong `centralized-data-service/internal/service/source/scan_service.go`:
- Mục đích của Introspection/Scan Array chỉ là **khám phá cấu trúc trường (schema discovery)** của phần tử bên trong mảng để tạo các mapping rules mẫu. Việc duyệt toàn bộ hàng triệu bản ghi là hoàn toàn không cần thiết và phản tác dụng. Tương tự như `ScanRawData` (vốn đã dùng `LIMIT 1000` cố định ở line 112), `ScanArrayFields` cần có trần bảo vệ:
  - Khi `mode == "sample"`: giữ nguyên `TABLESAMPLE SYSTEM(1)` và `LIMIT 1000`.
  - Khi `mode == "full"`: áp dụng trần an toàn `LIMIT 2000` (hoặc `LIMIT 1000`). Quét 1,000 - 2,000 bản ghi là thừa đủ bao phủ mọi biến thể cấu trúc JSON của một bảng.
  - Thiết lập timeout cho context DB truy vấn tối đa 10s: `queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second); defer cancel()`. Nếu query DB vượt quá 10s, trả lỗi rõ ràng về cho NATS thay vì để NATS connection bị timeout lơ lửng.

```go
// Demo Code trong scan_service.go
sampleClause := ""
limitClause := "LIMIT 2000" // Trần bảo vệ an toàn cho scan discovery
if mode == "sample" {
    sampleClause = "TABLESAMPLE SYSTEM(1)"
    limitClause = "LIMIT 1000"
}

queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

sqlQuery := fmt.Sprintf(`
    SELECT (_raw_data #> ?::text[])::text AS node_data
    FROM %s %s 
    WHERE _raw_data #> ?::text[] IS NOT NULL 
      AND jsonb_typeof(_raw_data #> ?::text[]) != 'null'
    %s`,
    sqlutil.QualifiedTable(schemaName, targetTable), sampleClause, limitClause)

rows, err := execDB.WithContext(queryCtx).Raw(sqlQuery, pgPath, pgPath, pgPath).Rows()
```

### B. Tầng CDC CMS Service - Chuyển default mode sang "sample" & tăng timeout RPC hợp lý
Trong `cdc-cms-service/internal/api/system/introspection_handler.go`:
- Tại `ScanArray` (line 338) và `ScanArrayAll` (line 416):
  - Chuyển `"mode": "sample"` thay vì hardcode `"full"`, hoặc cho phép request body truyền mode lên (mặc định `sample`).
  - Đảm bảo thời gian timeout RPC là `15*time.Second` hoặc `20*time.Second` đủ dư để nhận kết quả từ worker (khi query worker đã giới hạn < 2s).

```go
// Demo Code trong introspection_handler.go
mode := "sample"
payload, _ := json.Marshal(map[string]any{
    "target_table":      targetTable,
    "explode_path":      body.ExplodePath,
    "master_binding_id": body.MasterBindingID,
    "reply_to":          replySubj,
    "mode":              mode,
})
```

### C. Kiểm tra vận hành hạ tầng
- Chạy lệnh kiểm tra container/pod `cdc-worker` hoặc `centralized-data-service` trên cluster production/staging để xác nhận worker đang RUNNING và kết nối ổn định với NATS broker.
