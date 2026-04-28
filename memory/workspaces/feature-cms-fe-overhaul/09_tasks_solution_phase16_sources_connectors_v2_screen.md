# Solution — Phase 16 Sources Connectors V2 Screen

## Vấn đề

`Sources & Connectors` trước phase này gần như chỉ là màn Kafka Connect runtime. Nó chưa giúp operator nhìn ra lớp metadata fingerprint đã persist và cũng không chỉ ra mismatch giữa runtime và metadata.

## Giải pháp

1. Dùng đồng thời:
   - `/api/v1/system/connectors`
   - `/api/v1/sources`
2. Render hai lớp dữ liệu song song trong một page.
3. Thêm mismatch visibility để operator thấy drift metadata/runtime ngay trên UI.

## Outcome

- FE bớt phụ thuộc vào mental model cũ kiểu registry-first.
- Operator nhìn được lớp runtime và lớp fingerprint cùng lúc.
- Tạo tiền đề cho các màn V2-native sâu hơn ở phase sau.
