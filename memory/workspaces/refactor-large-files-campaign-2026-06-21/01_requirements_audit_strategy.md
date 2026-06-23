# Yêu cầu: Điều chỉnh chiến lược Refactor tránh băm nhỏ file (Flow-based Consolidation)

## 1. Bối cảnh & Lý do điều chỉnh
- Trong quá trình triển khai Phase 1 (Recon Module), model đã thực hiện phân rã các file lớn thành nhiều file nhỏ phụ thuộc như: `_models.go`, `_helpers.go`, `_actions.go`, `_recovery.go`, `_seed.go`, v.v.
- Việc phân tách này dẫn đến việc "băm nhỏ" mã nguồn của một luồng xử lý duy nhất (Single Flow) thành nhiều file nhỏ rời rạc, làm giảm khả năng đọc hiểu, theo dõi luồng thực thi và tăng chi phí bảo trì hệ thống.
- Ý kiến chỉ đạo của User: **"chúng ta đang phân tách các file thực hiện nhiều nhiệm vụ (không phải băm nhỏ ra) các nhóm func thực hiện cùng 1 flow thì vẫn nên để trong 1 file để dễ thực thi."**

## 2. Yêu cầu chi tiết
- **Nguyên tắc phân tách (SRP đúng nghĩa)**: Chỉ tách file khi file gốc thực hiện **nhiều nhiệm vụ khác nhau** (Multiple Responsibilities/Tasks). Ví dụ: vừa đăng ký metadata, vừa cung cấp API query, vừa tự động phát hiện schema.
- **Giữ nguyên luồng xử lý (Single Flow)**: Các hàm thuộc cùng một luồng thực thi nghiệp vụ (ví dụ: từ khi fetch shadow, mapping rules, cho đến khi upsert master) phải được giữ trong cùng một file chính (ví dụ: `transmuter.go`).
- **Phân loại cấu trúc file sau refactor**:
  1. **Core Flow File (Giữ lại)**: Luôn chứa toàn bộ luồng thực thi chính (orchestration, sequence of execution). File này có thể dài hơn 500 dòng nhưng tập trung, mạch lạc, dễ trace.
  2. **Task-Specific Files (Tách nếu có)**: Chỉ tách các chức năng/nhiệm vụ phụ trợ độc lập hoàn toàn khỏi luồng chính (như logging/reporting trạng thái chạy, hoặc các helper tính toán/coercion độc lập thuần túy).
- **Phạm vi kiểm toán & Điều chỉnh**:
  - Rà soát lại Phase 1 (Recon Module) và đề xuất kế hoạch gộp lại nếu cần.
  - Áp dụng triệt để nguyên tắc mới cho Phase 2 (`transmuter.go`, `schema_adapter.go`), Phase 3, và Phase 4.
