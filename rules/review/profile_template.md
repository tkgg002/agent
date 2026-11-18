# L3 Review Profile — `<project_name>` (template, ≤ 8KB)

> Copy sang `agent/memory/global/project/<project_name>/review_profile.md` rồi điền. Trỏ từ `<repo>/.agent-project.json` → `review.profile`. Chỉ file này (và manifest) được chứa tên service / path / symbol của dự án; L1/L2 phải giữ agnostic (ADR-07).
> Mục nào không có → ghi `none`, không xoá heading (lens parse theo heading).

## 1. Repos & kiến trúc
| Repo | Kiến trúc | Layer order (ngoài → trong) | Prefix path từng layer |
| :--- | :--- | :--- | :--- |
| `<repo-a>` | `<Clean Layered / Hexagonal+CQRS / Feature-hook React>` | `a > b > c > d` | `a: internal/a/`, ... |

## 2. Invariants (nâng severity khi vi phạm)
- `INV-1 (S0)` — <ví dụ: cấm sửa chữ ký hàm/trường struct đang phục vụ luồng cũ; chỉ mở rộng qua method/option mới>
- `INV-2 (S1)` — ...

## 3. Shared chokepoints (CP) & callers phải bảo toàn
| CP | Symbol / path | Callers (`file:line`) | Bất biến |
| :--- | :--- | :--- | :--- |
| `CP-1` | `pkg.Func` (`path.go:line`) | `caller1.go:line`, ... | <điều kiện không được phá> |

## 4. Contracts liên service
- **Events/subjects:** `<stream/subject>` → producer `<repo>` / consumers `<repo>`; trường bắt buộc: `...`
- **Bảng dùng chung & write-owner:** `<schema.table>` → owner `<repo>`; repo khác chỉ đọc/cập nhật cột `...`
- **HTTP API public:** `<METHOD /path>` → consumer `<...>`

## 5. God-files & nơi CẤM nhồi `if/else`
| File | LOC hiện tại | Khi thêm loại mới phải đặt ở | Interface phải implement |
| :--- | :---: | :--- | :--- |
| `<path>` | `n` | `<path pattern>` | `<Interface>` |

## 6. Money-path / dữ liệu nhạy cảm đặc thù
- Keywords bổ sung: `...`
- Bảng/collection chứa PII: `...` (mask rule: `...`)

## 7. Lệnh kiểm nhanh đặc thù (0 token)
```bash
# ví dụ: grep callers của 1 chokepoint
grep -rn "ResolveX(" internal/ --include=*.go
```
