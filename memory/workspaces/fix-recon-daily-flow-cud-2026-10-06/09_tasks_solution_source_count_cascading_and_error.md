# HỒ SƠ GIẢI PHÁP KỸ THUẬT (TECHNICAL SOLUTION): SỬA 3 LỖI SOURCE COUNT

## 1. BACKEND CMS SERVICE (`cdc-cms-service`)

### 1.1. Bổ sung `ListConnectionsWithFilter` trong `master_repo_gorm.go`
```go
func (r *masterRepoGorm) ListConnectionsWithFilter(ctx context.Context, roleFilter string) ([]master.MasterConnectionItem, error) {
	tableName := "cdc_system.connection_registry"
	if os.Getenv("GO_ENV") == "test" {
		tableName = "connection_registry"
	}
	var rows []master.MasterConnectionItem
	query := r.db.WithContext(ctx).Table(tableName).Where("status = 'active'")
	if roleFilter != "" && roleFilter != "all" {
		query = query.Where("role_type = ?", roleFilter)
	}
	err := query.Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
```
Và hàm `ListMasterConnections` hiện tại:
```go
func (r *masterRepoGorm) ListMasterConnections(ctx context.Context) ([]master.MasterConnectionItem, error) {
	return r.ListConnectionsWithFilter(ctx, "master")
}
```

### 1.2. Mở rộng handler `ListConnections` trong `master_registry_handler_connections.go`
```go
func (h *MasterRegistryHandler) ListConnections(c *fiber.Ctx) error {
	role := c.Query("role")
	if role == "" {
		role = c.Query("role_type")
	}
	// Nếu gọi endpoint chung /api/v1/connections và không chỉ định role -> trả về all
	if strings.Contains(c.Path(), "/connections") && !strings.Contains(c.Path(), "master-connections") && role == "" {
		role = "all"
	}

	items, err := h.masterRepo.ListConnectionsWithFilter(c.Context(), role)
	if err != nil {
		h.logger.Error("list connections failed", zap.Error(err), zap.String("role", role))
		return c.Status(500).JSON(fiber.Map{"error": "internal_error"})
	}
	return c.JSON(fiber.Map{"data": items, "count": len(items)})
}
```

### 1.3. Đăng ký router trong `internal/router/router.go`
```go
	shared.Get("/v1/connections", h.Master.Registry.ListConnections)
	dual("GET", shared, "/connections", h.Master.Registry.ListConnections)
```

---

## 2. WORKER SERVER (`centralized-data-service`)
Trong `internal/server/server_scheduler.go`:
```go
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("panic: %v", r)
			observability.Ctx(ctx, s.logger).Error(...)
			s.db.WithContext(ctx).Model(&mastermodel.WorkerSchedule{}).
				Where("id = ?", currentSched.ID).
				Update("last_error", panicMsg)
		}
	}()
```

---

## 3. FRONTEND CMS WEB (`ActivityManager.tsx`)

### 3.1. Query Connection API
```typescript
  const { data: rawConnections } = useQuery<any[]>({
    queryKey: ['connections-for-activity-manager'],
    queryFn: async () => {
      const r = await cmsApi.get('/api/v1/connections');
      return r.data.data || [];
    },
  });
```

### 3.2. Cascading Selector Metadata Lineage
```typescript
  // 1. TẦNG 2: Connection Options theo Layer
  const connectionOptions = useMemo(() => {
    if (!rawConnections) return [];
    if (selectedLayer === 'source') {
      return rawConnections.filter((c) => c.role_type === 'source');
    }
    if (selectedLayer === 'shadow') {
      return rawConnections.filter(
        (c) => c.role_type === 'shadow' || c.connection_code === 'default_shadow'
      );
    }
    if (selectedLayer === 'master') {
      return rawConnections.filter((c) => c.role_type === 'master');
    }
    return rawConnections;
  }, [rawConnections, selectedLayer]);

  // 2. TẦNG 3: Namespace Options theo Connection được chọn
  const namespaceOptions = useMemo(() => {
    const set = new Set<string>();
    const currentConn = (rawConnections || []).find((c) => c.connection_code === selectedConn);

    if (selectedLayer === 'source') {
      if (currentConn?.default_database) set.add(currentConn.default_database);
      sourceObjects.forEach((s) => {
        if (!selectedConn || s.source_connection_code === selectedConn) {
          if (s.source_db) set.add(s.source_db);
        }
      });
    } else if (selectedLayer === 'shadow') {
      if (currentConn?.default_database) set.add(currentConn.default_database);
      if (currentConn?.default_schema) set.add(currentConn.default_schema);
      (rawShadowBindings || []).forEach((s) => {
        if (s.shadow_schema) set.add(s.shadow_schema);
      });
      sourceObjects.forEach((s) => {
        if (s.shadow_schema) set.add(s.shadow_schema);
      });
    } else if (selectedLayer === 'master') {
      // 1. Lấy từ chính connection registry của Master đã chọn
      if (currentConn?.default_schema) set.add(currentConn.default_schema);
      if (currentConn?.default_database) set.add(currentConn.default_database);

      // 2. Lấy các master_schema từ master_binding thuộc đúng connection này
      (rawMasterBindings || []).forEach((m) => {
        const matchConn =
          (selectedConn && m.master_connection_code === selectedConn) ||
          (currentConn && m.master_connection_id === currentConn.id);
        if (matchConn) {
          if (m.master_schema) set.add(m.master_schema);
          if (m.master_database) set.add(m.master_database);
        }
      });
    }
    return Array.from(set).map((ns) => ({ label: ns, value: ns }));
  }, [selectedLayer, selectedConn, rawConnections, sourceObjects, rawShadowBindings, rawMasterBindings]);
```

### 3.3. Tooltip Hiển thị Lỗi Trực quan
```tsx
    {
      title: 'Lỗi gần nhất',
      dataIndex: 'last_error',
      width: 180,
      render: (v) =>
        v ? (
          <Tooltip
            placement="topLeft"
            color="#262626"
            title={
              <div style={{ maxWidth: 450, padding: 4 }}>
                <div style={{ fontWeight: 600, color: '#ff4d4f', marginBottom: 4, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span>Chi tiết lỗi thực thi:</span>
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0, height: 'auto', color: '#40a9ff' }}
                    onClick={(e) => {
                      e.stopPropagation();
                      navigator.clipboard.writeText(v);
                      message.success('Đã sao chép nội dung lỗi!');
                    }}
                  >
                    Sao chép
                  </Button>
                </div>
                <div style={{ wordBreak: 'break-word', whiteSpace: 'pre-wrap', fontFamily: 'monospace', fontSize: 11, color: '#f0f0f0' }}>
                  {v}
                </div>
              </div>
            }
          >
            <Tag
              icon={<CloseCircleOutlined />}
              color="error"
              style={{ cursor: 'pointer', maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}
            >
              Có lỗi (Chi tiết)
            </Tag>
          </Tooltip>
        ) : (
          <Tag icon={<CheckCircleOutlined />} color="success">
            OK
          </Tag>
        ),
    },
```
