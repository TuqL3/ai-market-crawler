# Shopee Smart Deal - AI Săn Sale Thông Minh

## Tổng quan

Hệ thống AI giúp người dùng săn sale thông minh trên Shopee:
- **Price Tracker**: Theo dõi lịch sử giá, phát hiện giảm giá ảo
- **Voucher Aggregator**: Tổng hợp mã giảm giá, nhắc lịch lấy voucher
- **AI Chat**: Tư vấn mua sắm thông minh dựa trên data thực
- **Telegram Bot**: Alert deal thật + nhắc voucher real-time

## Kiến trúc hệ thống

```
┌──────────────┐  GraphQL   ┌────────────────┐  RabbitMQ   ┌─────────────────┐
│  Next.js     │   / WS     │  Go Service    │  (async)    │  Python Service  │
│  Frontend    │◄──────────►│  - Crawler     │────────────►│  - AI Analysis   │
│  - Dashboard │            │  - GraphQL API │             │  - Price Predict │
│  - Chat      │            │  - Scheduler   │◄───── gRPC ─│  - RAG Chat      │
└──────────────┘            └──────┬─────────┘             └────────┬─────────┘
                                   │                                │
┌──────────────┐                   └──────────┬─────────────────────┘
│ Telegram Bot │◄── Go Service                │
└──────────────┘                        ┌─────▼──────┐
                                        │  Postgres  │
                                        │  +pgvector │
                                        └────────────┘
                                        ┌────────────┐
                                        │  RabbitMQ  │
                                        └────────────┘
```

### Luồng dữ liệu

```
1. CRAWL (mỗi 6 tiếng)
   Go Crawler ──► Shopee Affiliate API ──► Lưu products + giá vào DB

2. PRICE TRACKING (mỗi 12 tiếng)
   Go Scheduler ──► Crawl giá mới ──► So sánh với giá cũ ──► Phát hiện biến động

3. VOUCHER CRAWL (mỗi 3 tiếng + trước sự kiện sale)
   Go Crawler ──► Shopee voucher page ──► Lưu voucher vào DB

4. AI ANALYSIS (async qua RabbitMQ)
   Go publish ──► RabbitMQ ──► Python consume:
   - Phát hiện giảm giá ảo (so sánh giá 30 ngày)
   - Phân loại deal: DEAL THẬT / DEAL ẢO / BÌNH THƯỜNG
   - Tính điểm deal (giá hiện tại vs giá thấp nhất lịch sử)

5. ALERT (real-time)
   Python phát hiện deal tốt ──► RabbitMQ ──► Go ──► Telegram Bot ──► User

6. AI CHAT (real-time qua gRPC)
   User hỏi ──► GraphQL ──► Go ──► gRPC ──► Python RAG ──► Stream response
```

## Tech Stack

| Component | Technology | Mục đích |
|-----------|-----------|----------|
| Crawler + API | **Go** (gin, gqlgen, gorm, robfig/cron) | Crawl Shopee, GraphQL API, Scheduling |
| AI Processing | **Python** (anthropic SDK, pandas, SQLAlchemy) | Phân tích giá, phát hiện fake sale, RAG chat |
| Async Messaging | **RabbitMQ** | Go → Python async tasks (analysis, alerts) |
| Real-time Chat | **gRPC** (buf) | Python → Go streaming chat response |
| Database | **PostgreSQL + pgvector** | Products, giá, vouchers, embeddings |
| Frontend | **Next.js + TypeScript + Tailwind + Apollo** | Dashboard, Chat UI |
| Notification | **Telegram Bot API** | Alert deal + nhắc voucher |
| DevOps | **Docker Compose** | Local development |

## Cấu trúc thư mục

```
shopee-smart-deal/
├── docker-compose.yml
├── .env.example
│
├── proto/                              # gRPC definitions
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   └── deal/v1/
│       ├── analysis.proto              # AnalyzeDeal, DetectFakeSale
│       └── chat.proto                  # Ask, AskStream
│
├── go-service/
│   ├── Dockerfile
│   ├── go.mod
│   ├── cmd/
│   │   ├── api/main.go                # API gateway
│   │   └── crawler/main.go            # Crawler + scheduler
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── crawler/
│   │   │   └── shopee.go              # Shopee Affiliate API crawler
│   │   ├── scheduler/scheduler.go     # Cron jobs: crawl, track giá, voucher
│   │   ├── api/
│   │   │   ├── router.go              # HTTP router + GraphQL
│   │   │   └── middleware/
│   │   ├── graph/
│   │   │   ├── schema.graphqls
│   │   │   └── resolvers.go
│   │   ├── grpcclient/client.go       # gRPC client (chat only)
│   │   ├── rabbitmq/
│   │   │   ├── publisher.go           # Publish tasks to RabbitMQ
│   │   │   └── consumer.go            # Consume results from Python
│   │   ├── telegram/bot.go            # Telegram bot notifications
│   │   ├── store/
│   │   │   ├── postgres.go
│   │   │   ├── products.go
│   │   │   ├── prices.go
│   │   │   ├── vouchers.go
│   │   │   ├── deals.go
│   │   │   └── chat.go
│   │   └── models/
│   ├── gen/deal/v1/
│   └── migrations/
│
├── python-service/
│   ├── Dockerfile
│   ├── pyproject.toml
│   ├── src/
│   │   ├── main.py                    # gRPC server + RabbitMQ consumer
│   │   ├── config.py
│   │   ├── grpc_server/
│   │   │   └── chat_servicer.py       # AI Chat (RAG)
│   │   ├── ai/
│   │   │   ├── deal_analyzer.py       # Phân tích deal thật/ảo
│   │   │   ├── price_analyzer.py      # Phân tích biến động giá
│   │   │   └── rag.py                 # RAG pipeline cho chat
│   │   ├── rabbitmq/
│   │   │   ├── consumer.py            # Consume tasks from Go
│   │   │   └── publisher.py           # Publish results/alerts
│   │   ├── embeddings/store.py
│   │   └── db/connection.py
│   ├── gen/deal/v1/
│   └── tests/
│
└── frontend/
    ├── Dockerfile
    ├── package.json
    └── apps/web/
        ├── app/
        │   ├── layout.tsx
        │   ├── page.tsx               # Dashboard - deals hôm nay
        │   ├── products/
        │   │   ├── page.tsx           # Tìm kiếm sản phẩm
        │   │   └── [id]/page.tsx      # Chi tiết SP + biểu đồ giá
        │   ├── vouchers/page.tsx      # Danh sách voucher + lịch sale
        │   ├── alerts/page.tsx        # Quản lý price alerts
        │   └── chat/page.tsx          # AI chat tư vấn mua sắm
        ├── components/
        │   ├── dashboard/             # DealCard, StatsOverview, TopDeals
        │   ├── product/               # PriceChart, PriceHistory, DealBadge
        │   ├── voucher/               # VoucherCard, SaleCalendar, CountdownTimer
        │   ├── chat/                  # ChatWindow, MessageBubble, ChatInput
        │   └── ui/                    # Shared UI (shadcn)
        ├── hooks/
        │   ├── useProducts.ts
        │   ├── useDeals.ts
        │   ├── useVouchers.ts
        │   └── useChat.ts
        └── lib/
            ├── apollo.ts
            └── graphql/
                ├── queries.ts
                ├── mutations.ts
                └── subscriptions.ts
```

## Database Schema

### products
Thông tin sản phẩm từ Shopee.
```sql
CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shopee_id       BIGINT NOT NULL UNIQUE,
    shop_id         BIGINT NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT,
    image_url       TEXT,
    category        VARCHAR(255),
    rating          REAL DEFAULT 0,
    sold_count      INTEGER DEFAULT 0,
    shop_name       VARCHAR(255),
    shop_rating     REAL,
    url             TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### price_history
Lịch sử giá theo thời gian — core của price tracking.
```sql
CREATE TABLE price_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    price           BIGINT NOT NULL,                -- giá hiện tại (VND)
    original_price  BIGINT,                         -- giá gốc (trước giảm)
    discount_pct    REAL,                           -- % giảm giá hiển thị
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    INDEX idx_price_history_product_time (product_id, recorded_at DESC)
);
```

### vouchers
Mã giảm giá trên Shopee.
```sql
CREATE TABLE vouchers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shopee_voucher_id VARCHAR(255) NOT NULL UNIQUE,
    code            VARCHAR(100),
    title           TEXT NOT NULL,
    description     TEXT,
    discount_type   VARCHAR(20) NOT NULL,           -- percentage, fixed_amount, shipping
    discount_value  BIGINT NOT NULL,                -- giá trị giảm
    min_spend       BIGINT DEFAULT 0,               -- đơn tối thiểu
    max_discount    BIGINT,                         -- giảm tối đa
    usage_limit     INTEGER,                        -- tổng số lượng
    used_count      INTEGER DEFAULT 0,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### deal_analysis
Kết quả phân tích AI — deal thật hay ảo.
```sql
CREATE TABLE deal_analysis (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    verdict         VARCHAR(20) NOT NULL,           -- real_deal, fake_deal, normal
    deal_score      REAL NOT NULL,                  -- 0-100, càng cao càng tốt
    current_price   BIGINT NOT NULL,
    lowest_30d      BIGINT,                         -- giá thấp nhất 30 ngày
    highest_30d     BIGINT,                         -- giá cao nhất 30 ngày
    avg_30d         BIGINT,                         -- giá trung bình 30 ngày
    price_before_sale BIGINT,                       -- giá ngay trước đợt sale
    analysis_note   TEXT,                           -- AI giải thích
    analyzed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    INDEX idx_deal_analysis_verdict (verdict, deal_score DESC)
);
```

### price_alerts
User đặt alert theo dõi giá.
```sql
CREATE TABLE price_alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    target_price    BIGINT NOT NULL,                -- báo khi giá dưới mức này
    is_active       BOOLEAN DEFAULT true,
    triggered_at    TIMESTAMPTZ,                    -- lần cuối trigger
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### users
User đăng ký nhận alert.
```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_chat_id BIGINT UNIQUE,                 -- Telegram user ID
    username        VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### product_embeddings (pgvector)
Vector embeddings cho AI chat search.
```sql
CREATE TABLE product_embeddings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    embedding       vector(1536),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(product_id)
);
```

### chat_sessions + chat_messages
Lịch sử chat AI.
```sql
CREATE TABLE chat_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,           -- user, assistant
    content         TEXT NOT NULL,
    sources         JSONB,                          -- sản phẩm được trích dẫn
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## gRPC Services (chỉ dùng cho Chat real-time)

```protobuf
service ChatService {
  rpc Ask(AskRequest) returns (AskResponse);
  rpc AskStream(AskStreamRequest) returns (stream AskStreamResponse);
}
```

## RabbitMQ Queues

| Queue | Publisher | Consumer | Mục đích |
|-------|-----------|----------|----------|
| `deal.analyze` | Go (sau khi crawl giá) | Python | Phân tích deal thật/ảo |
| `deal.result` | Python (sau khi phân tích) | Go | Nhận kết quả phân tích |
| `alert.trigger` | Python (phát hiện deal tốt) | Go | Gửi Telegram alert cho user |
| `product.embed` | Go (sản phẩm mới) | Python | Tạo embedding cho AI chat |

## GraphQL API

**Endpoints:**
- `POST /graphql` — Queries & Mutations
- `WS /graphql` — Subscriptions (chat streaming)

```graphql
type Query {
  # Products
  products(filter: ProductFilter, page: Int, pageSize: Int): ProductConnection!
  product(id: ID!): Product
  searchProducts(query: String!, limit: Int): [Product!]!

  # Price
  priceHistory(productId: ID!, days: Int): [PricePoint!]!
  
  # Deals
  topDeals(limit: Int, category: String): [DealAnalysis!]!
  fakeDeals(limit: Int): [DealAnalysis!]!

  # Vouchers
  activeVouchers(page: Int, pageSize: Int): VoucherConnection!
  upcomingVouchers: [Voucher!]!

  # Alerts
  myAlerts: [PriceAlert!]!

  # Chat
  chatHistory(sessionId: ID!): [ChatMessage!]!
}

type Mutation {
  # Alert
  createAlert(productId: ID!, targetPrice: Int!): PriceAlert!
  deleteAlert(id: ID!): Boolean!

  # Chat
  createChatSession: ChatSession!
  sendMessage(sessionId: ID!, content: String!): ChatMessage!

  # Track
  trackProduct(shopeeUrl: String!): Product!
}

type Subscription {
  messageStream(sessionId: ID!): ChatChunk!
}

input ProductFilter {
  category: String
  minPrice: Int
  maxPrice: Int
  minRating: Float
  minDealScore: Float
  verdict: String                    # real_deal, fake_deal
}
```

## Các Phase triển khai

### Phase 1: Foundation (Tuần 1)
> Mục tiêu: Skeleton chạy được, services kết nối nhau.

1. Restructure project: đổi proto, models, migrations cho domain mới
2. Setup RabbitMQ trong docker-compose
3. Go: RabbitMQ publisher/consumer setup (amqp091-go)
4. Python: RabbitMQ consumer setup (aio-pika)
5. Giữ nguyên gRPC cho chat service
6. Database migration mới: products, price_history, vouchers, deal_analysis, users, price_alerts
7. **Verify**: Go → Postgres ✓, Go → RabbitMQ → Python ✓, Go → Python gRPC ✓

### Phase 2: Shopee Crawler (Tuần 2)
> Mục tiêu: Crawl được sản phẩm + giá từ Shopee.

1. Đăng ký Shopee Affiliate Program → lấy API key
2. Go crawler: gọi Shopee Affiliate API → lấy products theo category
3. Price tracker: crawl giá định kỳ → lưu vào price_history
4. Scheduler: crawl products mỗi 6h, crawl giá mỗi 12h
5. Upsert logic: sản phẩm đã có thì update, chưa có thì insert
6. **Verify**: Products + giá chảy vào DB liên tục ✓

### Phase 3: AI Deal Analysis (Tuần 3)
> Mục tiêu: AI phân tích deal thật/ảo.

1. Go: sau khi crawl giá mới → publish message vào queue `deal.analyze`
2. Python consumer: nhận message → phân tích:
   - Lấy price_history 30 ngày
   - So sánh: giá hiện tại vs giá thấp nhất / trung bình / trước sale
   - AI verdict: DEAL THẬT / DEAL ẢO / BÌNH THƯỜNG
   - Tính deal_score (0-100)
3. Python: publish kết quả vào queue `deal.result`
4. Go: consume kết quả → lưu vào deal_analysis table
5. **Verify**: Crawl giá → tự động phân tích → kết quả trong DB ✓

### Phase 4: Voucher + Telegram Alert (Tuần 4)
> Mục tiêu: Tổng hợp voucher, gửi alert cho user.

1. Voucher crawler: crawl voucher Shopee → lưu DB
2. Users table + Telegram bot setup (go-telegram-bot-api)
3. Telegram commands:
   - `/track <shopee_url>` — bắt đầu track sản phẩm
   - `/alert <shopee_url> <price>` — đặt alert giá
   - `/deals` — xem top deals hôm nay
   - `/vouchers` — xem voucher đang có
4. Alert system: khi giá giảm dưới target → gửi Telegram message
5. Daily digest: mỗi sáng gửi "Top 5 deal thật hôm nay"
6. Nhắc voucher: trước event sale (4.4, 5.5...) gửi reminder
7. **Verify**: User track SP qua Telegram → nhận alert khi giá giảm ✓

### Phase 5: AI Chat (Tuần 5)
> Mục tiêu: User chat hỏi tư vấn mua sắm.

1. Embedding pipeline: sản phẩm mới → Go publish → RabbitMQ → Python tạo embedding
2. RAG chat:
   - User: "Tai nghe bluetooth dưới 500k nào tốt?"
   - Embed câu hỏi → search pgvector → lấy top sản phẩm phù hợp
   - Claude trả lời dựa trên data thật: giá, rating, deal score, lịch sử giá
3. gRPC streaming: Python → Go → GraphQL subscription → UI
4. **Verify**: Chat hoạt động, trả lời dựa trên data thật ✓

### Phase 6: Frontend (Tuần 6-7)
> Mục tiêu: Dashboard + Chat UI hoàn chỉnh.

1. **Dashboard** (`/`):
   - Top deals hôm nay (deal_score cao nhất, verdict = real_deal)
   - Thống kê: tổng SP tracking, deal thật/ảo hôm nay
   - Upcoming sale events + voucher countdown

2. **Tìm kiếm sản phẩm** (`/products`):
   - Search bar: tìm theo tên
   - Filter: category, khoảng giá, rating, deal score
   - Product cards: ảnh, giá, badge DEAL THẬT/ẢO

3. **Chi tiết sản phẩm** (`/products/[id]`):
   - Biểu đồ giá 30/60/90 ngày (recharts)
   - Deal analysis: verdict + giải thích AI
   - Nút "Track giá" + "Đặt alert"
   - Voucher áp dụng được
   - Link gốc Shopee (affiliate link)

4. **Voucher** (`/vouchers`):
   - Danh sách voucher active + sắp mở
   - Countdown timer cho voucher sắp mở
   - Lịch sale events (4.4, 5.5, 6.6...)

5. **Alert** (`/alerts`):
   - Quản lý price alerts
   - Lịch sử alert đã trigger

6. **Chat** (`/chat`):
   - Chat AI tư vấn mua sắm
   - Trích dẫn sản phẩm cụ thể + link

### Phase 7: Polish & Monetize
> Mục tiêu: Production-ready + kiếm tiền.

1. Affiliate link integration: mọi link Shopee → affiliate link
2. SEO: landing pages cho top deals
3. Structured logging + health checks
4. Rate limiting + caching (Redis nếu cần)
5. Analytics: track click, conversion rate

## Environment Variables (.env.example)

```env
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/shopee_smart_deal

# Shopee
SHOPEE_AFFILIATE_APP_ID=xxx
SHOPEE_AFFILIATE_SECRET=xxx

# AI
ANTHROPIC_API_KEY=sk-ant-xxx

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# Telegram
TELEGRAM_BOT_TOKEN=xxx

# Services
PYTHON_GRPC_ADDR=python-service:50052
GO_API_PORT=8080

# Frontend
NEXT_PUBLIC_GRAPHQL_URL=http://localhost:8080/graphql
NEXT_PUBLIC_GRAPHQL_WS_URL=ws://localhost:8080/graphql
```

## Key Design Decisions

- **RabbitMQ cho async tasks**: Analysis, embedding, alerts — Go không cần chờ Python xử lý xong
- **gRPC chỉ cho Chat**: Real-time streaming cần request-response trực tiếp
- **Telegram Bot thay vì mobile app**: Ship nhanh, user không cần install app, notification instant
- **Shopee Affiliate API**: Hợp pháp, có sẵn data, kiếm tiền từ affiliate commission
- **pgvector**: Một database duy nhất cho cả relational data + vector search
- **price_history time series**: Core feature — không có lịch sử giá thì không phát hiện được fake sale

## Run Project

```bash
docker compose up -d
```
