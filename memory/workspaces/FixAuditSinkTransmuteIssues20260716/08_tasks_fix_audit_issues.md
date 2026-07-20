# Task List - Fix Audit Sink & Transmute Issues

- [x] `P0-1 & P0-2`: Tối ưu offset tracking & flush timing gap (CommitInterval = 0, Highest Offset map, Rebalance handling)
- [x] `P0-3`: Thêm log và metrics cho 4 điểm silent drop (Empty value, Nil afterData, Source not registered, Missing PK)
- [x] `P0-4`: Bổ sung recover cho transmute goroutine và cancel context khi panic để giải phóng resource
- [x] `P0-5`: Fix bare type assertions trong dedup (switch type để tránh float64 json parsing trap)
- [x] `P1-1`: Thêm retry logic cho bulkUpsertMaster
- [x] `P1-2`: Chuyển NATS Subscribe sang QueueSubscribe cho transmute command
- [x] `P1-3`: Thêm log chi tiết khi rules bị filter
- [x] `P1-4`: Xử lý default value cho non-nullable rules khi missing field
- [x] `P1-5`: Fix DLQ write error swallow
- [ ] `P2-1`: Concurrency optimization (sẽ thực hiện ở phase sau)
- [ ] `P2-2`: Flatten orphan cleanup
- [ ] `P2-3`: Reconciliation tự động
- [ ] `P2-4`: Scheduler stuck cleanup
