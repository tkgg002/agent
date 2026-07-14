# Phân Tích Kiến Trúc Luồng Đối Soát & Chữa Lành (Recon & Heal Pipeline)

## Tổng Quan Kiến Trúc Hiện Tại

Hệ thống đối soát và chữa lành dữ liệu CDC gồm **3 thành phần chính** hoạt động qua **NATS messaging**:

```mermaid
graph LR
    subgraph "CMS Frontend (cdc-cms-web)"
        FE_Check["Nút 'Bắt đầu đối soát'<br/>openCheckAll() / openCheckTable()"]
        FE_Heal["Nút 'Chữa lành' (per-row, drift only)<br/>openHeal() → useHealMutation"]
        FE_ExecHeal["Nút 'Thực thi chữa lành' (per-row, drift only)<br/>openExecuteHeal() → ExecuteHealModal"]
        FE_Reports["Bảng Report + Checkboxes"]
    end

    subgraph "API Gateway (cdc-cms-service)"
        API_CheckAll["POST /reconciliation/check<br/>TriggerCheckAll()"]
        API_CheckTable["POST /reconciliation/check/:table<br/>TriggerCheck()"]
        API_Heal["POST /reconciliation/heal<br/>TriggerHeal()"]
        API_ExecHeal["POST /reconciliation/execute-heal<br/>TriggerExecuteHeal()"]
        API_Reports["GET /reconciliation/report/:table/unhealed<br/>GetUnhealedReports()"]
    end

    subgraph "CDC Worker (centralized-data-service)"
        WK_Check["HandleReconCheck<br/>cdc.cmd.recon-check"]
        WK_Heal["HandleReconHeal<br/>cdc.cmd.recon-heal"]
        WK_ExecHeal["HandleExecuteHeal<br/>cdc.cmd.execute-heal"]
    end

    DB[(cdc_reconciliation_report)]

    FE_Check --> API_CheckAll -->|NATS| WK_Check --> DB
    FE_Check --> API_CheckTable -->|NATS| WK_Check
    FE_Heal --> API_Heal -->|NATS| WK_Heal --> DB
    FE_ExecHeal --> API_ExecHeal -->|NATS| WK_ExecHeal --> DB
    FE_Reports --> API_Reports --> DB
```

---

## GIAI ĐOẠN 0: KÍCH HOẠT ĐỐI SOÁT (Recon Check)

### Sequence Flow

```mermaid
sequenceDiagram
    actor Admin
    participant FE as CMS Frontend
    participant API as API Gateway
    participant NATS as NATS Bus
    participant WK as CDC Worker
    participant DB as PostgreSQL

    Admin->>FE: Bấm "Bắt đầu đối soát"<br/>(Kiểm tra / Kiểm tra tất cả)
    Note over FE: Per-row: openCheckTable(record)<br/>→ useCheckTableMutation<br/>Global: openCheckAll()<br/>→ useCheckAllMutation
    FE->>API: POST /api/reconciliation/check/:table?tier=2<br/>(hoặc POST /api/reconciliation/check cho checkAll)
    Note over FE,API: Body: {reason, table,<br/>source_database?, source_table?,<br/>shadow_schema?, shadow_table?, lookback?}
    Note over API: Handler: TriggerCheck()<br/>File: reconciliation_handler_commands.go:18
    API->>API: resolveTargetTable(scope)<br/>→ xác định shadow table name
    API->>NATS: Publish ReconCheckCommand<br/>Subject: cdc.cmd.recon-check
    Note over API: Wire payload: {tier, table, lookback}
    API-->>FE: 202 {message, tier, table, job_id}
    
    NATS->>WK: Subscribe handler: HandleReconCheck()
    Note over WK: File: recon_handler_run.go:18<br/>Worker parse thêm: segment, deep,<br/>start_time, end_time (reserved fields)

    alt Segment A (source↔shadow) — Default
        WK->>WK: RunTier2(ctx, entry)<br/>So sánh IDs + Timestamps<br/>giữa Source (Mongo/PG) và Shadow
        Note over WK: Kết quả:<br/>• missingIDs (missing_from_dest)<br/>• staleIDs.mismatched<br/>• staleIDs.missing_from_src<br/>• orphan_count
    else Segment B (shadow↔master)
        WK->>WK: RunSegmentBFor(ctx, table, deep)<br/>So sánh Shadow ↔ Master
        Note over WK: Kết quả:<br/>• missingIDs (missing in master)<br/>• staleIDs.stale_ids (timestamp lệch)<br/>• staleIDs.orphan_in_master
    end

    WK->>DB: INSERT INTO cdc_reconciliation_report<br/>(target_table, segment, missing_ids,<br/>stale_ids, missing_count, stale_count,<br/>orphan_count, source_count, checked_at)
    WK-->>NATS: Respond {status, tables_checked}
```

### Chi Tiết Kỹ Thuật

| Thành phần | File | Function | NATS Subject |
|---|---|---|---|
| FE Hook | [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts#L171) | `useCheckTableMutation()` | — |
| FE Hook (All) | [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts#L162) | `useCheckAllMutation()` | — |
| API Route (1 table) | [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go#L174) | `POST /reconciliation/check/:table` → `TriggerCheck()` | — |
| API Route (all) | [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go#L173) | `POST /reconciliation/check` → `TriggerCheckAll()` | — |
| API Handler | [reconciliation_handler_commands.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go#L18) | `TriggerCheck()` | — |
| Command | [recon_check.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_check.go#L27) | `ReconCheckCommand` | `cdc.cmd.recon-check` |
| Worker Handler | [recon_handler_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go#L18) | `HandleReconCheck()` | Subscribe `cdc.cmd.recon-check` |
| Recon Engine | recon_tier_a.go / recon_tier_b.go | `RunTier2()` / `RunSegmentBFor()` | — |

### Payload — Từ FE đến Worker (Data Flow)

**① FE gửi tới API Gateway:**
```
POST /api/reconciliation/check?tier=2
```
```json
{
  "reason": "Manual check by admin",
  "table": "payment_bills",
  "source_database": "mongo_fintech",    // optional — dùng resolve scope
  "source_table": "payment_bills",       // optional
  "shadow_schema": "cdc_shadow_gpay",    // optional
  "shadow_table": "payment_bills",       // optional
  "lookback": "hot"                      // optional: "hot" (2h) / "cold" (7d)
}
```

**② API Gateway build Command (wire payload qua NATS):**
```json
{
  "tier": "2",
  "table": "cdc_shadow_gpay.payment_bills",
  "lookback": "hot"
}
```
> `tier` lấy từ query string (default `"1"`). `table` là kết quả `resolveTargetTable()` — FQN shadow table. Các field scope (`source_database`, `shadow_schema`...) chỉ dùng để resolve, KHÔNG đi vào wire payload.

**③ Worker parse (có thể nhận thêm reserved fields):**
```go
var payload struct {
    Tier      string `json:"tier"`
    Table     string `json:"table"`
    Segment   string `json:"segment"`    // reserved — FE chưa gửi
    Deep      bool   `json:"deep"`       // reserved — FE chưa gửi
    StartTime *int64 `json:"start_time"` // reserved — epoch ms
    EndTime   *int64 `json:"end_time"`   // reserved — epoch ms
    Lookback  string `json:"lookback"`
}
```
> **Lưu ý:** `segment`, `deep`, `start_time`, `end_time` là reserved fields. Worker parse được nhưng FE + API Gateway hiện **KHÔNG gửi** các field này. Segment B check được trigger bởi logic nội bộ worker (tier1 tự scan tất cả segment).

**Validation tại Worker:**
- `start_time` và `end_time` phải đi cặp, `end >= start`, khoảng cách ≤ 24h
- Nếu `segment == "shadow_master"` → chuyển sang `handleReconCheckSegmentB()`
- Nếu `tier == "prune"` → chạy orphan prune (soft-delete ghost shadow rows)

---

## GIAI ĐOẠN 1: TRUY VẤN DANH SÁCH LỖI (Unhealed Reports)

### Sequence Flow

```mermaid
sequenceDiagram
    actor Admin
    participant FE as CMS Frontend
    participant API as API Gateway
    participant DB as PostgreSQL

    Admin->>FE: Bấm "Thực thi chữa lành" (ThunderboltOutlined)<br/>→ openExecuteHeal(record)
    Note over FE: ExecuteHealModal mở ra,<br/>auto-fetch danh sách unhealed reports<br/>qua useUnhealedReports(table, shadowSchema)
    FE->>API: GET /api/reconciliation/report/:table/unhealed<br/>?shadow_schema=cdc_shadow_xxx
    Note over API: Handler: GetUnhealedReports()<br/>File: reconciliation_handler_execute_heal.go:15
    API->>DB: SELECT * FROM cdc_reconciliation_report<br/>WHERE target_table = :table<br/>AND healed_at IS NULL<br/>ORDER BY checked_at DESC
    DB-->>API: Rows[]
    API-->>FE: {data: [...reports], total: N}
    FE->>FE: Render bảng với Checkboxes<br/>cho từng report row
```

### Chi Tiết Kỹ Thuật

| Thành phần | File | Function |
|---|---|---|
| API Gateway | [reconciliation_handler_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go#L15) | `GetUnhealedReports()` |
| Query Handler | queries/recon/ | `ListUnhealedReportsQuery` |

**Cấu trúc Report trả về:**
```json
{
  "id": 42,
  "target_table": "payment_bills",
  "segment": "source_shadow",
  "missing_count": 15,
  "stale_count": 3,
  "orphan_count": 2,
  "missing_ids": ["id1", "id2", ...],
  "stale_ids": {"mismatched": [...], "missing_from_src": [...]},
  "checked_at": "2026-07-03T10:00:00Z",
  "healed_at": null,
  "status": "drift"
}
```

---

## GIAI ĐOẠN 2: THỰC THI CHỮA LÀNH (Execute Heal — Luồng Tương Tác)

> [!IMPORTANT]
> Đây là luồng **hoàn toàn mới**, tách biệt khỏi luồng heal cũ (background). Không chạy lại đối soát (`RunTier2`/`RunSegmentBFor`), chỉ thực thi chữa lành từ report IDs đã có.

### Sequence Flow

```mermaid
sequenceDiagram
    actor Admin
    participant FE as CMS Frontend
    participant API as API Gateway
    participant NATS as NATS Bus
    participant WK as CDC Worker
    participant Shadow as Shadow DB
    participant Master as Master DB
    participant Source as Source DB (Mongo)

    Admin->>FE: Tích chọn Report IDs +<br/>Action Checkboxes → "Thực hiện"
    FE->>API: POST /api/reconciliation/execute-heal
    Note over FE,API: Payload:<br/>{table, report_ids: [42,43],<br/>heal_mismatched: true,<br/>heal_missing_dest: true,<br/>prune_missing_src: false,<br/>force_heal: false}
    
    Note over API: Handler: TriggerExecuteHeal()<br/>File: reconciliation_handler_execute_heal.go:29
    API->>NATS: Publish ExecuteHealCommand<br/>Subject: cdc.cmd.execute-heal
    API-->>FE: 202 {message: "dispatched", job_id}

    NATS->>WK: Subscribe: HandleExecuteHeal()
    Note over WK: File: recon_execute_heal.go:29

    Note over WK: ── Safety Gate ──<br/>Tính tổng IDs từ tất cả report đã chọn.<br/>Nếu > 50,000 + !force_heal → BLOCK
    
    alt totalIDs > 50K && !force_heal
        WK-->>NATS: {error: "execute-heal blocked:<br/>... exceeds safety threshold 50000"}
        Note over FE: FE bắt error →<br/>Modal.confirm() hỏi user<br/>"Xác nhận Force Heal?"
        FE->>API: POST execute-heal<br/>(force_heal: true)
        API->>NATS: Re-dispatch
    end

    loop Từng Report ID trong mảng
        WK->>WK: reportRepo.ClaimForHealing(id)
        Note over WK: ── Race Condition Guard ──<br/>Atomic UPDATE status='healing'<br/>WHERE status NOT IN ('healing','healed')
        
        alt Claim thất bại (worker khác đang xử lý)
            WK->>WK: Log warn + skip report
        else Claim thành công
            WK->>WK: Parse JSONB (stale_ids, missing_ids)

            alt Segment A (source_shadow)
                Note over WK: Parse StaleIDs → {mismatched, missing_from_src}<br/>Parse MissingIDs → flat array (missing_from_dest)

                opt heal_mismatched = true
                    WK->>Source: fetchAndWriteChunked(mismatched)<br/>Chunk 1000 IDs/batch → FetchAndWriteByIDs()
                    Source-->>Shadow: Upsert records (mỗi batch 200 docs)
                    WK->>WK: Ghi healed_mismatched_count, duration_ms
                end

                opt heal_missing_dest = true
                    WK->>Source: fetchAndWriteChunked(missing_ids)<br/>Chunk 1000 IDs/batch → FetchAndWriteByIDs()
                    Source-->>Shadow: Upsert records (mỗi batch 200 docs)
                    WK->>WK: Ghi healed_missing_dest_count, duration_ms
                end

                opt prune_missing_src = true
                    WK->>Shadow: UPDATE SET _deleted=true<br/>WHERE _source_id IN (missing_from_src)
                    Note over WK: ⚠️ TODO: Chưa implement thực tế<br/>(chỉ log count)
                    WK->>WK: Ghi pruned_missing_src_count, duration_ms
                end

            else Segment B (shadow_master)
                Note over WK: Parse StaleIDs → {stale_ids, orphan_in_master}<br/>Parse MissingIDs → flat array

                opt heal_mismatched = true
                    WK->>Shadow: mapGpayToSourceIDs(stale_ids)<br/>Map _gpay_id → _source_id
                    WK->>NATS: publishTransmuteChunked(source_ids)<br/>Subject: cdc.cmd.transmute
                    Note over WK: Chunked: 200 IDs/batch<br/>Delay: 200ms giữa các batch
                end

                opt heal_missing_dest = true
                    WK->>Shadow: mapGpayToSourceIDs(missing_ids)
                    WK->>NATS: publishTransmuteChunked(source_ids)
                end

                opt prune_missing_src = true
                    WK->>Master: UPDATE SET _deleted=true<br/>WHERE _gpay_id IN (orphan_in_master)
                    Note over WK: ⚠️ TODO: Chưa implement thực tế
                end
            end

            Note over WK: SegA/B handle error internally:<br/>log chunk fail + continue remaining.<br/>Partial heal vẫn set healed (idempotent).
            WK->>WK: UPDATE report SET<br/>healed_at = NOW(),<br/>status = "healed",<br/>healed_mismatched_count, ...,<br/>healed_missing_dest_count, ...,<br/>pruned_missing_src_count, ...
            Note over WK: ReleaseHealClaim chỉ gọi khi:<br/>• registry not found<br/>• unknown segment
        end
    end

    WK-->>NATS: Respond {status: "success",<br/>reports_processed, total_healed}
```

### Chi Tiết Kỹ Thuật

| Thành phần | File | Function | NATS Subject |
|---|---|---|---|
| FE Modal | [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx) | — | — |
| FE Page | [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx) | `openExecuteHeal()` | — |
| API Gateway | [reconciliation_handler_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go#L29) | `TriggerExecuteHeal()` | — |
| Command | [recon_async.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_async.go#L35) | `ExecuteHealCommand` | `cdc.cmd.execute-heal` |
| Worker Handler | [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L43) | `HandleExecuteHeal()` | Subscribe `cdc.cmd.execute-heal` |
| Worker SegA | [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L155) | `executeHealSegA()` | — |
| Worker SegB | [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L230) | `executeHealSegB()` | Publish `cdc.cmd.transmute` |
| NATS Sub | [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L346) | — | `cdc.cmd.execute-heal` |
| Route | [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go#L177) | — | `/reconciliation/execute-heal` |

**Payload API:**
```json
{
  "table": "payment_bills",
  "report_ids": [42, 43, 44],
  "heal_mismatched": true,
  "heal_missing_dest": true,
  "prune_missing_src": false,
  "force_heal": false
}
```

> `force_heal` (default false): Khi tổng IDs vượt ngưỡng 50,000, Worker block request. FE hiển thị `Modal.confirm()` hỏi user → nếu đồng ý, gửi lại với `force_heal: true`.

### Batching & Constants

| Constant | Value | Mô tả | File |
|---|---|---|---|
| `healChunkSize` | 200 | Số IDs mỗi batch transmute (Segment B) | recon_heal_v4.go |
| `healDelayMs` | 200ms | Delay giữa các batch (giảm tải I/O & Kafka) | recon_heal_v4.go |
| `healAutoMaxIDs` | 1,000 | Ngưỡng max IDs cho auto-heal (Background) | recon_heal_v4.go |
| `healAutoMaxDriftPct` | 5.0% | Ngưỡng max drift % cho auto-heal | recon_heal_v4.go |
| `healReportMaxAge` | 5 min | Report quá cũ → chạy lại check | recon_heal_v4.go |
| **`segAChunkSize`** | **1,000** | **Số IDs/batch cho FetchAndWriteByIDs (Segment A)** | **recon_execute_heal.go** |
| **`interactiveHealMaxIDs`** | **50,000** | **Ngưỡng Safety Gate cho Interactive Heal** | **recon_execute_heal.go** |

### Cơ Chế An Toàn (Safety Mechanisms)

| Cơ chế | Áp dụng cho | Mô tả |
|---|---|---|
| **Safety Gate (Threshold)** | Interactive Heal | Tổng IDs > 50K → block, cần `force_heal=true` |
| **Race Condition Guard** | Interactive Heal | `ClaimForHealing()` — atomic UPDATE status='healing', skip nếu bị worker khác claim |
| **Chunking SegA** | Interactive Heal SegA | `fetchAndWriteChunked()` — chunk 1000 IDs/batch trước khi gọi MongoDB `$in` |
| **healThresholdBlocked()** | Background Heal | Max 1000 IDs / 5% drift — tự động block nếu vượt |
| **Chunking SegB** | Cả 2 luồng | `publishTransmuteChunked()` — 200 IDs/batch + 200ms delay |

---

## Luồng Background Heal (HandleReconHeal — Chi Tiết Đầy Đủ)

> [!NOTE]
> Luồng này (`HandleReconHeal` / `cdc.cmd.recon-heal`) VẪN HOẠT ĐỘNG song song với luồng Interactive. Khác biệt chính: luồng này **TỰ CHẠY đối soát** trước khi heal, và hỗ trợ 2 mode: **Window** (mặc định) và **Full-diff** (quét theo time-range).

### Payload — Từ FE đến Worker (Data Flow)

**① FE gửi tới API Gateway (`useHealMutation`):**
```
POST /api/reconciliation/heal
```
```json
{
  "reason": "Manual heal by admin",
  "table": "payment_bills",
  "segment": "source_shadow",            // optional: "" / "source_shadow" / "shadow_master"
  "source_database": "mongo_fintech",    // optional — chỉ dùng resolve scope
  "source_table": "payment_bills",       // optional
  "shadow_schema": "cdc_shadow_gpay",    // optional
  "shadow_table": "payment_bills"        // optional
}
```

**② API Gateway build Command (`TriggerHeal`):**
```json
{
  "table": "cdc_shadow_gpay.payment_bills",
  "segment": "source_shadow"
}
```
> API chỉ extract `table` (qua `resolveTargetTable()`) + `segment`. Các field scope chỉ dùng resolve, **KHÔNG** đi vào wire payload. `ReconHealCommand` struct **CHỈ CÓ 2 FIELD**: `Table` + `Segment`.

**③ Worker parse (có reserved fields — FE/API KHÔNG gửi):**
```go
var payload struct {
    Table     string `json:"table"`
    Segment   string `json:"segment"`    // FE gửi
    Legacy    bool   `json:"legacy"`     // reserved — FE không gửi → false
    Mode      string `json:"mode"`       // reserved — FE không gửi → "" (= window)
    StartTime string `json:"start_time"` // reserved — RFC3339, FE không gửi
    EndTime   string `json:"end_time"`   // reserved — RFC3339, FE không gửi
    Lookback  string `json:"lookback"`   // reserved — FE không gửi → ""
}
```

> [!IMPORTANT]
> **GAP hiện tại:** Worker hỗ trợ đầy đủ `mode=full_diff` + `start_time/end_time` (time-range heal) và `lookback=hot/cold`, nhưng API Gateway (`ReconHealCommand`) **KHÔNG CÓ** các field này → FE **KHÔNG THỂ** trigger `full_diff` mode qua UI. Muốn bật full_diff phải gửi NATS message trực tiếp hoặc bổ sung fields vào `ReconHealCommand`.


### Sequence Flow — Segment A (Source↔Shadow)

```mermaid
sequenceDiagram
    actor Admin
    participant FE as CMS Frontend
    participant API as API Gateway
    participant NATS as NATS Bus
    participant WK as CDC Worker
    participant Source as Source DB (Mongo)
    participant Shadow as Shadow DB
    participant DB as Report DB

    Admin->>FE: Bấm "Chữa lành" (MedicineBoxOutlined)<br/>→ openHeal(record) → useHealMutation
    FE->>API: POST /api/reconciliation/heal<br/>(hoặc /api/reconciliation/heal/:table)
    Note over FE,API: Payload: {table, segment,<br/>mode, lookback, start_time, end_time}
    API->>NATS: Publish ReconHealCommand<br/>Subject: cdc.cmd.recon-heal
    API-->>FE: 202 {message: "heal dispatched"}

    NATS->>WK: HandleReconHeal()
    Note over WK: File: recon_handler_run.go:206<br/>Dispatch: healSegmentA()

    alt Mode = "full_diff" (Quét theo time-range)
        Note over WK: 🔸 NHÁNH FULL-DIFF<br/>File: recon_heal_v4.go:293
        WK->>WK: Validate time range<br/>(RFC3339, end >= start, max 30 ngày)
        WK->>WK: TimeBoundedDiffMissingFromShadow(entry, start, end)
        Note over WK: So sánh IDs trong khoảng [start, end]<br/>giữa Source và Shadow → tìm missing[]
        
        alt missing = 0
            WK-->>NATS: {status: "noop"}
        else missing > 0
            WK->>WK: healThresholdBlocked?<br/>(max 1000 IDs / 5% drift)
            WK->>Source: FetchAndWriteByIDs(missing)
            Source-->>Shadow: Upsert records trực tiếp
            WK-->>NATS: {status: "healed",<br/>healed_count, missing_count,<br/>src_total, dispatch_path: "direct_fetch_write"}
        end

    else Mode = "window" (Mặc định — Quét RunTier2)
        Note over WK: 🔸 NHÁNH WINDOW<br/>File: recon_heal_v4.go:353
        
        alt lookback = "hot"
            WK->>WK: RunTier2(ctx, entry)<br/>Lookback = 2 giờ
        else lookback = "cold"
            WK->>WK: RunTier2(ctx, entry)<br/>Lookback = 7 ngày
        end
        
        Note over WK: RunTier2 trả về report mới:<br/>• missingIDs (missing_from_dest)<br/>• staleIDs.mismatched<br/>• staleIDs.missing_from_src

        alt Tất cả = 0
            WK-->>NATS: {status: "noop"}
        else Có drift
            WK->>WK: healThresholdBlocked?
            WK->>WK: Gộp: healIDs = missing + mismatched + missing_from_src
            WK->>Source: FetchAndWriteByIDs(healIDs)
            Source-->>Shadow: Upsert records trực tiếp
            WK->>DB: UPDATE report SET<br/>healed_at, healed_count,<br/>healed_duration_ms, status="healed"
            WK-->>NATS: {status: "healed",<br/>healed_count, missing_count,<br/>mismatched_count, orphan_count}
        end
    end
```

### Sequence Flow — Segment B (Shadow↔Master)

```mermaid
sequenceDiagram
    participant WK as CDC Worker
    participant Shadow as Shadow DB
    participant NATS as NATS Bus
    participant DB as Report DB

    Note over WK: healSegmentB()<br/>File: recon_heal_v4.go:90

    WK->>DB: GetLatestByTable(masterFQN, "shadow_master")
    
    alt Report null / đã healed / quá 5 phút
        WK->>WK: RunSegmentBFor(table, deep=true)<br/>Chạy lại đối soát shadow↔master
    else Report hợp lệ
        WK->>WK: Dùng report hiện có
    end

    Note over WK: Parse JSONB:<br/>• missingGpayIDs (flat array)<br/>• staleObj.stale_ids<br/>• staleObj.orphan_in_master

    WK->>WK: Gộp: gpayIDs = missing + stale + orphan
    WK->>WK: healThresholdBlocked?<br/>(max 1000 IDs / 5% drift)
    WK->>Shadow: mapGpayToSourceIDs(gpayIDs)<br/>Map _gpay_id → _source_id (chunked 200)
    
    loop Chunk 200 IDs, delay 200ms
        WK->>NATS: Publish cdc.cmd.transmute<br/>{master_table, _source_ids, triggered_by}
    end

    WK->>DB: UPDATE report SET<br/>healed_at, healed_count,<br/>healed_duration_ms
    WK-->>NATS: {status: "dispatched",<br/>healed_count, missing/mismatched/orphan counts}
```

### Chi Tiết Kỹ Thuật — Background Heal

| Thành phần | File | Function | NATS Subject |
|---|---|---|---|
| API Gateway | [reconciliation_handler_heal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_heal.go#L17) | `TriggerHeal()` | — |
| Command | [recon_async.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_async.go) | `ReconHealCommand` | `cdc.cmd.recon-heal` |
| Worker Entry | [recon_handler_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go#L206) | `HandleReconHeal()` | Subscribe `cdc.cmd.recon-heal` |
| Segment A | [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L257) | `healSegmentA()` | — |
| Full-diff scan | recon_heal_v4.go:293 | `TimeBoundedDiffMissingFromShadow()` → `FetchAndWriteByIDs()` | — |
| Window scan | recon_heal_v4.go:353 | `RunTier2()` → `FetchAndWriteByIDs()` | — |
| Segment B | [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L90) | `healSegmentB()` | Publish `cdc.cmd.transmute` |
| Safety Gate | recon_heal_v4.go:48 | `healThresholdBlocked()` | — |

### So Sánh 2 Mode trong healSegmentA

| Tiêu chí | Window (mặc định) | Full-diff |
|---|---|---|
| **Trigger** | `mode` = "" / "window" | `mode` = "full_diff" |
| **Input time** | `lookback`: "hot" (2h) / "cold" (7d) | `start_time` + `end_time` (RFC3339) |
| **Max range** | Cố định theo lookback | Tối đa 30 ngày |
| **Engine** | `RunTier2()` → full report | `TimeBoundedDiffMissingFromShadow()` |
| **Output** | missing + mismatched + missing_from_src | Chỉ missing (thiếu ở shadow) |
| **Heal path** | `FetchAndWriteByIDs()` trực tiếp | `FetchAndWriteByIDs()` trực tiếp |
| **Report update** | ✅ Cập nhật healed_at, count, duration | ❌ Không cập nhật report |

---

### So Sánh Tổng Quan: Background Heal vs Interactive Heal

| Tiêu chí | Background Heal | Interactive Execute Heal |
|---|---|---|
| **NATS Subject** | `cdc.cmd.recon-heal` | `cdc.cmd.execute-heal` |
| **API Route** | `POST /reconciliation/heal/:table` | `POST /reconciliation/execute-heal` |
| **Trigger** | Nút "Chữa lành" trên FE | Nút "Thực thi chữa lành" trên FE |
| **Chạy đối soát?** | ✅ Có (RunTier2/RunSegmentBFor/TimeBoundedDiff) | ❌ Không — chỉ dùng report đã có |
| **Input** | `{table, segment, mode, lookback, start/end_time}` | `{table, report_ids[], checkboxes}` |
| **Modes** | Window (hot/cold) + Full-diff (time-range) | Không có mode — lấy từ report |
| **Granularity** | Toàn bộ drift phát hiện | Chọn cụ thể report + loại action |
| **Safety Gate** | `healThresholdBlocked()` — max 1000 IDs / 5% drift | Không (user đã chọn thủ công) |
| **Handler** | `HandleReconHeal()` | `HandleExecuteHeal()` |
| **File Worker** | [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go) | [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go) |

---

## ⚠️ Known Issues (TODO)

> [!WARNING]
> Hai điểm TODO chưa implement trong code hiện tại:

1. **Segment A — Prune Missing Src**: `executeHealSegA()` line 148-158 — chỉ log count, chưa thực thi SQL `UPDATE _deleted = true` trên Shadow DB.
2. **Segment B — Prune Orphan in Master**: `executeHealSegB()` line 216-225 — chỉ log count, chưa thực thi SQL `UPDATE _deleted = true` trên Master DB.

---

## Schema: `cdc_reconciliation_report`

| Column | Type | Mô tả |
|---|---|---|
| `id` | BIGSERIAL | PK |
| `target_table` | TEXT | Tên bảng đối soát |
| `segment` | TEXT | `source_shadow` / `shadow_master` |
| `source_db` | TEXT | Qualified name của source |
| `missing_count` | INT | Số bản ghi thiếu ở đích |
| `stale_count` | INT | Số bản ghi lệch timestamp |
| `orphan_count` | INT | Số bản ghi thừa ở đích |
| `missing_ids` | JSONB | `["id1","id2",...]` |
| `stale_ids` | JSONB | SegA: `{mismatched:[],missing_from_src:[]}` / SegB: `{stale_ids:[],orphan_in_master:[]}` |
| `source_count` | BIGINT | Tổng bản ghi nguồn |
| `checked_at` | TIMESTAMPTZ | Thời điểm check |
| `healed_at` | TIMESTAMPTZ | Thời điểm heal (NULL = chưa heal) |
| `status` | TEXT | State machine (xem bên dưới) |
| `healed_mismatched_count` | INT | Số bản ghi mismatched đã heal |
| `healed_mismatched_duration_ms` | INT | Thời gian heal mismatched (ms) |
| `healed_missing_dest_count` | INT | Số bản ghi missing dest đã heal |
| `healed_missing_dest_duration_ms` | INT | Thời gian heal missing dest (ms) |
| `pruned_missing_src_count` | INT | Số bản ghi pruned |
| `pruned_missing_src_duration_ms` | INT | Thời gian prune (ms) |

### State Machine — `status` Column

```mermaid
stateDiagram-v2
    [*] --> ok : Recon clean (no drift)
    [*] --> drift : Recon found differences
    drift --> healing : ClaimForHealing() thành công
    healing --> healed : Heal hoàn tất
    healing --> drift : ReleaseHealClaim() (heal fail/crash)
    ok --> [*]
    healed --> [*]
```

| Status | Mô tả | Transition từ |
|---|---|---|
| `ok` | Report sạch, không có drift | Recon check |
| `drift` | Phát hiện missing/stale/orphan, chờ heal | Recon check |
| `healing` | Đang được 1 worker xử lý (race lock) | `ClaimForHealing()` |
| `healed` | Heal hoàn tất | `executeHeal()` / `healSegmentA/B()` |

> **Lưu ý:** `healing` status ngăn luồng thứ 2 (Background / Interactive) can thiệp cùng report. Nếu worker crash giữa chừng, status vẫn là `healing` → cần mechanism cleanup (manual hoặc TTL-based revert) trong tương lai.

