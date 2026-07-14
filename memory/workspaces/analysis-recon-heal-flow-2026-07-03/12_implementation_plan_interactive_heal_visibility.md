# Kế hoạch Triển khai: Sửa lỗi hiển thị modal Chữa lành đối soát (Interactive Heal Visibility)

## 1. Nguyên nhân lỗi
Khi mở modal "Chữa lành đối soát", API được gọi là:
`GET /api/reconciliation/report/shadow_testexp.export_jobs/unhealed?shadow_schema=shadow_testexp`

Tại backend (`recon_read_repo_gorm.go`), hàm xử lý `ListUnhealedReports` nhận `table="shadow_testexp.export_jobs"` và `shadowSchema="shadow_testexp"`. Logic truy vấn hiện tại:
```go
q := r.db.WithContext(ctx).
	Table("cdc_system.cdc_reconciliation_report").
	Where("(shadow_table = ? OR master_table = ?)", table, table)
```
Do cột `shadow_table` trong bảng `cdc_reconciliation_report` chỉ lưu tên bảng thô (`export_jobs`), điều kiện `shadow_table = 'shadow_testexp.export_jobs'` trả về kết quả rỗng.
Lỗi tương tự xảy ra ở hàm `GetTableHistory` phục vụ API lịch sử bảng.

## 2. Giải pháp kỹ thuật

### Tệp tin thay đổi: [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)

#### Hàm `ListUnhealedReports`:
Bổ sung đoạn logic chuẩn hóa tên bảng ở đầu hàm:
```go
if strings.Contains(table, ".") {
	parts := strings.Split(table, ".")
	if len(parts) > 1 {
		if shadowSchema == "" {
			shadowSchema = parts[0]
		}
		table = parts[len(parts)-1]
	}
}
```

#### Hàm `GetTableHistory`:
Bổ sung chuẩn hóa cho cả `table` và `masterTable`:
```go
if strings.Contains(table, ".") {
	parts := strings.Split(table, ".")
	if len(parts) > 1 {
		if shadowSchema == "" {
			shadowSchema = parts[0]
		}
		table = parts[len(parts)-1]
	}
}
if strings.Contains(masterTable, ".") {
	parts := strings.Split(masterTable, ".")
	masterTable = parts[len(parts)-1]
}
```

## 3. Kế hoạch xác minh

### Kiểm thử tự động (Build):
- Di chuyển vào thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service` và thực hiện `go build ./...` để đảm bảo code biên dịch thành công.

### Kiểm thử thủ công (cURL):
- Gọi endpoint:
  `curl -s -H "Authorization: Bearer dev-token" "http://localhost:8083/api/reconciliation/report/shadow_testexp.export_jobs/unhealed?shadow_schema=shadow_testexp"`
- Xác nhận dữ liệu trả về chứa danh sách các reports chưa được chữa lành (có `healed_at IS NULL` và các count lệch > 0).
