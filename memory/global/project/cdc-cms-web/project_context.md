# Project Context — cdc-cms-web

> **Last Updated**: 2026-11-09
> **Maintained by**: Brain (Antigravity)

## 📌 TÀI LIỆU TỔNG THỂ BẮT BUỘC ĐỌC TRƯỚC MỌI TASK
Trước khi phân tích, thiết kế hay sửa code trên `cdc-cms-web`, **BẮT BUỘC** đọc tài liệu kiến trúc & Design Pattern tổng thể của `data-hub` tại:

👉 **`/Users/trainguyen/Documents/work/agent/memory/global/project/data-hub/system_overview_and_design_pattern.md`**

- **Kiến trúc chuẩn của `cdc-cms-web`:** **Feature/Hook-Driven React Architecture** (`src/pages/<Page>.tsx` → `src/hooks/use<Domain>.ts` → `src/services/api.ts` + `src/components/<domain>/` + `src/types/index.ts`).
- **Danh mục đầy đủ 99 Actions & Hợp đồng DTO với `cdc-cms-service`:** Xem Mục 2.3 & Mục 3.3 trong `../data-hub/system_overview_and_design_pattern.md`.
