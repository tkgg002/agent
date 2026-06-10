# Walkthrough: Table Registry UI Enhancement

## Các thay đổi đã thực hiện
1. **Thêm padding cho Switch container**:
   - Thêm style `paddingBottom: '5px'` vào thẻ `div` bọc `Switch` trong `TableRegistry.tsx` (cột "Trạng thái table").
2. **Loại bỏ Space của Ant Design trong AsyncRowActions**:
   - Thay các thẻ `<Space>` và `<Space wrap>` bằng cấu trúc thẻ `div` đơn giản, giúp loại bỏ hoàn toàn các class CSS `ant-space ...` sinh ra lỗi vỡ giao diện trong một số trường hợp.
3. **Phản ánh trạng thái trực tiếp trên nút "Quét field"**:
   - Khi đang quét (`isScanning`): nút có viền xanh dương (`borderColor: '#1677ff', color: '#1677ff'`) và hiển thị icon loading (nhờ Ant Design `loading={isScanning}`).
   - Khi quét thành công (`isSuccess`): nút có viền xanh lá (`borderColor: '#52c41a', color: '#52c41a'`) và hiển thị icon check mark (`<CheckOutlined />`).
   - Xoá hoàn toàn component `<DispatchStatusBadge>` để tránh chật giao diện, vì trạng thái đã được biểu thị trực tiếp trên nút, và các thông báo cụ thể (error, success) đã được hiển thị qua `message.error`/`message.success` toast.
   - Xoá import không sử dụng `DispatchStatusBadge` để tránh lỗi TS6133.
4. **Sắp xếp button theo chiều dọc và rộng 100%**:
   - Chuyển đổi các khối `<Space>` bao bọc các button hành động ở các cột `Source Actions` (chứa nút Scan fields), `Shadow Actions` (chứa Create, Mapping, Snapshot) và `Master Actions` (chứa Create) thành các thẻ `div` flex-box với thuộc tính `flexDirection: 'column'` và `width: '100%'`.
   - Cấu hình style `width: '100%'` cho toàn bộ các Button để các nút tự động dãn rộng bằng nhau và được căn lề thẳng hàng từ trên xuống dưới một cách trực quan, đồng bộ.
   - **Mới**: Gộp hai cột `Approve` và `Action` của bảng trong `MasterRegistry.tsx` thành một cột `Actions` duy nhất. Các nút `Edit` và `Mappings` được chuyển sang cấu trúc flex-box dọc (`flexDirection: 'column'`) với khoảng cách `5px` và rộng `100%` để xếp thẳng hàng từ trên xuống dưới.
   - **Mới**: Di chuyển hai nút `Approve` và `Reject` sang cột `Status`, hiển thị xếp dọc ngay phía dưới Tag trạng thái giúp bố cục khoa học hơn. Khôi phục lại nút `Swap` đang bị comment out theo thiết kế ban đầu.
   - **Mới**: Loại bỏ cột `Active` và chuyển component `Switch` (Active/Inactive) sang cột `Sync` (xếp dọc phía trên nút `Sync`) để tối ưu hóa không gian hiển thị của bảng.
   - **Mới**: Khắc phục lỗi dùng sai prop `orientation="vertical"` thành `direction="vertical"` cho component `<Space>` của Ant Design tại các cột `Source` và `DB Master`.
5. **Căn lề đỉnh (Vertical Align Top) cho toàn bộ Cell trong Table**:
   - Thêm quy tắc CSS `.top-aligned-table .ant-table-cell { vertical-align: top !important; }` vào file `index.css`.
   - Gán `className="top-aligned-table"` cho cả hai component `<Table />` trong `TableRegistry.tsx` (bảng hiển thị Shadow Objects và bảng hiển thị Shadow Bindings).
   - **Mới**: Gán `className="top-aligned-table"` vào component `<Table />` của `MasterRegistry.tsx` để đồng bộ căn chỉnh lề đỉnh của toàn bộ bảng hiển thị Master table bindings.
   - Nhờ đó, tất cả các cell của hàng sẽ luôn được căn thẳng lề ở đỉnh (top), tránh việc các button hay text bị lệch giữa (middle alignment) do chiều cao dòng thay đổi, giữ giao diện sạch sẽ và chuẩn hóa.
6. **Cải thiện độ trực quan của Checkbox bị disabled và hàng (row) tương ứng**:
   - Cấu hình prop `rowClassName` cho Table trong `MasterMappingFieldsPage.tsx` để tính toán trạng thái disabled (nếu mapping chưa tồn tại trên shadow hoặc trạng thái shadow khác `approved`) và tự động gán class `.disabled-row`.
   - Thêm style cho `.disabled-row` và `.disabled-row td` trong `index.css`:
     - Thiết lập màu nền xám nhạt (`#f5f5f5 !important`) để đánh dấu trực quan toàn bộ hàng.
     - Thiết lập `opacity: 0.55;` để làm mờ nhẹ toàn bộ hàng (bao gồm văn bản, các tag trạng thái, select box, switch và nút hành động của hàng đó).
   - Cập nhật quy tắc CSS của `.ant-checkbox-disabled`:
     - Giảm `opacity` xuống `0.4 !important` và thay đổi màu nền (`#e8e8e8`/`#d9d9d9`), màu viền (`#bfbfbf`), màu dấu check (`#595959`) để checkbox disabled hiển thị xám đậm và mờ rõ ràng so với checkbox enabled thông thường.
7. **Hiển thị phân cấp cây trực quan cho Mapping Fields**:
   - Cập nhật interface `MasterRule` trong `MasterMappingFieldsPage.tsx` để chấp nhận thuộc tính `children?: MasterRule[]`.
   - Viết logic `treeData` bằng `useMemo` để chuyển đổi mảng rules phẳng thành cấu trúc cây dựa trên tiền tố của `source_field` (ví dụ `parent.child` là con của `parent`).
   - **Mới**: Thêm thuật toán sinh tự động các node cha ảo trung gian (virtual parents) từ các đường dẫn shadow fields phẳng vì database chỉ lưu trữ các node lá thực tế. Thiết lập `id` âm độc lập cho các node ảo để tránh trùng key.
   - **Mới**: Cập nhật render cho tất cả các cột trong `columns` để ẩn checkbox (trả về `display: 'none'` trong `getCheckboxProps`), ẩn các switch active, các nút hành động Xoá và select type cho các node ảo trung gian này nhằm giữ giao diện sạch sẽ, trực quan.
   - Cập nhật cột `Shadow Field` render để chỉ hiển thị phần tên trường con cuối cùng thay vì hiển thị toàn bộ đường dẫn dài dòng (ví dụ hiển thị `id` thay vì `raw_data.user.id`). Bọc component hiển thị trong `<Tooltip title={text}>` giúp hover xem đường dẫn đầy đủ dễ dàng.
   - Cập nhật `<Table />` sử dụng `dataSource={treeData}` và thêm cấu hình `expandable={{ defaultExpandAllRows: true }}` để mặc định mở rộng tất cả các hàng, cùng `checkStrictly: true` trong `rowSelection` giúp tick chọn các hàng độc lập chuẩn xác.
8. **Cải tiến dropdown "Chọn nhanh field JSON" (Select)**:
    - **Frontend:** Huỷ bỏ logic rename target_column và phân giải backend phức tạp do User yêu cầu revert.
    - **Deduplicate explode_path:** Thêm logic filter bằng cách sử dụng `seen Set` để lọc trùng lặp option theo `value` (tức là `explode_path` dạng `${r.source_field}[*]`). Nếu có nhiều rule có chung trường JSON nguồn, dropdown chỉ hiển thị 1 option duy nhất thay vì hiển thị lặp lại nhiều dòng hoặc ẩn sạch đi, nâng cao tính chính xác và trải nghiệm người dùng (UX).

## Kết quả kiểm thử
- Đã chạy `npm run build` trong `cdc-cms-web` thành công 100% không gặp bất kỳ lỗi biên dịch TypeScript hay Vite.
- Xác nhận thay đổi hoạt động đúng logic: dropdown "Chọn nhanh field JSON" chỉ hiển thị các field JSON không trùng lặp.
