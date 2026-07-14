# Yêu cầu: Thêm Số lượng lệch & Thời gian xử lý vào Nhật ký đối soát

## 1. Bối cảnh
- Trong giao diện DrillDown của một pipeline đối soát (khi click vào một dòng trên master grid), bảng "Nhật ký đối soát (30 phiên gần nhất)" hiện tại hiển thị các cột: Phiên lúc, Khoảng quét, Loại scan, Kết quả, Chi tiết.
- Cột "Chi tiết" hiển thị dạng text ghép `source_count → dest_count • thiếu ... • stale ...`.
- Người dùng muốn bổ sung cột riêng biệt để quan sát trực tiếp "Số lượng lệch" (drift/diff) và "Thời gian xử lý" (execution duration) cho mỗi phiên đối soát một cách trực quan, giúp tăng cường observability của hệ thống.

## 2. Yêu cầu chi tiết
- **Cột Số lượng lệch (Lệch):**
  - Hiển thị mức độ lệch `r.diff` giữa 2 trạm.
  - Sử dụng hàm format `fmtDrift(r.diff)` đã có sẵn trong file để hiển thị:
    - `0` (màu xanh lá) nếu không có lệch.
    - `-X (thiếu)` (màu đỏ) nếu trạm sau thiếu dữ liệu.
    - `+X (thừa)` (màu vàng/gold `#d4a017`) nếu trạm sau thừa dữ liệu (orphan).
  - Độ rộng cột phù hợp (ví dụ: `130px`).
- **Cột Thời gian xử lý:**
  - Hiển thị `duration_ms` (thời gian thực thi đối soát).
  - Định dạng thời gian xử lý:
    - Nếu dưới 1000 ms: hiển thị số ms (ví dụ: `450ms`).
    - Nếu từ 1000 ms trở lên: hiển thị số giây với 2 chữ số thập phân (ví dụ: `1.25s`).
  - Độ rộng cột phù hợp (ví dụ: `110px`).
- **Đảm bảo giao diện cân đối:**
  - Các cột được sắp xếp hợp lý, tránh làm tràn/vỡ layout bảng lịch sử phiên đối soát.
