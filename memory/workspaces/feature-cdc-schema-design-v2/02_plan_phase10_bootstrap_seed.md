# Plan — Phase 10 Bootstrap Seed

## Kế hoạch

1. Đọc lại model/migration V2 để lấy đúng field và constraint.
2. Thiết kế một flow mẫu hoàn chỉnh:
   - source connection
   - shadow connection
   - master connection
   - source object
   - shadow binding
   - master binding
   - mapping rules
   - transmute schedule
3. Viết file SQL template riêng ngoài migration chain.
4. Tự rà lại các `ON CONFLICT`, unique key và naming convention.
5. Ghi workspace artifact cho phase này.
