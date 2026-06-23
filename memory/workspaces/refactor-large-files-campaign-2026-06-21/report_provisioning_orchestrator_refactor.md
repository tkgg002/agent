# Báo cáo dòng thay đổi: provisioning_orchestrator.go Refactor

Báo cáo chi tiết số lượng dòng code thay đổi trước và sau khi thực hiện phân rã file `provisioning_orchestrator.go` thuộc package `recon`.

## So sánh kích thước file

| File Name | Trước Refactor (LoC) | Sau Refactor (LoC) | Tỷ lệ giảm |
| :--- | :---: | :---: | :---: |
| `provisioning_orchestrator.go` | 873 | 34 | -96.1% |

## Danh sách các file helper mới được tạo ra

| File Name | Số dòng (LoC) | Vai trò |
| :--- | :---: | :--- |
| `provisioning_orchestrator_models.go` | 56 | Chứa hằng số cấu hình TTL, max entry log và `stepLogEntry` struct. |
| `provisioning_orchestrator_helpers.go` | 159 | Chứa trace injection, helper queries đọc state/mode, CAS UPDATE, và NATS publish. |
| `provisioning_orchestrator_seed.go` | 146 | Chứa logic seed master_binding và các truy vấn phụ phục vụ Advance. |
| `provisioning_orchestrator_actions.go` | 284 | Chứa API cốt lõi điều phối luồng: Advance, SetMode, Pause, Resume, Retry, Archive. |
| `provisioning_orchestrator_recovery.go` | 190 | Xử lý sự kiện hoàn thành bước (HandleStepCompleted) và vòng lặp khôi phục timeout. |
| **Tổng số dòng mới** | **869** | (Giảm 4 dòng so với file gốc do loại bỏ imports thừa). |

## Xác nhận tính nhất quán và biên dịch
- **Biên dịch**: `go build ./...` PASS.
- **Unit Tests**:
  - `go test -v ./internal/service/recon/...` PASS.
  - `go test ./...` PASS.
