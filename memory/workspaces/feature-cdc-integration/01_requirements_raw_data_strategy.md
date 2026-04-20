# Requirements: Raw Data Strategy — 100% không miss dữ liệu

> Date: 2026-04-14
> Phase: raw_data_strategy
> Triggered by: User yêu cầu `_raw_data` phải chứa đầy đủ data từ source, kể cả field mới mà destination chưa có

## Bối cảnh

- Data source: 10-50 triệu records
- `_raw_data` hiện tại dùng `to_jsonb(*)` = chỉ pack columns destination đã có
- Field mới từ source mà destination chưa có → MISS trong `_raw_data`
- User yêu cầu: 100% không miss bất kỳ dữ liệu nào

## Yêu cầu

### R1: `_raw_data` phải chứa toàn bộ data gốc từ source
- Kể cả field mới mà destination chưa có column
- Không phụ thuộc vào Airbyte propagate schema

### R2: Performance phải chấp nhận được cho 50M rows
- Ghi không gây lock table
- Query trên typed columns vẫn nhanh

### R3: Không miss dữ liệu khi Airbyte overwrite
- Airbyte mode `full_refresh + overwrite` → xoá + ghi lại data
- Sau overwrite, `_raw_data` phải được populate lại

## Phân tích các option (Brain)

### Option A: Airbyte raw mode song song
- Thêm 1 connection raw mode → ghi JSON gốc vào table prefix `raw_`
- Typed connection giữ nguyên (performance)
- `_raw_data` copy từ raw table
- **Pro**: JSON gốc 100% từ source, không miss
- **Con**: 2x connections, 2x storage, quản lý phức tạp

### Option B: Giữ typed mode + `to_jsonb(*)` + chấp nhận delay
- `_raw_data` = snapshot destination (thiếu field mới cho đến khi Airbyte sync)
- **Pro**: Đơn giản
- **Con**: MISS field mới trong khoảng chờ — KHÔNG đáp ứng R1

### Option C: Airbyte đổi sang raw mode hoàn toàn
- 1 connection, Airbyte ghi `_airbyte_data JSONB`
- CDC system extract typed columns từ `_airbyte_data`
- **Pro**: JSON gốc 100%, 1 connection
- **Con**: Query chậm hơn trên JSONB, cần GIN index, transform layer phải hoạt động tốt

### Option D: Typed mode + MongoDB Change Stream trực tiếp
- Giữ Airbyte typed mode cho batch sync
- Thêm Go Worker đọc MongoDB change stream → ghi raw JSON vào `_raw_data` realtime
- **Pro**: Realtime, 100% không miss
- **Con**: Phức tạp, cần Debezium hoặc custom Go consumer

## Chờ User chọn option
