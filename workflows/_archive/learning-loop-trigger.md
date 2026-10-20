# Workflow: Hermes Learning Loop Trigger

Hướng dẫn kích hoạt quy trình học tập khép kín tự động (Hermes Learning Loop) ở cuối mỗi phiên làm việc (session).

## 1. Mục tiêu / Objective
Đảm bảo mọi tri thức, kinh nghiệm, cạm bẫy và quy tắc rút ra từ session hiện tại được lưu vết, đóng gói thành kỹ năng chuẩn hóa (`SKILL.md`) và cập nhật luật hiến pháp (`GEMINI.md`/`CLAUDE.md`) trước khi kết thúc phiên.

## 2. Cách thức kích hoạt / Execution
Trước khi báo hoàn thành nhiệm vụ và kết thúc phiên làm việc với User, Muscle hoặc Brain BẮT BUỘC phải thực thi script tích hợp sau:

```bash
./agent/scripts/wrapup.sh
```

### Các bước thực thi tự động bên trong `wrapup.sh`:
1. **SOP Patcher (`sop_patcher.py`)**: Quét các chỉ thị sửa đổi quy trình dạng `[SOP_PATCH]` để vá tự động các tệp `.md` trong `agent/workflows/`.
2. **Skill Miner (`skill_miner.py`)**: Quét nhật ký tiến độ (`05_progress.md`) của workspace hiện tại để sinh ra tệp `SKILL.md` tương ứng trong `agent/skills/`.
3. **Rule Promoter (`rule_promoter.py`)**: Quét tệp `lessons.md` để đếm tần suất các tag. Nếu có bất kỳ tag lỗi nào xuất hiện $\ge 3$ lần, hệ thống tự động sinh luật và chèn vào `agent/GEMINI.md`, đồng bộ sang `/Users/trainguyen/Documents/work/CLAUDE.md`.

## 3. Quy định hậu kiểm / Post-flight Validation
Sau khi chạy script `wrapup.sh`, hãy xác minh:
- Thư mục kỹ năng `agent/skills/` có được cập nhật hay không.
- Các quy tắc mới có được phản ánh đồng bộ trên cả `GEMINI.md` và `CLAUDE.md`.
- Các tệp sao lưu `.bak` được tạo để tránh rủi ro phá hủy dữ liệu.
