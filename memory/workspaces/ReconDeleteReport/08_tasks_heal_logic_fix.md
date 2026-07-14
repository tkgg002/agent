# Danh sách Task: Sửa lỗi hiển thị & thực thi chữa lành

- [x] Task 1: Cập nhật backend `centralized-data-service` tại `recon_execute_heal_handler.go` để triển khai giải pháp NATS Request-Reply đồng bộ hóa transmute, cập nhật Healed counts bằng số lượng inserted + updated thực tế.
- [x] Task 2: Cập nhật frontend `cdc-cms-web` tại `ExecuteHealModal.tsx` để lọc healedReports chỉ chứa status 'healed' hoặc healed_at != null.
- [x] Task 3: Biên dịch backend thành công (`go build ./...`).
- [x] Task 4: Kiểm tra tĩnh frontend thành công (`npx tsc --noEmit`).
- [x] Task 5: Cập nhật `transmuter.go` để trả về lỗi rõ ràng khi không tìm thấy approved mapping rules cho master binding.
