# 09_solution_pattern.md — System Design: Transform Strategy Registry

> Ground: `transmuter.go` (đã đọc full), `transform_registry.go` (cell-level registry mẫu), `child_explode.go` (`extractArrayByPath` — logic flatten).
> Nguyên tắc: **mirror pattern registry sẵn có** (minimal impact, demand elegance). Engine KHÔNG sửa khi thêm type mới.

---

## 1. Nguyên lý phân tách (altitude)

| Thành phần | Vai trò | Ai sửa khi thêm type? |
|------------|---------|------------------------|
| **Engine** (`TransmuterModule`) | paging shadow, gate, OCC upsert, runtime state, hash | ❌ KHÔNG bao giờ |
| **Registry** (`strategy_registry.go`) | map `transform_type → Strategy` | ❌ KHÔNG (chỉ `init()` tự đăng ký) |
| **Strategy file** (`transmute/<type>.go`) | logic transform thuần (no I/O) | ✅ copy 1 file mới |
| **Spec** (`master_binding.transform_spec` JSONB) | cấu hình per-type do operator khai báo | ✅ khai báo, không code |

→ "Thêm loại mới = copy 1 file + 1 dòng register + khai báo type" đạt được vì engine dispatch động.

---

## 2. Source structure contract (system design)

Luồng dữ liệu & hợp đồng giữa các tầng (đã tồn tại, ta chuẩn hoá lại):

```
source_object_registry  (cái gì cần CDC)
  └─ shadow_binding      (đổ raw về đâu: _raw_data JSONB + system cols)
       └─ mapping_rule_v2 (source_path → target_column, data_type, transform_fn, mask)
            └─ master_binding (transform_type + transform_spec → chiếu lên master)
                 └─ transmute_schedule (khi nào chạy: cron/immediate/post_ingest)
```

**Hợp đồng 1 Strategy nhận / trả** (thuần, không chạm DB):

```
INPUT  (RunContext + ShadowRow):
  - Raw      []byte            // _raw_data (JSONB của 1 shadow row)
  - SourceID string            // _source_id gốc
  - SourceTs int64, Deleted bool
  - Rules    []MappingRule     // source_path, target_column, data_type, transform_fn
  - Spec     TransformSpec     // parse từ master_binding.transform_spec (per-type)
  - helpers  ApplyTransform(), TypeResolver, Masking   // tái dùng, KHÔNG tự viết lại

OUTPUT ([]Emit):
  - Cols       map[string]any  // chỉ cột nghiệp vụ
  - KeySuffix  string          // "" cho 1:1; "#<idx>" cho flatten (giải fan-out _source_id)
```

Engine tự gắn system cols (`_gpay_id, _source_id(+suffix), _source, _source_ts, _synced_at, _hash, _deleted, _version`) và OCC-upsert. Strategy **chỉ lo transform**.

---

## 3. Interface + Registry (mirror `transform_registry.go`)

```go
// internal/service/transmute/strategy.go
package transmute

// Strategy = 1 loại sync (master_binding.transform_type). Thuần, không I/O.
type Strategy interface {
    Type() string                                  // "copy_1_to_1", "flatten", ...
    ValidateSpec(spec []byte) error                // CMS approve-time validate transform_spec
    BuildEmits(rc RunContext, row ShadowRow) ([]Emit, error) // 0..N record từ 1 shadow row
}

type Emit struct {
    Cols      map[string]any
    KeySuffix string // "" | "#0" | "#1"... cho fan-out
}

// --- registry: y hệt style transformRegistry ---
var registry = map[string]Strategy{}

func Register(s Strategy)            { registry[s.Type()] = s }
func Get(t string) (Strategy, bool) { s, ok := registry[t]; if !ok { s, ok = registry["copy_1_to_1"] }; return s, ok }
func IsWhitelisted(t string) bool   { _, ok := registry[t]; return ok || t == "" }
func List() []string                { /* sorted keys cho UI dropdown */ }
```

## 4. File-per-type — "copy 1 file"

### `transmute/copy_1_to_1.go` (tách từ `buildMasterRow` hiện tại, KHÔNG đổi hành vi)
```go
func init() { Register(copyOneToOne{}) }
type copyOneToOne struct{}
func (copyOneToOne) Type() string          { return "copy_1_to_1" }
func (copyOneToOne) ValidateSpec(b []byte) error { return nil } // không cần spec
func (copyOneToOne) BuildEmits(rc RunContext, row ShadowRow) ([]Emit, error) {
    cols, ok := rc.ExtractColumns(row.Raw, rc.Rules) // = logic buildMasterRow cũ
    if !ok { return nil, nil }                        // miss non-nullable → skip
    return []Emit{{Cols: cols}}, nil                  // đúng 1 record
}
```

### `transmute/flatten.go` (mới — promote `extractArrayByPath`)
```go
func init() { Register(flatten{}) }
type flatten struct{}
type flattenSpec struct { ExplodePath string `json:"explode_path"` } // vd "after.items[*]"
func (flatten) Type() string { return "flatten" }
func (flatten) ValidateSpec(b []byte) error { /* explode_path bắt buộc, path hợp lệ */ }
func (flatten) BuildEmits(rc RunContext, row ShadowRow) ([]Emit, error) {
    var sp flattenSpec; _ = json.Unmarshal(rc.Spec, &sp)
    elems := extractArrayByPath(rc.ToMap(row.Raw), sp.ExplodePath) // tái dùng helper
    out := make([]Emit, 0, len(elems))
    for idx, el := range elems {
        cols, ok := rc.ExtractColumnsFromElement(el, rc.Rules) // rule chiếu trên element
        if !ok { continue }
        out = append(out, Emit{Cols: cols, KeySuffix: fmt.Sprintf("#%d", idx)})
    }
    return out, nil // N record từ 1 shadow row
}
```

> **2 nghĩa "trải phẳng" — phân biệt rõ:**
> - **Object → cột độc lập** (vd `payment.fee` → cột `fee`): ĐÃ làm được bằng `copy_1_to_1` với `source_path` dạng dot. KHÔNG cần code mới.
> - **Array → row độc lập** (vd `items[]` → N row): chính là strategy `flatten` mới ở trên.

## 5. Dispatch — sửa DUY NHẤT trong `processBatch`

```go
// transmuter.go processBatch — thay khối buildMasterRow bằng:
strat, _ := transmute.Get(binding.TransformType)        // fallback copy_1_to_1
emits, err := strat.BuildEmits(rc, toShadowRow(row))
if err != nil { out.skipped++; continue }
for _, e := range emits {
    rec := e.Cols
    addSystemCols(rec, row, e.KeySuffix)  // _source_id = row.SourceID + e.KeySuffix
    if _, err := t.upsertMaster(ctx, binding, rec); err != nil { out.skipped++; continue }
    // count inserted/updated
}
```
`binding.TransformType` đã có sẵn trong `masterBindingRuntime` (transmuter.go:73) — hiện đang bị bỏ qua.

## 6. Ràng buộc thiết kế PHẢI xử lý (đã phát hiện từ code)

1. **`_source_id` fan-out**: master upsert `ON CONFLICT (_source_id)`. Flatten emit N row cùng `_source_id` → bị gộp còn 1. → Giải bằng `KeySuffix` (`_source_id = orig + "#idx"`), giống child_explode dùng `(_parent_source_id, _array_index)`.
2. **Delete-orphan khi array co lại**: update document làm array ngắn đi → row thừa ở master phải bị soft-delete. (child_explode dùng DELETE+INSERT per parent; ở master nên mark `_deleted` các suffix không còn — ghi nhận cho phase impl.)
3. **Hash-dedup**: vẫn đúng per emitted row (hash theo cột nghiệp vụ).
4. **Whitelist an toàn**: CMS validate `transform_type ∈ List()` + `ValidateSpec` trước approve — đúng triết lý "typo không thành code execution" của registry hiện tại.
5. **Backward-compat**: master_binding cũ `transform_type='copy_1_to_1'` chạy y nguyên (strategy tách ra phải giữ logic buildMasterRow).

## 7. Phạm vi phase đầu (framework + 2 type)
- [P-FW] tạo package `internal/service/transmute/`: `strategy.go` (interface+registry), `runcontext.go` (helpers bọc ExtractColumns/ApplyTransform/TypeResolver/Masking), `copy_1_to_1.go`.
- [P-FW] refactor `processBatch` → dispatch; giữ `copy_1_to_1` hành vi cũ (regression test).
- [P-FLAT] `flatten.go` + promote `extractArrayByPath` thành helper dùng chung + xử lý `_source_id` suffix + orphan soft-delete.
- [DOC] `transmute/README.md`: "Cách thêm 1 loại sync mới" (copy file → implement → init register → thêm enum).
- Option sau (mỗi cái 1 file): `filter.go` (predicate row-skip), `aggregate.go`/`group_by.go`/`join.go` (cân nhắc emit SQL hoặc đẩy xuống mart — quyết định khi tới).
