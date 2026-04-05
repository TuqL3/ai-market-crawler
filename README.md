# AI Problem Aggregator

## Tổng quan

Hệ thống AI crawl GitHub Issues để tổng hợp các vấn đề phổ biến mà developers đang gặp phải. Người dùng xem trending problems trên dashboard và chat với AI (RAG) dựa trên data crawl thực tế.

## Kiến trúc hệ thống

```
                  WebSocket              HTTP SSE
Frontend ◄────────────────► Go Service ────────────────► Python Service
                            │                            (FastAPI)
                            │         RabbitMQ
                            │ ─────────────────────────► Python Service
                            │  (classify, embed,         (consumer)
                            │   cluster, trends)
                            │
                            ▼
                      ┌──────────┐
                      │ Postgres │
                      │ +pgvector│
                      └──────────┘
```

### Luồng dữ liệu

```
1. CRAWL (mỗi 30 phút)
   Go Crawler ──► GitHub API ──► Lưu raw_problems vào DB

2. AI ANALYSIS (async qua RabbitMQ)
   Go publish ──► RabbitMQ ──► Python consume:
   - ClassifyProblems: phân loại vào 12 categories (Claude API)
   - EmbedProblems: tạo vector embeddings (Voyage API)
   - ClusterProblems: gom nhóm bằng HDBSCAN
   - DetectTrends: phát hiện vấn đề đang tăng
   - SummarizeCluster: tóm tắt + key themes + solutions

3. AI CHAT (real-time qua HTTP SSE)
   User hỏi ──► WebSocket ──► Go ──► HTTP SSE ──► Python (FastAPI)
   Python: embed câu hỏi → search pgvector → Claude trả lời → stream chunks
   Go nhận chunks ──► forward qua WebSocket ──► Frontend hiển thị
```

## Tech Stack

| Component | Technology | Mục đích |
|-----------|-----------|----------|
| Crawler + API Gateway | **Go** (gin, gqlgen, gorm, robfig/cron) | Crawl GitHub, GraphQL API, WebSocket server |
| AI Processing | **Python** (FastAPI, anthropic SDK, scikit-learn, SQLAlchemy) | Phân loại, clustering, trend detection, RAG chat |
| Async Messaging | **RabbitMQ** | Go → Python async tasks (analysis pipeline) |
| Real-time Chat | **HTTP SSE** (Go → Python) + **WebSocket** (Frontend → Go) | Chat streaming |
| Database | **PostgreSQL + pgvector** | Lưu trữ data + vector embeddings |
| Frontend | **Next.js + TypeScript + Tailwind + Apollo Client** | Dashboard + Chat UI |
| Monitoring | **Prometheus + Grafana + Loki** | Metrics, dashboard, centralized logging |
| CI/CD | **GitHub Actions** | Lint, test, build, deploy |
| DevOps | **Docker Compose** | Local development + deployment |

## Cấu trúc thư mục

```
ai-problem-aggregator/
├── docker-compose.yml
├── .env.example
├── .github/
│   └── workflows/
│       ├── go-service.yml             # CI/CD cho Go service
│       ├── python-service.yml         # CI/CD cho Python service
│       └── frontend.yml               # CI/CD cho Frontend
│
├── go-service/
│   ├── Dockerfile
│   ├── go.mod
│   ├── cmd/
│   │   ├── api/main.go                # API gateway + WebSocket server
│   │   └── crawler/main.go            # Crawler + scheduler
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── crawler/
│   │   │   └── github.go              # GitHub Issues crawler
│   │   ├── scheduler/scheduler.go     # Cron jobs
│   │   ├── api/
│   │   │   ├── router.go              # HTTP router + GraphQL + WebSocket
│   │   │   └── middleware/
│   │   ├── graph/
│   │   │   ├── schema.graphqls
│   │   │   └── resolvers.go
│   │   ├── rabbitmq/
│   │   │   ├── publisher.go           # Publish tasks to RabbitMQ
│   │   │   └── consumer.go            # Consume results from Python
│   │   ├── store/
│   │   │   ├── postgres.go
│   │   │   ├── problems.go
│   │   │   └── chat.go
│   │   └── models/
│   └── migrations/
│
├── python-service/
│   ├── Dockerfile
│   ├── pyproject.toml
│   ├── src/
│   │   ├── main.py                    # FastAPI server + RabbitMQ consumer
│   │   ├── config.py
│   │   ├── api/
│   │   │   └── chat.py                # SSE endpoint: /chat/stream
│   │   ├── ai/
│   │   │   ├── classifier.py          # Problem classification (Claude API)
│   │   │   ├── clusterer.py           # HDBSCAN clustering
│   │   │   ├── trend_detector.py      # Trend detection
│   │   │   ├── summarizer.py          # Cluster summarization (Claude API)
│   │   │   └── rag.py                 # RAG pipeline cho chat
│   │   ├── rabbitmq/
│   │   │   ├── consumer.py            # Consume tasks from Go
│   │   │   └── publisher.py           # Publish results
│   │   ├── embeddings/store.py        # pgvector operations
│   │   └── db/connection.py
│   └── tests/
│
├── monitoring/
│   ├── prometheus/prometheus.yml      # Prometheus config + scrape targets
│   ├── grafana/
│   │   ├── provisioning/              # Auto-provision datasources + dashboards
│   │   └── dashboards/               # Dashboard JSON files
│   └── loki/loki-config.yml          # Loki config
│
└── frontend/
    ├── Dockerfile
    ├── package.json
    └── apps/web/
        ├── app/
        │   ├── layout.tsx
        │   ├── page.tsx               # Dashboard - trending problems
        │   ├── problems/
        │   │   ├── page.tsx           # Problem list + filters
        │   │   └── [id]/page.tsx      # Problem detail + cluster info
        │   └── chat/page.tsx          # AI chat interface
        ├── components/
        │   ├── dashboard/             # TrendChart, CategoryFilter, ProblemCard, StatsOverview
        │   ├── chat/                  # ChatWindow, MessageBubble, ChatInput
        │   └── ui/                    # Shared UI (shadcn)
        ├── hooks/
        │   ├── useProblems.ts
        │   ├── useTrends.ts
        │   └── useChat.ts
        └── lib/
            ├── apollo.ts
            └── graphql/
                ├── queries.ts
                ├── mutations.ts
                └── subscriptions.ts
```

## Database Schema

### raw_problems
Dữ liệu crawl thô từ GitHub.
```sql
CREATE TABLE raw_problems (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source          VARCHAR(20) NOT NULL,
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,
    content         TEXT NOT NULL,
    sources         JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### crawl_jobs
Tracking trạng thái crawl.
```sql
CREATE TABLE crawl_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source          VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    items_crawled   INTEGER DEFAULT 0,
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## RabbitMQ Queues

| Queue | Publisher | Consumer | Mục đích |
|-------|-----------|----------|----------|
| `problem.classify` | Go (sau khi crawl) | Python | Phân loại problems vào categories |
| `problem.embed` | Go (sau khi crawl) | Python | Tạo vector embeddings |
| `problem.cluster` | Go (scheduler, mỗi 6h) | Python | Gom nhóm problems tương tự |
| `problem.trends` | Go (scheduler, mỗi 6h) | Python | Phát hiện trends |
| `problem.summarize` | Go (sau khi cluster) | Python | Tóm tắt clusters |
| `analysis.result` | Python (sau khi xử lý) | Go | Trả kết quả về Go lưu DB |

## HTTP SSE Endpoint (Chat)

Python FastAPI cung cấp endpoint cho chat streaming:

```
POST /chat/stream
Content-Type: application/json
Accept: text/event-stream

Request:
{
    "session_id": "uuid",
    "question": "React hydration bug là gì?"
}

Response (SSE):
data: {"content": "React hydration"}
data: {"content": " bug xảy ra khi..."}
data: {"content": "...", "done": true, "sources": [...]}
```

## GraphQL API

**Endpoints:**
- `POST /graphql` — Queries & Mutations
- `WS /graphql` — WebSocket (chat streaming)

```graphql
type Query {
  problems(filter: ProblemFilter, page: Int, pageSize: Int): ProblemConnection!
  problem(id: ID!): RawProblem
  clusters(page: Int, pageSize: Int): ClusterConnection!
  cluster(id: ID!): ProblemCluster
  trends(windowDays: Int): [TrendSnapshot!]!
  categories: [CategoryCount!]!
  chatHistory(sessionId: ID!): [ChatMessage!]!
}

type Mutation {
  createChatSession: ChatSession!
  sendMessage(sessionId: ID!, content: String!): ChatMessage!
}

type Subscription {
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

### Phase 1: DevOps Foundation
> Mục tiêu: CI/CD + Docker sẵn sàng trước khi code.

**Docker:**
1. Multi-stage Dockerfile cho Go, Python, Frontend
2. docker-compose.yml: postgres (pgvector), rabbitmq, go-service, python-service, frontend
3. .dockerignore tối ưu build context
4. Health checks cho mỗi service
5. Docker Compose profiles: `dev`, `prod`

**CI/CD (GitHub Actions):**
1. Pipeline cho mỗi service:
   - Lint + test → Build Docker image → Push lên Container Registry
   - Auto trigger khi push/merge vào main
2. Database migration tự động trong CI
3. Environment: staging + production branches

**Verify**: `docker compose up -d` → tất cả services chạy + kết nối nhau ✓

### Phase 2: Foundation
> Mục tiêu: Skeleton chạy được, services giao tiếp qua RabbitMQ + HTTP.

1. Setup RabbitMQ trong docker-compose
2. Go: RabbitMQ publisher/consumer (amqp091-go)
3. Python: FastAPI + RabbitMQ consumer (aio-pika)
4. Database migration (giữ nguyên schema hiện tại)
5. Bỏ toàn bộ gRPC: proto files, generated code, grpcclient
6. **Verify**: Go → Postgres ✓, Go → RabbitMQ → Python ✓, Go → Python HTTP SSE ✓

### Phase 3: Crawlers
> Mục tiêu: Data chảy vào database từ GitHub.

1. GitHub crawler: search issues by labels (bug, help wanted), pagination, rate limit
2. Scheduler: robfig/cron — crawl mỗi 30 phút
3. Sau mỗi crawl batch → publish messages vào RabbitMQ (classify + embed)
4. Upsert logic: ON CONFLICT (source, source_id) DO UPDATE
5. Crawl job tracking
6. **Verify**: GitHub issues chảy vào DB liên tục ✓

### Phase 4: AI Processing Pipeline
> Mục tiêu: Data được phân loại, clustering, phát hiện trends.

1. Python RabbitMQ consumers:
   - `problem.classify` → Classifier (Claude API) → publish result
   - `problem.embed` → Embedding (Voyage API) → lưu pgvector
   - `problem.cluster` → HDBSCAN clustering → publish result
   - `problem.trends` → Growth rate detection → publish result
   - `problem.summarize` → Claude summarization → publish result
2. Go RabbitMQ consumer: nhận results từ `analysis.result` → lưu DB
3. Go scheduler: mỗi 6h publish cluster + trends tasks
4. **Verify**: Crawl → tự động classify + embed → mỗi 6h cluster + trends ✓

### Phase 5: RAG Chat
> Mục tiêu: User chat với AI dựa trên data crawl.

1. Python FastAPI endpoint `/chat/stream` (SSE):
   - Nhận question → embed → search pgvector top-K
   - Build context từ matched problems
   - Call Claude với context → stream response
2. Go WebSocket handler:
   - Frontend gửi message qua WS
   - Go gọi Python `/chat/stream` → nhận SSE chunks
   - Forward chunks qua WS về Frontend
3. Chat history: lưu messages vào DB
4. **Verify**: User hỏi → AI trả lời dựa trên GitHub issues thật ✓

### Phase 6: Frontend
> Mục tiêu: Dashboard và Chat UI hoàn chỉnh.

1. **Dashboard** (`/`):
   - StatsOverview: tổng problems, active clusters, sources
   - TrendChart: trending clusters (recharts)
   - ProblemCard list: latest problems với category tags
   - CategoryFilter sidebar

2. **Problem Detail** (`/problems/[id]`):
   - Full text, metadata, link gốc GitHub
   - Related problems cùng cluster
   - Cluster summary + key themes

3. **Chat** (`/chat`):
   - ChatWindow + message history
   - WebSocket streaming
   - Source citations (link đến GitHub issues)
   - Markdown rendering

### Phase 7: Monitoring & Production
> Mục tiêu: Observability + deploy lên cloud.

**Monitoring & Observability:**
1. **Prometheus**: metrics từ Go (gin-prometheus), Python (prometheus-fastapi), RabbitMQ (rabbitmq_exporter)
2. **Grafana**: dashboard — crawl rate, queue depth, API latency, AI processing time
3. **Loki + Promtail**: centralized logging
4. **Alerting**: queue depth bất thường, crawl fail, API error rate cao

**Code Quality:**
1. Structured logging (Go: slog, Python: structlog)
2. Error handling + retry cho RabbitMQ consumers
3. Dead letter queue cho failed messages

**Cloud Deploy:**
1. VPS (DigitalOcean/Vultr/AWS Lightsail):
   - Docker Compose trên 1 VPS
   - Nginx reverse proxy + SSL (Let's Encrypt)
2. Hoặc Kubernetes khi cần scale:
   - Helm charts, HPA cho Go API + Python workers
   - Managed PostgreSQL + Managed RabbitMQ

**Backup & Recovery:**
1. PostgreSQL automated backup (pg_dump cron)
2. RabbitMQ durable queues + persistent messages
3. Disaster recovery plan

## Environment Variables (.env.example)

```env
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/ai_aggregator

# API Keys
GITHUB_TOKEN=ghp_xxx
ANTHROPIC_API_KEY=sk-ant-xxx

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# Services
PYTHON_SERVICE_URL=http://python-service:8000
GO_API_PORT=8080

# Frontend
NEXT_PUBLIC_GRAPHQL_URL=http://localhost:8080/graphql
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
```

## Key Design Decisions

- **RabbitMQ thay gRPC cho analysis pipeline**: Async, retry tự động, decouple Go/Python, scale workers độc lập
- **HTTP SSE cho chat streaming**: Đơn giản, không cần protobuf/code generation, Python dùng FastAPI native
- **WebSocket (Frontend ↔ Go)**: Real-time chat UI, Go làm proxy + auth + rate limit
- **pgvector thay vì vector DB riêng**: 1 database duy nhất, đủ cho scale hiện tại
- **GraphQL (gqlgen + Gin)**: Flexible queries, frontend fetch đúng data cần
- **Dead letter queue**: Messages fail nhiều lần → chuyển sang DLQ để debug, không mất data
- **DevOps first**: CI/CD + Docker setup trước khi code, đảm bảo mọi thay đổi đều qua pipeline

## Run Project

```bash
docker compose up -d
```
