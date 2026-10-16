# Audit Log Tiến độ: Audit Số liệu c,u,d & Loại bỏ Read(r)

- [2026-10-06T17:35:00+07:00] [Brain:Claude-3.5] Khởi tạo workspace fix-recon-daily-flow-cud-2026-10-06, thiết lập 01_requirements, 05_progress.
- [2026-10-06T17:36:00+07:00] [Brain:Claude-3.5] Truy vấn PostgreSQL cdc_dw (bảng cdc_system.cdc_activity_log): Xác thực 100% 1,562 events Kafka consumer và 2,084 records Transmute ngày 2026-09-30 xuất phát từ 3 lần chạy snapshot (07:22, 07:25, 07:50) + 1 lần run-now manual trigger (07:29) + events stream.
- [2026-10-06T17:37:00+07:00] [Brain:Claude-3.5] Xác định nguyên nhân c,u,d bị lệch: Backend batch_buffer.go:365 không đếm c,u,d vào details; Frontend có fallback ép mù 100% batchSize vào c.
- [2026-10-06T17:38:00+07:00] [Muscle:Chief] Cập nhật ReconPipelineGrid.tsx: Xóa hoàn toàn Read(r) khỏi Box ngoài, Modal columns, totals calculation và Table.Summary.
- [2026-10-06T17:39:00+07:00] [Muscle:Chief] Chạy npx tsc --noEmit: PASS 100%, 0 lỗi.
- [2026-10-06T17:39:30+07:00] [Brain:Claude-3.5] Hoàn thành 09_tasks_solution_audit_cud_kafka_flow.md, 12_implementation_plan_audit_cud_kafka_flow.md và 13_analysis_audit_cud_kafka_flow.md.
- [2026-10-07T09:15:00+07:00] [Brain:Claude-3.5] Phân tích kiến trúc 2 BatchBuffer theo nhắc nhở của User: cdcBatchBuffer (isSnapshotBuffer=false, realtime streaming, concurrency 20, commit offsets) và snapshotBatchBuffer (isSnapshotBuffer=true, initial load, concurrency 4, không commit offset, ép Op='c').
- [2026-10-07T09:16:00+07:00] [Brain:Claude-3.5] Xác định tác động chéo giữa 2 buffer: Cả 2 đều dùng batchUpsert trong batch_buffer.go; snapshotBatchBuffer hiện tại cũng ghi operation='kafka-consumer', dẫn tới số liệu snapshot bị gom chung vào thống kê Kafka CDC bắn về.
- [2026-10-07T10:40:00+07:00] [Brain:Claude-3.5] User phê duyệt kế hoạch triển khai ("a duyệt"). Uỷ quyền Muscle tiến hành thực thi code Backend Go: cdc_event.go, event_handler.go, bridge_handler.go, batch_buffer.go.
- [2026-10-07T11:15:00+07:00] [Muscle:Chief] Sửa event_handler.go: Gán Op = event.Data.Op vào shadow.UpsertRecord.
- [2026-10-07T11:15:30+07:00] [Muscle:Chief] Sửa bridge_handler.go: Gán Op = event.Op vào shadow.UpsertRecord.
- [2026-10-07T11:16:00+07:00] [Muscle:Chief] Sửa batch_buffer.go: Đếm c, u, d từ records; phân định isSnapshotBuffer để gán triggered_by và buffer_type ('snapshot' vs 'cdc-stream'); bổ sung c, u, d, buffer_type, is_snapshot vào details của cdc_activity_log.
- [2026-10-07T11:17:45+07:00] [Muscle:Chief] Thêm test TestBatchBuffer_OpFieldHandling vào snapshot_cdc_isolation_test.go; chạy go test ./internal/handler/shadow/... và ./internal/handler/source/... đạt PASS 100%.
- [2026-10-07T11:17:55+07:00] [Muscle:Chief] Chạy npx tsc --noEmit ở cdc-cms-web: PASS 100%, 0 lỗi.
- [2026-10-07T11:18:30+07:00] [Brain:Claude-3.5] Hoàn thành DoD G1-G8, xuất báo cáo 11_report_audit_cud_kafka_flow.md và 14_walkthrough_audit_cud_kafka_flow.md.
- [2026-10-07T13:40:00+07:00] [Brain:Claude-3.5] Chạy quy trình QC gắt gao, đối soát psql thực tế và thẩm định qua Laya Router (/v1/predict); phát hiện lỗ hổng lọc snapshot ở Frontend và lập tài liệu audit_report_adversarial_cud_and_batch_buffer.md.
- [2026-10-07T13:49:56+07:00] [Brain:Claude-3.5] User phê duyệt kế hoạch triển khai Phase 4 ("thực hiện"). Uỷ quyền Muscle tiến hành cập nhật ReconPipelineGrid.tsx.
- [2026-10-07T13:50:11+07:00] [Muscle:Chief] Cập nhật ReconPipelineGrid.tsx: Phân định rõ ràng isSnapshot batch; tách biệt hoàn toàn giữa Snapshot Runner và Kafka Streaming.
- [2026-10-07T13:50:15+07:00] [Muscle:Chief] Chạy npx tsc --noEmit: PASS 100%, 0 lỗi.
- [2026-10-07T13:50:25+07:00] [Muscle:Chief] Chạy go test snapshot_cdc_isolation_test.go: PASS 100%.
- [2026-10-07T13:51:00+07:00] [Brain:Claude-3.5] Hoàn tất Phase 4, hoàn tất toàn bộ yêu cầu của User.
- [2026-10-07T14:45:00+07:00] [Brain:Claude-3.5] Tiến hành kiểm toán phản biện độc lập qua Laya Router v2 và dump dữ liệu 8 rows vật lý; phát hiện heuristic batchSize >= 500 là suy diễn (Rule #0); lập tức loại bỏ heuristic, thiết lập chuẩn Explicit Metadata Only và đề xuất SQL Data Sanitation Migration.
- [2026-10-07T14:50:27+07:00] [Brain:Claude-3.5] User phê duyệt triển khai SQL Data Sanitation Migration ("làm đi"). Uỷ quyền Muscle thực thi.
- [2026-10-07T14:52:42+07:00] [Muscle:Chief] Chạy SQL Migration UPDATE 6 bản ghi snapshot lịch sử (ID 100915, 100916, 100924, 100925, 101020, 101021) trong cdc_system.cdc_activity_log: bổ sung buffer_type='snapshot', is_snapshot=true, triggered_by='snapshot'.
- [2026-10-07T14:53:00+07:00] [Muscle:Chief] Đối soát trực tiếp psql: 6 bản ghi snapshot đã có đầy đủ cờ tường minh; 2 bản ghi stream giữ nguyên.
- [2026-10-07T14:59:50+07:00] [Muscle:Chief] Chạy npx tsc --noEmit: PASS 100%, 0 lỗi.
- [2026-10-07T14:59:59+07:00] [Muscle:Chief] Chạy go test snapshot_cdc_isolation_test.go: PASS 100%.
- [2026-10-07T15:00:30+07:00] [Brain:Claude-3.5] Hoàn thành trọn vẹn, lập biên bản audit_report_adversarial_cud_and_batch_buffer_v2.md, báo cáo kết quả hoàn tất cho User.
- [2026-10-07T16:00:00+07:00] [Brain:Claude-3.5] User thông báo sự cố nghiêm trọng: Sau khi User UPDATE 1 record thực tế trong ngày 2026-10-07, UI vẫn báo 3 Insert (3 | 3 | 0 | 0 | 3 | 3 | 0 | 0 | 3 | 0 | 0 | 0).
- [2026-10-07T16:15:00+07:00] [Brain:Claude-3.5] Thực hiện Root Cause Investigation không suy diễn:
  1. Kiểm tra PostgreSQL cdc_system.cdc_activity_log lúc 08:16:54 UTC: ID 107595 (kafka-consumer) ĐÃ GHI ĐÚNG details = '{"c": 0, "d": 0, "u": 1, "written": 1, "buffer_type": "cdc-stream", "is_snapshot": false}'; ID 107596 (transmute) ĐÃ GHI ĐÚNG details = '{"scanned": 1, "skipped": 0, "updated": 1, "inserted": 0}'.
  2. Kiểm tra shadow table shadow_traitestces.export_jobs: Record _id='6ac5c54caaad9d67d4844dbd' có _created_at='2026-10-07 04:06:41' và _updated_at='2026-10-07 08:16:54' -> ĐÃ UPDATE THẬT trên Database.
  3. Phát hiện Bug cốt lõi tại Frontend ReconPipelineGrid.tsx: dòng 414 và 474 dùng `typeof k.details === 'object' && k.details !== null ? k.details : {}`. Trong khi API wire contract (activity_log_read_models.go) trả về `details` là một JSON String (`*string`).
  4. Do `typeof string === 'object'` là FALSE -> `details` luôn bị fallback thành `{}` (object rỗng) -> Toàn bộ `c, u, d` và `inserted, updated` bị bỏ qua -> Frontend chạy vào fallback `row.kafka.c += written` và `row.shadow.c += written` -> Biến 100% 3 records thành 3 Insert và 0 Update! Master biến thành 0 inserted, 0 updated!
- [2026-10-07T16:20:00+07:00] [Brain:Claude-3.5] Lập giải pháp triệt để: Bổ sung helper parseLogDetails(raw) parse an toàn cả JSON String lẫn Object, sửa dứt điểm dòng 414 và 474; chuẩn hóa việc cộng tổng Kafka/Shadow/Master theo đúng số lượng op thật.
- [2026-10-07T16:21:00+07:00] [Brain:Claude-3.5] User phê duyệt kế hoạch triển khai Phase 6 ("anh duyệt, làm đàng hoàng cho anh"). Uỷ quyền Muscle thực thi toàn diện.
- [2026-10-07T16:24:00+07:00] [Muscle:Chief] Cập nhật ReconPipelineGrid.tsx:
  - Thêm helper parseLogDetails(raw) parse an toàn JSON String và Object.
  - Sửa dòng 414 và 474 để dùng parseLogDetails.
  - Chuẩn hóa batchSize = hasExplicitOps ? (c + u + del) : written (tránh lệch tổng Kafka).
  - Chuẩn hóa Master Transmute bóc tách inserted, updated, skipped từ details.
  - Dọn dẹp unused icon imports (SwapOutlined, WarningOutlined, CheckCircleOutlined).
- [2026-10-07T16:24:30+07:00] [Muscle:Chief] Chạy npm run build (tsc -b && vite build): PASS 100% trong 539ms (0 errors).
- [2026-10-07T16:24:55+07:00] [Muscle:Chief] Chạy go test snapshot_cdc_isolation_test.go: PASS 100%.
- [2026-10-07T16:25:00+07:00] [Muscle:Chief] Chạy simulation test trên dữ liệu thực tế ngày 2026-10-07: Kết quả hiển thị chính xác 100% 2 Insert, 1 Update ở cả 3 trạm (3 2 1 0 | 3 2 1 0 | 3 2 1 0).


