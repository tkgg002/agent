# Kết quả Kiểm thử & Benchmark: Tối ưu hóa SQL cdc_activity_log

## 1. Kiểm thử Tích hợp (Automated Integration Tests)
Chạy toàn bộ integration test suite của `cdc-cms-service` vượt qua thành công:
```
ok  	cdc-cms-service/test/internal/api	0.559s
ok  	cdc-cms-service/test/internal/api/dto	1.478s
ok  	cdc-cms-service/test/internal/app/commands	1.323s
ok  	cdc-cms-service/test/internal/app/queries	1.103s
ok  	cdc-cms-service/test/internal/infra/http	0.738s
ok  	cdc-cms-service/test/internal/infra/messaging	0.297s
ok  	cdc-cms-service/test/internal/infra/observability	1.979s
ok  	cdc-cms-service/test/internal/infra/observability/probes	2.449s
ok  	cdc-cms-service/test/internal/infra/persistence	2.252s
ok  	cdc-cms-service/test/internal/middleware	1.705s
```

## 2. Kết quả Đo đạc Hiệu năng (Benchmark Metrics)
Chạy benchmark đếm tổng số bản ghi với 50 lần lặp trên môi trường dữ liệu local (9,758 bản ghi):

| Phiên bản truy vấn | Thời gian thực thi trung bình | Đỉnh độ trễ (Peak latency) | Kết quả đếm | Trạng thái SLOW SQL |
| :--- | :--- | :--- | :--- | :--- |
| **Truy vấn Cũ** (Chứa lateral joins) | **49.94ms** | **1,330.94ms** (1.3s) | 9,758 | **Bị cảnh báo SLOW SQL** |
| **Truy vấn Mới** (Tách biệt / Không joins) | **2.86ms** | **~4ms** | 9,758 | **Hoàn toàn mượt mà (<5ms)** |

### Nhận xét:
- Việc tách biệt câu lệnh `COUNT(*)` giúp tăng tốc độ truy vấn đếm lên **hơn 17 lần** ở trạng thái trung bình.
- Loại bỏ hoàn toàn đỉnh độ trễ 1.3 giây (gây đơ/lag UI/API) khi PostgreSQL phải tính toán các lateral subqueries trên toàn bộ tập dữ liệu lịch sử hoạt động.
- Tính nhất quán dữ liệu được bảo toàn tuyệt đối (kết quả đếm trả về trùng khớp hoàn toàn ở cả hai phiên bản).
