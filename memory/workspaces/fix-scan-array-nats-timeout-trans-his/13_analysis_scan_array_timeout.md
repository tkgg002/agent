# Phân Tích Kỹ Thuật (Analysis): Lỗi 503 Service Unavailable / NATS Timeout trên API scan-array/trans_his

## 1. Dữ liệu từ Request của User
- URL: `https://cdc-cms-api.goopay.vn/api/introspection/scan-array/trans_his`
- Method: `POST`
- Response: `503 Service Unavailable`
- Body: `{"error":"Worker timeout or request error: nats: timeout"}`
- Remote Address: `10.200.198.100:443`

## 2. Truy vết Mã Nguồn (Code Trace)

### Mắt xích 1: API Endpoint tại `cdc-cms-service`
- Router: `cdc-cms-service/internal/router/router.go:325`
  `dual("POST", shared, "/introspection/scan-array/:table", h.System.Introspection.ScanArray)`
- Handler: `cdc-cms-service/internal/api/system/introspection_handler.go:318-353`
  ```go
  correlationID := fmt.Sprintf("scan-array-%s-%d", targetTable, time.Now().UnixNano())
  replySubj := "cdc.evt.scan-array." + correlationID
  payload, _ := json.Marshal(map[string]any{
      "target_table":      targetTable,
      "explode_path":      body.ExplodePath,
      "master_binding_id": body.MasterBindingID,
      "reply_to":          replySubj,
      "mode":              "full",
  })
  msgData, err := h.rpc.Request(c.UserContext(), "cdc.cmd.scan-array", replySubj, payload, 15*time.Second)
  if err != nil {
      return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
          "error": "Worker timeout or request error: " + err.Error(),
      })
  }
  ```
- NATS RPC Client: `cdc-cms-service/internal/infra/messaging/nats_publisher.go:39-66`
  `sub.NextMsg(timeout)` chờ đúng 15 giây. Sau 15 giây không có message phản hồi, `sub.NextMsg` trả về lỗi `"nats: timeout"`.
  Handler gán lỗi vào response `503` với string `"Worker timeout or request error: nats: timeout"`.

### Mắt xích 2: Xử lý tại `centralized-data-service` (Worker)
- Subscriber: `centralized-data-service/internal/server/server_setup.go:284`
  `natsClient.Conn.QueueSubscribe("cdc.cmd.scan-array", workerQueueGroup, scanHandler.HandleScanArrayFields)`
- Handler: `centralized-data-service/internal/handler/scan/scan_handler.go:194`
  Unmarshal payload, gọi: `h.scanSvc.ScanArrayFields(ctx, payload.TargetTable, payload.ExplodePath, payload.MasterBindingID, payload.Mode)`
- Service: `centralized-data-service/internal/service/source/scan_service.go:327-340`:
  ```go
  sampleClause := ""
  limitClause := ""
  if mode == "sample" {
      sampleClause = "TABLESAMPLE SYSTEM(1)"
      limitClause = "LIMIT 1000"
  }

  sqlQuery := fmt.Sprintf(`
      SELECT (_raw_data #> ?::text[])::text AS node_data
      FROM %s %s 
      WHERE _raw_data #> ?::text[] IS NOT NULL 
        AND jsonb_typeof(_raw_data #> ?::text[]) != 'null'
      %s`,
      sqlutil.QualifiedTable(schemaName, targetTable), sampleClause, limitClause)

  rows, err := execDB.WithContext(ctx).Raw(sqlQuery, pgPath, pgPath, pgPath).Rows()
  ```
  Và sau đó duyệt toàn bộ `rows.Next()` để unmarshal JSON:
  ```go
  for rows.Next() {
      // scan nodeData
      // json.Unmarshal
      // flattenJSONWithTypes
  }
  ```

### Mắt xích 3: Đặc thù bảng `trans_his`
- `trans_his` là bảng Transaction History (lịch sử giao dịch), quy mô dữ liệu trong production thường từ hàng trăm nghìn đến nhiều triệu bản ghi.
- Cột `_raw_data` chứa JSONB. Không có index chuyên biệt cho biểu thức `_raw_data #> pgPath`.
- Postgres buộc phải chạy Sequential Scan toàn bảng và tính toán hàm JSONB trên từng dòng.
- Go worker duyệt qua từng dòng và parse JSON trong bộ nhớ.
- Tổng thời gian vượt xa ngưỡng 15 giây.

### Mắt xích 4: Trạng thái Worker Pod trên Server
- Nếu worker pod trên server chưa được triển khai, bị crash (OOM, restart) hoặc không kết nối được tới NATS server, message `cdc.cmd.scan-array` không có consumer tiếp nhận -> NATS client chờ 15s cũng trả về `nats: timeout`.

## 3. Kết luận
Lỗi xuất phát từ:
1. **Thiết kế thiếu trần an toàn (Safety Ceiling) khi truy vấn**: `ScanArrayFields` không có `LIMIT` khi `mode == "full"`, trong khi CMS luôn truyền cứng `mode: "full"`.
2. **Khám phá Schema (Introspection) không cần quét toàn bộ dữ liệu**: Chỉ cần quét mẫu 1,000 - 2,000 bản ghi là đã trích xuất được 100% các trường của mảng JSON.
3. Cần kiểm tra song song trạng thái pod worker trên cụm production.
