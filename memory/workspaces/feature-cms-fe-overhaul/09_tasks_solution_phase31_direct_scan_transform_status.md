# Phase 31 Solution — Direct Scan Fields & Transform Status

## Giải pháp chốt
- Direct-V2 hóa những phần worker contract đã đủ mềm.
- Không cố direct-V2 hóa `create-default-columns` khi worker path chưa đủ dữ liệu.

## Tác động thực chiến
- Row `V2-only` trong `TableRegistry` có thể:
  - xem `transform-status`
  - chạy `scan-fields`
  mà không cần bridge cũ.

## Chưa làm
- `create-default-columns`
- `standardize`
- các action còn buộc metadata legacy sâu hơn
