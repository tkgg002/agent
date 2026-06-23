# Architectural Decisions - Refactor Large Files Campaign

Tài liệu này ghi nhận các quyết định kiến trúc quan trọng được đưa ra trong chiến dịch refactor dọn dẹp các file lớn (> 500 LoC) của centralized-data-service.

## ADR 1: Phân tách recon_heal.go thành các helper chuyên biệt

### Bối cảnh
File `recon_heal.go` ban đầu có kích thước 900 dòng, chứa các logic quản lý buffer audit log, logic heal missing/orphaned records, logic Debezium signal trigger, và các helpers xử lý MongoDB document. Để giảm độ phức tạp, tăng tính đóng gói, ta cần phân tách file này thành các file nhỏ hơn dựa trên Single Responsibility Principle (SRP).

### Quyết định
Phân rã file này thành 6 file nhỏ chuyên biệt trong package `recon`:
1. `recon_heal.go`: Chỉ giữ lại cấu trúc lõi `ReconHealer` và constructor chính.
2. `recon_heal_models.go`: Chứa structs cấu hình, `HealResult`.
3. `recon_heal_audit.go`: Chứa struct và logic của `healAuditBatcher` (ghi nhận log batched).
4. `recon_heal_action.go`: Thực thi các core APIs heal và windowing logic.
5. `recon_heal_utils.go`: Hàm helper hỗ trợ băm, trích xuất dữ liệu document.
6. `recon_heal_legacy.go`: Chứa shims test helpers cho các test file bên ngoài.

### Hệ quả
- **Tích cực**: Kích thước file chính giảm 93.7% (từ 900 dòng xuống còn 56 dòng). Tách bạch rõ ràng logic nghiệp vụ của healer và logic ghi nhận nhật ký của audit batcher.
- **Tiêu cực**: Tăng số lượng file vật lý trong package `recon`.

## ADR 2: Quy định "Flow-based Consolidation" (Hợp nhất theo luồng chạy)

### Bối cảnh
Sau khi hoàn thành Giai đoạn 1, việc phân tách các tệp tin nghiệp vụ lớn thành quá nhiều file phụ trợ (`_helpers.go`, `_actions.go`, `_models.go`, v.v.) đã gây ra tình trạng băm nhỏ code của cùng một luồng nghiệp vụ duy nhất (Single Flow), làm tăng độ phức tạp khi bảo trì và khó theo dõi luồng thực thi.

### Quyết định
1. Chuyển đổi sang nguyên tắc **Flow-based Consolidation**: Giữ toàn bộ luồng chạy nghiệp vụ chính (piping flow, core execution, orchestration) tập trung trong một file chính duy nhất (ví dụ: `recon_tier_a.go`, `recon_heal.go`, `provisioning_orchestrator.go`, `transmuter.go`). File này có thể dài hơn 500 dòng nhưng đảm bảo tính liên tục của luồng code.
2. Chỉ phân tách các phần phụ trợ thực sự là các nhiệm vụ độc lập khác hoàn toàn luồng chính (chẳng hạn như ghi nhận trạng thái persistence, hoặc các hàm tiện ích chuyển đổi kiểu dữ liệu thuần túy).
3. Thực hiện audit và hợp nhất (re-merge) các file đã bị băm nhỏ quá mức ở Phase 1.

### Hệ quả
- **Tích cực**: Luồng code nghiệp vụ liên tục, dễ dàng debug và trace lỗi mà không cần chuyển đổi liên tục giữa nhiều file. Số lượng file vật lý trong package được thu gọn đáng kể.
- **Tiêu cực**: Các file chính sẽ dài hơn (từ 300 đến 600 dòng), nhưng vẫn nằm trong tầm kiểm soát và đảm bảo tính tập trung.

