# Requirements: Data Flow Core — BÀI TOÁN GỐC CHƯA GIẢI

> Date: 2026-04-14
> Phase: data_flow_core
> Priority: P0 — QUAN TRỌNG NHẤT, phải giải quyết trước mọi thứ khác

## Vấn đề

Phase 1 CDC yêu cầu: dữ liệu từ source (MongoDB) phải vào hệ thống CDC đầy đủ, 100% không miss.

**Hiện trạng**: CHƯA CÓ GIẢI PHÁP HOẠT ĐỘNG.

- Airbyte ghi typed columns vào Postgres, KHÔNG ghi `_raw_data`
- Bridge `to_jsonb(*)` chỉ pack columns destination đã có → miss field mới
- Airbyte config `propagate_columns` tự thêm column mới nhưng có delay (schedule 24h)
- Giữa 2 lần sync → field mới MISS hoàn toàn

## Yêu cầu tuyệt đối

### R1: Mọi data từ source phải vào `_raw_data` đầy đủ — kể cả field mới
### R2: Không miss bất kỳ document nào — 10M, 50M records
### R3: Performance chấp nhận được — không lock table, không timeout

## Options đã phân tích

### Option A: Airbyte raw mode song song
- Thêm 1 connection raw → JSON gốc vào table `raw_*`
- Typed connection giữ nguyên
- Pro: 100% data gốc. Con: 2x connection, 2x storage

### Option C: Airbyte đổi sang raw mode hoàn toàn
- 1 connection raw, CDC extract typed columns
- Pro: 100% data gốc, 1 connection. Con: query chậm trên JSONB

### Option D: Typed mode + MongoDB Change Stream
- Airbyte giữ typed (batch 24h)
- Go Worker kết nối MongoDB Change Stream → ghi `fullDocument` JSON gốc → `_raw_data` realtime
- Pro: realtime < 1s, 100% data gốc, field mới có ngay
- Con: phức tạp, cần MongoDB driver trong Worker

## User chưa chọn option — cần quyết định ở session mới

## Context cho session mới

### Files cần đọc
1. `agent/GEMINI.md` — core agent rules
2. `agent/memory/workspaces/feature-cdc-integration/01_requirements_data_flow_core.md` — FILE NÀY
3. `agent/memory/workspaces/feature-cdc-integration/01_requirements_raw_data_strategy.md` — phân tích chi tiết 4 options
4. `agent/memory/workspaces/feature-cdc-integration/04_decisions.md` — ADR-014 (_raw_data deferred)
5. `agent/memory/workspaces/feature-cdc-integration/00_current.md` — trạng thái hiện tại

### Airbyte config thực tế (đã verify)
- 1 destination: Postgres
- 2 sources: MongoDB
- 1 connection: MongoDb → Postgres (6 active streams, 2 non-active)
- nonBreakingChangesPreference: `propagate_columns`
- destMode per stream: `overwrite`
- schedule: 24h
- MongoDB đã bật ReplicaSet (`--replSet rs0`)

### Những gì ĐÃ hoạt động
- Airbyte sync data → typed columns trong Postgres ✅
- CMS quét streams từ Airbyte (DiscoverSchema) ✅
- CMS tạo registry entries ✅
- `ensureCDCColumns` thêm `_raw_data` column vào Airbyte tables ✅
- Activity Log + Schedule Manager ✅

### Những gì CHƯA hoạt động
- `_raw_data` populate đúng (chỉ pack destination columns, miss field mới)
- Bridge chạy chunk cho 50M rows (hiện UPDATE cả table 1 lần)
- Data flow end-to-end test pass
