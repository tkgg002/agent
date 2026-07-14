# Danh sách Task chi tiết - Tracing Reconciliation Detail

- [x] `[Phase 1: Rà soát & Thiết kế]` Rà soát lại tất cả các method gọi từ CheckHandler xuống ReconCore và các database agents để tìm các chỗ thiếu tracing span hoặc mất context propagation.
- [x] `[Phase 2: Triển khai Tracing cho ReconCore]`
  - [x] Bổ sung/cải thiện các child span trong `recon_tier_a.go` (`RunDeepCheck`, `RunHashWindowCheck`).
  - [x] Bổ sung/cải thiện các child span trong `recon_tier_b.go` (`RunDeepCheckB`, `RunHashWindowCheckB`).
- [x] `[Phase 3: Triển khai Tracing cho Database Agents]`
  - [x] Thêm child spans cho `ReconSourceAgent` (`CountDocuments`, `HashWindow`, `BucketHash`, `ListIDTsInWindow`, `ListIDsInWindow`, `ListAllIDs`, `StreamAllIDs`, `listIDsInWindowPostgres`, `streamAllIDsPostgres`, `ListIDTsInWindow`, `listIDTsInWindowPostgres`, `StreamIDsInTimeRange`, `streamIDsPostgresInTimeRange`).
  - [x] Thêm child spans cho `ReconDestAgent` (`CountRows`, `CountDeletedRows`, `EstimatedCountRows`, `CountInWindow`, `BucketCounts`, `ListIDTsInWindow`, `MaxWindowTs`, `HashWindow`, `BucketHash`).
- [x] `[Phase 4: Sửa lỗi lock timeout & Tối ưu hóa Index]`
  - [x] Thêm map `ensuredMasters map[string]bool` vào `TransmuterModule` struct và khởi tạo trong `NewTransmuterModule`.
  - [x] Cập nhật `TransmuterModule.Run` để kiểm tra cache trước khi chạy `EnsureMaster`.
  - [x] Cập nhật `TransmuterModule.InvalidateRuleCache` để xóa cache cho master table khi rules thay đổi.
  - [x] Cập nhật `EnsureCDCColumnsInSchema` trong `schema_adapter.go` để tự động tạo partial index cho `_deleted`.
  - [x] Cập nhật `Generate` trong `master_ddl_generator.go` để tự động tạo partial index cho `_deleted` và index cho timestamp nghiệp vụ.
- [x] `[Phase 5: Kiểm thử & Xác minh]` Chạy unit tests để đảm bảo code compile thành công và tracing + caching + index logic hoạt động đúng đắn.
- [x] `[Phase 6: Triển khai Smart Tracing & Chống Span Storm]`
  - [x] Khai báo context key `skipWindowTraceKey` để bypass việc tạo child span cho các window sạch.
  - [x] Cập nhật `RunHashWindowCheck` và `RunHashWindowCheckB` để inject cờ bypass vào context của vòng lặp window.
  - [x] Cập nhật `HashWindow` trong `ReconSourceAgent` và `ReconDestAgent` để kiểm tra cờ bypass trước khi tạo span con.
- [x] `[Phase 7: Kiểm thử & Xác minh hiệu năng]`
  - [x] Đảm bảo code compile thành công và chạy thử test suite để xác minh không bị ảnh hưởng đến logic đối soát.

- [x] `[Phase 8: Sửa lỗi Segment B (Báo cáo & Chữa lành)]`
  - [x] Cập nhật `internal/service/recon/recon_tier_b.go`: đổi CheckType thành `"hash_window"`/`"bucket_hash"`, sửa stale JSON format, gán `SourceDB: ""`, cộng thêm orphan vào StaleCount.
  - [x] Cập nhật `internal/service/recon/recon_engine_segment_b.go`: gán trống trường Source trong `stampB`.
  - [x] Cập nhật `internal/handler/recon/recon_base_handler.go`: đổi struct `staleSegmentB` và hàm parse tương ứng.
  - [x] Cập nhật `internal/handler/recon/recon_execute_heal_handler.go`: lấy shadow FQN từ schema+table nếu segment là shadow_master, đổi sang mismatched/missing_from_src.
  - [x] Cập nhật `internal/handler/recon/recon_check_heal_handler.go`: sửa gọi `.Mismatched` và `.MissingFromSrc`.
  - [x] Rà soát tĩnh codebase đảm bảo sạch cấu trúc cũ 100%.
  - [x] Thực hiện sửa đổi thành công và chuẩn bị biên dịch/unit test.

- [x] `[Phase 9: Sửa đổi logic phân dải timestamp đối soát Segment B sang cột nghiệp vụ]`
  - [x] Chỉnh sửa file `internal/service/recon/recon_tier_b.go` để cập nhật `measureAndResolveWatermarksB`, `RunHashWindowCheckB`, `RunDeepCheckB`, và `TimeBoundedDiffMissingFromMaster`.
  - [x] Biên dịch và chạy unit tests thành công pass 100%.

- [x] `[Phase 10: Sửa đổi logic Shadow Schema & Bổ sung Activity Log cho SinkWorker]`
  - [x] Sửa đổi `internal/sinkworker/worker.go` để query shadow target từ DB (không fallback) và ghi nhận activity log.
  - [x] Sửa đổi các unit test cũ bị fail do format SQL và cấu trúc mock data.
  - [x] Biên dịch và chạy thành công test suite pass 100%.

