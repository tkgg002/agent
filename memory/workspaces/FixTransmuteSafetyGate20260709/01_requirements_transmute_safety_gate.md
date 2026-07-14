# Yêu cầu: Khắc phục lỗi transmute safety gate batchSize

## 1. Bối cảnh
Khi chạy transmute với số lượng `onlySourceIDs` lớn hơn 2000, hệ thống trả về lỗi từ safety gate:
`transmute safety gate: len(onlySourceIDs) = 3822 vượt quá batchSize = 2000, hãy chia lô nhỏ hơn`
Điều này làm gián đoạn tiến trình đồng bộ dữ liệu shadow DB sang master DB đối với các lô lớn (ví dụ shadow_test_ss.schedule_histories).

## 2. Mục tiêu
- Loại bỏ lỗi safety gate bằng cách tự động chia nhỏ (chunking) `onlySourceIDs` thành nhiều lô nhỏ hơn hoặc bằng `batchSize` (2000) trực tiếp trong hàm `Run` của `TransmuterModule`.
- Từng lô nhỏ sau khi được chia ra sẽ được transmutation tuần tự (hoặc song song, nhưng tuần tự là an toàn và khớp với logic checkpoint hiện tại) và cộng dồn kết quả vào `TransmuteResult`.
- Đảm bảo tính nhất quán của logic Orphan Master (so sánh danh sách ID của lô hiện tại với dữ liệu shadow tương ứng để phát hiện bản ghi mồ côi và soft-delete trên Master).
- Toàn bộ test case hiện có của `transmuter` phải PASS và không bị ảnh hưởng.
- Viết thêm test case kiểm thử việc truyền danh sách ID lớn hơn `batchSize` để chứng minh giải pháp hoạt động đúng đắn.

## 3. Ràng buộc & Tiêu chuẩn Gates (DoD)
- Không chỉnh sửa source code trực tiếp từ Brain, phải delegate qua Muscle subagent.
- Mọi thay đổi phải được phản ánh vào `05_progress_transmute_safety_gate.md`.
- Chạy toàn bộ test suite của service để đảm bảo không bị regression.
- Không tự ý commit git.
