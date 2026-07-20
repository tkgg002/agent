# Kế hoạch triển khai: Thêm Chặng vào Nhật ký đối soát

## 1. Mục tiêu
Hiển thị cột "Chặng" (segment) trong bảng lịch sử đối soát ("Nhật ký đối soát") trong `ReconPipelineGrid.tsx`.

## 2. Thay đổi đề xuất

### Component: cdc-cms-web
#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
Thêm cột Segment vào mảng columns trong Table:
```tsx
            {
              title: 'Chặng',
              dataIndex: 'segment',
              key: 'segment',
              width: 120,
              render: (s: string | undefined) => (
                <Tag color={s === 'shadow_master' ? 'purple' : 'blue'} style={{ margin: 0 }}>
                  {s === 'shadow_master' ? 'Shadow → Master' : 'Source → Shadow'}
                </Tag>
              ),
            },
```

## 3. Kế hoạch kiểm tra
- Chạy lệnh build/compile trong `cdc-cms-web` để đảm bảo không lỗi biên dịch.
- Chạy script linter `verify_governance.py`.
