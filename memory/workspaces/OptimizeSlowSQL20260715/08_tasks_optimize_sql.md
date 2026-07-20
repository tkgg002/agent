# Danh sách Task chi tiết: Tối ưu hóa SQL chậm

## Phase 1: Phân tích & Lập phương án
- [x] Đọc logs và xác định các câu lệnh SQL chậm
- [/] Phân tích cấu trúc bảng, các index hiện tại và lý do chậm
- [ ] Lập kế hoạch tối ưu hóa chi tiết (Implementation Plan) và đề xuất giải pháp tốt nhất
- [ ] Xin phê duyệt từ User

## Phase 2: Thực thi tối ưu hóa
- [ ] Ủy quyền cho Muscle chỉnh sửa mã nguồn trong `recon_read_repo_gorm.go`
- [ ] Tạo các index bổ sung nếu cần thiết qua SQL migration hoặc trực tiếp

## Phase 3: Kiểm thử & Xác thực
- [ ] Chạy kiểm thử tích hợp (Integration Test) để đảm bảo tính đúng đắn của dữ liệu
- [ ] Đo lường hiệu năng sau tối ưu, so sánh với log trước đó
- [ ] Chạy linter quy trình và báo cáo kết quả
