# Decisions: Hermes Closed Learning Loop Architecture

## Decision 1: Cấu trúc thư mục Skill và Tiêu chuẩn hóa
- **Bối cảnh**: Chúng ta cần một định dạng kỹ năng dễ đọc, dễ viết và tương thích với cả các hệ thống agent bên ngoài sau này.
- **Quyết định**: Áp dụng định dạng `SKILL.md` theo tiêu chuẩn *agentskills.io*. Mỗi kỹ năng sẽ là một thư mục riêng nằm dưới `agent/skills/<category>/<skill-name>/` chứa:
  - `SKILL.md` (metadata yaml + instructions + pitfalls + verification).
  - Thư mục `scripts/` (các công cụ bổ trợ đi kèm skill đó).
  - Thư mục `references/` (tài liệu tham khảo chuyên sâu).

## Decision 2: Ngôn ngữ phát triển Công cụ Học tập (Learning Tools)
- **Bối cảnh**: Các tác vụ xử lý tệp tin, trích xuất dữ liệu markdown và so khớp chuỗi đòi hỏi sự linh hoạt, tốc độ và khả năng xử lý cấu trúc dữ liệu tốt.
- **Quyết định**: Sử dụng **Python 3** làm ngôn ngữ phát triển các công cụ cốt lõi (`sop_patcher.py`, `skill_miner.py`, `rule_promoter.py`). Python có thư viện xử lý markdown, regex mạnh mẽ, và dễ bảo trì hơn so với Bash Script thuần túy cho logic phức tạp.

## Decision 3: Cơ chế vá lỗi SOP (SOP Patching Method)
- **Bối cảnh**: Tránh rủi ro LLM ghi đè hoặc phá hủy cấu trúc của các file kịch bản quy trình quan trọng trong `agent/workflows/`.
- **Quyết định**: Áp dụng phương pháp **Semantic Patching**. Script sẽ không viết lại toàn bộ file kịch bản, mà sẽ tìm phần khối thông tin (ví dụ: danh sách lệnh thực thi hoặc bước kiểm thử) và thay thế chính xác bằng nội dung tối ưu đã học được, tương tự cơ chế `replace` tool của Gemini CLI để tối thiểu hóa rủi ro phá hoại dữ liệu.
