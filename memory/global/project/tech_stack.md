# 🏛️ Core Principles & Tech Stack (Global Fundamentals)

> **BẢN CHẤT**: File này chứa các "Nguyên tắc cốt lõi" (Core Principles) và Kỷ luật Vận hành. Đây là DNA của Agent, MẶC ĐỊNH PHẢI TUÂN THỦ 100% nhằm loại bỏ các bẫy hành vi, ảo giác (hallucination) và over-engineering.

## 1. Kỷ luật Phân vai (Brain ↔ Muscle) & Thực thi
- **Brain Plan, Muscle Execute (Rule 1 & 12)**: Brain CHỈ lập kế hoạch, review và delegate. Brain TUYỆT ĐỐI KHÔNG ĐƯỢC dùng tools sửa code trực tiếp (`file_edit`, `replace_file_content`). Việc chạm vào code là của Muscle.
- **Không Báo Cáo Láo (Anti-Fraud)**: Tuyệt đối không tự ý sửa Plan/Requirement (ví dụ đổi thành `Deferred`) để che đậy việc code lỗi hoặc chưa làm. Nếu bị block, báo cáo trung thực error log cho User.
- **Minimal Impact (Zero Scope Creep)**: Yêu cầu sửa 1 điều kiện -> sửa ĐÚNG 1 điều kiện đó. Không over-engineer, không đẻ thêm file, test hay spawn hàng loạt sub-agent không cần thiết cho 1 task <5 files. Blast radius = 0.
- **Analyze ≠ Implement**: Giao task "phân tích", output là Design Doc + Code Demo. Cấm tự ý sửa source code thật khi chưa được User `Approve`.
- **Kỷ luật Commit**: Khi refactor lớn (move/rename/split), KHÔNG tự `git commit` từng bước nhỏ. Gom toàn bộ thay đổi -> `go build` verify -> để User review diff qua IDE -> User quyết định commit.

## 2. Kỷ luật Không gian (Workspace) & Tri thức (Memory)
- **Workspace-First Rule (Rule 9)**: Nhận task mới (kể cả viết Docs) BẮT BUỘC tạo thư mục `agent/memory/workspaces/<feature>`. 1 Mạch việc / 1 Task = 1 Workspace độc lập. Không dùng khung chat làm bộ nhớ tạm (Shadow Document).
- **Audit-Log Bất biến (Rule 11)**: Mọi file log (`05_progress.md`, `lessons.md`) TUYỆT ĐỐI CHỈ APPEND. Cấm dùng cờ `Overwrite: true` gây xóa sạch dữ liệu lịch sử.
- **No Lazy Archaeology**: Phải tự dùng tool đọc hết workspace (ADR, `00_context`, `04_decisions`) trước khi hỏi User.
- **Bảo tồn Comments**: Khi rewrite file, GIỮ NGUYÊN mọi comment (đặc biệt là tiếng Việt và design notes). Cấm tự ý xóa hoặc dịch sang tiếng Anh.

## 3. Tiêu chuẩn Hoàn thành (DoD) & Kiểm thử
- **Build Pass ≠ Test Pass ≠ Done**: Compile (`go build`) thành công chỉ là bước 1. Báo "Done" PHẢI có bằng chứng chạy runtime thực tế hoặc E2E.
- **Verify by Delta**: Test phải tạo ra sự thay đổi (DELTA) ở DB/Log. Kết quả test giống hệt trạng thái No-op (không chạy gì) là test giả (False Confidence).
- **Safe Process Kill (CẤM DÙNG LSOF)**: Trước khi test lại service, phải kill tiến trình. **TUYỆT ĐỐI CẤM** dùng `lsof -ti :PORT | xargs kill -9` (sẽ giết lây SSH Tunnel/DBeaver của hệ thống). Bắt buộc dùng `pgrep -f "exact-binary-name" | xargs kill -9`.
- **Precondition Check**: Test trên Fixture thất bại, hãy check xem Fixture đó có thoả mãn điều kiện nghiệp vụ không (vd: thiếu status approved) TRƯỚC khi đổ lỗi cho code và revert bừa bãi.
- **Xác nhận Intent**: Nếu gặp trường hợp "Empty State", phải hỏi User: *"Empty là trạng thái hợp lệ hay là BUG?"*. Không tự ý vẽ UI che giấu lỗi thiếu dữ liệu.

## 4. Kiến trúc & Thiết kế Core (Architecture)
- **Hexagonal Boundaries**: Tầng HTTP/API Handler KHÔNG chạm trực tiếp hạ tầng (`gorm.DB`, `nats.Conn`, `gorm.ErrRecordNotFound`). Logic DDL (Schema Setup) và DML (Data Transform) phải được tách biệt ở 2 service/handler riêng.
- **Single Source of Truth**: Resource name động, config, phải đọc từ DB/Registry tập trung. Cấm hardcode.
- **Fail-Loud / No Smart Coercion**: DSN thiếu credentials, Input sai schema -> Quăng lỗi (Fail-fast). KHÔNG tự ý ép kiểu thông minh (Smart coercion) che giấu lệch Schema làm hỏng data thầm lặng.
- **Validation Order**: Pipeline config: `Read -> Env Override -> VALIDATE -> Fallbacks`. Validate TRƯỚC fallback để bắt đúng lúc user truyền input rỗng.