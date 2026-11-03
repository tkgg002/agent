# 13 Analysis: TASK-11-M05 (`centralized-data-service`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)` | `VERIFIED` | `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_test_helpers.go` (missing on disk and not in diff.patch: centralized-data-service:internal/service/source/metadata_registry_test_helpers.go) |
| Hợp đồng Đầu vào (`Wave 2`) | `VERIFIED` | `UpstreamReady = false` (DependsOn: `TASK-11-M03, TASK-11-M04`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-11/task-11-m05` (`.orchestrator/worktrees/TASK-11/centralized-data-service`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_test_helpers.go`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_test_helpers.go (lines 13-37)
  13 | type MetadataRegistryServiceForTestOpts struct {
  14 | 	SourceCache    map[string]*source.TableRegistry
  15 | 	RouteCache     map[string][]*metadata.ResolvedSourceRoute
  16 | 	TargetRouteMap map[string]*metadata.ResolvedSourceRoute
  17 | 	MappingCache   map[int64][]mastermodel.MappingRule
  18 | 	TargetCache    map[string]*source.TableRegistry
  19 | 	CloneRoutes    map[int64][]*metadata.ResolvedSourceRoute
  20 | }
  21 | 
  22 | // NewMetadataRegistryServiceForTest preloads a MetadataRegistryService with the supplied cache snapshots.
  23 | func NewMetadataRegistryServiceForTest(opts MetadataRegistryServiceForTestOpts) *MetadataRegistryService {
  24 | 	rs := &MetadataRegistryService{
  25 | 		logger:         zap.NewNop(),
  26 | 		sourceCache:    opts.SourceCache,
  27 | 		routeCache:     opts.RouteCache,
  28 | 		targetRouteMap: opts.TargetRouteMap,
  29 | 		mappingCache:   opts.MappingCache,
  30 | 		targetCache:    opts.TargetCache,
  31 | 		cloneRoutes:    opts.CloneRoutes,
  32 | 	}
  33 | 	if rs.mappingCache == nil {
  34 | 		rs.mappingCache = make(map[int64][]mastermodel.MappingRule)
  35 | 	}
  36 | 	return rs
  37 | }
```
