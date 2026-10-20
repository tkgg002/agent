# 11 — Summary Report: Mandatory Final Adversarial Audit & Anti-Smoke-Speculation Gate (`mandatory_final_audit`)

## 1. Tóm tắt Thay đổi

Nhằm giải quyết triệt để tình trạng Agent suy diễn từ smoke test / mockup và buộc User phải nhắc chạy Audit phản biện thủ công sau mỗi task, Phase 7 (`mandatory_final_audit`) đã tích hợp **Cổng Chống Suy Diễn Smoke Test (Anti-Smoke-Speculation Gate)** và **Quy trình Audit Phản Biện Cuối Cùng Tự Động (`audit_report_<suffix>.md` — 6 phần)** vào toàn bộ vòng đời thực thi của Agent:

1. **Chặn trước khi viết code (`H4` — `handle_pre_write`):** Chặn cứng (`deny`) mọi hành vi tiêm hàm mock/closure giả (`*Fn func(`, `lookupMasterTargetsFn`, `runEventsFn`, `sqlmock`, `fakemodel`) vào file source code production (`not is_test_file(f)`).
2. **Tự động kích hoạt ngay sau khi sửa code (`H6` — `handle_post_invocation`):** Mỗi khi phát hiện chỉnh sửa source code (`new_edits`), bên cạnh việc chạy `checks`, Hook `H6` **luôn tự động bơm `FINAL_AUDIT_MSG` (6 bước)** vào `injectSteps` để ép Agent thực hiện Audit phản biện từng dòng diff, kiểm tra 3 trục lỗi thực tế (Concurrency/Multi-worker, Timeout/Heartbeat, Data Continuity/Watermark), phối hợp với User trace runtime value khi cần, áp dụng Self-Improvement Loop, chạy `/laya_router` `DG3`, và lưu `audit_report_<suffix>.md`.
3. **Chặn cứng ở cổng kết thúc (`H7` — `handle_stop` & `tooling/verify_governance.py`):** Cấm đóng phiên/báo cáo hoàn tất nếu chưa tạo/cập nhật `audit_report_<suffix>.md` sau lần sửa code cuối, hoặc nếu `audit_report_<suffix>.md` thiếu bất kỳ mục nào trong 6 mục bắt buộc, hoặc nếu chưa chạy lại `/laya_router` (`DG3` = `PASS`) sau lần sửa code cuối.

## 2. Danh sách File đã Thay đổi / Tạo mới

| File | Loại thay đổi | Mô tả chi tiết |
|------|---------------|----------------|
| `/Users/trainguyen/Documents/work/agent/GEMINI.md` | Cập nhật (`147` dòng) | Cập nhật Rules #4, #9 (`S4`, `S5`, `S6`), #12, #14 (`G3`, `G4`, `G7`, `G8`), #18 bổ sung Anti-Smoke-Speculation & `audit_report_<suffix>.md`. |
| `/Users/trainguyen/Documents/work/agent/skills/qa-agent/SKILL.md` | Cập nhật (`122` dòng) | Bổ sung Ma trận Chống Suy Diễn Smoke Test (L001/L044/L056/L061) ở `G3` và cấu trúc 6 phần bắt buộc của `audit_report_<suffix>.md` ở `G7`/`G8`. |
| `/Users/trainguyen/Documents/work/agent/skills/session-wrapup/SKILL.md` | Cập nhật (`58` dòng) | Bổ sung `audit_report_<suffix>.md` vào Bước 1 đóng phiên `S6`. |
| `/Users/trainguyen/Documents/work/agent/skills/workspace-docs/SKILL.md` | Cập nhật (`70` dòng) | Bổ sung `audit_report_*.md` vào bảng Tier `T1`/`T2`/`T3` và Mandatory Doc Registry. |
| `/Users/trainguyen/Documents/work/agent/skills/muscle-execute/SKILL.md` | Cập nhật (`69` dòng) | Cấm làm bẩn production struct bằng mock closure ở Bước 4, chống suy diễn smoke test & phối hợp User trace runtime ở Bước 5, lập `audit_report_<suffix>.md` ở Bước 6. |
| `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py` | Cập nhật (`636` dòng) | Thêm `PROD_MOCK_POLLUTION`, `FINAL_AUDIT_MSG`, `AUDIT_REQUIRED_PATTERNS`, `is_test_file()`, `audit_report_issues()`; nâng cấp `H4`, `H6`, `H7`. |
| `/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py` | Cập nhật (`686` dòng) | Thêm 4 unit test mới (`test_21`..`test_24`), tổng cộng `24/24` PASS. |
| `/Users/trainguyen/Documents/work/agent/tooling/verify_governance.py` | Cập nhật (`293` dòng) | Thêm `audit_report_*.md` vào `TIER_RULES["T2"]`/`["T3"]`, `SUFFIX_SCOPED_PREFIXES`, và kiểm tra 6 mục `AUDIT_REQUIRED_PATTERNS` ở chặng `S6`. |
| `/Users/trainguyen/Documents/work/agent/tooling/tests/test_verify_governance.py` | Cập nhật (`184` dòng) | Cập nhật test `T2` và thêm `test_incomplete_audit_report_fails_in_t2_s6`, tổng cộng `14/14` PASS (`tooling`). |
| `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` & `lessons/INDEX.md` | Append (`1930` dòng) | Ghi nhận bài học mới (`2026-10-19`) và đồng bộ `247/247` entries vào `INDEX.md`. |
