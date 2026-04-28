# Gap Analysis — Phase 9 Namespace Finalization

## Đã xử xong

- System table runtime path -> `cdc_system`
- Shadow naming -> `shadow_<source_db>`
- Legacy runtime dependency vào `cdc_internal` -> removed
- Final migration drop `cdc_internal` -> added

## Residual debt còn lại nhưng không block bootstrap

1. Migration history cũ vẫn ghi lại việc object từng được tạo ở `public` hoặc `cdc_internal`.
2. Một số comment/log text cũ có thể còn nhắc tên schema cũ nhưng không còn ảnh hưởng runtime.
3. `public` schema mặc định của PostgreSQL vẫn tồn tại ở mức DB engine; điều cần bảo đảm là app tables không còn nằm ở đó.

## Nếu muốn clean hơn nữa ở vòng sau

1. rewrite các migration lịch sử cũ để ngay từ đầu tạo object trong `cdc_system`
2. purge các helper legacy ở `public` nếu xác nhận không còn consumer
3. refactor CMS service để CRUD metadata V2 hoàn toàn, không mang semantics V1 trong UI/API
