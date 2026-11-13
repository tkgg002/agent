---
tier: T2
suffix: design-pattern
status: PROPOSED — chờ User cho phép; KHÔNG đụng recon-heal
---
# 09 Solution — Giải quyết triệt để: Agent phá code cũ + design pattern lộn xộn

## 0. Nguyên tắc
1. **Không tin lời Agent** — "OK" do máy quyết định (test, arch_check, api_diff, exit code).
2. **Một chuẩn cho mỗi service, viết ra, máy kiểm** — không còn "mỗi phiên một kiểu".
3. **Nợ cũ chỉ được giảm** — mọi vi phạm hiện có nằm trong allowlist/baseline; thêm mới = bị chặn.
4. **Không big-bang, không đụng recon-heal** — dọn theo đợt nhỏ, mỗi đợt 1 nhánh tách từ commit đã lên prod.

## 1. Hiện trạng đo bằng máy (arch_check.py, read-only, 2026-11-06 17:11)
Script: `scratch/arch_check.py` · baseline: `scratch/arch_allowlist_baseline.txt` (**225 vi phạm**).

| Rule | Ý nghĩa | Số file |
|---|---|---|
| DS-R1 | SQL `.Exec/.Raw` nằm ngoài `internal/repository` (data-service) | **74** |
| DS-R2 | handler import thẳng repository | 17 |
| DS-R3 | handler giữ `*gorm.DB` | 18 |
| CS-R1 | cms `api` import thẳng `infra` (29 lần là `infra/messaging` — publish NATS từ api thay vì qua `app/ports/publisher.go`) | 33 |
| CS-R2 | cms `api` tự viết SQL | 5 |
| CS-R4 | cms `app` import `infra` | 4 |
| CS-R5 | cms dùng `internal/model` (mô hình cũ) ngoài `infra/persistence` | 24 |
| ALL-R6 | file song song `_v2/legacy/_bk` | 7 |
| ALL-R7 | file > 600 dòng | 40 |
| ALL-R8 | code Go trong `docs/`, `scratch/` (làm `go build ./...` đỏ) | 3 thư mục |
| — | Test đỏ sẵn (go test full) | 15 package |
| — | Hàm xác định PK/timestamp trùng lặp | 9 hàm / 7 file |

Đính chính so với bản trước: CS-R1 chủ yếu là **messaging**, không phải persistence (chỉ ~6 file đụng persistence).

## 2. Chuẩn đích cho từng service (nội dung ARCHITECTURE.md — User duyệt)

### 2.1 centralized-data-service (layered)
```
cmd/* → internal/server (wiring DUY NHẤT)
handler  → service  → repository → model
             ↘ pkgs/* (hạ tầng dùng chung)
```
- handler: nhận message/HTTP, validate, gọi service. **Không** SQL, **không** `*gorm.DB`, **không** import repository.
- service: luật nghiệp vụ. Gọi repository qua interface khai báo **trong package service** (consumer-owned interface).
- repository: **nơi duy nhất** có SQL/GORM/Mongo query. Một bảng = một repository sở hữu.
- Dynamic SQL (bảng shadow/master sinh động): gom vào `internal/repository/dynamic/` (DDL builder, upsert builder, dialect) — không viết rải rác trong service.
- Test: đặt cạnh package (`_test.go`); `test/internal/...` chỉ giữ integration test và **luôn được chạy**.

### 2.2 cdc-cms-service (hexagonal/CQRS — giữ đúng cái đang có)
```
router → api (HTTP adapter) → app/commands|queries → domain
                                   ↕ app/ports (interface)
infra/{persistence,messaging,http,cache} implements ports
bootstrap = wiring DUY NHẤT
```
- api: chỉ gọi command/query bus; **không** import infra, **không** SQL.
- app: chỉ phụ thuộc `domain` + `ports`.
- domain: thuần, không GORM/infra.
- `internal/model` = struct GORM **chỉ dùng trong infra/persistence**, map sang domain.

### 2.3 cdc-cms-web
- pages → hooks → `services/api.ts`. UI không gọi `axios/fetch` trực tiếp (hiện 0 vi phạm — giữ nguyên).
- Kiểu dữ liệu API sinh/kiểm từ contract của cms-service (xem 3.4).

### 2.4 Luật chung
- **Tương thích ngược mặc định:** không đổi chữ ký hàm/interface exported, field JSON API, NATS subject/payload, cột DB. Cần đổi → thêm cái mới, giữ cái cũ, đánh dấu `Deprecated:`, xoá ở đợt riêng khi đã hết caller.
- **Không tạo bản song song** (`_v2`, `legacy`, `_bk`); thay thế thì xoá cái cũ trong cùng thay đổi.
- **Một khái niệm = một chủ sở hữu** (PK, timestamp, registry lookup, route resolve).
- File mới ≤ 600 dòng; file cũ trong allowlist không được dài thêm.

## 3. Hàng rào máy (chặn từ gốc, Agent không lách được)
| ID | Hàng rào | Chạy ở | Chặn khi |
|---|---|---|---|
| 3.1 | `go build ./... && go vet ./...` (sau khi dọn docs/scratch) | H6 sau mỗi lần sửa | đỏ |
| 3.2 | `go test -short` của package bị sửa **+ package phụ thuộc ngược + test/internal liên quan** | H6 | đỏ |
| 3.3 | `go test -short ./...` full so với `BASELINE_FAILS.txt` | H7 trước DONE | có test đỏ mới / baseline dài thêm |
| 3.4 | `api_diff` (exported Go API) + snapshot contract HTTP (cms↔web) + NATS payload (cms↔data-service) | H7 | có breaking change chưa khai báo & chưa được User duyệt |
| 3.5 | `arch_check.py --allowlist` | H6 | vi phạm mới ngoài allowlist |
| 3.6 | Allowlist/baseline: hook chỉ cho **xoá dòng**, cấm thêm | pre-tool hook | Agent thêm dòng |
| 3.7 | Hook thiếu `.agent-project.json` → BLOCK (hiện đang bỏ qua im lặng `agy_guard.py:1305`) | H6/H7 | thiếu manifest |
| 3.8 | Bỏ tự resolve FLAG Laya (`laya_gate.py:392`); Laya chỉ tham khảo | core agent | — |
| 3.9 | Diff ngoài danh sách file đã khai báo trong `08_tasks` | H7 | file lạ |

## 4. Hợp đồng hành vi của Agent mỗi task
1. Mở đầu: "Em hiểu anh muốn … / sẽ sửa các file … / không đụng …". Mơ hồ → hỏi, chưa làm.
2. Đọc **file mẫu cùng loại** đang tuân chuẩn và ghi đường dẫn vào kế hoạch (`## Pattern Reference`).
3. Trước khi đổi bất kỳ symbol exported nào: liệt kê **toàn bộ** caller (`grep`/`go list`, gồm `test/`), ghi vào kế hoạch.
4. Báo Done = dán output hook 3.1–3.5, không tự viết "PASS".
5. Không đụng nhánh User đang release (hiện: `recon-heal`).

## 5. Lộ trình dọn nợ (sau khi recon-heal lên prod, mỗi đợt 1 nhánh từ commit prod)
| Đợt | Nội dung | Thước đo xong |
|---|---|---|
| W0 | Cài hàng rào 3.1–3.9 + manifest + baseline test + allowlist 225 | Hook chạy, thử làm bể → bị chặn |
| W1 | Xoá/di dời `docs/*_bk*`, `scratch/`; xử lý 7 file `_v2/legacy` (gộp hoặc xoá bản thừa) | R6=0, R8=0, build/vet xanh |
| W2 | Sửa 15 package test đỏ (Red→Green, 1 test 1 commit); ưu tiên routing multi-connection | baseline = 0 |
| W3 | **Một nguồn sự thật PK/TS**: tạo 1 package `internal/service/keyspec` (hoặc tên User chọn) sở hữu luật PK/timestamp; 9 hàm hiện có (`resolveMasterPKFromSpec`, `resolveMasterTimestampField`, `resolveSourceTimestampField`, `resolveSourceAndDestTSFields`, `resolveTSFields`, `resolvePKFields`, `resolveTimestampField`, `resolveShadowPKField`, `resolveSegmentBTSCol`) chuyển thành wrapper gọi nó; test so khớp hành vi cũ trước khi gộp | 1 chủ sở hữu, test hành vi xanh |
| W4 | data-service: chuyển SQL về repository theo **từng domain** (recon → master → shadow → source); mỗi PR 1 domain | DS-R1 74→0, DS-R2/R3 → 0 |
| W5 | cms: api publish qua `app/ports` thay vì `infra/messaging`; 5 file SQL trong api về persistence | CS-R1, CS-R2, CS-R4 → 0 |
| W6 | cms: thay `internal/model` ngoài persistence bằng domain type + mapper | CS-R5 24→0 |
| W7 | Tách 40 file > 600 dòng — **chỉ khi đợt khác chạm vào**, có test phủ trước khi tách | R7 giảm dần |
| W8 | Contract test cms↔web, cms↔data-service (NATS) | breaking change bị bắt tự động |

Mỗi đợt: không đổi hành vi runtime (trừ W2 sửa bug), so sánh output trước/sau bằng test, allowlist phải giảm.

## 6. Việc làm được NGAY mà không đụng recon-heal
- Đã làm: arch_check.py + baseline allowlist 225 (trong workspace, read-only với repo).
- Có thể làm tiếp nếu User đồng ý: soạn nháp `ARCHITECTURE.md` (trong workspace), sửa hook core `agent/` cho 3.6–3.9 (không đụng data-hub), viết `api_diff` script trong workspace.

## 7. User cần quyết
1. Chuẩn 2.1/2.2/2.3 có đúng ý anh không (đặc biệt: data-service giữ layered, cms giữ hexagonal)?
2. Tên package nguồn sự thật PK/TS (W3).
3. `docs/*_bk*`, `scratch/`: xoá hay di dời?
4. Cho phép em làm mục 6 ngay (chỉ core agent + workspace, không đụng data-hub)?
