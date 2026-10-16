# Hồ sơ Giải pháp Kỹ thuật: Tái cấu trúc Bảng Đối soát Tách bạch Snapshot và Luồng Kafka CDC

## 1. Mô hình Dữ liệu Chuẩn hóa (`DailyFlowRecord`)

```typescript
interface DailyFlowRecord {
  date: string;
  // 1. Khối Snapshot / Re-sync (Nạp mẻ ban đầu hoặc re-sync)
  snapshot: {
    ops: number;          // Số Ops snapshot
    shadowWritten: number;// Số dòng DB ghi vào Shadow từ snapshot
    masterWritten: number;// Số dòng DB ghi vào Master từ snapshot
    masterInserted: number;
    masterUpdated: number;
    traceIds: Set<string>;
  };
  // 2. Khối Kafka CDC Stream (Bắn về thời gian thực)
  kafka: {
    total: number;        // Tổng Ops = c + u + d
    c: number;            // Inserts
    u: number;            // Updates
    d: number;            // Deletes
  };
  // 3. Khối DB Shadow (từ CDC Stream)
  shadow: {
    total: number;        // Tổng Ops tiếp nhận từ CDC stream
    written: number;      // Dòng DB vật lý bị tác động sau ON CONFLICT
    c: number;
    u: number;
    d: number;
  };
  // 4. Khối DB Master (từ CDC Stream Transmute)
  master: {
    total: number;        // Tổng dòng xử lý (inserted + updated + skipped)
    written: number;      // Dòng DB ghi vào Master (inserted + updated)
    inserted: number;     // Dòng mới chèn vào Master
    updated: number;      // Dòng cập nhật trên Master
    skipped: number;      // Dòng bỏ qua do OCC / filter
  };
  // 5. Khối Đối soát & DLQ
  recon: {
    totalRuns: number;
    dlq: number;
    missing: number;
    stale: number;
    healed: number;
  };
  status: 'BALANCED' | 'DRIFT' | 'WARNING';
}
```

---

## 2. Chi tiết Reducer Phân loại Dữ liệu (`dailyStats`)

```typescript
  const dailyStats = useMemo(() => {
    const map = new Map<string, DailyFlowRecord>();

    const getOrCreate = (d: string): DailyFlowRecord => {
      const key = d || '—';
      if (!map.has(key)) {
        map.set(key, {
          date: key,
          snapshot: {
            ops: 0,
            shadowWritten: 0,
            masterWritten: 0,
            masterInserted: 0,
            masterUpdated: 0,
            traceIds: new Set<string>(),
          },
          kafka: { total: 0, c: 0, u: 0, d: 0 },
          shadow: { total: 0, written: 0, c: 0, u: 0, d: 0 },
          master: { total: 0, written: 0, inserted: 0, updated: 0, skipped: 0 },
          recon: { totalRuns: 0, dlq: 0, missing: 0, stale: 0, healed: 0 },
          status: 'BALANCED',
        });
      }
      return map.get(key)!;
    };

    // 1. Phân loại Log Kafka Consumer (CDC Stream vs Snapshot Buffer)
    (kafkaLog?.data || []).forEach((k) => {
      const d = (k.started_at || (k as any).created_at || '').slice(0, 10);
      if (!d) return;
      const row = getOrCreate(d);
      const details = parseLogDetails(k.details);
      const written = Number(details.written) || Number(k.rows_affected) || 0;
      
      const hasOps = details.c != null || details.u != null || details.d != null ||
                     details.inserted != null || details.updated != null || details.deleted != null;
      const c = Number(details.c) || Number(details.inserted) || 0;
      const u = Number(details.u) || Number(details.updated) || 0;
      const del = Number(details.d) || Number(details.deleted) || 0;
      const ops = hasOps ? (c + u + del) : (Number(details.batch_size) || written);
      const isFailed = k.status === 'failed' || Boolean(k.error_message);
      
      const isSnapshot = details.buffer_type === 'snapshot' || 
                         details.is_snapshot === true || 
                         k.triggered_by === 'snapshot' ||
                         k.operation === 'snapshot.v2' ||
                         k.operation === 'snapshot';

      if (details.trace_id) {
        if (isSnapshot) {
          row.snapshot.traceIds.add(details.trace_id);
        }
      }

      if (isSnapshot) {
        // TÁCH RIÊNG: Toàn bộ thông số mẻ Snapshot đi vào khối Snapshot
        row.snapshot.ops += ops;
        row.snapshot.shadowWritten += written;
      } else {
        // LUỒNG CDC STREAM THUẦN TÚY:
        row.kafka.total += ops;
        row.kafka.c += (hasOps ? c : written);
        row.kafka.u += u;
        row.kafka.d += del;

        if (!isFailed) {
          row.shadow.total += ops;
          row.shadow.written += written;
          row.shadow.c += (hasOps ? c : written);
          row.shadow.u += u;
          row.shadow.d += del;
        } else {
          row.recon.dlq += Math.max(1, ops - written);
        }
      }
    });

    // 2. Phân loại Log Master Transmute (CDC Hook vs Snapshot/Admin Transmute)
    (transmuteLog?.data || []).forEach((t) => {
      const d = (t.started_at || (t as any).created_at || '').slice(0, 10);
      if (!d) return;
      const row = getOrCreate(d);
      const details = parseLogDetails(t.details);
      const ins = Number(details.inserted) || 0;
      const upd = Number(details.updated) || 0;
      const skip = Number(details.skipped) || 0;
      const isFailed = t.status === 'failed' || Boolean(t.error_message);
      const hasExplicit = details.inserted != null || details.updated != null || details.skipped != null;
      const rows = Number(t.rows_affected) || 0;

      const isSnapshotTransmute = t.triggered_by === 'admin' ||
                                  t.triggered_by === 'snapshot' ||
                                  t.triggered_by === 'snapshot-hook' ||
                                  (details.trace_id && row.snapshot.traceIds.has(details.trace_id));

      if (isSnapshotTransmute) {
        row.snapshot.masterWritten += (ins + upd || rows);
        row.snapshot.masterInserted += ins;
        row.snapshot.masterUpdated += upd;
      } else {
        if (!isFailed) {
          if (hasExplicit) {
            row.master.total += (ins + upd + skip);
            row.master.written += (ins + upd);
            row.master.inserted += ins;
            row.master.updated += upd;
            row.master.skipped += skip;
          } else {
            row.master.total += rows;
            row.master.written += rows;
            row.master.inserted += rows;
          }
        } else {
          row.recon.dlq += 1;
        }
      }
    });

    // 3. Nguồn Deep Recon & Smoke Recon Reports
    const allReports = [...(historyRecon?.data || []), ...(history?.data || [])];
    const sortedReports = allReports
      .filter((r, idx, arr) => r.id != null && arr.findIndex((x) => x.id === r.id) === idx)
      .sort((a, b) => (a.checked_at || '').localeCompare(b.checked_at || ''));

    sortedReports.forEach((r) => {
      const d = (r.checked_at || '').slice(0, 10);
      if (!d) return;
      const row = getOrCreate(d);
      row.recon.totalRuns += 1;
      row.recon.healed += ((r as any).healed_count || 0);

      const isSmoke = r.check_type === 'smoke' || r.check_type === 'segment_b_smoke';
      if (!isSmoke || (r.missing_count || 0) > 0) {
        row.recon.missing = r.missing_count || 0;
        row.recon.stale = r.stale_count || 0;
      }
    });

    // 4. Máy trạng thái (FSM Evaluation)
    map.forEach((row) => {
      // 1. Có lỗi DLQ hoặc phiên đối soát gần nhất phát hiện mất tin -> DRIFT (Đỏ)
      if (row.recon.dlq > 0 || row.recon.missing > 0) {
        row.status = 'DRIFT';
        return;
      }
      // 2. Có bản ghi stale -> Cảnh báo
      if (row.recon.stale > 0) {
        row.status = 'WARNING';
        return;
      }
      // 3. Đánh giá luồng CDC: Nếu Kafka CDC có tin (kafka.total > 0) mà Shadow không nhận được tin nào
      if (row.kafka.total > 0 && row.shadow.total === 0) {
        row.status = 'WARNING';
        return;
      }
      // 4. Nếu Shadow CDC có ghi dòng (shadow.written > 0) mà Master CDC hoàn toàn đứng im (= 0)
      if (pipeline?.masterName && row.shadow.written > 0 && row.master.total === 0) {
        row.status = 'WARNING';
        return;
      }
      row.status = 'BALANCED';
    });

    return Array.from(map.values()).sort((a, b) => b.date.localeCompare(a.date));
  }, [kafkaLog, transmuteLog, historyRecon, history, pipeline?.masterName]);
```

---

## 3. Cấu trúc Cột Table trong Modal (`columns`)

Bảng chia làm 6 khối cột rõ ràng:

1. **Ngày** (Fixed Left, width 105)
2. **1. SNAPSHOT / RE-SYNC** (Group Header):
   - `Shadow (Nạp)`: `r.snapshot.ops > 0 ? `${fmtNum(r.snapshot.ops)} Ops` : '—'`
   - `Master (Sync)`: `r.snapshot.masterWritten > 0 ? `${fmtNum(r.snapshot.masterWritten)} ghi` : '—'`
3. **2. KAFKA CDC BẮN VỀ** (Group Header):
   - `Tổng (Ops)`: `<b>{fmtNum(v)}</b>`
   - `Insert(c)`: `<span style={{ color: '#52c41a' }}>{fmtNum(v)}</span>`
   - `Update(u)`: `<span style={{ color: '#faad14' }}>{fmtNum(v)}</span>`
   - `Delete(d)`: `<span style={{ color: '#ff4d4f' }}>{fmtNum(v)}</span>`
4. **3. GHI VÀO DB SHADOW** (Group Header):
   - `Tổng xử lý`: `<b>{fmtNum(v)} Ops</b>` kèm `({fmtNum(r.shadow.written)} dòng DB)`
   - `Insert(c)`: `<span style={{ color: '#52c41a' }}>{fmtNum(v)}</span>`
   - `Update(u)`: `<span style={{ color: '#faad14' }}>{fmtNum(v)}</span>`
   - `Delete(d)`: `<span style={{ color: '#ff4d4f' }}>{fmtNum(v)}</span>`
5. **4. GHI VÀO DB MASTER** (Group Header):
   - `Tổng xử lý`: `<b>{fmtNum(v)} Ops</b>` kèm `({fmtNum(r.master.written)} ghi DB)`
   - `Inserted`: `<span style={{ color: '#52c41a' }}>{fmtNum(v)}</span>`
   - `Updated`: `<span style={{ color: '#faad14' }}>{fmtNum(v)}</span>`
   - `Skipped`: `<Text type="secondary">{fmtNum(v)}</Text>`
6. **5. ĐỐI SOÁT & DLQ (LỖI)** (Group Header):
   - `DLQ / Lỗi`: `v > 0 ? <span style={{ color: '#cf1322', fontWeight: 600 }}>...</span> : 0`
   - `Lệch thiếu`: `r.recon.totalRuns === 0 ? '—' : v`
   - `Lệch stale`: `r.recon.totalRuns === 0 ? '—' : v`
   - `Đã heal`: `<span style={{ color: '#52c41a' }}>{fmtNum(v)}</span>`
7. **ĐÁNH GIÁ** (Fixed Right, width 120):
   - `Tag: CÂN BẰNG / CẢNH BÁO / CÓ LỆCH`

---

## 4. Bỏ Phân trang & Chuẩn hóa Dòng TỔNG CỘNG Footer

```tsx
<Table<DailyFlowRecord>
  size="small"
  bordered
  rowKey="date"
  loading={isLoadingKafka || isLoadingTransmute}
  dataSource={dailyStats}
  scroll={{ y: 450, x: 1850 }}
  pagination={false}
  summary={() => {
    const isOverallDrift = totals.dlqTotal > 0 || (latestRecon?.missing || 0) > 0;
    const isOverallWarning = (latestRecon?.stale || 0) > 0 || (totals.kafkaTotal > 0 && totals.shadowTotal === 0);

    return (
      <Table.Summary fixed>
        <Table.Summary.Row style={{ background: '#fafafa', fontWeight: 600 }}>
          <Table.Summary.Cell index={0} align="center" fixed="left"><b>TỔNG CỘNG</b></Table.Summary.Cell>
          
          {/* Snapshot */}
          <Table.Summary.Cell index={1} align="right">
            {totals.snapshotOps > 0 ? <span style={{ color: '#722ed1' }}>{fmtNum(totals.snapshotOps)}</span> : '—'}
          </Table.Summary.Cell>
          <Table.Summary.Cell index={2} align="right">
            {totals.snapshotMasterWritten > 0 ? <span style={{ color: '#722ed1' }}>{fmtNum(totals.snapshotMasterWritten)}</span> : '—'}
          </Table.Summary.Cell>

          {/* Trạm 1: Kafka CDC */}
          <Table.Summary.Cell index={3} align="right"><b>{fmtNum(totals.kafkaTotal)}</b></Table.Summary.Cell>
          <Table.Summary.Cell index={4} align="right"><span style={{ color: '#52c41a' }}>{fmtNum(totals.kafkaC)}</span></Table.Summary.Cell>
          <Table.Summary.Cell index={5} align="right"><span style={{ color: '#faad14' }}>{fmtNum(totals.kafkaU)}</span></Table.Summary.Cell>
          <Table.Summary.Cell index={6} align="right"><span style={{ color: '#ff4d4f' }}>{fmtNum(totals.kafkaD)}</span></Table.Summary.Cell>

          {/* Trạm 2: Shadow */}
          <Table.Summary.Cell index={7} align="right">
            <div>
              <b>{fmtNum(totals.shadowTotal)} Ops</b>
              <div style={{ fontSize: 10, color: '#8c8c8c' }}>({fmtNum(totals.shadowWritten)} dòng)</div>
            </div>
          </Table.Summary.Cell>
          <Table.Summary.Cell index={8} align="right"><span style={{ color: '#52c41a' }}>{fmtNum(totals.shadowC)}</span></Table.Summary.Cell>
          <Table.Summary.Cell index={9} align="right"><span style={{ color: '#faad14' }}>{fmtNum(totals.shadowU)}</span></Table.Summary.Cell>
          <Table.Summary.Cell index={10} align="right"><span style={{ color: '#ff4d4f' }}>{fmtNum(totals.shadowD)}</span></Table.Summary.Cell>

          {/* Trạm 3: Master */}
          <Table.Summary.Cell index={11} align="right">
            {pipeline?.masterName ? (
              <div>
                <b>{fmtNum(totals.masterTotal)} Ops</b>
                <div style={{ fontSize: 10, color: '#8c8c8c' }}>({fmtNum(totals.masterWritten)} ghi)</div>
              </div>
            ) : <Text type="secondary">—</Text>}
          </Table.Summary.Cell>
          <Table.Summary.Cell index={12} align="right">{pipeline?.masterName ? <span style={{ color: '#52c41a' }}>{fmtNum(totals.masterInserted)}</span> : <Text type="secondary">—</Text>}</Table.Summary.Cell>
          <Table.Summary.Cell index={13} align="right">{pipeline?.masterName ? <span style={{ color: '#faad14' }}>{fmtNum(totals.masterUpdated)}</span> : <Text type="secondary">—</Text>}</Table.Summary.Cell>
          <Table.Summary.Cell index={14} align="right">{pipeline?.masterName ? <Text type="secondary">{fmtNum(totals.masterSkipped)}</Text> : <Text type="secondary">—</Text>}</Table.Summary.Cell>

          {/* Trạm 4: Đối soát & DLQ */}
          <Table.Summary.Cell index={15} align="right">{fmtNum(totals.dlqTotal)}</Table.Summary.Cell>
          <Table.Summary.Cell index={16} align="right">
            {!latestRecon ? <Text type="secondary">—</Text> : (
              <span style={{ color: latestRecon.missing > 0 ? '#cf1322' : '#52c41a' }}>{fmtNum(latestRecon.missing)}</span>
            )}
          </Table.Summary.Cell>
          <Table.Summary.Cell index={17} align="right">{!latestRecon ? <Text type="secondary">—</Text> : fmtNum(latestRecon.stale)}</Table.Summary.Cell>
          <Table.Summary.Cell index={18} align="right"><span style={{ color: '#52c41a' }}>{fmtNum(totals.healedTotal)}</span></Table.Summary.Cell>

          {/* Đánh giá tổng hợp */}
          <Table.Summary.Cell index={19} align="center" fixed="right">
            {dailyStats.length === 0 ? <Text type="secondary">—</Text> : (
              <Tag color={isOverallDrift ? 'error' : isOverallWarning ? 'gold' : 'success'}>
                {isOverallDrift ? 'CÓ LỆCH' : isOverallWarning ? 'CẢNH BÁO' : 'CÂN BẰNG'}
              </Tag>
            )}
          </Table.Summary.Cell>
        </Table.Summary.Row>
      </Table.Summary>
    );
  }}
/>
```
