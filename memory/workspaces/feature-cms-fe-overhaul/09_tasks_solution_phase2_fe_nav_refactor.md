# Solution — Phase 2 FE Nav Refactor

## Chốt giải pháp

Refactor FE theo hướng "bỏ concept sai khỏi UX trước, rồi mới thay data model sâu hơn":

1. Loại `CDCInternalRegistry` khỏi menu vì đây là legacy namespace không còn được phép là concept first-class.
2. Giữ `TableRegistry` tạm thời nhưng đổi cách gọi thành `Source Objects` để chuẩn bị chuyển semantics sang `source_object_registry`.
3. Đưa điều hướng về 3 cụm:
   - `Setup`
   - `Operate`
   - `Advanced`
4. Vá những text/operator hint còn lộ `cdc_internal`.
5. Không cố rewrite toàn bộ page data-flow trong cùng một phase để tránh blast radius quá lớn.

## Tác động

- Operator nhìn UI sẽ bớt hiểu nhầm rằng `cdc_internal` vẫn là control-plane chính.
- FE bắt đầu phản ánh đúng target architecture V2 ngay cả khi backend vẫn còn transitional endpoints.
- Phase sau có thể tập trung vào thay API/data model từng page thay vì còn phải dọn "tên sai" ở navigation.
