# HỒ SƠ GIẢI PHÁP KỸ THUẬT: KHẮC PHỤC LỖI LỆCH -7 GIỜ Ở CỘT "LẦN CHẠY CUỐI" VÀ "LẦN CHẠY KẾ" TRÊN MÔI TRƯỜNG TESTING

**Ngày lập:** 2026-10-15  
**Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
**Trạng thái:** `PROPOSED` (Chờ lệnh `APPROVE` từ User theo Rule #1 & Rule #13)

---

## 1. Bối cảnh & Hiện tượng (Symptom)
- **Thời điểm thực tế người dùng kiểm tra trên Testing:** `15/10/2026 16:47` (giờ Việt Nam, `UTC+7`).
- **Tác vụ vừa chạy lúc:** `15/10/2026 16:46:33` (`UTC+7`), cấu hình chạy hằng ngày lúc `00:05`.
- **Hiển thị thực tế trên bảng `/activity-manager` (Testing):**
  - **Lần chạy cuối (`last_run_at`):** `15/10/2026 09:46:33` (lệch đúng **-7 giờ** so với `15/10/2026 16:46:33`).
  - **Lần chạy kế (`next_run_at`):** `15/10/2026 17:05:00` (lệch đúng **-7 giờ** so với mốc `00:05` sáng ngày mai `16/10/2026 00:05:00`).

---

## 2. Phân tích Nguyên nhân Gốc rễ (Root Cause Analysis — 100% Thực chứng từ Source Code)

Qua đối chiếu trực tiếp mã nguồn Backend (`centralized-data-service`, `cdc-cms-service`) và Frontend (`cdc-cms-web`), nguyên nhân gốc rễ nằm ở sự kết hợp của **2 mắt xích**:

### 2.1. Mắt xích 1: Trên môi trường Testing (Container `TZ=UTC`), API `GET /api/worker-schedule` trả về chuỗi ISO theo múi giờ UTC (`...Z`) thay vì `+07:00`
1. Khi tác vụ thực thi lúc **`15/10/2026 16:46:33` (`UTC+7`)**:
   - Trong [`centralized-data-service/internal/server/server_scheduler.go` (dòng 82–102 & 229–262)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_scheduler.go#L82-L102):
     + `now := time.Now()` mang giá trị thời điểm thực tương ứng với **`2026-10-15 09:46:33 UTC`** (`16:46:33 +07:00`).
     + `calculateNextRun(sched, now)` quy đổi `now` sang `locHCM` (`UTC+7`), thấy `16:46:33` đã qua mốc `00:05:00` của ngày `15/10/2026` nên tính `nextRun = targetToday.AddDate(0, 0, 1)` = **`2026-10-16 00:05:00 +07:00`**, tương ứng với mốc UTC là **`2026-10-15 17:05:00 UTC`**.
   - Cả hai mốc thời gian được lưu xuống bảng `cdc_system.cdc_worker_schedule` (kiểu `TIMESTAMPTZ` / chuẩn hóa UTC bởi driver `pgx/v5`):
     + `last_run_at` = `2026-10-15 09:46:33+00`
     + `next_run_at` = `2026-10-15 17:05:00+00`
2. Khi Frontend gọi `GET /api/worker-schedule` ([`worker_schedule_read_repo_gorm.go` dòng 60–147](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go#L60-L147)):
   - **Trên máy Local Mac:** OS có `time.Local = Asia/Ho_Chi_Minh (+07:00)`, nên khi `pgx` scan `TIMESTAMPTZ` vào `*time.Time`, Go gắn `Location = +07:00` và serialize JSON thành `"2026-10-15T16:46:33+07:00"` và `"2026-10-16T00:05:00+07:00"`.
   - **Trên môi trường Testing (Docker/K8s container):** Container `cdc-cms-service` (và phiên PostgreSQL trong [`cdc-cms-service/pkgs/database/postgres.go` dòng 24–27](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/database/postgres.go#L24-L27)) chạy với múi giờ mặc định **`UTC`** (`time.Local = UTC`). Vì vậy `json.Marshal` xuất ra chuỗi RFC3339 ở múi giờ **UTC (`Z`)**:
     + `last_run_at`: **`"2026-10-15T09:46:33.xxxxxxZ"`**
     + `next_run_at`: **`"2026-10-15T17:05:00Z"`**

### 2.2. Mắt xích 2 (Trực tiếp gây hiển thị sai trên UI): Hàm `formatNaiveDateTime` trong `cdc-cms-web/src/utils/datetime.ts` vứt bỏ hậu tố `Z` và in thẳng con số giờ UTC!
- Tại [`cdc-cms-web/src/pages/ActivityManager.tsx` (dòng 1532–1543)](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L1532-L1543):
  ```tsx
  {
    title: 'Lần chạy cuối',
    dataIndex: 'last_run_at',
    width: 160,
    render: (v) => (v ? formatNaiveDateTime(v) : <Tag>Chưa chạy</Tag>),
  },
  {
    title: 'Lần chạy kế',
    dataIndex: 'next_run_at',
    width: 160,
    render: (v) => (v ? formatNaiveDateTime(v) : '-'),
  },
  ```
- Tại [`cdc-cms-web/src/utils/datetime.ts` (dòng 1–11)](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/datetime.ts#L1-L11):
  ```ts
  export function formatNaiveDateTime(iso: string | null | undefined): string {
    if (!iso) return '';
    const m = iso.match(/^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})(?::(\d{2}))?/);
    if (!m) return iso;
    const [, y, mo, d, h, mi, s] = m;
    return `${d}/${mo}/${y} ${h}:${mi}:${s ?? '00'}`;
  }
  ```
- **Bản chất lỗi:** Hàm `formatNaiveDateTime` dùng Regex cắt lấy phần chữ số `YYYY-MM-DD HH:mm:ss` ở đầu chuỗi và **bỏ qua hoàn toàn hậu tố `Z` (UTC)** ở cuối chuỗi!
  - Với `last_run_at = "2026-10-15T09:46:33Z"` (tức `16:46:33` giờ VN) $\rightarrow$ cắt số in ra **`15/10/2026 09:46:33`**.
  - Với `next_run_at = "2026-10-15T17:05:00Z"` (tức `16/10/2026 00:05:00` giờ VN) $\rightarrow$ cắt số in ra **`15/10/2026 17:05:00`**.

---

## 3. Giải pháp Đề xuất (The Single Best Approach)

Chuẩn hóa đồng bộ ở cả **Frontend** và **Backend** để mọi môi trường (`Local`, `Testing`, `Staging`, `Production`) đều hiển thị chính xác giờ Việt Nam (`Asia/Ho_Chi_Minh`, `UTC+7`):

### 3.1. Tại Frontend: [`cdc-cms-web/src/utils/datetime.ts`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/datetime.ts)
Nếu chuỗi ISO có chứa thông tin múi giờ (`Z` hoặc `[+-]\d{2}:?\d{2}`), thực hiện quy đổi sang múi giờ `Asia/Ho_Chi_Minh` (`UTC+7`) trước khi format `DD/MM/YYYY HH:mm:ss`:

```ts
// Format chuỗi datetime từ backend sang giờ Việt Nam (Asia/Ho_Chi_Minh, UTC+7)
// theo định dạng DD/MM/YYYY HH:mm:ss.
// - Nếu chuỗi là RFC3339 có hậu tố timezone ("Z" hoặc "+07:00", "+00:00"):
//   chuyển đổi chính xác sang múi giờ Asia/Ho_Chi_Minh (tránh lỗi lệch -7h khi
//   container backend/DB trên Testing chạy ở TZ=UTC và trả về "...Z").
// - Nếu chuỗi là naive timestamp không có hậu tố timezone: giữ nguyên các cụm số.
export function formatNaiveDateTime(iso: string | null | undefined): string {
  if (!iso) return '';
  const trimmed = iso.trim();

  // Kiểm tra xem chuỗi có hậu tố timezone (Z hoặc ±HH:mm / ±HHmm) hay không
  if (/(?:Z|[+-]\d{2}:?\d{2})$/i.test(trimmed)) {
    const date = new Date(trimmed);
    if (!Number.isNaN(date.getTime())) {
      const parts = new Intl.DateTimeFormat('en-GB', {
        timeZone: 'Asia/Ho_Chi_Minh',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      }).formatToParts(date);

      const map: Record<string, string> = {};
      for (const p of parts) {
        map[p.type] = p.value;
      }
      const hour = map.hour === '24' ? '00' : map.hour;
      return `${map.day}/${map.month}/${map.year} ${hour}:${map.minute}:${map.second ?? '00'}`;
    }
  }

  const m = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})(?::(\d{2}))?/);
  if (!m) return iso;
  const [, y, mo, d, h, mi, s] = m;
  return `${d}/${mo}/${y} ${h}:${mi}:${s ?? '00'}`;
}
```

### 3.2. Tại Backend: Chuẩn hóa `Location = Asia/Ho_Chi_Minh` khi trả về JSON từ `GET /api/worker-schedule`
Tại [`cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go#L116-L147):
- Chuyển `LastRunAt`, `NextRunAt`, `CreatedAt`, `UpdatedAt` sang `locHCM` (`time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)`) khi tạo `WorkerScheduleResponse`. Nhờ đó, ngay cả trên container Testing (`TZ=UTC`), API `GET /api/worker-schedule` vẫn luôn trả về chuỗi ISO mang hậu tố `+07:00` (`"2026-10-15T16:46:33+07:00"`, `"2026-10-16T00:05:00+07:00"`).
