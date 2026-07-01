# Lessons Learned: Hide Disabled Master Tables in Data Integrity

## Lessons Log
- **Lesson 1**: Lọc dữ liệu hiển thị (filtering) ở frontend là cách tiếp cận nhẹ nhàng và an toàn hơn cho các trường hợp dữ liệu lịch sử (historical reports) vẫn tồn tại ở backend nhưng thực thể cấu hình tương ứng (master binding) đã bị tắt hoặc xóa.
- **Lesson 2**: Cần cẩn trọng khi sử dụng query hook có queryKey trùng nhau trong React Query. Khi đưa query lên component cha, các component con vẫn có thể sử dụng lại query hook đó mà không lo bị duplicate network requests nhờ vào cơ chế caching tự động của React Query.
