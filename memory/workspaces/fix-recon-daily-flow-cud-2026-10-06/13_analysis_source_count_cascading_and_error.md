# PHÂN TÍCH GỐC RỄ & THIẾT KẾ GIẢI PHÁP: 3 LỖI SOURCE COUNT TRÊN ACTIVITY MANAGER

## 1. PHÂN TÍCH GỐC RỄ (ROOT CAUSE ANALYSIS)

### LỖI 1: Tầng dữ liệu chọn `source` hoặc `shadow` nhưng không hiện Connection
- **Triệu chứng:** Người dùng mở Form `source_count`, chọn Tầng dữ liệu `source` hoặc `shadow`, dropdown "Mã kết nối (Connection Code)" rỗng không có lựa chọn nào.
- **Root Cause trong Code:**
  1. File `ActivityManager.tsx` (dòng 405-411):
     ```typescript
     const { data: rawConnections } = useQuery<any[]>({
       queryKey: ['connections-for-activity-manager'],
       queryFn: async () => {
         const r = await cmsApi.get('/api/v1/master-connections');
         return r.data.data || [];
       },
     });
     ```
     `ActivityManager` đang gọi API `/api/v1/master-connections` để nạp danh sách connection cho form.
  2. File `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go` (dòng 552):
     ```go
     func (r *masterRepoGorm) ListMasterConnections(ctx context.Context) ([]master.MasterConnectionItem, error) {
         ...
         Where("status = 'active' AND role_type = 'master'").
         Order("id ASC").
         Find(&rows).Error
     ```
     Hàm `ListMasterConnections` lọc cứng `role_type = 'master'`.
     Vì vậy, API `/api/v1/master-connections` CHỈ TRẢ VỀ các kết nối Master (`default_master`, `master_2`, `postgres-develop`).
     Nó HOÀN TOÀN KHÔNG trả về các kết nối Source (`traitestpbs`, `traitestces`, `walletservice`) hay Shadow (`default_shadow`).
  3. Khi `selectedLayer === 'source'`, `connectionOptions` trong `CascadingSourceSelector` filter mảng `rawConnections` với điều kiện `c.role_type === 'source'` $\rightarrow$ Không có phần tử nào $\rightarrow$ Options bị rỗng!
- **Giải pháp tối ưu:**
  - Trên backend `cdc-cms-service`:
    1. Bổ sung endpoint tổng quát `GET /api/v1/connections` (hoặc mở rộng `GET /api/v1/master-connections?role=all` / `?role_type=all`):
       Khi có param `role=all` hoặc gọi qua endpoint `GET /api/v1/connections`, backend truy vấn bảng `cdc_system.connection_registry` với điều kiện `status = 'active'` (lấy toàn bộ các kết nối `source`, `shadow`, `master`).
       Nếu không có param trên `/api/v1/master-connections`, nó vẫn giữ `role_type = 'master'` để bảo vệ tương thích ngược 100%.
  - Trên frontend `ActivityManager.tsx`:
    - Gọi API `cmsApi.get('/api/v1/connections')` (hoặc `cmsApi.get('/api/v1/master-connections', { params: { role: 'all' } })`).

---

### LỖI 2: Chọn Layer `master`, chọn connection nhưng không hiện đúng Namespace
- **Triệu chứng:** Người dùng chọn Layer `master`, chọn mã kết nối (ví dụ `master_2` hoặc `postgres-develop`), dropdown Namespace không hiển thị đúng các schema/database của connection đó.
- **Root Cause trong Code:**
  1. Trong `ActivityManager.tsx` (dòng 154-158):
     ```typescript
     } else if (selectedLayer === 'master') {
       (rawMasterBindings || []).forEach((m) => {
         if (m.master_schema) set.add(m.master_schema);
       });
     }
     ```
     - Nhánh `selectedLayer === 'master'` duyệt qua toàn bộ `rawMasterBindings` mà HOÀN TOÀN KHÔNG KIỂM TRA xem `m.master_connection_code === selectedConn` hay `m.master_connection_id === currentConn.id`!
     - Một connection master (ví dụ `master_2`) có `default_database = 'goopay_master_2'`, `default_schema = 'public'`, và các `master_binding` có `master_schema = 'master_centrallized_export_service'`.
     - Một connection master khác (ví dụ `postgres-develop`) có `default_database = 'core-trans-proxy-history-service'`, `default_schema = 'public'`. Nhưng do chưa có binding nào trong `master_binding` cho connection này, code cũ không đọc `default_database` hay `default_schema` từ chính connection nên dropdown bị rỗng hoặc chỉ hiện schema của connection khác!
- **Giải pháp tối ưu:**
  - Trong `CascadingSourceSelector` (`ActivityManager.tsx`):
    Khi `selectedLayer === 'master'`:
    - Tìm connection hiện tại: `const currentConn = (rawConnections || []).find((c) => c.connection_code === selectedConn);`
    - Bổ sung `currentConn.default_schema` (nếu có, vd `public`) và `currentConn.default_database` (nếu có).
    - Lọc `rawMasterBindings`: Chỉ lấy các `m` có `m.master_connection_code === selectedConn || (currentConn && m.master_connection_id === currentConn.id)`. Thêm `m.master_schema` và `m.master_database` (nếu có) vào danh sách Namespace.
    - Áp dụng tương tự cho nhánh `selectedLayer === 'source'` và `selectedLayer === 'shadow'`:
      - Source: Lọc `sourceObjects` theo `s.source_connection_code === selectedConn`, kèm theo `currentConn.default_database`.
      - Shadow: Lọc `rawShadowBindings` và lấy `default_database` / `default_schema` của shadow connection.

---

### LỖI 3: Có error nhưng không hiện trực quan khi chạy source count
- **Triệu chứng:** Khi tác vụ `source_count` chạy thất bại (ví dụ bảng không tồn tại, sai định danh, lỗi kết nối SQL/Mongo), cột lỗi trên bảng chưa cung cấp trải nghiệm xem lỗi trực quan, người dùng khó nhận biết hoặc không xem được chi tiết lỗi.
- **Root Cause & Rà soát End-to-End:**
  1. *Presentation Layer (`ActivityManager.tsx` dòng 852-859):*
     ```tsx
     render: (v) =>
       v ? (
         <Tooltip title={v}>
           <Tag color="red">Có lỗi</Tag>
         </Tooltip>
       ) : (
         <Tag color="green">OK</Tag>
       ),
     ```
     Chỉ là một thẻ `<Tag color="red">Có lỗi</Tag>` thụ động. Không có icon cảnh báo, tooltip mặc định hẹp và khó nhìn đối với stack trace dài.
  2. *Worker Server (`centralized-data-service/internal/server/server_scheduler.go` dòng 127-133):*
     ```go
     defer func() {
         if r := recover(); r != nil {
             observability.Ctx(ctx, s.logger).Error(...)
             // Không ghi lại last_error vào DB khi bị panic!
         }
     }()
     ```
     Nếu goroutine chạy job bị panic (recover), `last_error` trong `cdc_worker_schedule` KHÔNG ĐƯỢC CẬP NHẬT $\rightarrow$ Bảng DB vẫn lưu trạng thái cũ, che giấu lỗi nghiêm trọng.
  3. *Runner (`source_count_runner.go` dòng 137):*
     Đã có lệnh `r.dbControl.Model(&mastermodel.WorkerSchedule{}).Where("id = ?", sched.ID).Update("last_error", countErr.Error())`. Tuy nhiên cần đảm bảo khi thành công thì reset `last_error = nil`.
- **Giải pháp tối ưu:**
  - Ở Frontend:
    - Nâng cấp cột "Lỗi gần nhất" thành một ô tương tác cao:
      - Khi có lỗi: Hiển thị `<Tag icon={<CloseCircleOutlined />} color="error" style={{ cursor: 'pointer' }}>Có lỗi</Tag>`.
      - Tooltip được bọc trong container định dạng đẹp: Tiêu đề "Chi tiết lỗi thực thi:", background đậm/rõ nét, `maxWidth: 500px`, `wordBreak: 'break-word'`, `whiteSpace: 'pre-wrap'`, kèm nút sao chép nội dung lỗi (Copy to clipboard) để Quản trị viên dễ dàng paste gửi team kỹ thuật.
  - Ở Worker Server:
    - Trong `server_scheduler.go`: Bổ sung ghi `last_error = fmt.Sprintf("panic: %v", r)` vào database trong khối `recover()`.
