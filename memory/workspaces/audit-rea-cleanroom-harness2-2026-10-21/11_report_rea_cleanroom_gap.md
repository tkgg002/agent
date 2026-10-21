# 11 — Report `rea_cleanroom_gap`

## Đã làm (sau User APPROVE 09:46)
| ID | Vấn đề | Sửa | Bằng chứng |
|---|---|---|---|
| G-08 | Lesson chỉ ghi biên bản, không chặn trước | Lesson có `ID/Trigger/Check/Require` → `tripwires.json`; H1 tiêm Check khớp prompt; H4t chặn ghi kết luận khi chưa có bằng chứng | Replay transcript thật chặn đúng lượt 1; M1, M6, M7 KILLED |
| G-01 | `APPROVE` khớp "đã được Jev duyệt"; thiếu `BẮT ĐẦU` | Neo đầu dòng; clean-room chỉ mở bằng dòng `BẮT ĐẦU`/`APPROVE` | M3 KILLED |
| G-02 | Đọc code gốc qua `run_command`/URL lọt | `READ_ARG_KEYS` | M2 KILLED |
| G-03 | Spec chỉ cần tiêu đề | Mỗi `R<n>` phải có `REF: path:line` (H4 + verify_governance) | M4 KILLED |
| G-04 | Prompt subagent chứa code gốc | H7 cảnh báo theo path/dòng code; resolve path tương đối (AF-2) | M8 KILLED |
| G-05 | `run-recursion` dùng proposer/scorer giả | Xoá stub, bắt buộc lệnh thật, base tree = harness thật | M5 KILLED |
| G-06 | REA chưa đăng ký | Dòng `reverse-engineer-anything` trong `OPERATOR_MAP.md` (REA do `rea-agents setup` cài lúc 09:56) | validate-surface VALIDATED |

## Kết quả
hooks 49/49, tooling 37/37, mutation 10/10 KILLED + SHA restore, DG1/DG2/DG3 PASS (FLAG đã resolve bằng REF + mutant). Vòng H7 stop-audit tìm thêm AF-4 (bằng chứng trỏ vào chính workspace/`$VAR`) và AF-5 (chuỗi đường dẫn giả được tính là bằng chứng) — đã sửa.

## Còn mở
1. 260/262 lesson cũ chưa có tripwire — cần backfill dần (ưu tiên lesson tái phạm).
2. `Require` chưa gắn đúng repo của Trigger.
3. Chưa chạy `run-recursion` với CLI agent thật; chưa trace hook trong subagent; chưa có task clean-room thật để chạy H4t/H7 end-to-end trong IDE.
4. Có phiên song song đang ghi `lessons.md` — nên tránh 2 phiên cùng sửa `/agent`.
