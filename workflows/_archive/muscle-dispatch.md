---
description: Quy trình tự động tra cứu, lựa chọn và thực thi Muscle workflow phù hợp nhất cho task kỹ thuật.
---

# Muscle Dispatch (Autonomous Selector)

Dùng quy trình này khi Brain cần thực hiện một công việc kỹ thuật chuyên sâu (Technical Task) và muốn sử dụng "vũ khí" Muscle mạnh nhất từ Hạt nhân mới (`agent/workflows/`).

## 1. Tìm kiếm (Discovery)

Brain thực hiện tra cứu:
1. Đọc `[OPERATOR_MAP.md](file:///Users/trainguyen/Documents/work/agent/OPERATOR_MAP.md)` để xác định Nhóm công cụ phù hợp.
2. Tìm keywords trong danh mục workflows tại `[agent/workflows/](file:///Users/trainguyen/Documents/work/agent/workflows/)`.
3. Nếu task yêu cầu tính năng **PRP (Plan-Record-Perform)** → Ưu tiên `/prp-plan`.
4. Nếu task yêu cầu **TDD (Test-Driven Development)** → Ưu tiên `/tdd`.
5. Nếu task là **Fix Bug** nhạy cảm → Ưu tiên `/santa-loop`.

## 2. Thử nghiệm (Simulation / Review)

Sau khi chọn được workflow mục tiêu:
1. Brain đọc (`view_file`) file `.md` của workflow đó trong `agent/workflows/`.
2. Phân tích các bước thực hiện và yêu cầu đầu vào ($ARGUMENTS).
3. Đánh giá rủi ro: Nếu workflow có khả năng thay đổi code diện rộng → Ghi chú lại.

## 3. Thực thi (Execution)

Brain thực hiện uỷ quyền cho Muscle:
1. Gọi `/muscle-execute <workflow_name> <arguments>`.
2. Theo dõi tiến độ qua `command_status`.
3. Cập nhật `05_progress.md` trong workspace hiện tại.

## 4. Bảo mật Tự động (Security Auto-Check)

> [!IMPORTANT]
> **Sau mỗi lần thực thi workflow có thay đổi code (Write/Edit)**: Brain **BẮT BUỘC** gọi `/security-agent` để rà soát lỗ hổng trước khi báo cáo kết quả cho User.

## 5. Rút kinh nghiệm (Learning)

Nếu workflow Muscle hoạt động hiệu quả hoặc gặp lỗi:
1. Ghi nhận Pattern thành công vào `agent/memory/global/lessons.md`.
2. Nếu workflow cần điều chỉnh cho dự án → Đề xuất update `OPERATOR_MAP.md`.
