# Technical Solutions

## Solution 1: Bỏ cascade active
Sử dụng python script để xóa block code từ dòng 305 đến 315 của `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go`.

## Solution 2: Sửa lateral join hiển thị sai tên bảng shadow
Sử dụng python script để thay thế đoạn SQL JOIN lateral trong `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go` bằng đoạn SQL JOIN trực tiếp dựa trên `sp.shadow_binding_id`.

## Solution 3: Sửa nạp chéo rules trong cache
Sử dụng python script để thêm kiểm tra `v2.ShadowBindingID != nil && *v2.ShadowBindingID != bindingID` trong `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go`.

## Solution 4: Sửa lệch parameters trong query ListMasterTablesByShadowIdentity
Sử dụng python script để thay thế danh sách parameters truyền vào ở cuối `ListMasterTablesByShadowIdentity` trong `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/master/master_binding_repo.go` để loại bỏ biến `shadowConnectionKey` bị dư thừa ở cuối.

