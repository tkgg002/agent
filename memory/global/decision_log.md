# Decision Log

> **Maintained by**: Brain (Antigravity)
> **Purpose**: Ghi lại các quyết định quan trọng của Brain — tại sao, từ context nào, kết quả ra sao.

---

## [2026-02-25] Chọn File-based Memory thay vì Vector DB

- **Context**: Upgrade Core System — cần chọn cơ chế memory cho hệ thống Brain/Muscle
- **Options Considered**:
  - [A] File-based markdown (`lessons.md`, `active_plans.md`...) — Simple, readable, no infra
  - [B] Vector DB (ChromaDB/Qdrant) — Semantic search, scalable, nhưng cần setup infra
- **Decision**: Chọn [A] File-based
- **Rationale**: Phù hợp với quy mô hiện tại. Vector DB là over-engineering cho 1 user + 1 Brain + vài workspaces. File-based đã đáp ứng đủ nhu cầu với tag-based filtering trong `lessons.md`
- **Outcome**: Hoạt động tốt — Brain đọc/ghi nhanh, không cần dependency ngoài

## [2026-02-25] Thực hiện toàn bộ P1+P2+P3 trong 1 phiên

- **Context**: User yêu cầu "upgrade core hoàn chỉnh nhất" trước GooPay Refactor
- **Decision**: Thực hiện tất cả P1→P3 trong cùng 1 phiên thay vì chia nhỏ
- **Rationale**: Không có dependency nào giữa các P. Không có risk cao. Hoàn thành 1 lần sẽ unblock workspace feature-refactor-2026 ngay
- **Outcome**: ✅ P1+P2 done. P3 đang thực hiện.
