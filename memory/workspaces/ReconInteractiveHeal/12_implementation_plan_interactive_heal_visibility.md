# Kế hoạch triển khai: Khắc phục hiển thị dữ liệu chưa Heal trong modal "Chữa lành đối soát"

## Mục tiêu
Đảm bảo khi người dùng nhấn nút "Chữa lành" (MedicineBoxOutlined) trên giao diện đối soát, hệ thống sẽ mở modal `ExecuteHealModal` thay vì `ConfirmDestructiveModal`. Modal này sẽ tải và hiển thị danh sách các phiên đối soát chưa được xử lý (unhealed reports) lấy từ API backend đã được chuẩn hóa FQN. Đồng thời, đổi tiêu đề của `ExecuteHealModal` thành "Chữa lành đối soát cho [table]" để mang lại trải nghiệm người dùng nhất quán và chính xác.

## Đánh giá tác động & Rủi ro (User Review Required)
> [!NOTE]
> Thay đổi này chỉ tác động đến tầng giao diện (Frontend) của ứng dụng `cdc-cms-web`. Không thay đổi logic nghiệp vụ của backend hay cấu trúc cơ sở dữ liệu.
> Nút "Chữa lành" thay vì kích hoạt task background (gọi `useHealMutation`) sẽ chuyển sang chế độ tương tác trực tiếp (gọi `useExecuteHealMutation` sau khi hiển thị các bản ghi bị lệch).

## Câu hỏi mở (Open Questions)
Không có câu hỏi mở nào. Giải pháp đã rõ ràng và thống nhất với yêu cầu của User.

## Thay đổi đề xuất

### Frontend Component (`cdc-cms-web`)

#### [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- Cập nhật hàm `openHeal` để thiết lập `executeHealTarget` thay vì `modalPlan`.
- Code trước thay đổi:
  ```typescript
  const openHeal = (record: ReconReport) =>
    setModalPlan({
      action: { kind: 'heal', table: record.target_table, segment: record.segment, record, isHeal: true },
      title: `Chữa lành đối soát cho ${record.target_table}`,
      description:
        'Tác vụ sẽ chạy ngầm để quét và chữa lành dữ liệu bị drift/lệch.',
      targetName: record.target_table,
      actionLabel: 'Chữa lành',
      danger: true,
    });
  ```
- Code sau thay đổi:
  ```typescript
  const openHeal = (record: ReconReport) =>
    setExecuteHealTarget({
      table: record.target_table,
      segment: record.segment,
      shadowSchema: record.shadow_schema || undefined,
    });
  ```

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Đổi tiêu đề hiển thị từ `"Chữa lành drift — "` sang `"Chữa lành đối soát cho "`.
- Code trước thay đổi:
  ```typescript
  title={
    <span>
      <MedicineBoxOutlined style={{ color: '#ff4d4f', marginRight: 8 }} />
      Chữa lành drift — {table}
    </span>
  }
  ```
- Code sau thay đổi:
  ```typescript
  title={
    <span>
      <MedicineBoxOutlined style={{ color: '#ff4d4f', marginRight: 8 }} />
      Chữa lành đối soát cho {table}
    </span>
  }
  ```

## Kế hoạch xác minh

### Biên dịch & Build
- Chạy kiểm tra kiểu tĩnh của TypeScript:
  `npx tsc --noEmit` trong thư mục `cdc-cms-web` để đảm bảo không phát sinh lỗi biên dịch.

### Xác minh thủ công
- Click nút "Chữa lành" trên bảng đối soát và kiểm tra xem modal hiển thị danh sách các phiên chưa được xử lý, tiêu đề của modal hiển thị chính xác "Chữa lành đối soát cho [table name]".
