# Hồ sơ giải pháp kỹ thuật cụ thể

## 1. Sửa hàm `getDiffIDs`
Trong file `ExecuteHealModal.tsx` dòng 85-128:
* Với Segment B (`shadow_master`), parse `stale_ids` và đọc các trường `missing_from_master`, `missing_from_shadow`, và `mismatched`.
* Với Segment A (`source_shadow`), parse `stale_ids` và đọc `missing_from_dest`, `missing_from_src`, và `mismatched`.

Đoạn code sửa đổi cụ thể:
```typescript
  const getDiffIDs = (r: any): string[] => {
    const ids: string[] = [];
    if (r.missing_ids) {
      try {
        const parsed = typeof r.missing_ids === 'string' ? JSON.parse(r.missing_ids) : r.missing_ids;
        if (Array.isArray(parsed)) {
          ids.push(...parsed.map(String));
        }
      } catch (e) {
        console.error(e);
      }
    }
    if (r.stale_ids) {
      try {
        const parsed = typeof r.stale_ids === 'string' ? JSON.parse(r.stale_ids) : r.stale_ids;
        if (parsed) {
          if (r.segment === 'shadow_master') {
            if (Array.isArray(parsed.missing_from_master)) {
              ids.push(...parsed.missing_from_master.map(String));
            }
            if (Array.isArray(parsed.missing_from_shadow)) {
              ids.push(...parsed.missing_from_shadow.map(String));
            }
            if (Array.isArray(parsed.mismatched)) {
              ids.push(...parsed.mismatched.map(String));
            }
            if (Array.isArray(parsed)) {
              ids.push(...parsed.map(String));
            }
          } else {
            if (Array.isArray(parsed.missing_from_dest)) {
              ids.push(...parsed.missing_from_dest.map(String));
            }
            if (Array.isArray(parsed.missing_from_src)) {
              ids.push(...parsed.missing_from_src.map(String));
            }
            if (Array.isArray(parsed.mismatched)) {
              ids.push(...parsed.mismatched.map(String));
            }
            if (Array.isArray(parsed)) {
              ids.push(...parsed.map(String));
            }
          }
        }
      } catch (e) {
        console.error(e);
      }
    }
    return Array.from(new Set(ids.filter(Boolean)));
  };
```

## 2. Nâng cấp Popover ID lệch để phân chia 3 loại và ẩn tag ngoài bảng
Sửa logic render cột `diff_ids` (dòng 320-356) của bảng `reportColumns` trong `ExecuteHealModal.tsx`.
* Ẩn việc render tag ID ra ngoài bảng.
* Chỉ hiển thị một Button nhỏ hình tròn với icon `UnorderedListOutlined` (giống bảng đã heal).
* Khi click vào icon list này, hiển thị Popover phân tách rõ ràng 3 danh sách ID theo 3 loại:
  * **Thiếu ở Shadow** (màu đỏ)
  * **Thiếu ở Master/Source** (màu cam)
  * **Lệch dữ liệu** (màu vàng)

Hàm render cột mới:
```typescript
    {
      title: 'ID lệch',
      key: 'diff_ids',
      width: 100,
      render: (record: any) => {
        const ids = getDiffIDs(record);
        if (ids.length === 0) return <Text type="secondary">—</Text>;

        let parsedStale: any = null;
        if (record.stale_ids) {
          try {
            parsedStale = typeof record.stale_ids === 'string' ? JSON.parse(record.stale_ids) : record.stale_ids;
          } catch (e) {
            console.error(e);
          }
        }

        let missingFromDest: string[] = [];
        let missingFromSrc: string[] = [];
        let mismatched: string[] = [];

        if (record.segment === 'shadow_master') {
          if (parsedStale) {
            missingFromDest = (parsedStale.missing_from_shadow || []).map(String);
            missingFromSrc = (parsedStale.missing_from_master || []).map(String);
            mismatched = (parsedStale.mismatched || []).map(String);
          }
        } else {
          if (parsedStale) {
            missingFromDest = (parsedStale.missing_from_dest || []).map(String);
            missingFromSrc = (parsedStale.missing_from_src || []).map(String);
            mismatched = (parsedStale.mismatched || []).map(String);
          }
        }

        if (record.missing_ids) {
          try {
            const parsedMiss = typeof record.missing_ids === 'string' ? JSON.parse(record.missing_ids) : record.missing_ids;
            if (Array.isArray(parsedMiss)) {
              if (record.segment === 'shadow_master') {
                missingFromSrc = Array.from(new Set([...missingFromSrc, ...parsedMiss.map(String)]));
              } else {
                missingFromDest = Array.from(new Set([...missingFromDest, ...parsedMiss.map(String)]));
              }
            }
          } catch (e) {
            console.error(e);
          }
        }

        const popoverContent = (
          <div style={{ maxHeight: 350, overflowY: 'auto', width: 320 }}>
            <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #f0f0f0', paddingBottom: 6 }}>
              <Text strong>Chi tiết ID lệch ({ids.length})</Text>
              <Button
                size="small"
                type="link"
                icon={<CopyOutlined />}
                onClick={() => {
                  navigator.clipboard.writeText(ids.join(', '));
                  message.success('Đã copy danh sách ID!');
                }}
              >
                Copy tất cả
              </Button>
            </div>
            
            {missingFromDest.length > 0 && (
              <div style={{ marginBottom: 12 }}>
                <Text type="danger" strong style={{ fontSize: 12 }}>
                  {record.segment === 'shadow_master' ? 'Thiếu ở Shadow (Missing from Shadow):' : 'Thiếu ở Shadow (Missing from Dest):'} ({missingFromDest.length})
                </Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
                  {missingFromDest.map(id => <Tag key={id} color="red" style={{ margin: 0 }}>{id}</Tag>)}
                </div>
              </div>
            )}

            {missingFromSrc.length > 0 && (
              <div style={{ marginBottom: 12 }}>
                <Text style={{ color: '#fa8c16', fontWeight: 'bold', fontSize: 12 }}>
                  {record.segment === 'shadow_master' ? 'Thiếu ở Master (Missing from Master):' : 'Thiếu ở Source (Missing from Src):'} ({missingFromSrc.length})
                </Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
                  {missingFromSrc.map(id => <Tag key={id} color="orange" style={{ margin: 0 }}>{id}</Tag>)}
                </div>
              </div>
            )}

            {mismatched.length > 0 && (
              <div style={{ marginBottom: 12 }}>
                <Text type="warning" strong style={{ fontSize: 12 }}>
                  Lệch dữ liệu (Mismatched): ({mismatched.length})
                </Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
                  {mismatched.map(id => <Tag key={id} color="gold" style={{ margin: 0 }}>{id}</Tag>)}
                </div>
              </div>
            )}
          </div>
        );

        return (
          <Popover content={popoverContent} trigger="click" placement="top">
            <Button size="small" shape="circle" icon={<UnorderedListOutlined />} style={{ fontSize: 10, width: 22, height: 22, minWidth: 22, padding: 0 }} />
          </Popover>
        );
      }
    }
```
