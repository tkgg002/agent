# 03_implementation_recon_source_agent_refactor

Thiết kế kỹ thuật chi tiết cho việc tái cấu trúc `recon_source_agent.go`.

## 1. Cấu trúc chia tách file đề xuất

### 1.1 `recon_models.go` [NEW]
- Chứa các struct định nghĩa dữ liệu đầu ra và cấu hình:
  - `ChunkHash`
  - `WindowResult`
  - `BucketHashResult`
  - `ReconSourceAgentConfig` (kèm phương thức `applyDefaults`)
- Hằng số mã lỗi: `ErrCodeSrcTimeout`, `ErrCodeSrcConnection`, etc.
- Logic phân loại lỗi MongoDB: `classifyMongoError`, `isMongoTransient` và test helper `ClassifyMongoErrorForTest`.

### 1.2 `recon_hash.go` [NEW]
- Chứa logic băm XOR, xxhash:
  - `HashWindow`
  - `BucketHash`
  - `hashIDPlusTs`, `hashIDPlusTsMs`
  - `bucketIndex`, `extractMongoID`, `extractTimestampMs`
- Các test helpers: `HashIDPlusTsForTest`, `HashIDPlusTsMsForTest`, `BucketIndexForTest`.

### 1.3 `recon_query.go` [NEW]
- Chứa logic đếm số lượng tài liệu và window watermark:
  - `resolveTimestampField`
  - `CountDocuments`
  - `EstimatedCount`
  - `BucketCounts`
  - `CountInWindow`
  - `CountInWindowWithFallback`
  - `MaxWindowTs`
  - `queryWithRetry`

### 1.4 `recon_stream.go` [NEW]
- Chứa logic stream dữ liệu keyset pagination:
  - `ListIDsInWindow`
  - `ListAllIDs` (Deprecated)
  - `StreamAllIDs`

### 1.5 `recon_legacy.go` [NEW]
- Chứa các shim tương thích cũ:
  - `GetChunkHashes`
  - `buildLegacyChunkHash`
  - `redactURL`

### 1.6 `recon_source_agent.go` [MODIFY]
- Chỉ giữ lại cấu trúc lõi `ReconSourceAgent`, constructor `NewReconSourceAgent` và `NewReconSourceAgentWithConfig`.
- Các helper quản lý connection: `getClient`, `getBreaker`, `selectOpts`, `secondaryColl`.

## 2. Quản lý Imports
Mỗi file sẽ chỉ import các package Go cần thiết cho chính logic bên trong nó để đảm bảo không bị thừa imports gây lỗi biên dịch Go:
- `recon_models.go`: `"errors"`, `"strings"`, `"time"`, `"github.com/sony/gobreaker"`, `"go.mongodb.org/mongo-driver/mongo"`
- `recon_hash.go`: `"context"`, `"encoding/binary"`, `"fmt"`, `"strconv"`, `"strings"`, `"time"`, `"github.com/cespare/xxhash/v2"`, `"go.mongodb.org/mongo-driver/bson"`, `"go.mongodb.org/mongo-driver/bson/primitive"`
- `recon_query.go`: `"context"`, `"fmt"`, `"time"`, `"centralized-data-service/internal/service/governance"`, `"go.mongodb.org/mongo-driver/bson"`, `"go.mongodb.org/mongo-driver/mongo"`
- `recon_stream.go`: `"context"`, `"fmt"`, `"go.mongodb.org/mongo-driver/bson"`, `"go.mongodb.org/mongo-driver/bson/primitive"`, `"go.mongodb.org/mongo-driver/mongo"`, `"go.mongodb.org/mongo-driver/mongo/options"`
- `recon_legacy.go`: `"context"`, `"crypto/md5"`, `"fmt"`, `"sort"`, `"strings"`
- `recon_source_agent.go`: `"context"`, `"fmt"`, `"strings"`, `"sync"`, `"time"`, `"centralized-data-service/pkgs/mongodb"`, `"github.com/sony/gobreaker"`, `"go.mongodb.org/mongo-driver/mongo"`, `"go.mongodb.org/mongo-driver/mongo/options"`, `"go.mongodb.org/mongo-driver/mongo/readpref"`, `"go.uber.org/zap"`, `"golang.org/x/time/rate"`
