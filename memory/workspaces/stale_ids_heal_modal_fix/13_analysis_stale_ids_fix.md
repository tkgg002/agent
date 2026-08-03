# Technical Analysis: Root Cause & Solution for Stale IDs in ExecuteHealModal

## 1. Root Cause
Trong `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go`:
```go
type StaleIDsPayload struct {
	Mismatched      []string `json:"mismatched"`
	MissingFromSrc  []string `json:"missing_from_src"`
	MissingFromDest []string `json:"missing_from_dest"`
}
```
Backend Go chỉ xuất bản duy nhất struct này vào DB JSONB `stale_ids` cho cả 2 chặng:
- `segment = "source_shadow"` (Chặng A):
  - `missing_from_src`: ID thiếu ở Source (có ở Shadow).
  - `missing_from_dest`: ID thiếu ở Shadow (có ở Source).
  - `mismatched`: ID có data/hash lệch nhau giữa Source và Shadow.
- `segment = "shadow_master"` (Chặng B):
  - `missing_from_src`: ID thiếu ở Shadow (có ở Master).
  - `missing_from_dest`: ID thiếu ở Master (có ở Shadow).
  - `mismatched`: ID có data/hash lệch nhau giữa Shadow và Master.

Tuy nhiên, ở Frontend `cdc-cms-web/src/components/ExecuteHealModal.tsx`:
```typescript
if (record.segment === 'shadow_master') {
  if (parsedStale) {
    missingFromDest = (parsedStale.missing_from_shadow || []).map(String);
    missingFromSrc = (parsedStale.missing_from_master || []).map(String);
    mismatched = (parsedStale.mismatched || []).map(String);
  }
}
```
Frontend đọc `missing_from_shadow` và `missing_from_master` vốn **KHÔNG TỒN TẠI** trong JSON payload của backend. Dẫn tới danh sách `missingFromDest` và `missingFromSrc` ở Chặng B luôn bị `[]` (rỗng), khiến Popover chi tiết ID lệch bị thiếu thông tin trầm trọng.

## 2. Proposed Solution
1. **Thống nhất key bóc tách từ `stale_ids`**:
   Bất kể `segment === 'source_shadow'` hay `segment === 'shadow_master'`, luôn đọc:
   - `missingFromDest = (parsedStale.missing_from_dest || []).map(String)`
   - `missingFromSrc = (parsedStale.missing_from_src || []).map(String)`
   - `mismatched = (parsedStale.mismatched || []).map(String)`

2. **Gán nhãn hiển thị theo ngữ nghĩa của từng chặng**:
   - **Với `missingFromDest`**:
     - `segment === 'shadow_master'`: Label **"Thiếu ở Master (Missing from Dest)"**
     - `segment !== 'shadow_master'`: Label **"Thiếu ở Shadow (Missing from Dest)"**
   - **Với `missingFromSrc`**:
     - `segment === 'shadow_master'`: Label **"Thiếu ở Shadow (Missing from Src)"**
     - `segment !== 'shadow_master'`: Label **"Thiếu ở Source (Missing from Src)"**
   - **Với `mismatched`**:
     - Cả 2 chặng: Label **"Lệch dữ liệu (Mismatched)"**
