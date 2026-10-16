# Danh mục Tasks: NATS Monitor UI

- [ ] Task 1: Tạo `internal/api/system/nats_monitor_handler.go` trong `cdc-cms-service`.
- [ ] Task 2: Cập nhật `internal/router/router.go` và `internal/server/server.go` để đăng ký handler và route `/api/system/nats/overview`.
- [ ] Task 3: Build và verify backend Go (`go build ./cmd/server`).
- [ ] Task 4: Tạo component `src/pages/NatsMonitor.tsx` trong `cdc-cms-web`.
- [ ] Task 5: Cập nhật `src/App.tsx` trong `cdc-cms-web` để thêm menu "NATS Monitor" và Route tương ứng.
- [ ] Task 6: Kiểm tra frontend build bằng TypeScript check (`npm run build` hoặc `npx tsc --noEmit`).
- [ ] Task 7: Khởi động lại service và nghiệm thu trên trình duyệt qua curl/browser.
