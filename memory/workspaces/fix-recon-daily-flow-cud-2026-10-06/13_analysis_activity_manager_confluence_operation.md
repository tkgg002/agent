# Phân Tích & Thiết Kế: Tác Vụ Confluence Daily Trên /activity-manager (Multi-Table Selection & Daily Source Count Integration)

- **Ngày khởi tạo:** 2026-10-12
- **Tác giả:** Brain (Chairman & Architect)
- **Yêu cầu từ User:**
  > *"rồi vụ Confluence daily thì sao. cũng ở operation tạo 1 cái confluence, cho chọn table. cho chọn cấu hình confluence để bắn lên atlassian.com. (nó sẽ vào daily_source_counts để lấy số lượng record table nếu có + số lượng  ) cái này đc chọn nhiều table 1 lúc"*

---

## I. MỤC TIÊU & BẢN CHẤT THIẾT KẾ

1. **Quản lý tập trung trên `/activity-manager`:**
   - Thêm tác vụ `confluence` (Báo cáo Confluence hàng ngày) vào danh mục `ALL_OPERATIONS`.
   - Admin có thể tạo nhiều lịch trình Confluence khác nhau:
     - Lịch trình 1: Bắn nhóm Pipeline Core Payment lên Confluence A (Space Core).
     - Lịch trình 2: Bắn nhóm Pipeline Partner / Clearing lên Confluence B (Space Partner).

2. **Cấu hình trên Form Modal:**
   - **Chọn Cấu hình Confluence Đích:** Dropdown chọn từ danh sách cấu hình lưu trong bảng `cdc_system.confluence_report_configs` (Base URL, Space Key, Parent Page ID, Email, API Token).
   - **Chọn Tables (Multi-Select nhiều table một lúc):** Cho phép chọn danh sách nhiều bảng/collection/file nguồn cùng lúc.
   - **Cấu hình thời gian chạy:** Mặc định chạy vào `06:00 AM` mỗi sáng (sau khi job `source-count` đã chốt sổ lúc `00:05`).

3. **Luồng thu thập dữ liệu & Bắn báo cáo:**
   - Khi job thức dậy lúc `06:00 AM`:
   - Duyệt qua từng table trong danh sách được chọn:
     - **Truy vấn 1:** Đọc từ bảng `cdc_system.daily_source_counts` để lấy số lượng bản ghi ngày tạo nghiệp vụ hôm qua ($D-1$) của table đó. Nếu có thì hiển thị số lượng, nếu chưa có/chưa chạy thì hiển thị `Chưa đếm` hoặc `-`.
     - **Truy vấn 2:** Đọc từ bảng thống kê lưu lượng CDC (`recon_hourly_pipeline_stats` hoặc daily telemetry) để lấy tổng số lượng:
       - `Kafka Ingested`: Số message CDC Kafka đã tiếp nhận trong ngày.
       - `Shadow Ingested`: Số dòng ghi vào Shadow DB (Insert / Update).
       - `Master Ingested`: Số dòng ghi vào Master DW (Upsert).
     - **Đối soát tự động:** So sánh `Source Creation Count` vs `CDC Flow` để gắn cờ trạng thái: `MATCH` (Khớp), `DRIFT` (Lệch), hoặc `PROCESSING`.
   - **Format XHTML chuẩn Confluence Storage Format.**
   - **Tạo trang con (Child Page):** Gọi Atlassian Confluence REST API để tạo trang con trực tiếp dưới `Parent Page ID` (ví dụ `463833042` - `Core Trans Proxy History Service`).

---

## II. THIẾT KẾ CSDL & SCHEMA

### 1. Bảng quản lý cấu hình Confluence đích `cdc_system.confluence_report_configs`
```sql
CREATE TABLE IF NOT EXISTS cdc_system.confluence_report_configs (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL UNIQUE,          -- Tên định danh, vd: 'core_trans_proxy_daily'
    display_title   VARCHAR(200) NOT NULL,                 -- Tiêu đề cấu hình, vd: 'Core Trans Proxy Daily Report'
    title_template  VARCHAR(255) NOT NULL,                 -- Template tiêu đề trang: '[Daily Data Check] {{.ReportDate}} - Core Trans Proxy'
    base_url        VARCHAR(255) NOT NULL,                 -- 'https://goopay.atlassian.net'
    space_key       VARCHAR(100) NOT NULL,                 -- '~712020a3dcebcbe93646fea463f119dd38b08c'
    parent_page_id  VARCHAR(50) NOT NULL,                  -- '463833042'
    user_email      VARCHAR(150) NOT NULL,                 -- Atlassian Account Email
    api_token       VARCHAR(255) NOT NULL,                 -- Atlassian API Token
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    last_run_at     TIMESTAMPTZ,
    last_status     VARCHAR(32),                           -- 'SUCCESS', 'FAILED'
    last_page_url   TEXT,                                  -- Link tới trang vừa tạo thành công
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 2. Dữ liệu lưu trong `cdc_worker_schedule.params` khi tạo operation `confluence`:
```json
{
  "confluence_config_id": 1,
  "report_date_mode": "yesterday",
  "timezone": "Asia/Ho_Chi_Minh",
  "selected_tables": [
    {
      "layer_type": "source",
      "connection_code": "mongo_ctphs_prod",
      "container_namespace": "traitest_ctphs",
      "object_name": "trans_his",
      "object_type": "collection",
      "display_name": "Core Proxy Transactions"
    },
    {
      "layer_type": "source",
      "connection_code": "pg_pbs_prod",
      "container_namespace": "master_payment_bill_service",
      "object_name": "payment_bills",
      "object_type": "table",
      "display_name": "Payment Bills Service"
    },
    {
      "layer_type": "source",
      "connection_code": "sftp_vcb_clearing",
      "container_namespace": "/reconciliation/inbox/vcb/",
      "object_name": "clearing_vcb_*.csv",
      "object_type": "file_batch",
      "display_name": "VCB Clearing Bank Statements"
    }
  ]
}
```

---

## III. GIAO DIỆN FORM MODAL TRÊN `/activity-manager` (`ActivityManager.tsx`)

Khi người dùng chọn `operation = 'confluence'`:
```tsx
{/* 1. CHỌN CẤU HÌNH CONFLUENCE ĐÍCH */}
<Form.Item 
  name="confluence_config_id" 
  label="Cấu hình Confluence đích (Atlassian)" 
  rules={[{ required: true, message: 'Vui lòng chọn cấu hình Confluence đích' }]}
>
  <Select placeholder="Chọn Confluence Space & Trang Cha">
    {confluenceConfigs.map(cfg => (
      <Select.Option key={cfg.id} value={cfg.id}>
        <Space>
          <Tag color="blue">{cfg.space_key}</Tag>
          <strong>{cfg.display_title}</strong>
          <Text type="secondary" style={{ fontSize: 11 }}>(Parent Page ID: {cfg.parent_page_id})</Text>
        </Space>
      </Select.Option>
    ))}
  </Select>
</Form.Item>

{/* 2. CHỌN NHIỀU TABLES / PIPELINES CÙNG LÚC (MULTI-SELECT) */}
<Form.Item 
  name="selected_table_keys" 
  label="Danh sách Tables / Pipelines muốn đưa vào báo cáo (Chọn nhiều)" 
  rules={[{ required: true, message: 'Vui lòng chọn ít nhất 1 table/pipeline' }]}
  tooltip="Hệ thống sẽ lấy số lượng từ daily_source_counts (nếu có) cộng với số lượng lưu lượng CDC trong ngày của từng table để đối soát."
>
  <Select 
    mode="multiple" 
    placeholder="Chọn các tables/pipelines (vd: trans_his, payment_bills, orders...)"
    allowClear
    style={{ width: '100%' }}
    optionFilterProp="label"
  >
    {availableSourceObjects.map(obj => (
      <Select.Option key={obj.key} value={obj.key} label={`${obj.object_name} (${obj.engine_type})`}>
        <Space>
          <Tag color={getEngineColor(obj.engine_type)}>{obj.engine_type.toUpperCase()}</Tag>
          <strong>{obj.object_name}</strong>
          <Text type="secondary" style={{ fontSize: 11 }}>[{obj.container_namespace}]</Text>
          <Tag color="cyan">{obj.object_type}</Tag>
        </Space>
      </Select.Option>
    ))}
  </Select>
</Form.Item>

{/* 3. MỐC GIỜ CHẠY */}
<Form.Item name="run_at_time" label="Giờ chạy hàng ngày (HH:mm)" initialValue="06:00" rules={[{ required: true }]}>
  <Input placeholder="06:00" style={{ width: 140 }} />
</Form.Item>
```

---

## IV. BẢNG MẪU BÁO CÁO CONFLUENCE TỰ ĐỘNG SINH

Khi trang con được tạo dưới `Core Trans Proxy History Service` (Page ID: `463833042`), bảng tổng hợp có định dạng:

| Pipeline / Bảng | Loại Nguồn | Số Bản Ghi Ngày Tạo (`daily_source_counts`) | Lưu Lượng Kafka Ingested | Shadow Ghi Nhận (Insert / Update) | Master Ghi Nhận (Upsert) | Trạng Thái Đối Soát |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: |
| **`trans_his`** (Core Proxy) | `MongoDB Collection` | **150,230** | 150,230 | 150,230 (150k I / 230 U) | 150,230 | 🟢 **MATCH (Khớp 100%)** |
| **`payment_bills`** (Bills) | `PostgreSQL Table` | **85,410** | 85,410 | 85,410 (85k I / 410 U) | 85,410 | 🟢 **MATCH (Khớp 100%)** |
| **`clearing_vcb_*.csv`** | `SFTP File Batch` | **12,500** | 12,500 | 12,500 (12.5k I / 0 U) | 12,500 | 🟢 **MATCH (Khớp 100%)** |
| **`orders`** (Order Svc) | `MySQL Table` | *Chưa đếm* | 45,120 | 45,120 (45k I / 120 U) | 45,120 | 🟡 **CDC OK (Source N/A)** |

- Cột **"Số Bản Ghi Ngày Tạo"** đọc trực tiếp từ `cdc_system.daily_source_counts`.
- Cột **"Lưu Lượng Kafka / Shadow / Master"** đọc từ telemetry CDC trong ngày.
- Báo cáo đối soát chéo 2 chiều giữa số nghiệp vụ phát sinh tại nguồn và số bản ghi di chuyển qua luồng CDC.
