# Kế hoạch & Kết quả Kiểm thử: NATS Monitor UI

## 1. Test Cases

| ID | Kịch bản kiểm thử | Kỳ vọng | Kết quả |
| :--- | :--- | :--- | :--- |
| **TC1** | Unit test `TestNATSMonitorHandler_GetOverview_Success` | Handler gọi song song 3 endpoint mock (`/varz`, `/jsz`, `/connz`), trả về HTTP 200, status `ok` | **PASS** |
| **TC2** | Unit test `TestNATSMonitorHandler_GetOverview_Down` | Khi cả 3 endpoint không phản hồi, handler trả về HTTP 503, status `down` kèm error details | **PASS** |
| **TC3** | Go backend compilation | Toàn bộ package `cdc-cms-service` biên dịch sạch lỗi (`go build ./cmd/server`) | **PASS** |
| **TC4** | Frontend TypeScript check | Toàn bộ codebase `cdc-cms-web` build sạch lỗi (`tsc -b && vite build`) | **PASS** |
| **TC5** | UI Route & Navigation | Menu `NATS Monitor` xuất hiện trong nhóm Operate tại sidebar, điều hướng mượt mà tới `/nats-monitor` | **PASS** |
| **TC6** | Dữ liệu thời gian thực | Bảng Streams hiển thị đủ 4 stream (`CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `CDC_EVENTS`, `SCHEMA_CONFIG`), Drawer Consumers hiển thị chi tiết Ack Pending | **PASS** |

## 2. Bằng chứng kiểm thử vật lý (Physical Evidence)
- Backend test:
  ```bash
  go test -v ./internal/api/system -run TestNATSMonitorHandler
  # Output: PASS (TestNATSMonitorHandler_GetOverview_Success, TestNATSMonitorHandler_GetOverview_Down)
  ```
- Frontend build:
  ```bash
  npm run build
  # Output: tsc -b && vite build -> dist/assets/NatsMonitor-*.js built successfully.
  ```
