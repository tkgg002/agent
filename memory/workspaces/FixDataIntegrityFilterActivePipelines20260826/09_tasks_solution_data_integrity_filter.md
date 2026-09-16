# Hồ sơ Giải pháp Kỹ thuật & Demo Code: Tối ưu Giao diện Data Integrity & Lỗi đồng bộ

## 1. Yêu cầu Mới từ User
1. **Loại bỏ Tab "Tổng quan" (Overview)**: Do trùng chức năng với Tab Pipelines và hiển thị danh sách phẳng chưa lọc.
2. **Loại bỏ Tab "Backfill _source_ts"**: Loại bỏ tab này để làm gọn giao diện, bỏ các tính năng dư thừa.
3. **Lọc Tab & Card "Lỗi đồng bộ"**:
   - Hiện tại đếm 1,190 bao gồm 1,189 bản ghi đã `resolved` (đã xử lý).
   - Đổi sang chỉ đếm và hiển thị các bản ghi lỗi thực sự chưa xử lý (`status !== 'resolved'`).
4. **Giữ nguyên Tab "Pipelines"**: Đã lọc đúng chỉ hiển thị các Pipelines/Connectors đang hoạt động.

## 2. Mã Nguồn Demo Chi Tiết (Code Demo)

### File: `cdc-cms-web/src/pages/DataIntegrity.tsx`

```tsx
  // Lọc chỉ lấy các lỗi chưa được xử lý (status !== 'resolved')
  const unresolvedFailedLogs = useMemo(() => {
    return failedLogs.filter(f => f.status !== 'resolved');
  }, [failedLogs]);

  const unresolvedFailedTotal = unresolvedFailedLogs.length;
```

**Cấu hình Tabs rút gọn (Chỉ giữ 2 Tab: Pipelines & Lỗi đồng bộ):**
```tsx
      <Tabs
        defaultActiveKey="pipelines"
        items={[
          {
            key: 'pipelines',
            label: 'Pipelines',
            children: showReportLoading ? (
              <Skeleton active paragraph={{ rows: 6 }} />
            ) : (
              <ReconPipelineGrid
                rows={reportList}
                loading={reports.isFetching}
                onCheckTable={openCheckTable}
                onHeal={openHeal}
                onExecuteHeal={openExecuteHeal}
                onPrune={openPrune}
              />
            ),
          },
          {
            key: 'failed',
            label: `Lỗi đồng bộ (${unresolvedFailedTotal})`,
            children: failed.isLoading && !failed.data ? (
              <Skeleton active paragraph={{ rows: 6 }} />
            ) : failed.isError && !failed.data ? (
              <Alert
                type="error"
                showIcon
                title="Không tải được danh sách lỗi đồng bộ"
                description={
                  <Button
                    type="primary"
                    icon={<ReloadOutlined />}
                    onClick={() => failed.refetch()}
                  >
                    Thử lại
                  </Button>
                }
              />
            ) : unresolvedFailedLogs.length === 0 ? (
              <Empty description="Không có lỗi đồng bộ" />
            ) : (
              <Table
                columns={failedColumns}
                dataSource={unresolvedFailedLogs}
                rowKey="id"
                loading={failed.isFetching}
                size="small"
                pagination={{ pageSize: 30 }}
              />
            ),
          },
        ]}
      />
```

**Header Statistic Card Lỗi đồng bộ:**
```tsx
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Lỗi đồng bộ"
              value={unresolvedFailedTotal}
              styles={{ content: { color: unresolvedFailedTotal > 0 ? '#faad14' : '#3f8600' } }}
            />
          </Card>
        </Col>
```
