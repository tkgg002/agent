# 08 Tasks: Audit Mất Dữ Liệu Source (trans-his) -> Shadow & Dừng Kafka Consumer

- [x] **Task 1**: Truy vết span Recon `cdc.recon.hash_window` và `cdc.recon.drift_drill_down`.
- [x] **Task 2**: Thu thập metrics Prometheus live và xác định 100% bản ghi `trans-his` bị drop.
- [x] **Task 3**: Phân tích nguyên nhân gốc rễ Over-filtering tại `IsGenericConnectionCode` và `ResolveSourceRoutes`.
- [x] **Task 4**: Phân tích nguyên nhân vì sao từ hôm qua tới nay không có `kafka-consume` nào chạy (`snapshot.mode: no_data` + Kafka Lag = 0).
- [x] **Task 5**: Phân tích nguyên nhân vì sao không có Auto-Heal và Alerting.
- [x] **Task 6**: Trình bày báo cáo chi tiết cho User và đã nhận User Approve (Rule #13 Brain Code Prohibition).
- [x] **Task 7**: Uỷ quyền cho Muscle thực thi sửa code Config-driven prefix và chống silent drop.
- [x] **Task 8**: Chạy `go test` xác nhận 23/23 unit tests pass 100%.
- [ ] **Task 9**: Kích hoạt Recon Heal hoặc Oplog Bridge để bù 212 bản ghi bị thiếu vào `shadow_traitestctphs.trans_his`.
