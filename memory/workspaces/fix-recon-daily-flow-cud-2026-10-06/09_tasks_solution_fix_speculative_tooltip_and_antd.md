# 09 — Solution: Loại bỏ hoàn toàn suy diễn số liệu trong UI Tooltip & Chuẩn hóa AntD v6

Ngày: 2026-10-12
Tác giả: Brain (Chairman & Architect)
Trạng thái: **PROPOSE ONLY — Chờ User APPROVE**

---

## 1. Bản chất vi phạm & Phân tích Gốc rễ

### A. Sự cố: Tooltip tự suy diễn số liệu (Cheat / Phỏng đoán)
- **Hiện tượng vi phạm:** Trong component `MasterUpdatedCell`, code tự tính `absorbed = shadowU - masterUpdated` (ví dụ: 40 - 13 = 27), sau đó tự bịa ra câu giải thích khẳng định như thật trong Tooltip:
  > *"40 event update ở Shadow. 13 dòng đã có sẵn ở Master nên được update; 27 event còn lại rơi vào dòng mới tạo trong cùng mẻ nên đã nằm trong cột Inserted (giá trị mới nhất)."*
- **Bản chất sai trái:**
  1. Frontend **hoàn toàn không có dữ liệu telemetry** chứng minh 27 event đó là rơi vào dòng mới tạo trong cùng mẻ.
  2. 27 event đó hoàn toàn có thể là:
     - 27 update xảy ra trên **cùng một bản ghi PK cũ** (ví dụ 1 record bị update liên tục 28 lần $\rightarrow$ Master chỉ ghi 1 update cuối cùng).
     - Update bị skip do điều kiện version OCC (`_source_ts` cũ hơn record đã có trên Master).
     - Update bị lọc bỏ hoặc rơi rụng do bug nghiệp vụ.
  3. Việc tự tiện đưa một kịch bản lý thuyết suy đoán vào Tooltip của UI là hành vi **LỪA DỐI NGƯỜI DÙNG**, vi phạm trực tiếp Hiến pháp Rule #0 ("không đoán mò, không suy diễn") và Lesson `[2026-10-09]`.

---

## 2. Giải pháp kỹ thuật chuẩn tắc (The Single Best Approach)

### Nguyên tắc cốt lõi: Truth in Telemetry (Chỉ hiển thị sự thật đo lường)
1. **Xóa bỏ vĩnh viễn** hai component phỏng đoán `MasterTotalCell` và `MasterUpdatedCell`.
2. **Đồng bộ hóa đối xứng 100% giữa Khối 3 (Shadow) và Khối 4 (Master)**:
   - Cột "Tổng xử lý":
     - Dòng 1: `X Ops` (tổng số thao tác xử lý từ stream)
     - Dòng 2: `(Y dòng DB)` (số dòng vật lý thực tế biến đổi/ghi trong DB)
   - Các cột chi tiết (Inserted, Updated, Skipped): Hiển thị đúng con số đo lường thô từ API backend, không kèm bất kỳ suy diễn phỏng đoán nào.
3. Muốn biết dữ liệu có bị mất hay lệch: Nhìn vào **Khối 5 (Đối soát & DLQ)** do worker kiểm toán chuyên trách báo cáo (`recon.missing`, `recon.stale`, `recon.dlq`).

---

## 3. Chi tiết mã nguồn cần thay đổi trong `ReconPipelineGrid.tsx`

### Thay đổi 1: Sửa Khối 4 trong Table columns (Dòng 1344 - 1357)
```tsx
// 4. GHI VÀO DB MASTER (Từ CDC Stream)
{
  title: '4. GHI VÀO DB MASTER (Từ CDC Stream)',
  children: [
    {
      title: 'Tổng xử lý',
      dataIndex: ['master', 'total'],
      width: 140,
      align: 'right',
      sorter: (a, b) => a.master.total - b.master.total,
      render: (_v: number, r: DailyFlowRecord) => !pipeline?.masterName ? <Text type="secondary">—</Text> : (
        <div>
          <b>{fmtNum(r.master.total)} Ops</b>
          <div style={{ fontSize: 10, color: '#8c8c8c' }}>({fmtNum(r.master.written)} dòng DB)</div>
        </div>
      ),
    },
    {
      title: 'Inserted',
      dataIndex: ['master', 'inserted'],
      width: 80,
      align: 'right',
      render: (v: number) => !pipeline?.masterName ? <Text type="secondary">—</Text> : <span style={{ color: '#52c41a' }}>{fmtNum(v)}</span>
    },
    {
      title: 'Updated',
      dataIndex: ['master', 'updated'],
      width: 80,
      align: 'right',
      render: (v: number) => !pipeline?.masterName ? <Text type="secondary">—</Text> : <span style={{ color: '#faad14' }}>{fmtNum(v)}</span>
    },
    {
      title: 'Skipped',
      dataIndex: ['master', 'skipped'],
      width: 75,
      align: 'right',
      render: (v: number) => !pipeline?.masterName ? <Text type="secondary">—</Text> : <Text type="secondary">{fmtNum(v)}</Text>
    },
  ],
},
```

### Thay đổi 2: Sửa Khối 4 trong Table Summary Row (Dòng 1443 - 1451)
```tsx
{/* Khối 4: MASTER */}
<Table.Summary.Cell index={11} align="right">
  {pipeline?.masterName ? (
    <div>
      <b>{fmtNum(totals.masterTotal)} Ops</b>
      <div style={{ fontSize: 10, color: '#8c8c8c' }}>({fmtNum(totals.masterWritten)} dòng DB)</div>
    </div>
  ) : <Text type="secondary">—</Text>}
</Table.Summary.Cell>
<Table.Summary.Cell index={12} align="right">
  {pipeline?.masterName ? <span style={{ color: '#52c41a' }}>{fmtNum(totals.masterInserted)}</span> : <Text type="secondary">—</Text>}
</Table.Summary.Cell>
<Table.Summary.Cell index={13} align="right">
  {pipeline?.masterName ? <span style={{ color: '#faad14' }}>{fmtNum(totals.masterUpdated)}</span> : <Text type="secondary">—</Text>}
</Table.Summary.Cell>
<Table.Summary.Cell index={14} align="right">
  {pipeline?.masterName ? <Text type="secondary">{fmtNum(totals.masterSkipped)}</Text> : <Text type="secondary">—</Text>}
</Table.Summary.Cell>
```

### Thay đổi 3: Xóa bỏ 2 hàm suy diễn (Dòng 1494 - 1533)
Xóa bỏ hoàn toàn:
- `function MasterTotalCell(...) { ... }`
- `function MasterUpdatedCell(...) { ... }`

### Thay đổi 4: Sửa 3 điểm Deprecation Ant Design v6
1. Dòng 673: `<Space orientation="vertical" style={{ width: '100%' }} size={20}>`
2. Dòng 415 & 1264: `destroyOnHidden` cho `Modal`
3. Dòng 1958-1959: `size={1250}` và `destroyOnHidden` cho `Drawer`
