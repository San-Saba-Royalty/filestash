# Implementation Guide - Step by Step

## Prerequisites

Before starting, ensure you have:

1. **Go 1.21+** installed
2. **Docker** and **Docker Compose** installed
3. **Supabase account** created at https://supabase.com
4. **Azure OpenAI** resource provisioned
5. **PostgreSQL client tools** (psql, pgcli, etc.)

## Phase 1: Environment Setup (Day 1-2)

### 1.1 Supabase Setup

#### Create Supabase Project

1. Go to https://supabase.com/dashboard
2. Click "New Project"
3. Choose organization and set project name: `filestash-ai`
4. Choose region closest to your Filestash deployment
5. Set database password (save securely)
6. Wait for project provisioning (~2 minutes)

#### Get Connection Details

From your Supabase dashboard:
- **Project URL**: `https://[project-id].supabase.co`
- **API URL**: `https://[project-id].supabase.co/rest/v1`
- **Anon Key**: Found in Settings → API
- **Service Role Key**: Found in Settings → API (keep secret!)
- **Database URL**: Found in Settings → Database

#### Initialize Database Schema

1. Open SQL Editor in Supabase dashboard
2. Copy the complete schema from `DATABASE_SCHEMA.md`
3. Run the migration script
4. Verify tables created:

```sql
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name LIKE 'ai_%';
```

Expected output:
```
ai_documents
ai_embeddings
ai_search_queries
ai_processing_queue
ai_config
```

5. Verify pgvector extension:

```sql
SELECT * FROM pg_extension WHERE extname = 'vector';
```

### 1.2 Azure OpenAI Setup

#### Create Azure OpenAI Resource

1. Go to Azure Portal: https://portal.azure.com
2. Search for "Azure OpenAI"
3. Click "Create"
4. Fill in details:
   - **Subscription**: Your subscription
   - **Resource Group**: Create new or use existing
   - **Region**: Choose available region (e.g., East US, West Europe)
   - **Name**: `filestash-openai-[random]`
   - **Pricing Tier**: Standard S0
5. Click "Review + Create"

#### Deploy Models

Once resource is created:

1. Go to Azure OpenAI Studio: https://oai.azure.com
2. Navigate to "Deployments"
3. Create embedding deployment:
   - **Model**: text-embedding-3-large
   - **Deployment name**: `text-embedding-3-large`
   - **Model version**: Use latest
   - **Tokens per minute**: 120K (adjust based on needs)
4. Create chat deployment:
   - **Model**: gpt-4 or gpt-4o
   - **Deployment name**: `gpt-4o`
   - **Model version**: Use latest
   - **Tokens per minute**: 30K (adjust based on needs)

#### Get API Keys

1. In Azure Portal, go to your OpenAI resource
2. Navigate to "Keys and Endpoint"
3. Copy:
   - **Endpoint**: `https://[your-resource].openai.azure.com/`
   - **Key 1**: Your API key
   - **API Version**: `2024-02-01` (or latest)

### 1.3 Docker Setup for Unstructured.io

Create `docker-compose.ai.yml` in your Filestash root:

```yaml
version: '3.8'

services:
  unstructured-api:
    image: downloads.unstructured.io/unstructured-io/unstructured-api:latest
    container_name: unstructured-api
    ports:
      - "8000:8000"
    environment:
      - UNSTRUCTURED_ALLOWED_MIMETYPES=application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,text/plain,text/html,image/jpeg,image/png
    volumes:
      - ./unstructured-cache:/cache
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Start Unstructured.io:

```bash
docker-compose -f docker-compose.ai.yml up -d
```

Verify it's running:

```bash
curl http://localhost:8000/general/v0/general/docs
```

## Phase 2: Plugin Development (Day 3-10)

### 2.1 Create Plugin Directory Structure

```bash
cd /Users/gqadonis/Projects/sansaba/filestash
mkdir -p server/plugin/plg_ai_semantic
cd server/plugin/plg_ai_semantic
```

### 2.2 Initialize Go Module Dependencies

Add to `go.mod` in Filestash root:

```go
require (
    github.com/jackc/pgx/v5 v5.5.1
    github.com/pgvector/pgvector-go v0.1.1
)
```

Run:

```bash
go mod tidy
```

### 2.3 Create Core Plugin Files

#### File: `server/plugin/plg_ai_semantic/index.go`

```go
package plg_ai_semantic

import (
    "github.com/gorilla/mux"
    . "github.com/mickael-kerjean/filestash/server/common"
)

func init() {
    // Initialize plugin
    Hooks.Register.Onload(func() {
        Log.Info("plg_ai_semantic::loading")
        
        // Load configuration
        if err := loadAIConfig(); err != nil {
            Log.Error("plg_ai_semantic::config failed: %v", err)
            return
        }
        
        if !IsAIEnabled() {
            Log.Info("plg_ai_semantic::disabled via config")
            return
        }
        
        // Initialize clients
        if err := initSupabaseClient(); err != nil {
            Log.Error("plg_ai_semantic::supabase init failed: %v", err)
            return
        }
        
        if err := initAzureOpenAIClient(); err != nil {
            Log.Error("plg_ai_semantic::azure openai init failed: %v", err)
            return
        }
        
        if err := initUnstructuredClient(); err != nil {
            Log.Error("plg_ai_semantic::unstructured init failed: %v", err)
            return
        }
        
        // Start background indexer
        go startIndexerWorker()
        
        Log.Info("plg_ai_semantic::loaded successfully")
    })
    
    // Register HTTP endpoints
    Hooks.Register.HttpEndpoint(registerAIEndpoints)
    
    // Register middleware for file operations
    Hooks.Register.Middleware(fileOperationMiddleware)
    
    // Register static assets
    registerStaticAssets()
}

func registerAIEndpoints(r *mux.Router) error {
    base := WithBase("/api/ai")
    
    // Semantic search
    r.HandleFunc(base+"/search", aiSearchHandler).Methods("POST")
    
    // RAG chat
    r.HandleFunc(base+"/chat", ragChatHandler).Methods("POST")
    
    // Index management
    r.HandleFunc(base+"/index/status", indexStatusHandler).Methods("GET")
    r.HandleFunc(base+"/index/trigger", triggerIndexHandler).Methods("POST")
    r.HandleFunc(base+"/index/document/{id}", getDocumentStatusHandler).Methods("GET")
    
    // Stats and analytics
    r.HandleFunc(base+"/stats", getStatsHandler).Methods("GET")
    
    return nil
}
```

#### File: `server/plugin/plg_ai_semantic/config.go`

Copy the full implementation from `PLUGIN_ARCHITECTURE.md` section 2.

#### File: `server/plugin/plg_ai_semantic/types.go`

Copy the full implementation from `PLUGIN_ARCHITECTURE.md` section 3.

#### File: `server/plugin/plg_ai_semantic/supabase_client.go`

Full implementation:

```go
package plg_ai_semantic

import (
    "context"
    "fmt"
    "time"
    
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/pgvector/pgvector-go"
)

var supabasePool *pgxpool.Pool

func initSupabaseClient() error {
    if !IsAIEnabled() {
        return nil
    }
    
    // Parse Supabase URL to get connection string
    // Format: postgresql://postgres:[PASSWORD]@db.[PROJECT-ID].supabase.co:5432/postgres
    connString := fmt.Sprintf(
        "postgresql://postgres:%s@db.%s.supabase.co:5432/postgres",
        aiConfig.Supabase.Password,
        aiConfig.Supabase.ProjectID,
    )
    
    config, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return fmt.Errorf("unable to parse connection string: %w", err)
    }
    
    // Pool configuration
    config.MaxConns = 20
    config.MinConns = 5
    config.MaxConnLifetime = time.Hour
    config.MaxConnIdleTime = 30 * time.Minute
    config.HealthCheckPeriod = time.Minute
    
    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return fmt.Errorf("unable to create connection pool: %w", err)
    }
    
    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := pool.Ping(ctx); err != nil {
        return fmt.Errorf("unable to ping database: %w", err)
    }
    
    supabasePool = pool
    Log.Info("plg_ai_semantic::supabase pool established")
    return nil
}

func setUserContext(ctx context.Context, userID string) (context.Context, error) {
    // Set RLS context for user isolation
    _, err := supabasePool.Exec(ctx, "SELECT set_config('app.user_id', $1, false)", userID)
    return ctx, err
}

// StoreDocument creates or updates a document record
func StoreDocument(ctx context.Context, doc *Document) error {
    ctx, err := setUserContext(ctx, doc.UserID)
    if err != nil {
        return err
    }
    
    query := `
        INSERT INTO ai_documents (
            file_path, file_name, file_type, file_size, file_hash,
            storage_backend, storage_path, user_id, session_id, 
            status, title, created_date, modified_date
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        ON CONFLICT (file_path, user_id, storage_backend) 
        DO UPDATE SET
            file_name = EXCLUDED.file_name,
            file_hash = EXCLUDED.file_hash,
            file_size = EXCLUDED.file_size,
            modified_date = EXCLUDED.modified_date,
            last_updated = NOW(),
            status = EXCLUDED.status,
            index_version = ai_documents.index_version + 1
        RETURNING id, index_version
    `
    
    err = supabasePool.QueryRow(
        ctx, query,
        doc.FilePath, doc.FileName, doc.FileType, doc.FileSize, doc.FileHash,
        doc.StorageBackend, doc.StoragePath, doc.UserID, doc.SessionID,
        doc.Status, doc.Title, doc.CreatedDate, doc.ModifiedDate,
    ).Scan(&doc.ID, &doc.IndexVersion)
    
    return err
}

// StoreEmbedding saves a vector embedding
func StoreEmbedding(ctx context.Context, emb *Embedding) error {
    query := `
        INSERT INTO ai_embeddings (
            document_id, chunk_index, chunk_text, chunk_type, 
            page_number, section_title, embedding, token_count,
            element_type, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (document_id, chunk_index) 
        DO UPDATE SET
            chunk_text = EXCLUDED.chunk_text,
            embedding = EXCLUDED.embedding,
            token_count = EXCLUDED.token_count
        RETURNING id
    `
    
    // Convert []float32 to pgvector.Vector
    vec := pgvector.NewVector(emb.Vector)
    
    err := supabasePool.QueryRow(
        ctx, query,
        emb.DocumentID, emb.ChunkIndex, emb.ChunkText, emb.ChunkType,
        emb.PageNumber, emb.SectionTitle, vec, emb.TokenCount,
        emb.ElementType, emb.Metadata,
    ).Scan(&emb.ID)
    
    return err
}

// UpdateDocumentStatus updates processing status
func UpdateDocumentStatus(ctx context.Context, docID string, status string, chunkCount int, errorMsg string) error {
    query := `
        UPDATE ai_documents 
        SET status = $1, chunk_count = $2, error_message = $3, last_updated = NOW()
        WHERE id = $4
    `
    
    _, err := supabasePool.Exec(ctx, query, status, chunkCount, errorMsg, docID)
    return err
}

// SemanticSearch performs vector similarity search
func SemanticSearch(
    ctx context.Context,
    queryEmbedding []float32,
    userID string,
    maxResults int,
    threshold float64,
) ([]SearchResult, error) {
    ctx, err := setUserContext(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    vec := pgvector.NewVector(queryEmbedding)
    
    query := `
        SELECT * FROM semantic_search($1::vector, $2, $3, $4)
    `
    
    rows, err := supabasePool.Query(ctx, query, vec, userID, maxResults, threshold)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        err := rows.Scan(
            &r.DocumentID, &r.ChunkID, &r.ChunkText,
            &r.FilePath, &r.FileName, &r.Similarity,
            &r.PageNumber, &r.SectionTitle,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, r)
    }
    
    return results, rows.Err()
}

// GetRAGContext retrieves relevant chunks for RAG
func GetRAGContext(
    ctx context.Context,
    queryEmbedding []float32,
    userID string,
    maxChunks int,
) ([]SearchResult, error) {
    ctx, err := setUserContext(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    vec := pgvector.NewVector(queryEmbedding)
    
    query := `
        SELECT * FROM get_rag_context($1::vector, $2, $3)
    `
    
    rows, err := supabasePool.Query(ctx, query, vec, userID, maxChunks)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        err := rows.Scan(
            &r.ChunkText, &r.FileName, &r.FilePath,
            &r.PageNumber, &r.Similarity,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, r)
    }
    
    return results, rows.Err()
}

// DeleteDocument soft-deletes a document
func DeleteDocument(ctx context.Context, documentID string, userID string) error {
    ctx, err := setUserContext(ctx, userID)
    if err != nil {
        return err
    }
    
    query := `
        UPDATE ai_documents 
        SET deleted_at = NOW() 
        WHERE id = $1 AND user_id = $2
    `
    
    _, err = supabasePool.Exec(ctx, query, documentID, userID)
    return err
}

// GetDocumentByPath retrieves document by file path
func GetDocumentByPath(ctx context.Context, filePath string, userID string, backend string) (*Document, error) {
    ctx, err := setUserContext(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    query := `
        SELECT id, file_path, file_name, file_type, file_size, file_hash,
               storage_backend, storage_path, user_id, status, chunk_count,
               indexed_at, last_updated
        FROM ai_documents
        WHERE file_path = $1 AND user_id = $2 AND storage_backend = $3 
              AND deleted_at IS NULL
    `
    
    var doc Document
    err = supabasePool.QueryRow(ctx, query, filePath, userID, backend).Scan(
        &doc.ID, &doc.FilePath, &doc.FileName, &doc.FileType, &doc.FileSize,
        &doc.FileHash, &doc.StorageBackend, &doc.StoragePath, &doc.UserID,
        &doc.Status, &doc.ChunkCount, &doc.IndexedAt, &doc.LastUpdated,
    )
    
    if err == pgx.ErrNoRows {
        return nil, nil // Not found
    }
    
    return &doc, err
}

// QueueDocumentForIndexing adds document to processing queue
func QueueDocumentForIndexing(ctx context.Context, docID string, priority int) error {
    query := `
        INSERT INTO ai_processing_queue (document_id, priority, status)
        VALUES ($1, $2, 'queued')
        ON CONFLICT (document_id) DO UPDATE SET
            priority = EXCLUDED.priority,
            status = 'queued',
            retry_count = 0,
            queued_at = NOW()
    `
    
    _, err := supabasePool.Exec(ctx, query, docID, priority)
    return err
}

// GetNextQueuedDocument retrieves next document to process
func GetNextQueuedDocument(ctx context.Context, workerID string) (*IndexingJob, error) {
    // Lock and get highest priority queued item
    query := `
        UPDATE ai_processing_queue
        SET status = 'processing',
            worker_id = $1,
            started_at = NOW(),
            locked_at = NOW(),
            lock_expires_at = NOW() + INTERVAL '10 minutes'
        WHERE id = (
            SELECT id FROM ai_processing_queue
            WHERE status = 'queued'
            ORDER BY priority DESC, queued_at ASC
            LIMIT 1
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, document_id, priority, retry_count
    `
    
    var job IndexingJob
    err := supabasePool.QueryRow(ctx, query, workerID).Scan(
        &job.ID, &job.DocumentID, &job.Priority, &job.RetryCount,
    )
    
    if err == pgx.ErrNoRows {
        return nil, nil // No jobs available
    }
    
    return &job, err
}

// CompleteQueuedJob marks job as completed
func CompleteQueuedJob(ctx context.Context, jobID string) error {
    query := `
        UPDATE ai_processing_queue
        SET status = 'completed', completed_at = NOW()
        WHERE id = $1
    `
    
    _, err := supabasePool.Exec(ctx, query, jobID)
    return err
}

// FailQueuedJob marks job as failed and handles retries
func FailQueuedJob(ctx context.Context, jobID string, errorMsg string) error {
    query := `
        UPDATE ai_processing_queue
        SET status = CASE 
                WHEN retry_count >= max_retries THEN 'failed'
                ELSE 'queued'
            END,
            retry_count = retry_count + 1,
            error_message = $2,
            locked_at = NULL,
            worker_id = NULL
        WHERE id = $1
    `
    
    _, err := supabasePool.Exec(ctx, query, jobID, errorMsg)
    return err
}

// LogSearchQuery records a search query for analytics
func LogSearchQuery(ctx context.Context, query string, queryType string, userID string, results []SearchResult, searchTime int, llmTime int) error {
    docIDs := make([]string, len(results))
    for i, r := range results {
        docIDs[i] = r.DocumentID
    }
    
    sql := `
        INSERT INTO ai_search_queries (
            user_id, query_text, query_type, result_count,
            top_document_ids, search_time_ms, llm_time_ms
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
    
    _, err := supabasePool.Exec(
        ctx, sql,
        userID, query, queryType, len(results),
        docIDs, searchTime, llmTime,
    )
    
    return err
}
```

Continue with remaining files in next response...

### 2.4 Build Plugin

Add plugin to `server/plugin/index.go`:

```go
import (
    // ... existing imports ...
    _ "github.com/mickael-kerjean/filestash/server/plugin/plg_ai_semantic"
)
```

Build Filestash:

```bash
cd /Users/gqadonis/Projects/sansaba/filestash
make build_backend
```

## Phase 3: Configuration (Day 11-12)

### 3.1 Add Configuration to Filestash

Edit your Filestash config (via admin panel or config file):

```json
{
  "ai": {
    "enabled": true,
    "supabase": {
      "project_id": "your-project-id",
      "password": "your-database-password"
    },
    "azure_openai": {
      "endpoint": "https://your-resource.openai.azure.com/",
      "api_key": "your-api-key",
      "api_version": "2024-02-01",
      "embeddings_deployment": "text-embedding-3-large",
      "chat_deployment": "gpt-4o",
      "embedding_dimensions": 1536
    },
    "unstructured": {
      "url": "http://localhost:8000"
    },
    "indexing": {
      "auto_index_on_upload": true,
      "batch_size": 10,
      "max_file_size_mb": 100,
      "supported_mime_types": [
        "application/pdf",
        "application/msword",
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        "text/plain",
        "text/html"
      ]
    },
    "search": {
      "max_results": 20,
      "similarity_threshold": 0.7
    },
    "rag": {
      "max_context_chunks": 5,
      "temperature": 0.7,
      "max_tokens": 1000
    }
  }
}
```

### 3.2 Environment Variables

Create `.env` file:

```bash
# Supabase
SUPABASE_PROJECT_ID=your-project-id
SUPABASE_DB_PASSWORD=your-password

# Azure OpenAI
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZURE_OPENAI_API_KEY=your-key
AZURE_OPENAI_API_VERSION=2024-02-01

# Unstructured.io
UNSTRUCTURED_API_URL=http://localhost:8000
```

## Phase 4: Testing (Day 13-14)

### 4.1 Unit Tests

Create test files for each component.

### 4.2 Integration Tests

Test the complete pipeline:
1. Upload a PDF file
2. Check indexing queue
3. Wait for processing
4. Perform semantic search
5. Test RAG query

### 4.3 Load Testing

Use tools like `k6` or `hey` to test under load.

## Phase 5: Deployment (Day 15-16)

### 5.1 Production Checklist

- [ ] Supabase configured with proper indexes
- [ ] Azure OpenAI quotas set appropriately
- [ ] Unstructured.io Docker container running
- [ ] Backup strategy configured
- [ ] Monitoring and alerting set up
- [ ] Documentation updated

### 5.2 Deployment Steps

1. Build production binary
2. Deploy Filestash with new plugin
3. Run database migrations
4. Start Docker services
5. Monitor logs for errors
6. Verify AI endpoints responding

## Troubleshooting

### Common Issues

**Issue**: Supabase connection fails
- Check network connectivity
- Verify credentials
- Check IP whitelist in Supabase

**Issue**: Azure OpenAI rate limits
- Increase tokens per minute quota
- Implement request queuing
- Add retry logic with exponential backoff

**Issue**: Unstructured.io processing slow
- Increase Docker CPU/memory limits
- Use faster strategy (fast vs hi_res)
- Process smaller batches

**Issue**: Vector search returns no results
- Check embedding dimensions match
- Verify data was indexed
- Lower similarity threshold
- Check RLS policies

## Next Steps

1. Review [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for API specs
2. Review [FRONTEND_INTEGRATION.md](./FRONTEND_INTEGRATION.md) for UI
3. Review [TESTING_GUIDE.md](./TESTING_GUIDE.md) for testing strategies
