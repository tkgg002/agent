# Context: Metrics Active Count for Shadow & Master Tables

## Bối cảnh
Hệ thống hiện tại có các metrics liên quan đến shadow và master tables. User yêu cầu bổ sung thêm metric đếm số lượng bản ghi có trạng thái active (count active) cho cả shadow và master tables.

## Mục tiêu (DoD)
- Định vị nơi thu thập và bắn metrics liên quan đến shadow & master tables.
- Bổ sung logic đếm các bản ghi "active" (thường là filter theo status/active flag hoặc logic định nghĩa active tương ứng trong hệ thống).
- Xuất metric này (thường qua Prometheus hoặc hệ thống đo lường hiện tại).
- Viết unit test hoặc integration test để đảm bảo metrics được bắn chính xác.
