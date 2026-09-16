# Kế Hoạch Triển Khai: Xóa Sạch Triệt Để default_master & LIMIT 1 Mò Mẫm Trong Toàn Bộ Subsystem Recon

**Workspace:** `FeatureTransmuteShadowToMasterCustomDB20260914`  
**Ngày:** 2026-09-15  
**Tác giả:** Brain (Architect)  

---

## 1. Mục Tiêu
Triển khai giải pháp kỹ thuật toàn diện nhằm:
- Khắc phục triệt để lỗi fallback transaction aborted trong `CountRows` (`recon_dest_query.go`).
- Xóa bỏ 100% việc trỏ cứng vào `default_master` (`RoleDestination`) trong toàn bộ subsystem Recon (Recon Smoke, Recon Tier B, Chunk Stream Bucket Engine, Recon Setup).
- Xóa bỏ câu query `LIMIT 1` mò mẫm trong `recon_stream_bucket_engine.go`.
- Định tuyến chính xác mọi luồng đọc/kiểm định đối soát Segment B tới Target Database Connection thực tế của bảng Master.

---

## 2. Kế Hoạch Triển Khai Chi Tiết

### Pha 1: Fix Cục Bộ `CountRows` & Tối Ưu `scanExact` (CDS Engine)
- Sửa `internal/service/recon/recon_dest_query.go`:
  - Rollback transaction thử nghiệm ngay lập tức nếu lỗi, mở transaction read-only mới tinh trước khi chạy fallback `SELECT COUNT(*)`. Triệt tiêu hoàn toàn `SQLSTATE 25P02`.
- Sửa `internal/service/recon/recon_smoke.go`:
  - Trong `scanExact`: Khi `kind == "master"`, truyền `pkCol = ""` để chạy thẳng `SELECT COUNT(*)`.

### Pha 2: Mở Rộng Master Binding Ref & Query Database (CDS Engine)
- Sửa `internal/service/recon/recon_engine_segment_b.go`:
  - Thêm trường `MasterConnectionKey string `gorm:"column:master_connection_key"`` vào `MasterBindingRef`.
  - Sửa `ListActiveMasterBindings`: `LEFT JOIN cdc_system.connection_registry cr_ms` và SELECT `COALESCE(cr_ms.connection_code, 'default') AS master_connection_key`.

### Pha 3: Bổ Sung Dynamic Master Agent Pool Vào `ReconCore` & `server_setup.go` (CDS Engine)
- Sửa `internal/service/recon/recon_engine.go`:
  - Thêm `connMgr`, `masterAgentsMu`, `masterAgents` vào `ReconCore`.
  - Cài đặt `SetConnectionManager` và `GetMasterAgent(ctx, key)`.
- Sửa `internal/server/server_setup.go`:
  - Inject `reconCore.SetConnectionManager(connectionManager)`.

### Pha 4: Định Tuyến Pipeline Recon Smoke Đa Kết Nối (CDS Engine)
- Sửa `internal/service/recon/recon_smoke.go`:
  - Bổ sung `TargetKey` cho `ScanTarget`.
  - Cập nhật `smokeCountCache` lưu theo `targetKey` (`kind:connKey:rel`).
  - Trong `CheckAllUnified`: Lấy `msAgent` theo `r.MasterConnectionKey`.
  - Cập nhật `RunTotalOnlyB` và `reconDrillDownCheckB` sử dụng `msAgent` tương ứng.

### Pha 5: Mở Rộng Recon Tier B Sang Đa Kết Nối (CDS Engine)
- Sửa `internal/service/recon/recon_tier_b.go`:
  - Trong `RunTierB` và `RunFastLookbackSegmentB`: Thay thế `rc.masterAgent` bằng `msAgent` phân giải động qua `rc.GetMasterAgent(ctx, ref.MasterConnectionKey)`.

### Pha 6: Chuẩn Hóa Chunk Stream Bucket Engine & Xóa Bỏ `LIMIT 1` Mò Mẫm (CDS Engine)
- Sửa `internal/service/recon/recon_stream_bucket_engine.go`:
  - Thêm `connMgr` và `GetMasterAgent`.
  - Xóa bỏ `ORDER BY ... LIMIT 1` mò mẫm trong `lookupMasterRef`.
  - `lookupMasterRefExact`: SELECT thêm `master_connection_key`.
  - `executeSegmentB` và `checkDayChunkB`: Dùng `msAgent` tương ứng.
- Sửa `internal/server/server_setup.go`:
  - Inject `chunkEngine.WithConnectionManager(connectionManager)`.

### Pha 7: Kiểm Thử & Nghiệm Thu
- Chạy unit tests: `go test ./internal/service/recon/...`.
- Biên dịch worker: `go build ./cmd/worker`.
- Báo cáo kết quả và append audit log vào `05_progress.md`.
