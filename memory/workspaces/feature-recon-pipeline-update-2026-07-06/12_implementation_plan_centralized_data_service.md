# Kế hoạch triển khai - Refactor Tier sang TypeRecon trong Centralized Data Service

Kế hoạch này nhằm loại bỏ hoàn toàn các nợ kỹ thuật liên quan đến tên gọi `Tier` (RunTier1, RunTier2, RunTier3) trong core service `centralized-data-service`, chuyển đổi chúng sang các tên gọi nghiệp vụ tương ứng (`RunSmokeCheck`, `RunHashWindowCheck`, `RunDeepCheck`).

---

## 🔍 Phân tích hiện trạng các Callsite cần refactor

Dưới đây là các vị trí đang sử dụng các hàm `RunTier1`, `RunTier2`, `RunTier3` cần được đổi tên:

1. **`internal/service/recon/recon_tier_a.go`**:
   - `func (rc *ReconCore) RunTier1(...)` -> `func (rc *ReconCore) RunSmokeCheck(...)`
   - `func (rc *ReconCore) RunTier2(...)` -> `func (rc *ReconCore) RunHashWindowCheck(...)`
   - `func (rc *ReconCore) RunTier3(...)` -> `func (rc *ReconCore) RunDeepCheck(...)`
2. **`internal/handler/recon/recon_check_handler.go`**:
   - Dòng 226: `h.reconCore.RunTier3(...)` -> `h.reconCore.RunDeepCheck(...)`
   - Dòng 228: `h.reconCore.RunTier1(...)` -> `h.reconCore.RunSmokeCheck(...)`
   - Dòng 236: `h.reconCore.RunTier2(...)` -> `h.reconCore.RunHashWindowCheck(...)`
3. **`internal/handler/recon/recon_heal_handler.go`**:
   - Thay thế toàn bộ các lệnh gọi `RunTier2` thành `RunHashWindowCheck` (khoảng 4-5 vị trí trong luồng tự động chữa lành).
4. **`internal/service/recon/recon_engine_run.go`**:
   - Thay thế lệnh gọi `rc.RunTier1(...)` thành `rc.RunSmokeCheck(...)`.
5. **`internal/handler/recon/recon_heal_v4_test.go`**:
   - Cập nhật các mock và assert test liên quan đến `RunTier2` thành `RunHashWindowCheck`.
6. **`pkgs/metrics/prometheus.go`**:
   - Cập nhật các comment mô tả metric từ `RunTier1` -> `RunSmokeCheck`.

---

## 🛠️ Kế hoạch thực thi chi tiết (checklists)

- [ ] Bước 1: Sửa đổi file định nghĩa hàm `internal/service/recon/recon_tier_a.go`
- [ ] Bước 2: Sửa đổi handler gọi API `internal/handler/recon/recon_check_handler.go`
- [ ] Bước 3: Sửa đổi handler chữa lành `internal/handler/recon/recon_heal_handler.go`
- [ ] Bước 4: Sửa đổi scheduler engine `internal/service/recon/recon_engine_run.go`
- [ ] Bước 5: Cập nhật file unit test `internal/handler/recon/recon_heal_v4_test.go`
- [ ] Bước 6: Biên dịch thử nghiệm và chạy test suite của `centralized-data-service` (`go test ./...`)
