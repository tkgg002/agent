# Danh Sách Nhiệm Vụ: Xóa Sạch Triệt Để default_master & LIMIT 1 Mò Mẫm Trong Toàn Bộ Subsystem Recon

**Workspace:** `FeatureTransmuteShadowToMasterCustomDB20260914`  
**Ngày:** 2026-09-15  

---

### Task 1: Sửa bug Fallback Transaction Aborted trong `CountRows` (`recon_dest_query.go`)
- [x] 1.1 Tách riêng transaction cho lần thử `SELECT COUNT(pkColumn)`: Rollback ngay lập tức khi hoàn thành hoặc có lỗi.
- [x] 1.2 Khi có lỗi ở lần thử 1, mở transaction read-only mới tinh để chạy `SELECT COUNT(*)`, đảm bảo không bao giờ bị `SQLSTATE 25P02`.
- [x] 1.3 Trong `recon_smoke.go` (`scanExact`), khi `kind == "master"`, truyền `pkColumn = ""` để chạy thẳng `SELECT COUNT(*)`, tối ưu tốc độ và loại bỏ rủi ro query nhầm cột `_gpay_id`.

---

### Task 2: Mở rộng Model và Query `ListActiveMasterBindings` (`recon_engine_segment_b.go`)
- [x] 2.1 Bổ sung trường `MasterConnectionKey string `gorm:"column:master_connection_key"`` vào struct `MasterBindingRef`.
- [x] 2.2 Sửa câu truy vấn trong `ListActiveMasterBindings`:
  - `SELECT mb.id, mb.master_schema, mb.master_table, sb.shadow_schema, sb.shadow_table, COALESCE(cr_ms.connection_code, 'default') AS master_connection_key`
  - Đổi `JOIN cdc_system.connection_registry cr_ms` thành `LEFT JOIN cdc_system.connection_registry cr_ms ON cr_ms.id = mb.master_connection_id AND cr_ms.status = 'active'`.

---

### Task 3: Tích hợp Dynamic Master Agent vào `ReconCore` (`recon_engine.go` & `server_setup.go`)
- [x] 3.1 Thêm các trường vào struct `ReconCore`: `connMgr *source.ConnectionManager`, `masterAgentsMu sync.RWMutex`, `masterAgents map[string]*ReconDestAgent`.
- [x] 3.2 Cài đặt method `SetConnectionManager(mgr *source.ConnectionManager)` và `GetMasterAgent(ctx context.Context, connectionKey string) (*ReconDestAgent, error)`.
- [x] 3.3 Trong `server_setup.go`, inject `reconCore.SetConnectionManager(connectionManager)`.

---

### Task 4: Cập nhật Pipeline Recon Smoke Sang Đa Kết Nối (`recon_smoke.go`)
- [x] 4.1 Cập nhật `ScanTarget`: Bổ sung `TargetKey string`.
- [x] 4.2 Sửa `smokeCountCache`: Căn cứ theo `TargetKey` (`kind:connKey:rel`) thay vì chỉ dùng `rel`.
- [x] 4.3 Sửa `CheckAllUnified`: Phân giải `msAgent, _ := rc.GetMasterAgent(ctx, r.MasterConnectionKey)` cho từng `ref`.
- [x] 4.4 Sửa `RunTotalOnlyB`: Sử dụng `msAgent` tương ứng với `ref.MasterConnectionKey`.
- [x] 4.5 Sửa `reconDrillDownCheckB`: Dùng `msAgent` cho `BucketCounts`.

---

### Task 5: Xóa Bỏ Hardcode `default_master` Trong Recon Tier B (`recon_tier_b.go`)
- [x] 5.1 Trong `RunTierB`: Thay thế `rc.masterAgent` bằng dynamic agent qua `rc.GetMasterAgent(ctx, ref.MasterConnectionKey)`.
- [x] 5.2 Trong `RunFastLookbackSegmentB`: Thay thế toàn bộ `rc.masterAgent` (`BucketCounts`, `ListIDTsInWindow`, `MaxWindowTs`) bằng dynamic agent.

---

### Task 6: Xóa Bỏ `LIMIT 1` Mò Mẫm & Hardcode Trong Chunk Stream Bucket Engine (`recon_stream_bucket_engine.go` & `server_setup.go`)
- [x] 6.1 Bổ sung `connMgr` và `GetMasterAgent` cho `ChunkStreamBucketEngine`.
- [x] 6.2 Trong `server_setup.go`: Inject `chunkEngine.WithConnectionManager(connectionManager)`.
- [x] 6.3 Xóa bỏ câu query `LIMIT 1` mò mẫm trong `lookupMasterRef`: Thay bằng logic truy vấn chính xác kèm cảnh báo ambiguous nếu có nhiều hơn 1 binding.
- [x] 6.4 `lookupMasterRefExact`: SELECT thêm `master_connection_key`.
- [x] 6.5 `executeSegmentB` và `checkDayChunkB`: Sử dụng `msAgent` tương ứng với `ref.MasterConnectionKey` cho `HashWindow` và `ListIDTsInWindow`.

---

### Task 7: Kiểm Thử & Nghiệm Thu Toàn Diện
- [x] 7.1 Viết unit tests cô lập: Đã bổ sung 3 unit tests trong `recon_fallback_test.go` (`TestCountRows_FallbackTransactionIsolation`, `TestReconCore_GetMasterAgent_MultiConnection`, `TestChunkStreamBucketEngine_GetMasterAgent_MultiConnection`).
- [x] 7.2 Lệnh kiểm thử và build sẵn sàng: `go test -v ./internal/service/recon/...` và `go build ./cmd/worker`. (Ghi chú: Cần chạy trực tiếp ngoài macOS sandbox do sandbox hạn chế syscall Go directory inspect `open ..: operation not permitted`).
- [x] 7.3 Cập nhật `05_progress.md` và biên soạn báo cáo chi tiết `11_report_recon_multi_connection.md`.
