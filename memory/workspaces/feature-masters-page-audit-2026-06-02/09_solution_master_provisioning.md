# 09_solution_master_provisioning.md — Master schema/connection: đặt func ở đâu?

> Ground: workflow `analyze-master-schema-provisioning` (4 agent, evidence file:line). 2026-06-03.

## 1. SHADOW làm thế nào (mẫu chuẩn) — 2 LỚP, 2 SERVICE, 2 TRIGGER
| Lớp | Cái gì | SERVICE | Khi nào | Evidence |
|-----|--------|---------|---------|----------|
| **L1 — Metadata** | INSERT `connection_registry` role='shadow' code='default_shadow' | **CMS** | **lúc boot** (idempotent) | `bootstrap/shadow_connection.go:34-51` gọi từ `server.go:75` |
| **L2 — DDL vật lý** | `CREATE SCHEMA/TABLE/ALTER` shadow | **WORKER (sinkworker)** | **lazy khi có CDC ingest** | `schema_manager.go:246` (CREATE SCHEMA), `:89-143` |

→ Shadow = **CMS giữ metadata (boot); WORKER chạy DDL vật lý (theo data flow)**. CMS KHÔNG bao giờ chạy CREATE TABLE.

## 2. MASTER hiện trạng
| Cái gì | SERVICE | Khi nào | Trạng thái |
|--------|---------|---------|-----------|
| **Metadata** role='master' | (chưa có) | — | ❌ **GAP**: seed comment, KHÔNG có `EnsureDefaultMasterConnection`. Chỉ có endpoint tay tôi lỡ thêm. |
| **Schema/Table vật lý** | **WORKER** `MasterDDLGenerator.Apply` → `CREATE SCHEMA IF NOT EXISTS + CREATE TABLE` ở dest DB | **on-approve** qua NATS `cdc.cmd.master-create` / `cdc.cmd.master.bind` | ✅ **ĐÃ CÓ SẴN** (`master_ddl_generator.go:131,184-204`; handler `master_ddl_handler.go:53`; subscribe `worker_server.go:442,474`) |

## 3. Quyền truy cập DB (quyết định)
- **CMS** chỉ kết nối: control-plane (5433 cdc_dw) + shadow (5436). **KHÔNG có connection tới dest (5434 goopay_dest).** (`config-local.yml`; AppConfig CMS không có MasterDB.)
- **WORKER** có: control + shadow + **dest (5434) = RoleDestination — service DUY NHẤT chạm dest** (`connection_manager.go:51`, `multi.go:161`).
- ⇒ **Chỉ WORKER mới CREATE SCHEMA ở dest được.** CMS không thể, về mặt kiến trúc.

## 4. ĐỀ XUẤT (đặt func ở đâu) — bám đúng mẫu shadow

| Thành phần | Đặt ở | Cơ chế | Lý do |
|-----------|-------|--------|-------|
| **Master CONNECTION (metadata)** | **CMS — bootstrap lúc boot** | Thêm `bootstrap.EnsureDefaultMasterConnection` (clone `EnsureDefaultShadowConnection`) → INSERT `default_master` role='master' active, idempotent; gọi ở `server.go` ngay sau shadow | Mirror shadow L1. CMS sở hữu `connection_registry`. **Tự động, KHÔNG cần nút.** |
| **Master SCHEMA/TABLE (DDL vật lý)** | **WORKER — đã có** | `MasterDDLGenerator.Apply` chạy on-approve qua NATS. KHÔNG code mới | Mirror shadow L2. Chỉ worker chạm dest. |

## 5. Kết luận quan trọng
- **KHÔNG cần nút "Tạo schema master"**: schema vật lý **tự tạo khi approve master** (worker), y như shadow table tự tạo khi ingest. Người dùng chỉ cần: tạo master (đặt `master_schema` = vd `master_<connector>`) → **Approve** → worker `CREATE SCHEMA IF NOT EXISTS` + CREATE TABLE.
- **Lỗi `master_connection_not_found`** chỉ vì thiếu metadata row → fix bằng **bootstrap `default_master` lúc boot** (mirror shadow), KHÔNG phải bằng nút bấm tay.
- **Endpoint `POST /master-connections` tôi lỡ thêm = SAI hướng** (reinvent thủ công thay vì bootstrap như shadow) → **revert**.

## 6. Việc sẽ làm (sau khi User chốt)
1. **REVERT** ở CMS: `master_registry_handler_connection.go` (Create+List), route `/v1/master-connections` (đã thêm sai).
2. **ADD** `bootstrap.EnsureDefaultMasterConnection` (CMS) — INSERT default_master role='master' @boot, idempotent. Cần thêm stanza config master cho CMS (chỉ để seed metadata; CMS không mở connection tới dest).
3. (Tuỳ chọn UX) Khi tạo master từ connector trên /shadow: default `master_schema = master_<connector>` cho tiện. Status "đã tạo" = master đã approved (schema tồn tại sau approve).

## 7. Nếu User VẪN muốn nút per-connector + status
- Nút chỉ là **trigger control-plane**: gọi approve/bind để worker tạo schema; CMS KHÔNG tự CREATE SCHEMA. Status = query `cdc_system.master_binding.schema_status='approved'` cho connector đó. Nhưng đây là thừa so với luồng approve sẵn có.
