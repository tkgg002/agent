# Agent Lifecycle Hooks (Antigravity `hooks.json`)

Hệ thống cưỡng chế quy trình tự động gồm 7 cổng (`H1` – `H7`) được thực thi bởi bộ điều phối duy nhất tại [`agy/agy_guard.py`](./agy/agy_guard.py) và kiểm thử tự động bởi 16 kịch bản trong [`agy/tests/test_agy_guard.py`](./agy/tests/test_agy_guard.py):

| Hook | Sự kiện Antigravity | Chức năng cưỡng chế |
|------|---------------------|---------------------|
| **H1 boot** | `PreInvocation` | Bơm nhắc nhở Boot `S0` và kiểm tra trạng thái Laya Server (`:8200`) ở lượt đầu tiên của hội thoại. |
| **H2 memory_guard** | `PreToolUse` (`write_to_file`, `replace_file_content`) | Chặn cứng (`deny`) mọi thao tác `Overwrite: true` lên file đã tồn tại trong `agent/memory/`; chặn làm mất `TargetContent` trên các file append-only (`05_progress`, `04_decisions`, `lessons.md`). |
| **H3 secret_guard** | `PreToolUse` (`write_to_file`, `replace_file_content`) | Chặn cứng (`deny`) nếu ghi chuỗi giống Secret/Token/URI có mật khẩu vào `agent/memory/`; yêu cầu xác nhận (`force_ask`) nếu ghi vào source code. |
| **H4 change_gate** | `PreToolUse` (`write_to_file`, `replace_file_content`) | Chặn sửa source code nếu chưa tạo workspace, chưa append `05_progress`, hoặc chưa đạt `DG1 PASS`; yêu cầu xác nhận (`force_ask`) nếu chưa có lời `APPROVE` của User sau `09_tasks_solution`. |
| **H5 command_guard** | `PreToolUse` (`run_command`) | Chặn cứng (`deny`) `git push` (Rule #15), `git reset --hard`, `git clean -f` (Rule #19), `rm -rf` vào `memory`/`.git` (Rule #17); `force_ask` với lệnh ghi trực tiếp vào DB (Rule #12). |
| **H6 fast_feedback** | `PostInvocation` | Tự động tìm `.agent-project.json` gần nhất của file vừa sửa và chạy các lệnh `checks` của component đó. |
| **H7 done_gate** | `Stop` | Chặn kết thúc phiên (`decision: "continue"`) nếu thiếu `DG1/DG2/DG3`, thiếu cập nhật `05_progress`/`06_validation` sau lần sửa code cuối, thiếu `runtime_check` của component, hoặc `verify_governance.py` báo lỗi (tối đa 2 lần chặn/hội thoại). |

Các script shell cũ dành cho Claude Code được lưu trữ tại [`_archive_claude/`](./_archive_claude/).
