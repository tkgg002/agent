# Review Report — `<suffix>`

- **Repo:** `<repo>` | **Range:** `<base>..<head>` (`<base_sha>..<head_sha>`)
- **Ngày:** `<YYYY-MM-DD>` | **Agent:** `<model>` | **Mode:** Propose-Only
- **Rules nạp:** L1 base(4) | L2 packs: `<...|none>` | L3 profile: `<path|none>`
- **Scope file:** `scope_<suffix>.json` (R0, 0 token)

## 1. Verdict
**`BLOCK` | `REQUEST_CHANGES` | `APPROVE`** — S0: `n` | S1: `n` | S2: `n` | S3: `n`
> 2–4 câu: vì sao ra verdict này, finding nào quyết định.

## 2. Scope (từ `scope.py`)
| Chỉ số | Giá trị |
| :--- | :--- |
| Files changed / added / deleted | `n / n / n` |
| LOC +/- | `+n / -n` |
| Layers touched | `handler, service, ...` |
| `chokepoints_touched` | `CP-1, CP-4` hoặc `none` |
| `god_files` (> `god_file_loc`) | `path (loc)` hoặc `none` |
| `money_path` files | `n` |
| `layer_violations` | `n` |

## 3. Findings (xếp theo severity, mỗi finding 1 khối)
### [S0] `<rule-id>` — <tiêu đề ngắn>
- **REF:** `<abs_path>:<start>-<end>`
- **Quote:**
  ```
  <đúng dòng code trích từ head>
  ```
- **Vì sao:** <1–3 câu, nối với rule>
- **Blast-radius:** <callers/consumers bị ảnh hưởng, `file:line`>
- **Gợi ý sửa:** <1–2 câu, không viết code vào repo>
- **Gate:** `laya dg3 = PASS|FLAG|UNVALIDATED` | `EV-...` (nếu có test chứng minh)

### [S1] ...

## 4. Blast-radius tổng hợp
| Chokepoint / Contract | Thay đổi trong diff | Callers/Consumers phải kiểm | Trạng thái |
| :--- | :--- | :--- | :--- |
| `CP-x` | ... | `file:line`, ... | OK / RISK |

## 5. Gate (R5)
- Mode B `laya_gate.py review`: `06_laya_summary_review_<ts>.json` → `gate=<...>`, rounds `<n>/3`.
- Mode A `laya_gate.py run --stage dg3`: `06_laya_summary_dg3_<ts>.json` → `gate=<...>`; controls: `true=n`, `false=n`.
- Dropped (BLOCK ở R5, REF sai sau 1 lần kiểm lại): `<rule-id> — lý do>` hoặc `none`.

## 6. Evidence
| EV | Kind | Lệnh | exit |
| :--- | :--- | :--- | :---: |
| `EV-...` | unit | `...` | 0 |

## 7. Không-finding đáng ghi (đã kiểm, OK)
- `<rule-id>`: đã kiểm `file:line`, không vi phạm.
