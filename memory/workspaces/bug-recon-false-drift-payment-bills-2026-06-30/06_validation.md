# Validation: Fix False Drift on Recon payment_bills / Kiểm chứng: Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills

## 1. Kịch bản kiểm thử (Test Plan & Scenarios)
Chúng ta cần xác minh rằng `ReconDestAgent` thực hiện chính xác các truy vấn và tính toán hash theo hai hệ quy chiếu thời gian khác nhau (default `_source_ts` và dynamic domain timestamp).

| ID | Scenario | Field cấu hình | Loại query/hash | Kết quả mong đợi | Trạng thái |
| :--- | :--- | :--- | :--- | :--- | :--- |
| TC01 | CountInWindow (Default) | `_source_ts` | `CountInWindow` | Query lọc theo cột `_source_ts` kiểu bigint | PASS |
| TC02 | CountInWindow (DomainTS) | `lastUpdatedAt` | `CountInWindow` | Query lọc theo cột `lastUpdatedAt` kiểu timestamp | PASS |
| TC03 | BucketCounts (Default) | `_source_ts` | `BucketCounts` | Sinh bucket thống kê theo cột `_source_ts` | PASS |
| TC04 | BucketCounts (DomainTS) | `lastUpdatedAt` | `BucketCounts` | Sinh bucket thống kê theo cột `lastUpdatedAt` | PASS |
| TC05 | HashWindow (Default) | `_source_ts` | `HashWindow` | XOR Hash dựa trên cột `_source_ts` | PASS |
| TC06 | HashWindow (DomainTS) | `lastUpdatedAt` | `HashWindow` | XOR Hash dựa trên cột `lastUpdatedAt` | PASS |

## 2. Thực thi kiểm thử & Bằng chứng (Test Execution & Evidence)
Một bộ unit test đầy đủ bao quát toàn bộ 6 scenarios trên đã được triển khai tại file [recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go).

Lệnh thực thi kiểm thử:
```bash
go test -v ./internal/service/recon/...
```

Kết quả thực thi (Evidence):
```
=== RUN   TestDestAgent_CountInWindow_Default
--- PASS: TestDestAgent_CountInWindow_Default (0.00s)
=== RUN   TestDestAgent_CountInWindow_DomainTS
--- PASS: TestDestAgent_CountInWindow_DomainTS (0.00s)
=== RUN   TestDestAgent_BucketCounts_Default
--- PASS: TestDestAgent_BucketCounts_Default (0.00s)
=== RUN   TestDestAgent_BucketCounts_DomainTS
--- PASS: TestDestAgent_BucketCounts_DomainTS (0.00s)
=== RUN   TestDestAgent_ListIDTsInWindow_Default
--- PASS: TestDestAgent_ListIDTsInWindow_Default (0.00s)
=== RUN   TestDestAgent_ListIDTsInWindow_DomainTS
--- PASS: TestDestAgent_ListIDTsInWindow_DomainTS (0.00s)
=== RUN   TestDestAgent_HashWindow_Default
--- PASS: TestDestAgent_HashWindow_Default (0.00s)
=== RUN   TestDestAgent_HashWindow_DomainTS
--- PASS: TestDestAgent_HashWindow_DomainTS (0.00s)
=== RUN   TestDestAgent_BucketHash_Default
--- PASS: TestDestAgent_BucketHash_Default (0.00s)
=== RUN   TestDestAgent_BucketHash_DomainTS
--- PASS: TestDestAgent_BucketHash_DomainTS (0.00s)
PASS
ok  	centralized-data-service/internal/service/recon	0.573s
```
100% test cases đều pass, đảm bảo logic hoạt động chính xác trên cả hai hệ quy chiếu và không gây lỗi hồi quy.
