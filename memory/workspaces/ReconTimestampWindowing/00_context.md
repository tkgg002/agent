# Context: ReconTimestampWindowing

## Sự cố Timestamp Windowing
Trong quá trình đối soát Segment A (Source ↔ Shadow) bằng phương pháp quét các cửa sổ thời gian (time-windowing):
- Một record bị mất (miss) một bản cập nhật trên Shadow sẽ có timestamp cũ ở Shadow và timestamp mới ở Source.
- Khi quét window mới (chứa update mới ở Source): Bản ghi chỉ có ở Source, không có ở Shadow -> Báo **Fake Missing** (thực ra là update thiếu chứ không phải thiếu record).
- Khi quét window cũ (chứa update cũ ở Shadow): Bản ghi chỉ có ở Shadow, không có ở Source -> Báo **Fake Orphan** (thực ra là update thiếu chứ không phải mồ côi).
- Kết quả: Bản ghi bị xẻ làm 2, khiến CMS báo sai lệch thông tin và `StaleCount` đếm sót.

## Cơ chế an toàn chặn Heal bị ngắt
Trong `healSegmentA` của `recon_heal_v4.go`, nếu `StaleCount` (chỉ đếm `mismatchedFromDest` ban đầu) và `MissingCount` đều bằng 0, hệ thống coi là `noop` và không thực hiện re-trigger Debezium signal để heal. Do đó, việc xẻ sai bản ghi khiến `StaleCount` đếm sót và chặn đứng luồng tự chữa lành.

## Giải pháp khắc phục
1. Thêm luồng Post-Processing vào `RunTier2` của `recon_tier_a.go` để đối chiếu chéo (cross-check) danh sách `missingFromDest` trực tiếp với Shadow DB. Nếu bản ghi tồn tại ở Shadow DB thì định tuyến lại từ `missing` và `orphan` về đúng `mismatched` (stale update).
2. Cập nhật `StaleCount` trong report để gộp cả `mismatchedFromDest` và `missingFromSrc` (hoặc các phần mồ côi cần heal) để `healSegmentA` nhận diện chính xác trạng thái cần heal.
