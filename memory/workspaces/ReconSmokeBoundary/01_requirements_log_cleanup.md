# Yêu cầu (01_requirements_log_cleanup)

## Bối cảnh
Người dùng yêu cầu dọn dẹp các log cũ và không chính xác, cụ thể là tiền tố log `[tier2]` trong file `recon_tier_a.go` không còn chính xác vì đây thuộc tầng đối soát Tier A. Cần thay đổi tất cả các prefix/thông điệp log `[tier2]`, `tier2`, `tier3` trong `recon_tier_a.go` thành `tierA` / `[tierA]` để đồng bộ với `[tierB]` của `recon_tier_b.go`.

## Yêu cầu chi tiết
1. Quét toàn bộ `recon_tier_a.go` để tìm các tag `[tier2]`, `tier2`, `tier3` trong log.
2. Thay thế chúng bằng `[tierA]` hoặc `tierA` tương ứng.
3. Chạy kiểm thử tự động để đảm bảo việc thay đổi chuỗi log không làm hỏng code hoặc làm hỏng test.
