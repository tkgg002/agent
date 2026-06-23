# Phân tích so khớp: Reconciliation Engine

## 1. Phương pháp thực hiện
- Trích xuất toàn bộ thân hàm (function body) của 23 hàm nghiệp vụ từ `recon_core.go` cũ (tại `data-hub-bf`) và so khớp line-by-line với các file mới trong `centralized-data-service/internal/service/recon/` (`recon_engine.go`, `recon_tier_a.go`, `recon_tier_b.go`).
- Sử dụng script phân tích cú pháp tĩnh để lọc ra sự khác biệt.

## 2. Kết quả so khớp danh sách hàm
Cả hai phiên bản đều chứa đúng **23 hàm nghiệp vụ**, không thừa không thiếu:
- **recon_engine.go**: `CheckAll`, `DiffIDsForTest`, `IsOffPeakForTest`, `MasterRel`, `NewReconCore`, `NewReconCoreWithConfig`, `ReapOrphanRunsFromDeadInstances`, `ReapStaleRuns`, `SetMasterAgent`, `SetMetadataRegistry`, `SetPlaneDBs`, `ShadowRel`, `TableGroupForTest`.
- **recon_tier_a.go**: `AcquireLeader`, `PruneAllOrphans`, `RunOrphanPrune`, `RunTier1`, `RunTier2`, `RunTier3`.
- **recon_tier_b.go**: `CheckAllSegmentB`, `RunRowDiffB`, `RunSegmentB`, `RunSegmentBFor`.

## 3. Chi tiết các thay đổi phát hiện
Phát hiện có 13 hàm có sự khác biệt về mặt text, cụ thể:

### 3.1 Thay đổi Package & Struct Name (Cải tiến kiến trúc)
Các hàm `RunTier2`, `RunOrphanPrune`, `PruneAllOrphans`, `RunTier3`, `CheckAll`, `RunSegmentB`, `RunSegmentBFor`, `CheckAllSegmentB` có sự thay đổi kiểu dữ liệu:
- Thay thế `model.TableRegistry` -> `source.TableRegistry`.
- Thay thế `model.ReconciliationReport` -> `system.ReconciliationReport`.
- Tại `CheckAll`: `rc.db.Model(&model.TableRegistry{})` -> `rc.db.Model(&source.TableRegistry{})`.

### 3.2 Thay đổi Namespace helper function (Cải tiến module hóa)
Tại hàm `RunRowDiffB`:
- Thay thế `IsTransformWhitelisted` -> `master.IsTransformWhitelisted`.
- Thay thế `ApplyTransform` -> `master.ApplyTransform`.

## 4. Kết luận
- **100% logic nghiệp vụ được bảo toàn**: Không có sự thay đổi hay thiếu sót nào về mặt giải thuật (watermark, adaptive freeze, chunking, logic block khi vượt blast radius, v.v.).
- Thay đổi chỉ tập trung vào việc chuẩn hóa namespace, phân rã package theo kiến trúc phân tầng (Domain-Driven Design) mới của service.
