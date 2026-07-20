# Hồ sơ giải pháp kỹ thuật chi tiết (Technical Solutions)

Dưới đây là chi tiết thay đổi mã nguồn cần thực hiện trên Frontend.

## 1. Cập nhật `useReconStatus.ts`
Vị trí thay đổi: [useReconStatus.ts#L77-L97](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts#L77-L97)
Thêm các thuộc tính `source_count` và `dest_count` vào interface `UnhealedReport`:

```diff
 export interface UnhealedReport {
   id: number;
   target_table: string;
   segment: string;
   source_db: string;
   missing_count: number;
   stale_count: number;
   orphan_count: number;
   check_type: string;
   checked_at: string;
   shadow_schema?: string;
   shadow_table?: string;
   healed_at: string | null;
   healed_mismatched_at?: string | null;
   healed_missing_src_at?: string | null;
   healed_missing_dest_at?: string | null;
   healed_mismatched_count?: number;
   healed_missing_dest_count?: number;
   pruned_missing_src_count?: number;
+  source_count?: number | null;
+  dest_count?: number | null;
 }
```

## 2. Cập nhật `ExecuteHealModal.tsx`
Vị trí thay đổi 1: [ExecuteHealModal.tsx#L316-L323](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx#L316-L323)
Bổ sung cấu hình cột "Nguồn" và "Đích" vào mảng `reportColumns`:

```diff
     {
       title: 'Khoảng thời gian',
       key: 'time_range',
       width: 220,
       render: (record: any) => formatTimeRange(record.recon_start_time, record.recon_end_time),
     },
+    {
+      title: 'Nguồn',
+      dataIndex: 'source_count',
+      width: 95,
+      render: (v: number | null | undefined) => v == null ? '—' : v.toLocaleString(),
+    },
+    {
+      title: 'Đích',
+      dataIndex: 'dest_count',
+      width: 95,
+      render: (v: number | null | undefined) => v == null ? '—' : v.toLocaleString(),
+    },
     {
       title: 'ID lệch',
       key: 'diff_ids',
```

Vị trí thay đổi 2: [ExecuteHealModal.tsx#L648-L658](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx#L648-L658)
Cập nhật cấu hình scroll ngang cho table ở tab `unhealed`:

```diff
           <Table
             size="small"
             rowKey="id"
             rowSelection={rowSelection}
             dataSource={reports}
             columns={reportColumns}
             pagination={false}
-            scroll={{ y: 200 }}
+            scroll={{ x: 'max-content', y: 200 }}
             style={{ marginBottom: 4 }}
           />
```
