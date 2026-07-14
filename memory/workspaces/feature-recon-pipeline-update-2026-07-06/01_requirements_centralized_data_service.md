# Yêu cầu Refactor Tier sang TypeRecon trong Centralized Data Service

Yêu cầu cụ thể:
1. Loại bỏ hoàn toàn các hàm mang tên `RunTier1`, `RunTier2`, `RunTier3` trong module đối soát core (`centralized-data-service`).
2. Thay thế bằng các tên phản ánh chính xác nghiệp vụ:
   - `RunTier1` -> `RunSmokeCheck` (Đối soát tổng thể active records counts toàn thời gian)
   - `RunTier2` -> `RunHashWindowCheck` (Đối soát lookback window sử dụng XOR hash)
   - `RunTier3` -> `RunDeepCheck` (Đối soát sâu toàn bộ bảng sử dụng 256-bucket fingerprint)
3. Đảm bảo toàn bộ các module gọi đến (API handler, Heal daemon, Scheduler engine) và các unit test suites biên dịch và hoạt động chính xác 100%.
