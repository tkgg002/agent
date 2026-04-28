# Requirements — Phase 5 Operational Pages Practical

## Mục tiêu

- Làm `ActivityManager` và `DataIntegrity` usable hơn cho operator thực tế.
- Không để các action vận hành chỉ bám vào `target_table` hiển thị trần.
- Dùng metadata hiện có để hiển thị:
  - source object
  - shadow target
  - scope thực thi

## Phạm vi

- `src/pages/ActivityManager.tsx`
- `src/pages/DataIntegrity.tsx`

## Điều kiện hoàn thành

1. Operator nhìn schedule biết nó áp vào source object / shadow target nào.
2. Operator nhìn recon / failed logs biết bảng đó thuộc source/shadow nào.
3. UI nói rõ contract legacy còn tồn tại ở backend thay vì che giấu.
4. Build FE pass.
