# Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution): Confluence Daily Report

- **Parent Page:** `Core Trans Proxy History Service` (ID: `463833042`)
- **Space Key:** `~712020a3dcebcbe93646fea463f119dd38b08c`

---

## 1. Cấu Trúc Payload REST API Confluence

Để một trang mới được tạo ra làm **con trực tiếp** của trang `Core Trans Proxy History Service` (ID `463833042`), payload HTTP gửi lên `POST https://goopay.atlassian.net/wiki/rest/api/content` như sau:

```json
{
  "type": "page",
  "title": "Daily Data Check & Recon Report - 2026-10-12",
  "space": {
    "key": "~712020a3dcebcbe93646fea463f119dd38b08c"
  },
  "ancestors": [
    {
      "id": "463833042"
    }
  ],
  "body": {
    "storage": {
      "value": "<h2>1. Tóm Tắt Luồng Dữ Liệu Ngày 2026-10-12</h2>...",
      "representation": "storage"
    }
  }
}
```

---

## 2. Mã Nguồn Mẫu (Demo Code)

### 2.1. Confluence Client (`internal/infra/external/confluence/client.go`)

```go
package confluence

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Config struct {
	BaseURL      string
	UserEmail    string
	APIToken     string
	SpaceKey     string
	ParentPageID string
}

type Client struct {
	cfg        Config
	authHeader string
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	rawAuth := fmt.Sprintf("%s:%s", cfg.UserEmail, cfg.APIToken)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(rawAuth))
	return &Client{
		cfg:        cfg,
		authHeader: authHeader,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type PageResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
		Base  string `json:"base"`
	} `json:"_links"`
}

type SearchResponse struct {
	Results []PageResponse `json:"results"`
}

// FindChildPage tìm xem trang đã tồn tại dưới parent hay chưa
func (c *Client) FindChildPage(ctx context.Context, title string) (*PageResponse, error) {
	reqURL := fmt.Sprintf("%s/wiki/rest/api/content?spaceKey=%s&title=%s&expand=version",
		c.cfg.BaseURL, url.QueryEscape(c.cfg.SpaceKey), url.QueryEscape(title))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence get error status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}
	if len(searchResp.Results) > 0 {
		return &searchResp.Results[0], nil
	}
	return nil, nil
}

// CreateChildPage tạo trang con dưới ParentPageID: 463833042
func (c *Client) CreateChildPage(ctx context.Context, title string, storageXHTML string) (*PageResponse, error) {
	reqURL := fmt.Sprintf("%s/wiki/rest/api/content", c.cfg.BaseURL)

	payload := map[string]interface{}{
		"type":  "page",
		"title": title,
		"space": map[string]string{"key": c.cfg.SpaceKey},
		"ancestors": []map[string]string{
			{"id": c.cfg.ParentPageID}, // 463833042
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          storageXHTML,
				"representation": "storage",
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence create error status %d: %s", resp.StatusCode, string(body))
	}

	var pageResp PageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
		return nil, err
	}
	return &pageResp, nil
}

// UpdatePage cập nhật trang nếu đã tồn tại
func (c *Client) UpdatePage(ctx context.Context, pageID string, title string, storageXHTML string, newVersion int) (*PageResponse, error) {
	reqURL := fmt.Sprintf("%s/wiki/rest/api/content/%s", c.cfg.BaseURL, pageID)

	payload := map[string]interface{}{
		"type":  "page",
		"title": title,
		"version": map[string]int{
			"number": newVersion,
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          storageXHTML,
				"representation": "storage",
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence update error status %d: %s", resp.StatusCode, string(body))
	}

	var pageResp PageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
		return nil, err
	}
	return &pageResp, nil
}
```

---

### 2.2. XHTML Confluence Storage Format Builder

```go
func FormatConfluenceReport(dateStr string, records []system.DailyFlowRecord, reports []reconmodel.ReconciliationReport) string {
	var sb strings.Builder

	// Header Info Panel
	sb.WriteString(`<ac:structured-macro ac:name="info">
		<ac:rich-text-body>
			<p><strong>BÁO CÁO ĐỐI SOÁT &amp; CHẤT LƯỢNG DỮ LIỆU TỰ ĐỘNG (CDC DW)</strong></p>
			<p>Ngày chốt số liệu: <strong>` + dateStr + `</strong> | Hệ thống: <strong>Core Trans Proxy History Service</strong></p>
		</ac:rich-text-body>
	</ac:structured-macro>`)

	// Table Matrix
	sb.WriteString(`<h3>1. Chi Tiết Luồng Dữ Liệu Từng Bảng (Kafka CDC &rarr; Shadow &rarr; Master)</h3>`)
	sb.WriteString(`<table class="wrapped relative-table" style="width: 100%;">
		<thead>
			<tr>
				<th>Pipeline</th>
				<th>Kafka CDC In</th>
				<th>Shadow Written</th>
				<th>Master Inserted</th>
				<th>Master Updated</th>
				<th>Master Skipped</th>
				<th>Missing / Stale</th>
				<th>Trạng Thái</th>
			</tr>
		</thead>
		<tbody>`)

	for _, r := range records {
		statusColor := "Green"
		statusTitle := "BALANCED"
		if r.Status == "DRIFT" {
			statusColor = "Red"
			statusTitle = "DRIFT"
		} else if r.Status == "WARNING" {
			statusColor = "Yellow"
			statusTitle = "WARNING"
		}

		statusMacro := fmt.Sprintf(`<ac:structured-macro ac:name="status">
			<ac:parameter ac:name="colour">%s</ac:parameter>
			<ac:parameter ac:name="title">%s</ac:parameter>
		</ac:structured-macro>`, statusColor, statusTitle)

		sb.WriteString(fmt.Sprintf(`<tr>
			<td><strong>%s</strong></td>
			<td>%d (C:%d, U:%d)</td>
			<td>%d</td>
			<td><strong>%d</strong></td>
			<td>%d</td>
			<td>%d</td>
			<td>%d / %d</td>
			<td>%s</td>
		</tr>`, r.MasterTable, r.Kafka.Total, r.Kafka.C, r.Kafka.U,
			r.Shadow.Written, r.Master.Inserted, r.Master.Updated, r.Master.Skipped,
			r.Recon.Missing, r.Recon.Stale, statusMacro))
	}

	sb.WriteString(`</tbody></table>`)
	return sb.String()
}
```
