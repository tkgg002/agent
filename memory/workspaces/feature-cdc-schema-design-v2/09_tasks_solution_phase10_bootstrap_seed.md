# Solution — Phase 10 Bootstrap Seed

## Output

Đã tạo file:

- [bootstrap_cdc_system_v2_template.sql](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/sql/bootstrap_cdc_system_v2_template.sql)

## Cách dùng

1. Chạy migrations tới `038`.
2. Copy file template này thành file riêng cho môi trường.
3. Sửa:
   - `connection_code`
   - `host/port`
   - `default_database/default_schema`
   - `secret_ref`
   - source db/schema/object thật
   - master schema/table thật
   - mapping rules thật
4. Chạy file seed trên DB system.
5. Start service.

## Lợi ích

- Không còn phụ thuộc trí nhớ tay khi bootstrap lại.
- Có một flow mẫu end-to-end bám đúng V2.
- Giảm rủi ro seed sai namespace trong đợt wipe lớn.
