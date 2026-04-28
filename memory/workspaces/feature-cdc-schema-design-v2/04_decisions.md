# Decisions

## Decision 1

- Chọn `cdc_system` làm schema control plane mặc định cho V2.
- Lý do:
  - tránh nhầm `cdc_internal` là shadow payload schema
  - đọc tên là hiểu đây là metadata system

## Decision 2

- Không tiếp tục coi `cdc_table_registry` là trung tâm dữ liệu.
- Lý do:
  - bảng này đang trộn static metadata, routing, runtime state
  - không biểu diễn được multi-destination theo từng layer

## Decision 3

- Tách `source object` và `binding` thành hai lớp riêng.
- Lý do:
  - cùng một source object có thể đi tới nhiều master
  - shadow route và master route có lifecycle khác nhau

## Decision 4

- Dùng migration song song thay vì rewrite trực tiếp bảng cũ.
- Lý do:
  - hệ thống đang chạy thật
  - cần rollout an toàn và giữ backward compatibility

## Decision 5

- Runtime connection manager là bắt buộc, không phải optional improvement.
- Lý do:
  - nếu vẫn giữ một `db` duy nhất thì toàn bộ thiết kế multi-source/multi-destination chỉ là tài liệu trên giấy
