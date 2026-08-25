# Task List: Fix Lỗi Flatten Mongo Extended-JSON (_id.$oid)

- [ ] Task 1: Bổ sung nhận diện Mongo ExtJSON trong `flattenJSONWithTypes` (`scan_service.go` & `scan_handler.go`) để dừng loop đệ quy khi gặp `$oid`, `$date`, v.v.
- [ ] Task 2: Tích hợp `unwrapMongoExtJSON` vào `MapColumnsFromElement` trong `child_explode.go` và `extractColumns` trong `transmuter.go`.
- [ ] Task 3: Bổ sung Unit Tests kiểm thử luồng flatten với mảng chứa `"_id": {"$oid": "..."}` và `"$date"`.
- [ ] Task 4: Chạy `go test` kiểm định toàn bộ package transmute, shadow, và source.
