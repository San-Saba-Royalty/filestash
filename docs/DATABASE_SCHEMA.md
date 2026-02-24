# Database Schema - Supabase PostgreSQL with pgvector

## Overview

This document defines the database schema for storing vector embeddings and metadata in Supabase PostgreSQL using the pgvector extension.

## Prerequisites

### Enable pgvector Extension

```sql
-- Run this first in your Supabase SQL editor
CREATE EXTENSION IF NOT EXISTS vector;
```

## Core Tables

### 1. `ai_documents`

Stores metadata about indexed documents.

```sql
CREATE TABLE ai_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- File identification
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_type TEXT, -- MIME type
    file_size BIGINT,
    file_hash TEXT, -- SHA256 hash for change detection
    
    -- Storage backend info
    storage_backend TEXT NOT NULL, -- 's3', 'azure', 'ipfs', 'local', etc.
    storage_path TEXT NOT NULL, -- Backend-specific path
    
    -- User/session context
    user_id TEXT, -- Filestash user ID
    session_id TEXT, -- Filestash session ID
    workspace TEXT, -- Optional workspace/tenant isolation
    
    -- Document metadata
    title TEXT,
    author TEXT,
    created_date TIMESTAMPTZ,
    modified_date TIMESTAMPTZ,
    
    -- Indexing metadata
    indexed_at TIMESTAMPTZ DEFAULT NOW(),
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    index_version INTEGER DEFAULT 1,
    
    -- Processing status
    status TEXT DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    error_message TEXT,
    
    -- Document stats
    chunk_count INTEGER DEFAULT 0,
    total_tokens INTEGER,
    
    -- Soft delete
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT unique_file_per_user UNIQUE(file_path, user_id, storage_backend)
);

-- Indexes
CREATE INDEX idx_documents_user_id ON ai_documents(user_id);
CREATE INDEX idx_documents_file_path ON ai_documents(file_path);
CREATE INDEX idx_documents_status ON ai_documents(status);
CREATE INDEX idx_documents_storage_backend ON ai_documents(storage_backend);
CREATE INDEX idx_documents_indexed_at ON ai_documents(indexed_at DESC);
CREATE INDEX idx_documents_deleted_at ON ai_documents(deleted_at) WHERE deleted_at IS NOT NULL;
```

### 2. `ai_embeddings`

Stores vector embeddings for document chunks.

```sql
CREATE TABLE ai_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Link to parent document
    document_id UUID NOT NULL REFERENCES ai_documents(id) ON DELETE CASCADE,
    
    -- Chunk information
    chunk_index INTEGER NOT NULL, -- Position in document (0-based)
    chunk_text TEXT NOT NULL, -- Actual text content
    chunk_type TEXT, -- 'text', 'table', 'image_caption', etc.
    
    -- Page/section info (from Unstructured.io)
    page_number INTEGER,
    section_title TEXT,
    
    -- Vector embedding
    embedding vector(1536) NOT NULL, -- Azure OpenAI text-embedding-3-large
    
    -- Metadata from Unstructured.io
    element_type TEXT, -- 'Title', 'NarrativeText', 'ListItem', 'Table', etc.
    metadata JSONB, -- Additional metadata from Unstructured
    
    -- Token tracking
    token_count INTEGER,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_chunk UNIQUE(document_id, chunk_index)
);

-- Indexes
CREATE INDEX idx_embeddings_document_id ON ai_embeddings(document_id);
CREATE INDEX idx_embeddings_chunk_index ON ai_embeddings(document_id, chunk_index);

-- HNSW index for fast similarity search (CRITICAL for performance)
-- Using cosine distance (optimal for OpenAI embeddings)
CREATE INDEX idx_embeddings_vector ON ai_embeddings 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Alternative: IVFFlat index (less accurate but faster to build for large datasets)
-- CREATE INDEX idx_embeddings_vector ON ai_embeddings 
-- USING ivfflat (embedding vector_cosine_ops)
-- WITH (lists = 100);
```

### 3. `ai_search_queries`

Logs search queries for analytics and improvement.

```sql
CREATE TABLE ai_search_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- User context
    user_id TEXT NOT NULL,
    session_id TEXT,
    
    -- Query details
    query_text TEXT NOT NULL,
    query_type TEXT NOT NULL, -- 'semantic', 'rag', 'hybrid'
    
    -- Results
    result_count INTEGER,
    top_document_ids UUID[], -- Array of top result IDs
    
    -- Performance metrics
    search_time_ms INTEGER,
    llm_time_ms INTEGER, -- For RAG queries
    
    -- Feedback (optional)
    user_rating INTEGER, -- 1-5 stars
    user_feedback TEXT,
    
    -- Timestamp
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_queries_user_id ON ai_search_queries(user_id);
CREATE INDEX idx_queries_created_at ON ai_search_queries(created_at DESC);
CREATE INDEX idx_queries_query_type ON ai_search_queries(query_type);
```

### 4. `ai_processing_queue`

Queue for background document processing.

```sql
CREATE TABLE ai_processing_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Document reference
    document_id UUID REFERENCES ai_documents(id) ON DELETE CASCADE,
    
    -- Queue metadata
    priority INTEGER DEFAULT 5, -- 1 (highest) to 10 (lowest)
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- Status
    status TEXT DEFAULT 'queued', -- 'queued', 'processing', 'completed', 'failed'
    error_message TEXT,
    
    -- Worker info
    worker_id TEXT,
    locked_at TIMESTAMPTZ,
    lock_expires_at TIMESTAMPTZ,
    
    -- Timestamps
    queued_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    CONSTRAINT unique_document_in_queue UNIQUE(document_id)
);

-- Indexes
CREATE INDEX idx_queue_status ON ai_processing_queue(status);
CREATE INDEX idx_queue_priority ON ai_processing_queue(priority DESC, queued_at ASC);
CREATE INDEX idx_queue_lock ON ai_processing_queue(worker_id, lock_expires_at);
```

### 5. `ai_config`

Stores plugin configuration (optional - can use Filestash config instead).

```sql
CREATE TABLE ai_config (
    id SERIAL PRIMARY KEY,
    
    -- Config key-value
    config_key TEXT UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    
    -- Metadata
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT
);

-- Example initial config
INSERT INTO ai_config (config_key, config_value, description) VALUES
('indexing.auto_index_on_upload', 'true', 'Automatically index files on upload'),
('indexing.batch_size', '10', 'Number of files to process in parallel'),
('indexing.max_file_size_mb', '100', 'Maximum file size to index (MB)'),
('search.max_results', '20', 'Maximum search results to return'),
('rag.max_context_chunks', '5', 'Maximum chunks to include in RAG context'),
('rag.temperature', '0.7', 'LLM temperature for RAG responses');
```

## Row Level Security (RLS)

Enable RLS to ensure users only access their own data:

```sql
-- Enable RLS on all tables
ALTER TABLE ai_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_embeddings ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_search_queries ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_processing_queue ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their own documents
CREATE POLICY "Users can view own documents" 
ON ai_documents FOR SELECT 
USING (user_id = current_setting('app.user_id', true));

-- Policy: Users can only see embeddings of their own documents
CREATE POLICY "Users can view own embeddings" 
ON ai_embeddings FOR SELECT 
USING (
    document_id IN (
        SELECT id FROM ai_documents 
        WHERE user_id = current_setting('app.user_id', true)
    )
);

-- Policy: Users can view their own queries
CREATE POLICY "Users can view own queries" 
ON ai_search_queries FOR SELECT 
USING (user_id = current_setting('app.user_id', true));

-- Note: Set user_id in session before queries:
-- SELECT set_config('app.user_id', 'user-123', false);
```

## SQL Functions

### 1. Semantic Search Function

```sql
CREATE OR REPLACE FUNCTION semantic_search(
    query_embedding vector(1536),
    user_filter TEXT,
    max_results INTEGER DEFAULT 20,
    similarity_threshold FLOAT DEFAULT 0.7
)
RETURNS TABLE (
    document_id UUID,
    chunk_id UUID,
    chunk_text TEXT,
    file_path TEXT,
    file_name TEXT,
    similarity FLOAT,
    page_number INTEGER,
    section_title TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.document_id,
        e.id AS chunk_id,
        e.chunk_text,
        d.file_path,
        d.file_name,
        1 - (e.embedding <=> query_embedding) AS similarity, -- Cosine similarity
        e.page_number,
        e.section_title
    FROM ai_embeddings e
    INNER JOIN ai_documents d ON e.document_id = d.id
    WHERE 
        d.user_id = user_filter
        AND d.deleted_at IS NULL
        AND d.status = 'completed'
        AND (1 - (e.embedding <=> query_embedding)) >= similarity_threshold
    ORDER BY e.embedding <=> query_embedding -- Cosine distance (lower is better)
    LIMIT max_results;
END;
$$ LANGUAGE plpgsql;
```

### 2. Hybrid Search (Combining semantic + keyword)

```sql
CREATE OR REPLACE FUNCTION hybrid_search(
    query_embedding vector(1536),
    query_keywords TEXT,
    user_filter TEXT,
    max_results INTEGER DEFAULT 20,
    semantic_weight FLOAT DEFAULT 0.7,
    keyword_weight FLOAT DEFAULT 0.3
)
RETURNS TABLE (
    document_id UUID,
    chunk_id UUID,
    chunk_text TEXT,
    file_path TEXT,
    file_name TEXT,
    combined_score FLOAT,
    semantic_score FLOAT,
    keyword_score FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.document_id,
        e.id AS chunk_id,
        e.chunk_text,
        d.file_path,
        d.file_name,
        -- Weighted combination of semantic and keyword scores
        (semantic_weight * (1 - (e.embedding <=> query_embedding))) + 
        (keyword_weight * ts_rank(to_tsvector('english', e.chunk_text), 
                                   plainto_tsquery('english', query_keywords))) AS combined_score,
        (1 - (e.embedding <=> query_embedding)) AS semantic_score,
        ts_rank(to_tsvector('english', e.chunk_text), 
                plainto_tsquery('english', query_keywords)) AS keyword_score
    FROM ai_embeddings e
    INNER JOIN ai_documents d ON e.document_id = d.id
    WHERE 
        d.user_id = user_filter
        AND d.deleted_at IS NULL
        AND d.status = 'completed'
    ORDER BY combined_score DESC
    LIMIT max_results;
END;
$$ LANGUAGE plpgsql;
```

### 3. Get Document Chunks for RAG Context

```sql
CREATE OR REPLACE FUNCTION get_rag_context(
    query_embedding vector(1536),
    user_filter TEXT,
    max_chunks INTEGER DEFAULT 5
)
RETURNS TABLE (
    chunk_text TEXT,
    file_name TEXT,
    file_path TEXT,
    page_number INTEGER,
    similarity FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.chunk_text,
        d.file_name,
        d.file_path,
        e.page_number,
        1 - (e.embedding <=> query_embedding) AS similarity
    FROM ai_embeddings e
    INNER JOIN ai_documents d ON e.document_id = d.id
    WHERE 
        d.user_id = user_filter
        AND d.deleted_at IS NULL
        AND d.status = 'completed'
    ORDER BY e.embedding <=> query_embedding
    LIMIT max_chunks;
END;
$$ LANGUAGE plpgsql;
```

## Maintenance Tasks

### 1. Vacuum and Analyze

Run periodically to maintain performance:

```sql
-- Vacuum and analyze tables
VACUUM ANALYZE ai_documents;
VACUUM ANALYZE ai_embeddings;
VACUUM ANALYZE ai_search_queries;

-- Reindex vector index if needed
REINDEX INDEX CONCURRENTLY idx_embeddings_vector;
```

### 2. Cleanup Old Queries

```sql
-- Delete queries older than 90 days
DELETE FROM ai_search_queries 
WHERE created_at < NOW() - INTERVAL '90 days';
```

### 3. Purge Deleted Documents

```sql
-- Permanently delete documents marked for deletion > 30 days ago
DELETE FROM ai_documents 
WHERE deleted_at IS NOT NULL 
AND deleted_at < NOW() - INTERVAL '30 days';
```

## Performance Tuning

### pgvector Index Parameters

```sql
-- For HNSW index (recommended)
-- m: Max number of connections per layer (16 is default, higher = better recall, slower build)
-- ef_construction: Size of dynamic candidate list (64 is default, higher = better quality, slower build)

-- For large datasets (>1M vectors):
CREATE INDEX idx_embeddings_vector ON ai_embeddings 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 32, ef_construction = 128);

-- Query-time performance tuning
SET hnsw.ef_search = 100; -- Higher = better recall, slower search (default: 40)
```

### Connection Pooling

Recommended for production:
- **Supavisor** (Supabase's pooler): Use `pooler` subdomain
- **PgBouncer**: Configure transaction pooling
- Minimum pool size: 10 connections
- Maximum pool size: 100 connections

## Storage Estimates

### Vector Storage Size

For 1536-dimensional vectors (4 bytes per dimension):
- **Per vector**: ~6 KB (with metadata)
- **10,000 documents**: ~300 MB (assuming 5 chunks per document)
- **100,000 documents**: ~3 GB
- **1,000,000 documents**: ~30 GB

### Supabase Free Tier

- Database size: 500 MB
- Can store ~80,000 vectors (with metadata)
- Upgrade to Pro ($25/mo) for 8 GB storage

## Backup Strategy

### Automated Backups

Supabase provides:
- Daily backups (7-day retention on free tier)
- Point-in-time recovery (Pro tier)

### Manual Backup

```bash
# Export full database
pg_dump -h db.your-project.supabase.co \
        -U postgres \
        -d postgres \
        -F c \
        -f backup_$(date +%Y%m%d).dump

# Restore
pg_restore -h db.your-project.supabase.co \
           -U postgres \
           -d postgres \
           backup_20250129.dump
```

## Migration Scripts

### Initial Setup Script

```sql
-- Run this complete script in Supabase SQL Editor

-- 1. Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. Create all tables
-- (Copy all CREATE TABLE statements from above)

-- 3. Create indexes
-- (Copy all CREATE INDEX statements from above)

-- 4. Create functions
-- (Copy all CREATE OR REPLACE FUNCTION statements from above)

-- 5. Enable RLS
-- (Copy all RLS policy statements from above)

-- 6. Insert default config
-- (Copy INSERT INTO ai_config statements from above)
```

## Next Steps

1. Set up Supabase project at https://supabase.com
2. Run the migration script in SQL Editor
3. Copy your Supabase URL and service role key
4. Configure Filestash with Supabase credentials
5. Proceed to [PLUGIN_ARCHITECTURE.md](./PLUGIN_ARCHITECTURE.md)
