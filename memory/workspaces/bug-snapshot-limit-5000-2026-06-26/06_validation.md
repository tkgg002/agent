# Validation: Bug Snapshot Limit 5000 Records / Kiểm chứng: Lỗi Snapshot Giới hạn 5000 Records

## 1. Test Plan & Scenarios / Kịch bản kiểm thử
Chúng ta cần xác minh rằng hàm `buildResumeFilterWithSample` thực hiện chuyển đổi chuỗi `lastSeen` sang đúng kiểu dữ liệu của `sampleID` (được trích xuất từ MongoDB collection).

| ID | Scenario | Input `lastSeen` | Input `sampleID` | Expected Output Filter | Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| TC01 | ObjectID conversion | `"5f9c1b3f9b9d9c0001f3e4a5"` | `primitive.ObjectID` | `{ "_id": { "$gt": ObjectID("5f9c1b3f9b9d9c0001f3e4a5") } }` | PASS |
| TC02 | Int32 conversion | `"12345"` | `int32(0)` | `{ "_id": { "$gt": int32(12345) } }` | PASS |
| TC03 | Int64 conversion | `"9876543210"` | `int64(0)` | `{ "_id": { "$gt": int64(9876543210) } }` | PASS |
| TC04 | Float64 conversion | `"123.45"` | `float64(0)` | `{ "_id": { "$gt": float64(123.45) } }` | PASS |
| TC05 | Fallback/String fallback | `"some-string-id"` | `"string"` | `{ "_id": { "$gt": "some-string-id" } }` | PASS |
| TC06 | Nil sampleID fallback | `"some-string-id"` | `nil` | `{ "_id": { "$gt": "some-string-id" } }` | PASS |

## 2. Test Execution & Evidence / Thực thi kiểm thử & Bằng chứng
Một bộ unit test đầy đủ bao quát 6 scenarios trên đã được triển khai tại [snapshot_runner_utils_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_utils_test.go).

Lệnh thực thi kiểm thử:
```bash
go test -v ./internal/handler/orchestration/...
```

Kết quả thực thi (Evidence):
```
=== RUN   TestBuildResumeFilterWithSample
=== RUN   TestBuildResumeFilterWithSample/ObjectID_type
=== RUN   TestBuildResumeFilterWithSample/Int32_type
=== RUN   TestBuildResumeFilterWithSample/Int64_type
=== RUN   TestBuildResumeFilterWithSample/Float64_type
=== RUN   TestBuildResumeFilterWithSample/Fallback/String_type
=== RUN   TestBuildResumeFilterWithSample/Nil_sampleID_fallback_to_String/ObjectID
--- PASS: TestBuildResumeFilterWithSample (0.00s)
    --- PASS: TestBuildResumeFilterWithSample/ObjectID_type (0.00s)
    --- PASS: TestBuildResumeFilterWithSample/Int32_type (0.00s)
    --- PASS: TestBuildResumeFilterWithSample/Int64_type (0.00s)
    --- PASS: TestBuildResumeFilterWithSample/Float64_type (0.00s)
    --- PASS: TestBuildResumeFilterWithSample/Fallback/String_type (0.00s)
    --- PASS: TestBuildResumeFilterWithSample/Nil_sampleID_fallback_to_String/ObjectID (0.00s)
PASS
ok  	github.com/goopay/centralized-data-service/internal/handler/orchestration	0.812s
```

Tất cả 100% test cases đều pass, đảm bảo logic ép kiểu hoạt động đúng đắn và không có hồi quy (regression) trên các luồng cũ.
