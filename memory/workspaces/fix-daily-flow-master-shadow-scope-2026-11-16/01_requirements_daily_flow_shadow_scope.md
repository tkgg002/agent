---
tier: T2
---
# Requirements — Daily Flow: khối Master bị gộp giữa 2 pipeline cùng tên bảng

## Triệu chứng (User)
Modal "Thống kê đối soát 3 trạm & DLQ theo ngày":
| Pipeline | Ngày | Kafka/Shadow Ops | Master Ops (dòng) | Master I/U |
|---|---|---|---|---|
| shadow_banvietbank_connector_service.bank_requests | 11-16 | 822 (626c/196u) | 1,764 (1,629) | 1,120/644 |
| shadow_bidv_connector_service.bank_requests | 11-16 | 942 (494c/448u) | 1,764 (1,629) | 1,120/644 |
| banvietbank | 11-15 | 372 | 1,532 | 935/597 |
| bidv | 11-15 | 1,160 | 1,532 | 935/597 |

Khối Master giống hệt nhau và = TỔNG 2 pipeline (822+942=1,764; 626+494=1,120; 196+448=644; 372+1,160=1,532) → Master bị gộp. Yêu cầu: tách rạch ròi theo từng shadow pipeline.

## Ràng buộc thực thi (từ lessons)
- L080: Master binding có thể `multi_shadow` (N shadow → 1 master) → không được giả định 1 master = 1 shadow.
- L087: đối soát đa chặng phải scope đúng cửa sổ/pipeline.
- L083/G5: không thêm concurrency mới trên cùng master binding (giữ concurrency=1 / tránh 40P01).
- L033/ADR-07: không hardcode tên bảng/schema.
- Propose-Only tới khi User APPROVE.

## DoD
- Daily Flow của 2 pipeline trả khối Master tách theo shadow_schema; log mới ghi `shadow_schema`/`shadow_table`.
- Test SQL thật (Postgres) với 2 shadow cùng tên bảng → master không gộp.
