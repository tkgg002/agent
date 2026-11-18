---
tier: T2
suffix: scan_nested_id
status: CODE_AND_UNIT_VERIFIED
---
# Audit Report — scan_nested_id

## 1. Diff thực tế & trạng thái repo hiện tại (cập nhật 2026-11-17 10:00)
- **cdc-cms-service** (`HEAD = 5a8bac7`, đã commit lên `origin/recon-heal`): đã chứa file migration `migrations/schema/cdc_system_model/083_mapping_rule_master_source_field_text.sql` (9 dòng, `ALTER COLUMN source_field TYPE TEXT`).
- **centralized-data-service** (`wc -l`: `scan_field_filter.go` = 42 dòng, `scan_field_filter_test.go` = 49 dòng, `scan_service.go` = 759 dòng):
  - `internal/service/source/scan_field_filter.go:21-24` (+4/-2): đã gỡ `"_id"` và `"id"` khỏi `extraNestedScanSkipFields`, chỉ giữ `_parent_source_id` và `_array_index`.
  - `internal/service/source/scan_service.go:401-403` (+2/-1): nhánh shadow (ghi cột `TargetColumn = relPath` không có tiền tố) đã bổ sung `shouldSkipTopLevelScanField(relPath) || existingV2[relPath]` để bảo vệ PK gốc `id`/`_id`.
  - `internal/service/source/scan_field_filter_test.go:27-49` (+24/-0): đã thêm `TestScanFieldFilter_AllowsNestedIDWhileBlockingTopLevelIDAndExplodeCols` kiểm chứng đủ 3 nhóm (`id`/`_id`/các field `merchantInfo` được giữ lại ở nested; `_parent_source_id`/`_array_index` bị chặn ở nested; `id`/`_id`/`_debezium_meta` bị chặn ở top-level).
  - *(Lưu ý git working tree)*: Trong `centralized-data-service` hiện còn có các file chưa commit từ task khác (`mongo_introspection.go`, `discover_handler_mongo.go`, `discover_handler_utils.go`); khi commit anh có thể chỉ định 3 file `scan_field_filter.go`, `scan_field_filter_test.go`, `scan_service.go` nếu muốn tách riêng.

## 2. Phản biện từng dòng (Line-by-Line Review)
- `scan_field_filter.go:21-24`: xóa `"_id"` và `"id"` khỏi `extraNestedScanSkipFields`. Ở `scan_service.go:387-398` (nhánh có `masterBindingID`, bao gồm cả flatten object `merchantInfo` và array explode), `objTarget = strings.ReplaceAll(basePath+"."+relPath, ".", "_")` sinh ra `merchantInfo_id` nên tuyệt đối không đụng cột PK `id` hay 11 cột hệ thống Shadow (`IsReservedSystemColumn` vẫn được gọi ở dòng 34).
- `scan_service.go:401-403`: nhánh `else if binding.ID > 0 && !isArrayExplode && !strings.Contains(relPath, ".")` (khi không có `masterBindingID`) đặt `TargetColumn = relPath` không tiền tố. Nhờ `shouldSkipTopLevelScanField(relPath)` được chèn ngay đầu nhánh, `id`/`_id` và `_debezium_*` vẫn bị chặn y hệt hành vi cũ, không phát sinh lỗi hồi quy.
- Migration `083`: `VARCHAR(255) -> TEXT` là binary-coercible trên PostgreSQL (không rewrite bảng); grep toàn bộ `migrations/schema` không có `VIEW` nào phụ thuộc `mapping_rule_master.source_field`; không có GORM `size` tag giới hạn 255.

## 3. Bằng chứng Kiểm thử & Chống Suy Diễn Smoke Test / Mockup
- Biên lai `EV-20261117T095810-88b6` (`exit=0`): thực thi `go test -v ./internal/service/source/ -run TestScanFieldFilter -count=1` chạy cả `TestScanFieldFilter_SkipsReservedShadowSystemColumns` lẫn `TestScanFieldFilter_AllowsNestedIDWhileBlockingTopLevelIDAndExplodeCols`.
- Biên lai `EV-20261117T095939-29c9` (`exit=0`): thực thi toàn bộ suite `go test ./internal/service/source/ -run TestScan -count=1`.
- Biên lai `EV-20261117T095840-a9ec` (`exit=0`): thực thi biên dịch toàn repo `go build ./...` tại `centralized-data-service`.
- Phần đối chiếu trực tiếp trên môi trường DB dev (chạy migration `083` và khởi động lại binary để bấm Scan `merchantInfo` trên giao diện) cần phối hợp cùng anh để trace dữ liệu trong bảng `cdc_system.mapping_rule_master`.

## 4. Hướng dẫn Commit & Kiểm chứng
1. Stage & commit 3 file thuộc issue này tại `centralized-data-service`: `scan_field_filter.go`, `scan_field_filter_test.go`, `scan_service.go`.
2. Khởi động lại service `centralized-data-service` và thực hiện Scan Flatten lại `merchantInfo` -> đối chiếu sự xuất hiện của rule `target_column = merchantInfo_id` (`source_path = merchantInfo.id`).
3. Đảm bảo migration `083` đã được áp trên DB đích (`\d cdc_system.mapping_rule_master` -> `source_field | text`).

## 5. Security
- Không có secret, password, token hay PII trong toàn bộ diff. Tuân thủ Rule #15 (không tự chạy lệnh đẩy code lên remote).

## Mục 3: Concurrency / Timeout-Heartbeat / Data Continuity
- Luồng Scan (`ScanArrayFields`) là tác vụ đọc mẫu `_raw_data` (`LIMIT 5000`/`1000`) và ghi rule trong một DB transaction ngắn, không can thiệp vào luồng multi-worker NATS/Kafka consumer, `TableDebouncer`, `AckWait`/`Heartbeat` hay `Watermark`/`Cursor` của pipeline CDC. `INSERT` vào `mapping_rule_master` đã có sẵn `WHERE NOT EXISTS (master_binding_id, target_column)` (`scan_service.go:446-451`) chống trùng lặp khi gọi song song.
- Migration `083` đổi `VARCHAR -> TEXT` chỉ giữ `AccessExclusiveLock` metadata cực ngắn trên bảng cấu hình `cdc_system.mapping_rule_master`, không rewrite dữ liệu.

## Mục 4: Kiểm tra suy diễn & báo cáo láo
- Trung thực 100%: không cài đặt closure giả vào struct production, không lấy kết quả unit test (`EV-20261117T095810-88b6`, `EV-20261117T095939-29c9`) hay kết quả build (`EV-20261117T095840-a9ec`) để bịa rằng đã bấm nút trên giao diện thật; bước khởi động lại binary và bấm Scan cần phối hợp cùng anh để xác nhận giá trị trên DB.

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop)
- Phản tỉnh: (1) khi VegaML `:8222` bị nghẽn queue bởi tiến trình của workspace khác gây timeout, việc chuyển sang engine fallback Laya `:8200/v1/predict` giúp thông suốt cổng kiểm định mà vẫn giữ nguyên chuẩn `trusted_rounds`; (2) khi gỡ một trường khỏi danh sách skip dùng chung, bắt buộc kiểm tra đủ mọi nhánh tiêu thụ (có tiền tố `basePath` vs không tiền tố ở nhánh shadow).

## Mục 6: Kết quả QC & nhận xét từ /laya_router (DG3)
- Đã thực thi `laya_gate.py run --stage dg3` sau lần sửa code cuối trên `claims_dg3_final.json` (`A-1`, `A-2`, `A-3` + `K1/K2 control_false` + `K3 control_true`) kèm biên lai kiểm thử `EV-20261117T095810-88b6` và `EV-20261117T095939-29c9`.
- File `06_laya_summary_dg3_20261117T095934.json` ghi nhận cổng kiểm định đạt (`trusted_rounds = ["r2", "r3"]`) với cả 3 claim `A-1`, `A-2`, `A-3` đều đạt quyết định `ACCEPT` (confidence vòng `r2`/`r3` từ `0.7537` đến `0.9353`).
