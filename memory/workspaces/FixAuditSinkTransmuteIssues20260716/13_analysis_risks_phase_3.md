# Phân Tích Rủi Ro và Thiết Kế Giải Pháp - Phase 3

Tài liệu này nghiên cứu chuyên sâu và đề xuất phương án giải quyết cho hai rủi ro cơ sở hạ tầng ở chặng Transmute:
1. **TX-H3: OCC Clock Skew** (Lệch múi giờ/thời gian hệ thống làm sai lệch logic Optimistic Concurrency Control).
2. **TX-H6: FNV-1a Hash Collision** (Rủi ro va chạm hash khi sử dụng thuật toán hash 32-bit/64-bit đơn giản).

---

## 1. Rủi ro TX-H3: OCC Clock Skew (Sai lệch clock trong kiểm soát đồng thời)

### 1.1. Hiện trạng & Vấn đề
Trong module Transmuter, khi thực hiện Bulk Upsert ghi dữ liệu từ Shadow Table sang Master Table, chúng ta sử dụng mệnh đề `ON CONFLICT DO UPDATE` với điều kiện bảo vệ OCC dựa trên `_source_ts` (mốc thời gian phát sinh event từ nguồn):
```sql
INSERT INTO master_table (...) VALUES (...)
ON CONFLICT (_gpay_id) DO UPDATE SET 
    ...,
    _updated_at = NOW()
WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(master_table._source_ts, 0)
```

Cơ chế này giả định rằng:
- `_source_ts` luôn tăng tuyến tính theo đúng thứ tự xảy ra thay đổi thực tế trên nguồn dữ liệu.
- Mọi event CDC đều có timestamp đồng nhất.

Tuy nhiên, trong môi trường phân tán thực tế, giả định này bị phá vỡ bởi các yếu tố sau:
1. **Clock Skew giữa các Database Node nguồn:** Nếu nguồn là một cluster MongoDB Sharded hoặc Replica Set, clock drift giữa các primary node có thể lên tới hàng chục mili-giây. Một event xảy ra sau trên Node B có thể mang timestamp nhỏ hơn event xảy ra trước trên Node A.
2. **Network Delay & Out-of-Order Delivery:** Nếu các event được đẩy qua Kafka hoặc NATS, các partition khác nhau có thời gian truyền dẫn khác nhau. Khi transmuter xử lý song song từ nhiều worker, một event cũ có thể bị trì hoãn mạng (delay) và được ghi vào Shadow muộn hơn event mới. Mặc dù Shadow lưu trữ đúng theo `_gpay_id` hoặc cursor, nhưng khi đồng bộ sang Master, nếu thứ tự xử lý bị đảo lộn (out-of-order) ở mức vi mô, điều kiện `EXCLUDED._source_ts >= master_table._source_ts` có thể chặn các cập nhật hợp lệ (nếu event mới có timestamp vô tình nhỏ hơn do clock skew).
3. **Multi-Source Merge:** Khi một bảng Master được tổng hợp (merge/flatten) từ nhiều Shadow Table độc lập (ví dụ bảng `users` được merge từ shadow `user_profile` và shadow `user_metadata`), mỗi nguồn có một nguồn phát sinh timestamp riêng. So sánh `_source_ts` chéo giữa các nguồn không đồng nhất sẽ dẫn đến tình trạng ghi đè sai lệch dữ liệu của nhau.

### 1.2. Thiết kế Giải pháp Đề xuất

Để giải quyết triệt để rủi ro Clock Skew mà không phá vỡ hiệu năng Bulk Upsert, chúng tôi đề xuất các hướng đi sau:

#### Phương án A: Vector Clocks cho Multi-Source (Khuyên dùng cho hệ thống tích hợp)
Thay vì sử dụng một cột `_source_ts` đơn lẻ, bảng Master sẽ duy trì một trường kiểu `JSONB` đại diện cho Vector Clocks/Version Tracking của từng nguồn:
- Cột Master: `_source_clocks JSONB` (Ví dụ: `{"profile_ts": 1783564316564, "metadata_ts": 1783564312000}`).
- Khi ghi từ shadow `user_profile`, câu lệnh SQL sẽ chỉ so sánh và cập nhật key tương ứng:
```sql
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    _source_clocks = jsonb_set(COALESCE(master_table._source_clocks, '{}'::jsonb), '{profile_ts}', to_jsonb(EXCLUDED._source_ts)),
    _updated_at = NOW()
WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE((master_table._source_clocks->>'profile_ts')::bigint, 0)
```
Giải pháp này cô lập hoàn toàn clock của từng nguồn, triệt tiêu rủi ro cross-source overwrite.

#### Phương án B: Sử dụng Logical Clocks (MongoDB Oplog `optime` / Postgres LSN)
Đối với đồng bộ 1-1 từ một nguồn duy nhất (như MongoDB):
- Thay vì dùng wall-clock `_source_ts` (milli-seconds), sử dụng logical clock của engine CDC nguồn.
- Trong MongoDB, đó là trường `optime` (bao gồm timestamp + increment counter `t` và `i`). Trong Postgres, đó là `LSN` (Log Sequence Number - số nguyên 64-bit tăng liên tục không bao giờ drift).
- Ghi nhận `_source_lsn` hoặc `_source_optime` (dưới dạng bigint) vào Shadow và Master.
- Điều kiện OCC sẽ chuyển thành:
```sql
WHERE EXCLUDED._source_lsn >= master_table._source_lsn
```
Vì LSN/optime được sinh ra bởi một engine log duy nhất trên database master nguồn, nó đảm bảo tính tăng đơn điệu nghiêm ngặt (strict monotonic), loại bỏ hoàn toàn ảnh hưởng của Clock Skew hệ thống.

---

## 2. Rủi ro TX-H6: FNV-1a Hash Collision (Va chạm Hash)

### 2.1. Hiện trạng & Vấn đề
Hiện tại, trong một số module đối soát hoặc tạo khoá phân tán, chúng ta sử dụng thuật toán FNV-1a (32-bit hoặc 64-bit) để băm dữ liệu bản ghi (ví dụ băm các trường để tạo checksum so khớp xxhash).
- FNV-1a là một thuật toán hash phi mã hóa (non-cryptographic hash) rất nhanh và dễ triển khai.
- Tuy nhiên, không gian của hash 32-bit chỉ có $2^{32} \approx 4.29$ tỷ giá trị. Theo nghịch lý ngày sinh (Birthday Paradox), tỷ lệ va chạm (collision probability) sẽ vượt quá 50% chỉ sau khoảng 77,000 bản ghi khác nhau.
- Với hash 64-bit, không gian lớn hơn ($2^{64}$), tỷ lệ va chạm 50% đạt được sau khoảng 5.06 tỷ bản ghi. Tuy nhiên, đối với các hệ thống tài chính quy mô lớn xử lý hàng trăm triệu giao dịch mỗi ngày, va chạm hash 64-bit vẫn có xác suất xảy ra thực tế.
- **Hậu quả:** Khi xảy ra va chạm hash, hai bản ghi có nội dung khác nhau sẽ sinh ra cùng một checksum hash.
  - Trong đối soát (Reconciliation): Luồng Tier1/Tier2 sẽ coi là khớp dữ liệu và bỏ qua, dẫn đến lọt lưới các sai lệch dữ liệu (false-negative).
  - Trong tạo khoá tự sinh: Gây xung đột ID vật lý hoặc ghi đè nhầm bản ghi khác.

### 2.2. Thiết kế Giải pháp Đề xuất

Để loại bỏ hoàn toàn rủi ro va chạm hash mà vẫn duy trì tốc độ xử lý cao, chúng tôi đề xuất:

#### Hướng đi 1: Thay thế bằng XXHash64 / XXH3 (Tối ưu về mặt hiệu năng)
- **XXHash** (đặc biệt là XXH3 64-bit hoặc 128-bit) là thuật toán hash non-cryptographic hiện đại nhất, nhanh hơn FNV-1a rất nhiều trên các CPU Intel/ARM hiện đại nhờ tận dụng tập lệnh vector (SIMD).
- Độ phân tán của XXHash cực kỳ tốt, vượt qua tất cả các bài test chất lượng hash (SMHasher).
- Chuyển đổi mã nguồn Go sang sử dụng thư viện `github.com/cespare/xxhash/v2` (dành cho XXHash64) hoặc `github.com/zeebo/xxh3`.
- Ở tầng database (PostgreSQL), có thể cài đặt extension `pg_xxhash` hoặc lưu trữ hash dưới dạng bigint.

#### Hướng đi 2: Sử dụng SHA-256 (Tối ưu về mặt an toàn)
- Với các nghiệp vụ đối soát yêu cầu độ tin cậy tuyệt đối và có tính pháp lý (audit/recon tài chính), SHA-256 là tiêu chuẩn bắt buộc.
- SHA-256 sinh ra hash 256-bit (32 bytes). Xác suất va chạm thực tế là bằng 0.
- **Cách triển khai:**
  - Trong Go: Dùng `crypto/sha256`.
  - Lưu checksum dưới dạng chuỗi hex (64 ký tự) oặc mảng byte (`[]byte`/`bytea` trong Postgres để tiết kiệm không gian lưu trữ xuống còn 32 bytes).
  - Postgres hỗ trợ native hàm `sha256(data::bytea)` nên việc đối soát chéo bằng SQL rất thuận tiện và chính xác.
  - Sử dụng index `hash` dạng `BTREE` hoặc hash index trên Postgres để tăng tốc độ tìm kiếm.

---

## 3. Lộ Trình Triển Khai Tiếp Theo

1. **Phase 4 (Kế hoạch):**
   - Viết các test case chứng minh rủi ro va chạm hash của FNV-1a trên tập mẫu lớn.
   - Thử nghiệm tích hợp thư viện `xxh3` vào module đối soát cục bộ và đo đạc latency.
2. **Phase 5 (Kế hoạch):**
   - Thiết kế migration plan để bổ sung cột `_source_lsn` / `_source_optime` vào các bảng shadow và master có tần suất cập nhật cao.
   - Chuyển đổi dần các điều kiện OCC sang logical clock.
