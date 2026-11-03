---
tier: T2
suffix: daily_flow_audit
workspace: audit-daily-flow-vs-daily-count-2026-10-27
mode: standard
created_at: 2026-10-27T09:53:30+07:00
---

# 01. REQUIREMENTS — Audit Bảng Daily Flow CDC vs Báo Cáo Đối Soát Hằng Ngày (`trans-his` / `cdc_transaction_history`)

## 1. Bối cảnh & Yêu cầu của User
User cung cấp 2 nguồn số liệu thực tế của luồng đồng bộ `trans-his` (`cdc_transaction_history` & `cdc_transaction_history_payload`) từ ngày `2026-10-14` đến `2026-10-27`:

1. **Bảng Daily Flow (Luồng CDC theo ngày trên UI):**
   - `2026-10-27`: Kafka `5,285` (`2,578 c`, `2,707 u`) | Shadow `5,285 Ops (2,589 dòng DB)` | Master `5,285 Ops (5,285 dòng DB)` (`2,578 Inserted`, `2,707 Updated`) -> `CÂN BẰNG`
   - `2026-10-26`: Kafka `24,990` (`12,300 c`, `12,690 u`) | Shadow `24,990 Ops (12,338 dòng DB)` | Master `24,992 Ops (24,992 dòng DB)` (`12,301 Inserted`, `12,691 Updated`, lệch `+2 Ops`) -> `CÂN BẰNG`
   - `2026-10-25`: Kafka `29,025` (`14,459 c`, `14,566 u`) | Shadow `29,025 Ops (14,485 dòng DB)` | Master `29,025 Ops (29,025 dòng DB)` -> `CÂN BẰNG`
   - `2026-10-24`: Kafka `26,690` (`13,185 c`, `13,505 u`) | Shadow `26,690 Ops (13,231 dòng DB)` | Master `26,690 Ops (26,690 dòng DB)` -> `CÂN BẰNG`
   - `2026-10-23`: Kafka `30,275` (`14,910 c`, `15,365 u`) | Shadow `30,275 Ops (14,984 dòng DB)` | Master `30,275 Ops (30,275 dòng DB)` -> `CÂN BẰNG`
   - `2026-10-22`: Kafka `22,448` (`11,019 c`, `11,429 u`) | Shadow `22,448 Ops (11,078 dòng DB)` | Master `22,448 Ops (22,448 dòng DB)` -> `CÂN BẰNG`
   - `2026-10-21`: Kafka `22,994` (`11,276 c`, `11,718 u`) | Shadow `22,994 Ops (11,337 dòng DB)` | Master `22,992 Ops (22,992 dòng DB)` (`11,275 Inserted`, `11,717 Updated`, thiếu `2 Ops`) -> `CẢNH BÁO`
   - `2026-10-20`: Kafka `22,184` (`10,908 c`, `11,276 u`) | Shadow `22,184 Ops (10,949 dòng DB)` | Master `22,184 Ops (22,184 dòng DB)` -> `CÂN BẰNG`
   - `2026-10-19`: Kafka `26,476` (`13,051 c`, `13,425 u`) | Shadow `26,476 Ops (13,104 dòng DB)` | Master `69,843 Ops (69,843 dòng DB)` (`51,468 Inserted`, `18,375 Updated`) -> vẫn hiện `CÂN BẰNG`!
   - `2026-10-18`: Kafka `30,601` (`15,240 c`, `15,361 u`) | Shadow `30,601 Ops (15,289 dòng DB)` | Master `0 Ops (0 dòng DB)` -> `CẢNH BÁO`
   - `2026-10-17`: Kafka `28,389` (`14,049 c`, `14,340 u`) | Shadow `28,389 Ops (14,096 dòng DB)` | Master `0 Ops (0 dòng DB)` -> `CẢNH BÁO`
   - `2026-10-16`: Kafka `31,928` (`15,760 c`, `16,168 u`) | Shadow `31,928 Ops (15,833 dòng DB)` | Master `13,470 Ops (13,470 dòng DB)` (`6,629 Inserted`, `6,841 Updated`) -> `CẢNH BÁO`
   - `2026-10-15`: Kafka `17,077` (`4,610 c`, `4,734 u` -> cộng lại mới là `9,344`, mất `7,733` ops!) | Shadow `17,077 Ops (12,375 dòng DB)` (`4,610 c`, `4,734 u`) | Master `17,082 Ops (17,082 dòng DB)` (`12,304 Inserted`, `4,778 Updated`) -> `CÂN BẰNG`
   - `TỔNG CỘNG`: Kafka `318,362` (`153,345 c`, `157,284 u`) | Shadow `318,362 Ops (161,688 dòng)` | Master `284,286 Ops (284,286 dòng)` (`161,036 Inserted`, `123,250 Updated`) -> `CẢNH BÁO`

2. **Báo Cáo Đối Soát Dữ Liệu Hằng Ngày (Source `trans-his` vs Master `cdc_transaction_history` & `cdc_transaction_history_payload`):**
   - `2026-10-18`: `15,239` vs `15,239` (Khớp 100%)
   - `2026-10-19`: `13,044` vs `13,044` (Khớp 100%)
   - `2026-10-20`: `10,906` vs `10,906` (Khớp 100%)
   - `2026-10-21`: `11,275` vs `11,275` (Khớp 100%)
   - `2026-10-22`: `11,019` vs `11,019` (Khớp 100%)
   - `2026-10-25`: `14,452` vs `14,452` (Khớp 100%)
   - `2026-10-26`: `12,297` vs `12,297` (Khớp 100%)

## 2. Mục tiêu Audit
- Điều tra toàn diện code (`cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`) kèm `file:line` để giải thích chính xác **tất cả các bất thường đang xảy ra** giữa Bảng Daily Flow và Báo cáo Count theo ngày.
- Giữ nghiêm ngặt chế độ **Propose-Only** (tuyệt đối không tự ý sửa code trước khi User `APPROVE`).
