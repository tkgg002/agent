# Progress Log: Khắc phục lỗi 503 Worker timeout (nats: timeout) API scan-array/trans_his

## Root Cause Analysis
- **Lỗi hiển thị**: `POST https://cdc-cms-api.goopay.vn/api/introspection/scan-array/trans_his` -> `503 Service Unavailable` `{"error":"Worker timeout or request error: nats: timeout"}`.
- **Vị trí phát sinh lỗi**:
  - `cdc-cms-service/internal/api/system/introspection_handler.go:341`: `h.rpc.Request(c.UserContext(), "cdc.cmd.scan-array", replySubj, payload, 15*time.Second)` chờ 15 giây.
  - Phía nhận là `centralized-data-service/internal/handler/scan/scan_handler.go:194` (`HandleScanArrayFields`) xử lý qua `scan_service.go:197` (`ScanArrayFields`).
  - Trong `introspection_handler.go:338`, payload gửi sang worker có `"mode": "full"`.
  - Trong `scan_service.go:327-338`: khi `mode == "full"`, câu query PostgreSQL chạy:
    `SELECT (_raw_data #> ?::text[])::text AS node_data FROM <schema>.trans_his WHERE _raw_data #> ?::text[] IS NOT NULL AND jsonb_typeof(...) != 'null'`
    không có `LIMIT`, không có index chuyên biệt, buộc Postgres phải quét toàn bộ bảng (full table scan) hàng trăm ngàn đến hàng triệu dòng của `trans_his`, rồi Go fetch từng dòng và unmarshal JSON. Quá trình này vượt quá 15 giây khiến NATS RPC client timeout.
  - Ngoài ra: nếu tiến trình worker (`centralized-data-service`) trên môi trường đó bị tắt/crash hoặc mất kết nối NATS, NATS cũng trả về `nats: timeout` sau 15s.

## Audit Log
- [2026-10-02T15:10:00+07:00] [Brain:Gemini-3.8-Flash-High] Tiếp nhận lỗi, đọc hiến pháp GEMINI.md và lessons.md, phân tích root cause từ mã nguồn cdc-cms-service và centralized-data-service.
- [2026-10-02T15:10:00+07:00] [Brain:Gemini-3.8-Flash-High] Khởi tạo workspace fix-scan-array-nats-timeout-trans-his với đầy đủ tài liệu: requirements, progress, tasks, solution, analysis, implementation plan.
- [2026-10-02T15:12:00+07:00] [Brain:Gemini-3.8-Flash-High] User phản hồi về hiện tượng 'Missing Span 15.00s' trong SigNoz. Phân tích cơ chế OTel W3C trace-context propagation: CMS gửi RPC sang cdc-worker qua NATS nhưng cdc-worker không trả span/kết quả về SigNoz, giải thích rõ cdc-cms-service không tự scan mà đã delegate sang worker nhưng worker không phản hồi.
