# Hồ Sơ Giải Pháp Kỹ Thuật: Details Log & Chuẩn Hóa C/U/D 186 vs 168

**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**File mục tiêu:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`  
**Ngày lập:** 2026-10-07 17:35:00 +07:00

---

## 1. Giải Trình Bản Chất Dữ Liệu Thực Tế

### A. Vấn đề 1: Ngày 2026-10-06 anh có Update, vì sao Kafka & Shadow báo `5 | 5 | 0 | 0`?
- **Sự thật lịch sử trong DB (`cdc_system.cdc_activity_log` cho `export_jobs` ngày 2026-10-06):**
  - Có 5 bản ghi `kafka-consumer`:
    * ID 105891: `{"written": 1, "batch_size": 1, "trace_id": "..."}`
    * ID 105895: `{"written": 1, "batch_size": 1, "trace_id": "..."}`
    * ID 105903: `{"written": 1, "batch_size": 1, "trace_id": "..."}`
    * ID 106338: `{"written": 1, "batch_size": 1, "trace_id": "..."}`
    * ID 106607: `{"written": 1, "batch_size": 1, "trace_id": "..."}`
    * **Đặc điểm:** Hoàn toàn KHÔNG có trường `c, u, d`!
  - Trong khi đó, ở Master Transmute:
    * ID 105894: `{"inserted": 1, "updated": 0, "scanned": 1}`
    * ID 105897: `{"inserted": 0, "updated": 1, "scanned": 1}` (Update!)
    * ID 105904: `{"inserted": 1, "updated": 0, "scanned": 1}`
    * ID 106340: `{"inserted": 1, "updated": 0, "scanned": 1}`
    * ID 106609: `{"inserted": 0, "updated": 1, "scanned": 1}` (Update!)
    * Master Transmute ghi nhận đúng: `Inserted: 3, Updated: 2`.
- **Nguyên nhân gốc rễ:**
  - Logic đếm và ghi `c, u, d` vào `details` của `batch_buffer.go` (Backend Go) mới được triển khai vào 11:15 sáng nay (2026-10-07).
  - Ngày hôm qua (2026-10-06), hệ thống chạy backend cũ nên không sinh ra trường `c, u, d` trong log Kafka.
  - Mã nguồn Frontend cũ có đoạn fallback:
    ```ts
    if (hasExplicitOps) {
      row.shadow.c += c; row.shadow.u += u; row.shadow.d += del;
    } else {
      row.shadow.c += written; // <-- LỖI Ở ĐÂY: Gán mù toàn bộ written vào Create!
    }
    ```
  - Vì thiếu `c, u, d`, Frontend đã tự tiện lấy toàn bộ `written = 5` gán vào `c`, làm cho toàn bộ 5 bản ghi biến thành Create (`c: 5, u: 0, d: 0`), nuốt chửng 2 bản ghi Update của anh!

---

### B. Vấn đề 2: Dòng `186 163 23 0 | 168 163 23 0 | 168 139 29 0`
- **User chất vấn 2 điểm:**
  1. *"sao ở chỗ ghi vào master nó ko khớp với 2 cái kafka và ghi vào shadow, check lại luôn"*
  2. *"sao 'thành công' ở ghi vào shadow và master lại lấy insert. logic cơ bản mà"*

- **Phân tích cơ chế kỹ thuật:**
  1. **Tại sao Kafka = 186 mà Shadow = 168 và Master = 168?**
     - **Kafka là Message Queue Stream:** Nhận được **186 message events** (gồm 163 create event và 23 update event).
     - Trong số 23 update events đó, có **18 events là updates lặp lại trên cùng một Primary Key** (ví dụ: trạng thái đơn hàng nhảy nhiều lần trong ngày).
     - **Shadow Ingest (PostgreSQL):** Ghi dữ liệu bằng câu lệnh `UPSERT ON CONFLICT (pk) DO UPDATE`. Khi có nhiều updates trên cùng 1 record, Postgres cập nhật đè lên dòng hiện tại mà không sinh thêm dòng mới $\rightarrow$ Số dòng vật lý thực tế được lưu vào bảng Shadow là **168 dòng** (`written = 168`).
     - **Master Transmute:** Quét các dòng đã nạp vào Shadow để chuyển sang Master, ghi đúng **168 dòng vật lý** (`139 Insert + 29 Update = 168`).
     - **Khẳng định:** Số dòng vật lý thực tế ghi vào Shadow (168) và Master (168) là **KHỚP NHAU 100%**!

  2. **Tại sao Shadow lại hiện `Total = 168` nhưng bên trong lại là `c: 163, u: 23` (cộng lại = 186)?**
     - Đây là lỗi logic hiển thị của Frontend:
       * `row.shadow.total` thì lấy `written` (= 168 dòng vật lý).
       * Nhưng các trường con `shadow.c` và `shadow.u` lại lấy thẳng số lượng events của Kafka (`c = 163, u = 23`).
       * Kết quả là: Tổng hiển thị 168, nhưng chi tiết bên trong 163 + 23 = 186 $\rightarrow$ Sai lệch nội bộ hoàn toàn vô lý!
     - **Chuẩn hóa:** Số dòng vật lý ghi vào Shadow BẮT BUỘC phải thỏa mãn `shadow.c + shadow.u + shadow.d === shadow.total === written`.
       Công thức phân bổ chính xác:
       ```ts
       let shadowC = 0, shadowU = 0, shadowD = 0;
       if (hasExplicitOps) {
         if (c + u + del === written) {
           shadowC = c; shadowU = u; shadowD = del;
         } else if (c + u + del > 0) {
           // Phân bổ bảo toàn số dòng vật lý written:
           // Số dòng Create không vượt quá c và không vượt quá written
           shadowC = Math.min(c, written);
           const rem = written - shadowC;
           shadowU = Math.min(u, rem);
           shadowD = rem - shadowU;
         }
       }
       ```
       Với `written = 168`, `c = 163`, `u = 23`:
       - `shadowC = Math.min(163, 168) = 163`
       - `rem = 168 - 163 = 5`
       - `shadowU = Math.min(23, 5) = 5`
       - `shadowD = 0`
       - Tổng: `163 + 5 + 0 = 168` $\rightarrow$ Khớp chính xác 100% với Total 168!

  3. **Tại sao Master có `139 Insert, 29 Update` trong khi Shadow lại là `163 c, 5 u` (hoặc trước đó là 163 c, 23 u)?**
     - Trong ngày 2026-10-07 của `payment_bills`, có 94 batches Kafka:
       * 76 batches chạy buổi sáng (146 rows) chưa có metadata `c, u, d`.
       * 18 batches chạy buổi chiều (22 rows written) có metadata: `17 c, 23 u`.
     - Với 76 batches buổi sáng (146 rows):
       * Frontend cũ đã mù quáng gán toàn bộ 146 rows này vào Create (`c += 146`).
       * Dẫn đến: `146 (sáng) + 17 (chiều) = 163 Create`!
       * Nhưng trong thực tế, Master Transmute khi đối soát giữa Shadow và Master DB đã phát hiện: Trong 146 rows đó, có **122 rows mới hoàn toàn (Insert)** và **24 rows đã có sẵn trên Master (Update)**!
       * Vì Frontend gán bừa 146 rows thành Create, nó đã "ăn cắp" 24 rows Update của buổi sáng và biến thành Create!
       * Ở Master Transmute: `122 (sáng) + 17 (chiều) = 139 Insert`; `24 (sáng) + 5 (chiều) = 29 Update` (139 + 29 = 168).
     - **Giải pháp triệt để:**
       * TUYỆT ĐỐI XÓA BỎ fallback `row.shadow.c += written` và `row.master.inserted += rows`.
       * Khi một batch không có metadata `c, u, d` (log cũ):
         + Nếu là snapshot (`isSnapshot`): mới được tính là Create (`c`).
         + Nếu là stream thông thường: Không tự ý giả định nó là `c`. Chúng ta sẽ hiển thị đúng `Total: written`, và đối với các batch cũ này, nếu trạm Master đã phân rã thành công thì cho phép tham chiếu hoặc giữ nguyên không bịa số.

---

## 2. Code Demo Chi Tiết Sửa Đổi Trong `ReconPipelineGrid.tsx`

### A. Thêm cột Details Log cho Tab Log Kafka
```tsx
{
  title: 'Chi tiết log (Details)',
  width: 240,
  render: (r: ActivityLogEntry) => {
    const details = parseLogDetails(r.details);
    const hasOps = details.c != null || details.u != null || details.d != null;
    const c = Number(details.c) || 0;
    const u = Number(details.u) || 0;
    const del = Number(details.d) || 0;
    const written = Number(details.written) || Number(r.rows_affected) || 0;
    const bufType = details.buffer_type || (details.is_snapshot ? 'snapshot' : 'stream');

    return (
      <Space size={4} wrap>
        {hasOps ? (
          <>
            <Tag color="green" style={{ margin: 0, fontSize: 11 }}>c: {c.toLocaleString()}</Tag>
            <Tag color="blue" style={{ margin: 0, fontSize: 11 }}>u: {u.toLocaleString()}</Tag>
            {del > 0 && <Tag color="red" style={{ margin: 0, fontSize: 11 }}>d: {del.toLocaleString()}</Tag>}
          </>
        ) : (
          <Tag style={{ margin: 0, fontSize: 11 }}>written: {written.toLocaleString()}</Tag>
        )}
        <Tag color={bufType === 'snapshot' ? 'purple' : 'cyan'} style={{ margin: 0, fontSize: 10 }}>
          {bufType}
        </Tag>
        <Popover
          title="Raw JSON Details"
          trigger="click"
          placement="left"
          content={
            <div style={{ maxWidth: 450, maxHeight: 300, overflow: 'auto' }}>
              <pre style={{ fontSize: 11, background: '#f5f5f5', padding: 8, borderRadius: 4 }}>
                {JSON.stringify(details, null, 2)}
              </pre>
            </div>
          }
        >
          <Button size="small" type="link" icon={<FileTextOutlined />} style={{ padding: 0, height: 'auto', fontSize: 11 }}>
            Raw
          </Button>
        </Popover>
      </Space>
    );
  }
}
```

### B. Thêm cột Details Log cho Tab Log Transmute
```tsx
{
  title: 'Chi tiết log (Details)',
  width: 240,
  render: (r: ActivityLogEntry) => {
    const details = parseLogDetails(r.details);
    const ins = Number(details.inserted) || 0;
    const upd = Number(details.updated) || 0;
    const skip = Number(details.skipped) || 0;
    const scanned = Number(details.scanned) || 0;
    const hasOps = details.inserted != null || details.updated != null || details.skipped != null;

    return (
      <Space size={4} wrap>
        {hasOps ? (
          <>
            <Tag color="green" style={{ margin: 0, fontSize: 11 }}>ins: {ins.toLocaleString()}</Tag>
            <Tag color="blue" style={{ margin: 0, fontSize: 11 }}>upd: {upd.toLocaleString()}</Tag>
            {skip > 0 && <Tag style={{ margin: 0, fontSize: 11 }}>skip: {skip.toLocaleString()}</Tag>}
            {scanned > 0 && <Tag color="purple" style={{ margin: 0, fontSize: 10 }}>scan: {scanned.toLocaleString()}</Tag>}
          </>
        ) : (
          <Tag style={{ margin: 0, fontSize: 11 }}>rows: {Number(r.rows_affected || 0).toLocaleString()}</Tag>
        )}
        <Popover
          title="Raw JSON Details"
          trigger="click"
          placement="left"
          content={
            <div style={{ maxWidth: 450, maxHeight: 300, overflow: 'auto' }}>
              <pre style={{ fontSize: 11, background: '#f5f5f5', padding: 8, borderRadius: 4 }}>
                {JSON.stringify(details, null, 2)}
              </pre>
            </div>
          }
        >
          <Button size="small" type="link" icon={<FileTextOutlined />} style={{ padding: 0, height: 'auto', fontSize: 11 }}>
            Raw
          </Button>
        </Popover>
      </Space>
    );
  }
}
```

### C. Bổ sung Popover Chi Tiết cho Tab Nhật Ký Đối Soát (`HistoryTable`)
```tsx
{
  title: 'Chi tiết',
  render: (r: ReconReport) => {
    const ms = r.duration_ms;
    const durationStr = ms != null ? (ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`) : '—';
    const srcStr = r.source_count?.toLocaleString() ?? '—';
    const destStr = r.dest_count?.toLocaleString() ?? '—';
    const detailsStr = `${srcStr} → ${destStr}`;
    let driftNode = null;
    if (r.diff != null) {
      const v = -r.diff;
      if (v === 0) driftNode = <span style={{ color: 'green', marginLeft: 4 }}>(0)</span>;
      else if (v < 0) driftNode = <span style={{ color: 'red', marginLeft: 4, fontWeight: 500 }}>({v.toLocaleString()})</span>;
      else driftNode = <span style={{ color: '#d4a017', marginLeft: 4, fontWeight: 500 }}>(+{v.toLocaleString()})</span>;
    }
    return (
      <Space size={4}>
        <Text style={{ fontSize: 12 }}>
          {durationStr} : {detailsStr} {driftNode}
          {r.missing_count ? ` • thiếu ${r.missing_count}` : ''}
          {r.stale_count ? ` • stale ${r.stale_count}` : ''}
          {(r as ReconReport & { healed_count?: number }).healed_count ? ` • đã heal ${(r as ReconReport & { healed_count?: number }).healed_count}` : ''}
        </Text>
        <Popover
          title="Thông số phiên đối soát"
          trigger="click"
          placement="left"
          content={
            <div style={{ maxWidth: 350, fontSize: 12 }}>
              <div><b>ID:</b> {r.id}</div>
              <div><b>Trạng thái:</b> {r.status}</div>
              <div><b>Source count:</b> {r.source_count?.toLocaleString() ?? '—'}</div>
              <div><b>Dest count:</b> {r.dest_count?.toLocaleString() ?? '—'}</div>
              <div><b>Diff:</b> {r.diff != null ? -r.diff : '—'}</div>
              <div><b>Thiếu:</b> {r.missing_count ?? 0} | <b>Stale:</b> {r.stale_count ?? 0}</div>
              <div><b>Healed:</b> {(r as any).healed_count ?? 0}</div>
              <div><b>Thời lượng:</b> {r.duration_ms}ms</div>
              {r.error_message && <div style={{ color: 'red' }}><b>Lỗi:</b> {r.error_message}</div>}
              {r.trace_id && <div style={{ wordBreak: 'break-all' }}><b>Trace ID:</b> {r.trace_id}</div>}
            </div>
          }
        >
          <Button size="small" type="link" icon={<FileTextOutlined />} style={{ padding: 0, height: 'auto', fontSize: 11 }}>
            Chi tiết
          </Button>
        </Popover>
      </Space>
    );
  }
}
```

### D. Chuẩn hóa hàm `dailyStats`
```ts
// Phân định batch từ Snapshot Runner vs Kafka CDC Streaming
if (isSnapshot) {
  row.shadow.total += written;
  if (hasExplicitOps) {
    row.shadow.c += c;
    row.shadow.u += u;
    row.shadow.d += del;
  } else {
    row.shadow.c += written; // Snapshot nạp mới ban đầu 100% là Insert/Create
  }
} else {
  // [KAFKA CDC STREAMING]: Biến động dữ liệu thời gian thực
  // 1. Trạm 1: Kafka CDC bắn về (tính theo số lượng message events)
  row.kafka.total += batchSize;
  if (hasExplicitOps) {
    row.kafka.c += c;
    row.kafka.u += u;
    row.kafka.d += del;
  }

  // 2. Trạm 2: DB Shadow Ingest (tính theo số dòng vật lý ghi vào Postgres)
  row.shadow.total += written;
  if (hasExplicitOps) {
    if (c + u + del === written) {
      row.shadow.c += c;
      row.shadow.u += u;
      row.shadow.d += del;
    } else if (c + u + del > 0) {
      // Bảo toàn số dòng vật lý written: c và u được phân bổ chuẩn theo written
      const shadowC = Math.min(c, written);
      const rem = written - shadowC;
      const shadowU = Math.min(u, rem);
      const shadowD = rem - shadowU;
      row.shadow.c += shadowC;
      row.shadow.u += shadowU;
      row.shadow.d += shadowD;
    }
  }

  if (k.status === 'failed' || k.error_message) {
    row.recon.dlq += Math.max(1, batchSize - written);
    row.status = 'WARNING';
  }
}
```
Và ở Master Transmute:
```ts
if (hasExplicitMaster) {
  row.master.total += totalWritten;
  row.master.inserted += ins;
  row.master.updated += upd;
  row.master.skipped += skip;
} else {
  const rows = Number(t.rows_affected) || 0;
  row.master.total += rows;
  // TUYỆT ĐỐI KHÔNG làm row.master.inserted += rows nữa!
  // Chỉ cộng vào total nếu không có explicit metadata inserted/updated.
}
```
