# Hồ sơ Giải pháp Kỹ thuật & Demo Code - Phân Tab Trang Schedules theo Mode

## 1. Yêu cầu Mới
Tách danh sách Schedules trên trang `http://localhost:5173/schedules` (`TransmuteSchedules.tsx`) thành các Tabs theo `mode` để tránh bị rối mắt:
1. **Tất cả**: Tất cả các schedules.
2. **Realtime**: Lọc các schedules có `mode === 'post_ingest'`.
3. **Sync ngay**: Lọc các schedules có `mode === 'immediate'`.
4. **Đặt lịch**: Lọc các schedules có `mode === 'cron'`.

## 2. Mã Nguồn Demo (Code Demo)

### File `src/pages/TransmuteSchedules.tsx`

```tsx
  const [activeTab, setActiveTab] = useState<string>('all');

  const sortedData = useMemo(() => {
    if (!data) return [];
    return [...data].sort((a, b) => {
      const timeA = a.last_run_at ? new Date(a.last_run_at).getTime() : 0;
      const timeB = b.last_run_at ? new Date(b.last_run_at).getTime() : 0;
      return timeB - timeA;
    });
  }, [data]);

  const counts = useMemo(() => {
    const all = sortedData.length;
    const post_ingest = sortedData.filter((r) => r.mode === 'post_ingest').length;
    const immediate = sortedData.filter((r) => r.mode === 'immediate').length;
    const cron = sortedData.filter((r) => r.mode === 'cron').length;
    return { all, post_ingest, immediate, cron };
  }, [sortedData]);

  const filteredData = useMemo(() => {
    if (activeTab === 'all') return sortedData;
    return sortedData.filter((r) => r.mode === activeTab);
  }, [sortedData, activeTab]);
```

Render Tabs & Table:
```tsx
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        style={{ marginTop: 12 }}
        items={[
          { key: 'all', label: `Tất cả (${counts.all})` },
          { key: 'post_ingest', label: `Realtime (${counts.post_ingest})` },
          { key: 'immediate', label: `Sync ngay (${counts.immediate})` },
          { key: 'cron', label: `Đặt lịch (${counts.cron})` },
        ]}
      />

      <Table
        style={{ marginTop: 8 }}
        size="middle"
        loading={isLoading}
        dataSource={filteredData}
        rowKey="id"
        columns={columns}
        pagination={false}
      />
```
