# Báo cáo Sửa đổi Logic Timestamp Đối soát Segment B

Báo cáo chi tiết về thay đổi mã nguồn trong centralized-data-service để sửa đổi logic đối soát Segment B (Shadow ↔ Master) sử dụng timestamp nghiệp vụ thay cho timestamp kĩ thuật.

---

## 1. Các file đã thay đổi
- `internal/service/recon/recon_tier_b.go`

---

## 2. Số lượng dòng code thay đổi
- Khoảng 15 dòng code được sửa đổi và tối ưu hóa.

---

## 3. Chi tiết thay đổi (Overview)

### A. Gán `ShadowSchema` cho registry entry
Trước đây, hàm `measureAndResolveWatermarksB` và `TimeBoundedDiffMissingFromMaster` thực hiện tìm registry entry bằng target table, sau đó trực tiếp gọi `resolveSourceAndDestTSFields(ctx, *entry)`. Tuy nhiên, registry entry lúc này bị thiếu schema thông tin của Shadow (`ShadowSchema`), khiến việc qualified target table trên Shadow DB không chính xác.
- **Giải pháp:** Gán `entry.ShadowSchema = ref.ShadowSchema` trước khi gọi `resolveSourceAndDestTSFields`.

### B. Tối ưu hóa so sánh ID trong `TimeBoundedDiffMissingFromMaster`
Hồ sơ giải pháp ban đầu đề xuất gán 3 biến trả về từ `diffIDs` bao gồm cả `staleIDs`. Tuy nhiên, hàm `diffIDs` trong codebase thực tế chỉ trả về 2 biến (`missing, orphan`). Để giải quyết lỗi compile này (như bài học số 12 đã cảnh báo):
- **Giải pháp:** Sử dụng signature đúng `missingFromMaster, _ := diffIDs(shadowIDs, masterIDs)` và trả về đúng kiểu của hàm gốc (`missingFromMaster, len(shadowIDs), len(masterIDs), nil`). Điều này vừa tối ưu hóa hiệu năng so sánh bằng map vừa đảm bảo không lỗi biên dịch.

---

## 4. Kết quả Kiểm thử & Xác minh

1. **Biên dịch:** Chạy `go build ./cmd/...` thành công không có lỗi cảnh báo.
2. **Unit Tests:** Chạy `go test -v ./internal/service/recon/...` thành công, đạt kết quả **PASS 100%** (cached & fresh test runs đều pass).
