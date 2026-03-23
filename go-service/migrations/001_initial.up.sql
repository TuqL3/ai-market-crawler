CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Raw crawled data
CREATE TABLE raw_problems (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
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

-- AI classification results
CREATE TABLE classified_problems (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    raw_problem_id  UUID NOT NULL REFERENCES raw_problems(id) ON DELETE CASCADE,
    category        VARCHAR(100) NOT NULL,
    subcategories   TEXT[],
    confidence      REAL NOT NULL,
    classified_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(raw_problem_id)
);

-- Problem clusters
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

-- Trend snapshots
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

-- Vector embeddings for RAG
CREATE TABLE problem_embeddings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    raw_problem_id  UUID NOT NULL REFERENCES raw_problems(id) ON DELETE CASCADE,
    embedding       vector(1536),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(raw_problem_id)
);

-- Chat
CREATE TABLE chat_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id      UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,
    content         TEXT NOT NULL,
    sources         JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Crawl job tracking
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

-- Indexes
CREATE INDEX idx_raw_problems_source ON raw_problems(source);
CREATE INDEX idx_raw_problems_crawled_at ON raw_problems(crawled_at);
CREATE INDEX idx_classified_problems_category ON classified_problems(category);
CREATE INDEX idx_trend_snapshots_snapshot_at ON trend_snapshots(snapshot_at);
CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id);
