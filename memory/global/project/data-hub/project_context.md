# Project Context — data-hub (Master Index)

> **Last Updated**: 2026-11-09
> **Maintained by**: Brain (Antigravity)

## 📌 BẮT BUỘC ĐỌC TRƯỚC MỌI TASK TRÊN DATA-HUB
Trước khi phân tích, thiết kế hay sửa bất kỳ dòng code nào trong hệ thống `data-hub` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`, `cdc-auth-service`), **BẮT BUỘC** đọc tài liệu tổng thể tại:

👉 **`/Users/trainguyen/Documents/work/agent/memory/global/project/data-hub/system_overview_and_design_pattern.md`**

### Nội dung cốt lõi trong tài liệu tổng thể:
1. **Mục 0 & 1 — Nguyên tắc Sống Còn & Tổng quan Hệ thống Đang Chạy (Live Production Baseline):**
   - Hệ thống đang chạy thực tế ổn định; tuyệt đối không đánh đồng lỗi môi trường test local hay thư mục lưu trữ (`docs/*_bk*`, `scratch/`) với lỗi hệ thống.
   - Giữ nguyên cấu trúc thư mục hiện hữu; mở rộng theo nguyên tắc **Open-Closed (100% Backward Compatibility)**.
2. **Mục 2 — Canonical Design Pattern của từng Service:**
   - `centralized-data-service`: **Clean Layered Architecture** (`server` → `handler` → `service` → `repository` → `model`) × 7 Functional Groups (`base`, `source`, `shadow`, `master`, `recon`, `governance`, `orchestration`).
   - `cdc-cms-service`: **Hexagonal + CQRS + Screaming Architecture** (`api` → `app/commands|queries` → `domain` & `ports` ← `infra/persistence|messaging|http`) × 7 Functional Groups (`source`, `shadow`, `master`, `governance`, `recon`, `scheduler`, `system`).
   - `cdc-cms-web`: **Feature/Hook-Driven React Architecture** (`pages` → `hooks` → `services/api.ts` + `components/<domain>`).
3. **Mục 3 — Bản đồ 5 Shared Chokepoints (`CP-1..CP-5`), Ma trận Đa hình 3 Chiều, Danh mục 99 Domain Actions ("Đủ chưa?") & Bản đồ Bảng/NATS:**
   - 5 Chokepoints dùng chung (`ResolveSourceRoutes`, `EventHandler.HandleRaw`, `BatchBuffer`, `TransmuteHandler/TransmuterModule`, `ReconCore/ExecuteHeal`).
   - Toàn bộ **99 Domain Actions** trên 7 miền (`Connector`, `Source`, `Shadow`, `Master`, `Recon & Heal`, `Governance`, `Scheduler/Wizard/System`) kèm đánh giá độ phủ đa Engine (`Mongo`, `Postgres`, `MySQL`, `ClickHouse`, `SFTP`).
4. **Mục 4 — Chi tiết Code-Level Design Patterns & Go Interface Contracts cho 4 Trục Đa Hình:**
   - Quy luật: **`1 Loại Mới = 1 File Thực Thi Mới + init() Register(...) = 0 câu lệnh if/else mới trong Core Engine`**.
   - Bao phủ 100% 99 Actions qua: `transmute.Strategy` (+ `DeletePolicyProvider`, `BatchPreEnricher`, `PostIncrementalCatchupHook`), `MasterSinkDriver`, `MasterDDLDriver` (đủ `Create`, `Alter`, `DropColumn`, `DropTarget`, `SwapTarget`), `ReconDestDriver`, `ConnectorEngineDriver`, `SourceIntrospectorDriver`, `SourceSnapshotDriver`, `BridgeStrategy`, `ReconSourceDriver`, `SourceHealFetcherDriver`, `FieldDirectiveFunc`, `TransformFunc`.
5. **Mục 5 & 6 — Quy trình 5 Bước Trước Khi Sửa Code (Blast-Radius = 0) & Lộ Trình Thực Thi Tổng Thể (`W1 → W6`).**
