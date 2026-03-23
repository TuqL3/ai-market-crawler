# AI Problem Aggregator - Implementation Plan

## Tổng quan
Hệ thống AI crawl GitHub Issues, StackOverflow, Reddit để tổng hợp các vấn đề phổ biến mà developers đang gặp phải. Người dùng xem trending problems trên dashboard và chat với AI.

## Kiến trúc hệ thống

```
┌─────────────┐ GraphQL ┌──────────────┐         ┌────────────────┐
│  Next.js    │  /WS    │  Go Service  │  gRPC   │ Python Service │
│  Frontend   │◄───────►│  - Crawler   │◄───────►│  - AI/NLP      │
│  - Dashboard│         │  - GraphQL   │         │  - RAG Chat    │
│  - Chat     │         │  - Scheduler │         │  - Clustering  │
└─────────────┘         └──────┬───────┘         └───────┬────────┘
                               │                         │
                               └────────┬────────────────┘
                                        ▼
                                  ┌──────────┐
                                  │ Postgres │
                                  │ +pgvector│
                                  └──────────┘
```

## Tech Stack

| Component | Technology | Mục đích |
|-----------|-----------|----------|
| Crawler + API Gateway | **Go** (gqlgen, chi, gorm, robfig/cron) | Crawl data, GraphQL API, WebSocket, scheduling |
| AI Processing | **Python** (anthropic SDK, scikit-learn) | Phân loại, clustering, trend detection, RAG |
| Communication | **gRPC** (buf) | Go ↔ Python service |
| Database | **PostgreSQL + pgvector** | Lưu trữ data + vector embeddings |
| Frontend | **Next.js + TypeScript + Tailwind + Apollo Client** | Dashboard + Chat UI (GraphQL) |
| DevOps | **Docker Compose** | Local development |

## Cấu trúc thư mục

```
ai-marketplace/
├── docker-compose.yml
├── Makefile
├── .env.example
│
├── proto/                              # gRPC definitions (shared)
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   └── aggregator/v1/
│       ├── analysis.proto              # ClassifyProblems, ClusterProblems, DetectTrends, SummarizeCluster, EmbedProblems
│       └── chat.proto                  # Ask, AskStream (server-side streaming)
│
├── go-service/                         # Go: Crawler + API Gateway
│   ├── Dockerfile
│   ├── go.mod / go.sum
│   ├── cmd/
│   │   ├── api/main.go                # API gateway entrypoint
│   │   └── crawler/main.go            # Crawler/scheduler entrypoint
│   ├── internal/
│   │   ├── config/config.go           # Env/yaml config loading
│   │   ├── crawler/
│   │   │   ├── crawler.go             # Crawler interface
│   │   │   ├── github.go              # GitHub Issues crawler (go-github)
│   │   │   ├── stackoverflow.go       # StackExchange API crawler
│   │   │   └── reddit.go              # Reddit API crawler (OAuth2)
│   │   ├── scheduler/scheduler.go     # Cron-based job scheduling
│   │   ├── api/
│   │   │   ├── router.go              # HTTP router (chi) + GraphQL endpoint
│   │   │   └── middleware/            # CORS, rate limit, logging
│   │   ├── graph/
│   │   │   ├── schema.graphqls        # GraphQL schema definition
│   │   │   ├── schema.resolvers.go    # Query/Mutation/Subscription resolvers
│   │   │   ├── model/models_gen.go    # Generated GraphQL models
│   │   │   └── generated.go           # gqlgen generated runtime
│   │   ├── grpcclient/client.go       # gRPC client to Python service
│   │   ├── store/                     # Postgres queries (pgx)
│   │   │   ├── postgres.go            # Connection pool
│   │   │   ├── problems.go
│   │   │   ├── crawldata.go
│   │   │   └── chat.go
│   │   └── models/                    # Go structs
│   ├── gen/aggregator/v1/             # Generated gRPC Go code
│   └── migrations/
│       ├── 001_initial.up.sql
│       └── 001_initial.down.sql
│
├── python-service/                     # Python: AI/NLP Processing
│   ├── Dockerfile
│   ├── pyproject.toml
│   ├── src/
│   │   ├── main.py                    # gRPC server entrypoint
│   │   ├── config.py
│   │   ├── grpc_server/
│   │   │   ├── analysis_servicer.py   # Implements AnalysisService
│   │   │   └── chat_servicer.py       # Implements ChatService
│   │   ├── ai/
│   │   │   ├── classifier.py          # Problem classification (Claude API)
│   │   │   ├── clusterer.py           # Similarity clustering (embeddings + HDBSCAN)
│   │   │   ├── trend_detector.py      # Trend detection (growth rate)
│   │   │   ├── summarizer.py          # Cluster summarization (Claude API)
│   │   │   └── rag.py                 # RAG pipeline for chat
│   │   ├── embeddings/store.py        # pgvector operations
│   │   └── db/connection.py           # Async postgres connection
│   ├── gen/aggregator/v1/             # Generated gRPC Python code
│   └── tests/
│
└── frontend/                           # Next.js Frontend
    ├── Dockerfile
    ├── package.json
    ├── src/
    │   ├── app/
    │   │   ├── layout.tsx
    │   │   ├── page.tsx               # Dashboard home
    │   │   ├── problems/
    │   │   │   ├── page.tsx           # Problem list + filters
    │   │   │   └── [id]/page.tsx      # Problem detail
    │   │   └── chat/page.tsx          # Chat interface
    │   ├── components/
    │   │   ├── dashboard/             # TrendChart, CategoryFilter, ProblemCard, StatsOverview
    │   │   ├── chat/                  # ChatWindow, MessageBubble, ChatInput
    │   │   └── ui/                    # Shared UI primitives (shadcn)
    │   ├── hooks/
    │   │   ├── useProblems.ts
    │   │   ├── useTrends.ts
    │   │   └── useChat.ts            # WebSocket hook
    │   ├── lib/
    │   │   ├── apollo.ts              # Apollo Client setup
    │   │   ├── graphql/
    │   │   │   ├── queries.ts         # GraphQL queries
    │   │   │   ├── mutations.ts       # GraphQL mutations
    │   │   │   └── subscriptions.ts   # GraphQL subscriptions (chat)
    │   │   └── ws.ts                  # WebSocket client
    │   └── types/index.ts
    └── public/
```

## Database Schema

### raw_problems
Dữ liệu crawl thô từ các nguồn.
```sql
CREATE TABLE raw_problems (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source          VARCHAR(20) NOT NULL,       -- github, stackoverflow, reddit
    source_id       VARCHAR(255) NOT NULL,
    url             TEXT NOT NULL,
    title           TEXT NOT NULL,
    body            TEXT,
    tags            TEXT[],
    score           INTEGER DEFAULT 0,
    answer_count    INTEGER DEFAULT 0,
    comment_count   INTEGER DEFAULT 0,
    source_created  TIMESTAMPTZ NOT NULL,
    crawled_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source, source_id)
);
```

### classified_problems
Kết quả phân loại bởi AI.
```sql
CREATE TABLE classified_problems (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    raw_problem_id  UUID NOT NULL REFERENCES raw_problems(id) ON DELETE CASCADE,
    category        VARCHAR(100) NOT NULL,
    subcategories   TEXT[],
    confidence      REAL NOT NULL,
    classified_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(raw_problem_id)
);
```

### problem_clusters + cluster_members
Nhóm các problems tương tự.
```sql
CREATE TABLE problem_clusters (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    label           VARCHAR(255) NOT NULL,
    summary         TEXT,
    key_themes      TEXT[],
    common_solutions TEXT[],
    cohesion_score  REAL,
    problem_count   INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cluster_members (
    cluster_id      UUID NOT NULL REFERENCES problem_clusters(id) ON DELETE CASCADE,
    raw_problem_id  UUID NOT NULL REFERENCES raw_problems(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, raw_problem_id)
);
```

### trend_snapshots
Dữ liệu trending theo time window.
```sql
CREATE TABLE trend_snapshots (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cluster_id      UUID REFERENCES problem_clusters(id),
    label           VARCHAR(255),
    problem_count   INTEGER,
    growth_rate     REAL,
    window_start    TIMESTAMPTZ NOT NULL,
    window_end      TIMESTAMPTZ NOT NULL,
    snapshot_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### problem_embeddings (pgvector)
Vector embeddings cho RAG chat.
```sql
CREATE TABLE problem_embeddings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    raw_problem_id  UUID NOT NULL REFERENCES raw_problems(id) ON DELETE CASCADE,
    embedding       vector(1536),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(raw_problem_id)
);
```

### chat_sessions + chat_messages
Lịch sử chat.
```sql
CREATE TABLE chat_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id      UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,        -- user, assistant
    content         TEXT NOT NULL,
    sources         JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### crawl_jobs
Tracking trạng thái crawl.
```sql
CREATE TABLE crawl_jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source          VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    items_crawled   INTEGER DEFAULT 0,
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## gRPC Services

### AnalysisService (analysis.proto)
```protobuf
service AnalysisService {
  rpc ClassifyProblems(ClassifyRequest) returns (ClassifyResponse);
  rpc ClusterProblems(ClusterRequest) returns (ClusterResponse);
  rpc DetectTrends(TrendRequest) returns (TrendResponse);
  rpc SummarizeCluster(SummarizeRequest) returns (SummarizeResponse);
  rpc EmbedProblems(EmbedRequest) returns (EmbedResponse);
}
```

### ChatService (chat.proto)
```protobuf
service ChatService {
  rpc Ask(AskRequest) returns (AskResponse);
  rpc AskStream(AskRequest) returns (stream AskChunk);  // Streaming cho real-time chat
}
```

## GraphQL API

**Endpoints:**
- `POST /graphql` — GraphQL queries & mutations
- `GET /playground` — GraphQL Playground (dev only)
- `WS /graphql` — GraphQL subscriptions (chat streaming)

### Schema Overview

```graphql
type Query {
  # Problems
  problems(filter: ProblemFilter, page: Int, pageSize: Int): ProblemConnection!
  problem(id: ID!): RawProblem

  # Clusters
  clusters(page: Int, pageSize: Int): ClusterConnection!
  cluster(id: ID!): ProblemCluster

  # Trends & Categories
  trends(windowDays: Int): [TrendSnapshot!]!
  categories: [CategoryCount!]!

  # Chat
  chatHistory(sessionId: ID!): [ChatMessage!]!
}

type Mutation {
  # Chat
  createChatSession: ChatSession!
  sendMessage(sessionId: ID!, content: String!): ChatMessage!
}

type Subscription {
  # Real-time chat streaming (replaces WebSocket)
  messageStream(sessionId: ID!): ChatChunk!
}

input ProblemFilter {
  source: String
  category: String
  tags: [String!]
  dateFrom: String
  dateTo: String
  minScore: Int
}
```

## Các Phase triển khai

### Phase 1: Foundation
> Mục tiêu: Skeleton chạy được, các service giao tiếp được với nhau.

1. Init monorepo: `go mod init`, `pyproject.toml`, `npx create-next-app`, `docker-compose.yml`
2. Viết proto files + buf config → generate Go/Python stubs
3. SQL migration `001_initial.up.sql` (dùng golang-migrate)
4. Go: config loading + Postgres connection pool (pgx)
5. Python: minimal gRPC server (stub responses)
6. Go: gRPC client kết nối Python service
7. Docker Compose: postgres (pgvector/pgvector:pg16), go-service, python-service, frontend
8. **Verify**: Go → Postgres ✓, Go → Python gRPC ✓

### Phase 2: Crawlers
> Mục tiêu: Data chảy vào database từ cả 3 nguồn.

1. Define `Crawler` interface trong Go:
   ```go
   type Crawler interface {
       Crawl(ctx context.Context, since time.Time) ([]models.RawProblem, error)
       Source() string
   }
   ```
2. GitHub crawler: `go-github` library, search issues by labels (bug, help wanted), pagination, rate limit handling
3. StackOverflow crawler: StackExchange API `/questions`, filter by activity/votes, quota management
4. Reddit crawler: OAuth2 client credentials → `/r/{subreddit}/search`, 1s delay between requests
5. Scheduler: `robfig/cron` — GitHub mỗi 30 phút, SO mỗi 1 giờ, Reddit mỗi 1 giờ
6. Upsert logic: `ON CONFLICT (source, source_id) DO UPDATE`
7. Crawl job tracking trong `crawl_jobs` table

### Phase 3: AI Processing Pipeline
> Mục tiêu: Data được phân loại, clustering, phát hiện trends.

1. **Classifier** (`classifier.py`): Claude API + structured prompt → JSON categories
2. **Embeddings** (`embeddings/store.py`): pgvector via asyncpg, lưu vector 1536-dim
3. **Clusterer** (`clusterer.py`): Embeddings + HDBSCAN cho initial clustering, Claude cho label generation
4. **Trend Detector** (`trend_detector.py`): So sánh cluster sizes giữa time windows, tính growth rate
5. **Summarizer** (`summarizer.py`): Claude → summary, key themes, common solutions
6. Wire up gRPC `analysis_servicer.py`
7. Go trigger: sau mỗi crawl batch → `ClassifyProblems` + `EmbedProblems`; mỗi 6h → `ClusterProblems` + `DetectTrends`

### Phase 4: GraphQL API Gateway
> Mục tiêu: Frontend có GraphQL API để query.

1. Setup gqlgen: schema definition (`schema.graphqls`), code generation
2. Implement resolvers: queries (problems, clusters, trends, categories), mutations (chat)
3. GraphQL subscriptions cho real-time chat streaming → gRPC `ChatService.AskStream`
4. Mount GraphQL handler trên Chi router (`/graphql`, `/playground`)
5. Middleware: CORS, rate limiting (`golang.org/x/time/rate`), structured logging
6. Pagination: cursor-based hoặc offset-based cho lists

### Phase 5: RAG Chat
> Mục tiêu: User hỏi → AI trả lời dựa trên crawled data.

1. RAG pipeline (`rag.py`):
   - Nhận question → generate embedding
   - Query pgvector top-K similar problems (cosine similarity)
   - Build context từ matched problems (title, body, source, URL)
   - Call Claude với system prompt + context + user question
   - Stream response
2. gRPC `chat_servicer.py` wiring
3. **E2E**: GraphQL Subscription → Go resolver → gRPC AskStream → RAG → streamed response → WebSocket → UI

### Phase 6: Frontend
> Mục tiêu: Dashboard và Chat UI hoàn chỉnh.

1. Next.js + TypeScript + Tailwind + Apollo Client setup
2. **Dashboard** (`/`):
   - `StatsOverview`: tổng problems, active clusters, sources breakdown
   - `TrendChart`: recharts line/bar chart trending clusters
   - `ProblemCard` list: latest/top problems với source badges, category tags
   - `CategoryFilter` sidebar: filter by category, source, date range
3. **Problem Detail** (`/problems/[id]`):
   - Full text, metadata, link gốc
   - Related problems cùng cluster
   - Cluster summary
4. **Chat** (`/chat`):
   - `ChatWindow` + message history
   - `ChatInput` với submit
   - `useChat` hook: GraphQL subscription lifecycle, reconnection, message state
   - `MessageBubble`: render markdown (react-markdown), source citations clickable
5. Responsive layout, dark mode

### Phase 7: Polish & Production
> Mục tiêu: Production-ready.

1. Structured logging (Go: `slog`, Python: `structlog`)
2. Health checks: `/healthz`, `/readyz`
3. Prometheus metrics: crawl counts, processing latency, API durations
4. Multi-stage Dockerfiles (build + runtime)
5. Integration tests: Docker Postgres + mock HTTP cho crawlers

## Environment Variables (.env.example)

```env
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/ai_aggregator

# API Keys
GITHUB_TOKEN=ghp_xxx
STACKOVERFLOW_API_KEY=xxx
REDDIT_CLIENT_ID=xxx
REDDIT_CLIENT_SECRET=xxx
ANTHROPIC_API_KEY=sk-ant-xxx

# Services
PYTHON_GRPC_ADDR=python-service:50052
GO_API_PORT=8080
GO_CRAWLER_ENABLED=true

# Frontend
NEXT_PUBLIC_GRAPHQL_URL=http://localhost:8080/graphql
NEXT_PUBLIC_GRAPHQL_WS_URL=ws://localhost:8080/graphql
```

## Key Design Decisions

- **2 Go binaries riêng biệt** (api + crawler): scaling khác nhau - API scale theo traffic, crawler chỉ cần 1 instance
- **pgvector thay vì vector DB riêng** (Pinecone, Qdrant): đơn giản, 1 database duy nhất, đủ cho scale hiện tại
- **buf thay vì protoc trực tiếp**: quản lý dependencies, linting, code gen dễ hơn
- **GraphQL thay vì REST**: Flexible queries, frontend chỉ fetch đúng data cần thiết, nested relationships (problem → cluster → trends) trong 1 request
- **gqlgen (schema-first)**: Type-safe, auto-generate resolvers từ schema, tích hợp tốt với Go ecosystem
- **Apollo Client**: Cache management, optimistic UI, GraphQL subscriptions cho real-time chat
- **Streaming gRPC cho chat**: Claude trả tokens từng phần → stream qua gRPC → GraphQL subscription → real-time UX
- **Rate limit strategy**: Mỗi platform có limit riêng (GitHub 5000/h, SO 10000/day, Reddit 1 req/s) → per-source limiter
