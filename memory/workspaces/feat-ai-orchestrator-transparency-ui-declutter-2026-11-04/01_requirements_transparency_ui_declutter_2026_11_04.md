---
tier: T2
mode: standard
target_project: ai-orchestrator
suffix: transparency_ui_declutter_2026_11_04
created_at: 2026-11-04T17:51:30+07:00
---

# 01. Requirements — Minh Bạch Thực Thi Model Theo Phiên/Task, Khởi Tạo Memory Định Nghĩa Repo & Tinh Gọn UI/UX (`ai-orchestrator`)

## 1. Yêu cầu gốc của User
1. **Feature 1 — Minh bạch tuyệt đối ở Frontend (Per-Task Session Model Call Ledger) & Quản trị Memory Định nghĩa Repo (Repo Definition Memory Bootstrap):**
   - **Ở mỗi phiên (session/round) của từng Task:** Hiển thị rõ ràng trên Frontend:
     - Gọi **Model nào** (`model`, `runner`/`cli`, `conv_id`).
     - **Prompt gửi đi như thế nào** (`system_prompt` + `user_prompt` nguyên bản, không cắt xén mập mờ).
     - **Model trả ra như thế nào** (`raw_response` nguyên bản, `status`, `duration_ms`, `timestamp`, `stage`).
   - **Cô lập tuyệt đối theo Task ("cái task nào phải ra task đó"):** Không trộn lẫn log, không copy/fallback sang task khác, không suy đoán hay liên kết chéo qua các task khác.
   - **Cơ chế Memory Define cho từng Repo (`Repo Memory Definition`):**
     - Khi bắt đầu làm việc với 1 repository (`targetRepo`), hệ thống kiểm tra trong bộ nhớ (`memory/global/project/<repo_name>/repo_definition.json` hoặc `.orchestrator/memory/repos/<repo_name>/repo_definition.json`) xem đã có bản định nghĩa (`define`) của repo đó chưa.
     - **Nếu có:** Đọc trực tiếp bản định nghĩa đó để nạp vào ngữ cảnh thực thi (hiển thị trạng thái `LOADED FROM MEMORY` trên FE).
     - **Nếu chưa có:** Tự động quét cấu trúc thực tế của repo (`language`, `module/package`, `entrypoints`, `build_cmd`, `test_cmd`, `key_dirs`, `current_branch`) để **init bản define** và lưu xuống đĩa dùng cho các lượt sau (hiển thị trạng thái `INITIALIZED & SAVED` trên FE). Tuyệt đối không đi lòng vòng, không suy đoán lan man, không liên kết qua các task khác.

2. **Feature 2 — Cắt Gọt & Tinh Gọn Toàn Diện UI/UX (UI/UX Decluttering & Label Audit):**
   - Hiện tại giao diện WebUI (`index.html`, `app.js`, `styles.css`) quá rối mắt, lặp đi lặp lại cùng một thông tin ở nhiều nơi (Topbar, Sidebar card, Workspace header, Stage banner, Sub-task table, Footer).
   - Bắt buộc **rà soát từng label text / badge / pill** trên toàn bộ giao diện:
     - Xác định rõ: Label nào **CẦN HIỂN THỊ**, label nào **XÓA BỎ (DO LẶP LẠI / VỚ VẨN)**, label nào **GOM NHÓM / THU GỌN**.
     - Thiết kế lại cách hiển thị trực quan, sạch sẽ, nhìn vào thấy ngay trọng tâm (Task đang chọn -> Repo Memory -> Sổ cái Minh bạch Model Call -> Tài liệu / Sub-tasks).

## 2. Ràng buộc Thực thi (Technical Constraints từ GEMINI.md & Lessons)
- **Rule #1, #9, #13 & `L015` (`Propose-Only`):** Giữ chế độ Read-Only 100% trên source code của `/Users/trainguyen/Documents/work/ai-orchestrator` ở pha `S1 -> S3`, trình bày Bảng Kiểm Toán Label UI + Thiết kế Giải pháp 1-1 + Code Demo trực tiếp trên chat và dừng chờ `APPROVE`.
- **`L001` (`2026-11-04` — `Per-Task Model Call Transparency Ledger, Repo Memory Bootstrap & UI Label Decluttering`):** Cô lập nhật ký lời gọi model theo từng Task, nạp/khởi tạo bộ nhớ định nghĩa kho mã nguồn độc lập không liên kết chéo task, và loại bỏ triệt để các nhãn giao diện lặp lại.
- **`L018` (`Single-Responsibility Module Split & Line-Budget Guard`):** Không nhồi thêm hàng trăm dòng vào các file lớn (`task_inspector.go`, `app.js`); tách module đơn trách nhiệm (`repo_memory_store.go`, `task_llm_trace.go`).
- **`L019` (`Full Real-LLM Wiring & Single Session Reuse`):** Lưu trữ và hiển thị 100% đầu vào/đầu ra thật của LLM cho từng Task, không dùng văn mẫu giả tạo hay liên kết chéo sang task khác.
- **`L001` (`2026-11-05` — `Preserve Workflow & Wave Topology Visualization During UI Label Decluttering`):** Không được ẩn bỏ thanh **Workflow & Wave của Task** (`#autopilot-flow` tại `internal/webui/static/index.html:290`). Bắt buộc hiển thị trực quan trên màn hình chính của Task cả (1) **Luồng Bước Quy Trình (`S1 Intake → S2.5 DG0 → S3 Plan & Demo → Human Gate → S4 Implement → S5 QA & DG3`)** đã cắt bỏ chữ lặp rườm rà và (2) **Dải Wave Thực Thi Sub-Tasks (`Wave 0 → Wave 1 → Wave 2...`)** của riêng Task đang chọn.
- **`L060` (`2026-11-05` — `Visual UI/UX Prototype, Semantic Color Palette & Progress Preview First Before Coding`):** Bắt buộc trình bày trước bản mẫu trực quan UI/UX (bảng màu ngữ nghĩa 5 trạng thái, thanh tiến độ tổng thể 0%→100% + mini progress bar từng bước/từng Wave, bố cục màn hình) để User xem và `APPROVE` trước khi chạm vào mã nguồn; tuyệt đối không được ẩn mất thanh tiến độ (`#live-telemetry-hud` / progress strip) hay xóa mất hệ màu trạng thái gốc của dự án.
