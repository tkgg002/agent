# 09_solution_flatten_discovery.md — Flatten: Scan → Review → Approve → Apply

> Yêu cầu User (2026-06-03): flatten KHÔNG auto-tạo toàn bộ. Phải **scan → hiện danh sách field → người duyệt → mới tạo cột vào master**. Giữ kiểm soát schema master.

## 1. Trấn an: worker flatten KHÔNG tự ý DDL master (đã verify code)
- `MasterDDLGenerator.Generate` **từ chối** nếu `schema_status != 'approved'` → `master_ddl_generator.go:69-71`.
- Transmuter chỉ nạp rule `status='approved' AND is_active=true` → `transmuter.go:293-294`.
- Flatten worker (vừa build) ghi qua `upsertMaster` vào **cột đã duyệt sẵn**; gọi `EnsureMaster` chỉ apply DDL cho binding approved. **Không bịa cột, không auto-create master.**
- → Bạn kiểm soát hoàn toàn table master. Auto-DDL (Finding 1) chỉ xảy ra ở **child SHADOW** (`child_explode.go` V3 MVP), KHÔNG phải master.

## 2. Luồng "scan → duyệt → apply" ĐÃ TỒN TẠI (cho field thường) — tái dùng 100%

| Bước | Cơ chế sẵn có | File |
|------|---------------|------|
| **Scan** raw_data | `POST /scan-raw-data` / `cdc.cmd.scan-raw-data` + `cdc.cmd.introspect` → worker quét `_raw_data` | `introspection_handler.go:244-350` |
| **Danh sách field đề xuất** | bảng `cdc_system.pending_fields`: field_name, sample_value, suggested_type, final_type(operator sửa), target_column_name, status='pending' | `model/pending_field.go` |
| **Preview giá trị** | `POST /api/v1/mapping-rules/preview` → eval JSONPath trên sample thật, xem extract ra gì TRƯỚC khi lưu | `mapping_preview_handler.go:24-114` |
| **Review/Approve** | pending → operator duyệt → tạo `mapping_rule_v2 (status='approved')`; có approve_schema_proposal | `approve_schema_proposal.go`, `mapping_rule_handler_*` |
| **Gate tạo master** | `MasterDDLGenerator.Generate` chỉ chạy khi `schema_status='approved'` | `master_ddl_generator.go:69` |
| **Re-scan = re-review** | re-scan tự reset rule về `status='pending'` để duyệt lại | `introspection_handler.go:335-346` |

→ Đây CHÍNH XÁC là mô hình bạn muốn. Không cần phát minh lại.

## 3. Phần THIẾU cho flatten (việc cần làm)
Scan hiện tại discover **field top-level** của `_raw_data`. Flatten cần discover field **bên trong array** tại `explode_path`:
1. **Array-element introspection** (MỚI): scan N sample row → đi tới `explode_path` (vd `after.items[*]`) → union keys của các phần tử → infer type → ghi vào `pending_fields` (scope theo flatten/child binding, đánh dấu là field-của-element).
2. Tái dùng **nguyên si**: pending_fields review UI → approve → mapping_rule_v2 (source_path tương đối element) → MasterDDLGenerator tạo cột (gate approved) → flatten worker chạy.

## 4. Luồng flatten end-to-end (control-plane)
```
1. Operator chọn master binding → transform_type=flatten, nhập explode_path
2. POST scan-array  ──▶  worker: sample N row, lấy elements tại explode_path,
                          union keys + infer type  ──▶  pending_fields(status=pending)
3. UI hiện DANH SÁCH field-của-element (sample, suggested_type)  ◀── operator
4. Operator tick chọn/sửa type/mask/nullable  ──▶  preview (tùy chọn)
5. APPROVE  ──▶  tạo mapping_rule_v2(status=approved, source_path tương đối element)
6. master_binding.schema_status=approved  ──▶  MasterDDLGenerator tạo cột master
7. flatten worker (đã build): explode array → N row → upsert cột đã duyệt
   ↑ KHÔNG bước nào auto; bước 5-6 là human gate.
```

## 5. Gợi ý nhất quán (tùy chọn)
- Đưa **child_explode auto-DDL (shadow)** về cùng gate `pending_fields` để nhất quán nguyên tắc "review trước khi tạo" (đồng thời fix Finding 1 SQLi data_type). Hiện child shadow tự CREATE/ALTER — trái nguyên tắc của bạn.

## 6. Việc đề xuất (phase tiếp)
- [FE] Wizard flatten: nhập explode_path → nút "Scan fields" → bảng review → approve.
- [BE-CMS] endpoint `scan-array` (hoặc mở rộng introspect nhận explode_path).
- [BE-Worker] introspect array elements tại explode_path → pending_fields.
- Tái dùng: pending_fields review/approve + MasterDDLGenerator + flatten worker (DONE).
