# Technical Specification: Hide Disabled Master Tables in Data Integrity

## Proposed Changes
Chúng ta sẽ thực hiện lọc tại Frontend trong file `cdc-cms-web/src/pages/DataIntegrity.tsx`.

### 1. Fetch metadata tại `DataIntegrity.tsx`
Để lọc danh sách `reportList` (dữ liệu thô từ `/api/reconciliation/report`), ta cần thông tin cấu hình `masters` từ `/api/v1/masters` ở component cha `DataIntegrity.tsx`.
Ta sẽ import và sử dụng query hook:
```typescript
  const { data: masters } = useQuery({
    queryKey: ['masters-recon-grid'],
    queryFn: async () => {
      const { data } = await cmsApi.get<{ data: any[] }>('/api/v1/masters', { params: { page: 1, page_size: 500 } });
      return data.data || [];
    },
    refetchInterval: 30000,
  });
```

### 2. Xác định các master bị tắt sync
Duyệt qua danh sách `reportList`.
Một report `r` đại diện cho một chặng của một bảng (segment `source_shadow` hoặc `shadow_master`).
Nếu bảng đó có chặng `shadow_master` hoặc có cấu hình master:
Chúng ta xác định tên master table:
- Nếu `r.segment === 'shadow_master'`, master table name là `r.target_table`.
- Hoặc ta có thể tìm `mstObj` trong `masters` thông qua `shadow_schema` và `shadow_table`:
```typescript
const mstObj = masters?.find(m => m.shadow_schema === r.shadow_schema && m.shadow_table === r.shadow_table);
```
Tuy nhiên, có trường hợp một shadow table có master binding nhưng đã bị xóa (do đó `mstObj` không còn trong danh sách `masters`), hoặc bị deactive (`!mstObj.is_active`).
Để bao phủ cả trường hợp master bị xóa/tắt:
Nếu một shadow table đã từng có segment `shadow_master` đối soát (tức là ta tìm thấy bất kỳ record nào trong `reportList` có `segment === 'shadow_master'` cho shadow table đó), thì:
- Nếu ta tìm thấy `mstObj` trong `masters` cho shadow table đó và `mstObj.is_active` là `false` -> Ẩn.
- Nếu không tìm thấy `mstObj` trong `masters` cho shadow table đó (nhưng lại có record segment `shadow_master` tồn tại trong database báo cáo) -> Có nghĩa là master config đã bị xóa hoặc bị tắt -> Ẩn.

Chúng ta sẽ xây dựng một tập hợp các shadow tables có master bị tắt:
```typescript
  const disabledShadowKeys = useMemo(() => {
    if (!masters || !reportList) return new Set<string>();
    
    // Tìm tất cả các shadow table FQN có segment shadow_master trong báo cáo
    const shadowTablesWithMaster = new Set<string>();
    reportList.forEach(r => {
      if (r.segment === 'shadow_master') {
        const key = `${r.shadow_schema || ''}.${r.shadow_table || ''}`;
        shadowTablesWithMaster.add(key);
      }
    });

    const disabledKeys = new Set<string>();
    shadowTablesWithMaster.forEach(key => {
      const parts = key.split('.');
      const schema = parts[0];
      const table = parts[1];
      const mst = masters.find(m => m.shadow_schema === schema && m.shadow_table === table);
      
      // Nếu không tìm thấy master binding tương ứng, hoặc master binding bị deactive
      if (!mst || !mst.is_active) {
        disabledKeys.add(key);
      }
    });

    return disabledKeys;
  }, [masters, reportList]);
```

Sau đó, ta lọc `reportList` ở component cha `DataIntegrity`:
```typescript
  const filteredReportList = useMemo(() => {
    return reportList.filter(r => {
      const key = `${r.shadow_schema || ''}.${r.shadow_table || r.target_table}`;
      return !disabledShadowKeys.has(key);
    });
  }, [reportList, disabledShadowKeys]);
```

### 3. Cập nhật các biến sử dụng `reportList` sang `filteredReportList`
- `reportList` truyền vào `ReconPipelineGrid` -> đổi thành `filteredReportList`.
- `reportList` truyền vào `Table` của tab overview -> đổi thành `filteredReportList`.
- `driftCount` và `okCount` tính toán dựa trên `filteredReportList` -> đổi thành `filteredReportList`.
- `tableOptions` và `reportByTarget` -> đổi thành `filteredReportList`.
