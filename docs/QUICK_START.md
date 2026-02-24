# Quick Start Guide - AI Integration in 30 Minutes

This guide will get your Filestash AI integration up and running in approximately 30 minutes.

## Prerequisites Check (2 minutes)

Verify you have:
```bash
# Go version
go version  # Should be 1.21+

# Docker version
docker --version
docker-compose --version

# Git (to clone/modify Filestash)
git --version

# curl (for testing)
curl --version
```

## Step 1: Supabase Setup (5 minutes)

### 1.1 Create Project
1. Go to https://supabase.com/dashboard
2. Click "New Project"
3. Fill in:
   - **Name**: `filestash-ai`
   - **Database Password**: Generate strong password (save it!)
   - **Region**: Choose closest to you
4. Wait ~2 minutes for provisioning

### 1.2 Initialize Database

1. Click "SQL Editor" in left sidebar
2. Click "New Query"
3. Copy and paste this complete schema:

```sql
-- Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;

-- Documents table
CREATE TABLE ai_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_type TEXT,
    file_size BIGINT,
    file_hash TEXT,
    storage_backend TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    user_id TEXT,
    session_id TEXT,
    status TEXT DEFAULT 'pending',
    chunk_count INTEGER DEFAULT 0,
    indexed_at TIMESTAMPTZ DEFAULT NOW(),
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_file_per_user UNIQUE(file_path, user_id, storage_backend)
);

-- Embeddings table
CREATE TABLE ai_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES ai_documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    chunk_text TEXT NOT NULL,
    page_number INTEGER,
    section_title TEXT,
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_chunk UNIQUE(document_id, chunk_index)
);

-- Indexes
CREATE INDEX idx_documents_user_id ON ai_documents(user_id);
CREATE INDEX idx_embeddings_document_id ON ai_embeddings(document_id);

-- Vector index (CRITICAL for performance)
CREATE INDEX idx_embeddings_vector ON ai_embeddings 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Search function
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
        1 - (e.embedding <=> query_embedding) AS similarity,
        e.page_number,
        e.section_title
    FROM ai_embeddings e
    INNER JOIN ai_documents d ON e.document_id = d.id
    WHERE 
        d.user_id = user_filter
        AND d.status = 'completed'
        AND (1 - (e.embedding <=> query_embedding)) >= similarity_threshold
    ORDER BY e.embedding <=> query_embedding
    LIMIT max_results;
END;
$$ LANGUAGE plpgsql;
```

4. Click "Run"
5. Verify tables created:

```sql
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name LIKE 'ai_%';
```

You should see: `ai_documents`, `ai_embeddings`

### 1.3 Save Connection Info

From Supabase Dashboard → Settings → Database:
- **Host**: `db.xxxxx.supabase.co`
- **Database**: `postgres`
- **Port**: `5432`
- **User**: `postgres`
- **Password**: (the one you set)

Save as:
```bash
# .env file
SUPABASE_HOST=db.xxxxx.supabase.co
SUPABASE_PASSWORD=your-password
```

## Step 2: Azure OpenAI Setup (5 minutes)

### 2.1 Create Resource

1. Go to https://portal.azure.com
2. Search "Azure OpenAI"
3. Click "Create"
4. Fill in:
   - **Resource Group**: Create new: `filestash-rg`
   - **Region**: East US or West Europe
   - **Name**: `filestash-openai-[random]`
   - **Pricing**: Standard S0
5. Click "Review + Create" → "Create"
6. Wait ~2 minutes

### 2.2 Deploy Models

1. Go to https://oai.azure.com
2. Click "Deployments" → "Create new deployment"
3. Deploy embedding model:
   - **Model**: text-embedding-3-large
   - **Deployment name**: `text-embedding-3-large`
   - **Tokens per minute**: 120K
4. Deploy chat model:
   - **Model**: gpt-4o
   - **Deployment name**: `gpt-4o`
   - **Tokens per minute**: 30K

### 2.3 Get Keys

1. Back in Azure Portal → Your OpenAI resource
2. Click "Keys and Endpoint"
3. Copy:
   - **Endpoint**: `https://xxxxx.openai.azure.com/`
   - **Key 1**: Your API key

Save as:
```bash
# .env file (append)
AZURE_OPENAI_ENDPOINT=https://xxxxx.openai.azure.com/
AZURE_OPENAI_KEY=your-key
```

## Step 3: Start Unstructured.io (5 minutes)

### 3.1 Create Docker Compose File

Navigate to your Filestash directory:
```bash
cd /Users/gqadonis/Projects/sansaba/filestash
```

Create `docker-compose.ai.yml`:
```yaml
version: '3.8'

services:
  unstructured-api:
    image: quay.io/unstructured-io/unstructured-api:latest
    container_name: unstructured-api
    ports:
      - "8000:8000"
    environment:
      - UNSTRUCTURED_PARALLEL_MODE=true
      - UNSTRUCTURED_PARALLEL_NUM_THREADS=4
    volumes:
      - ./unstructured-cache:/cache
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### 3.2 Start Container

```bash
# Create cache directory
mkdir -p unstructured-cache

# Start service
docker-compose -f docker-compose.ai.yml up -d

# Wait for it to start (30-60 seconds)
sleep 60

# Verify health
curl http://localhost:8000/healthcheck
```

Expected response:
```json
{"healthcheck":"OK"}
```

## Step 4: Configure Filestash (3 minutes)

### 4.1 Add Configuration

Edit your Filestash configuration file or use admin panel.

Add this to your config:
```json
{
  "ai": {
    "enabled": true,
    "supabase": {
      "host": "db.xxxxx.supabase.co",
      "password": "your-password"
    },
    "azure_openai": {
      "endpoint": "https://xxxxx.openai.azure.com/",
      "api_key": "your-key",
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
      "max_file_size_mb": 100
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

## Step 5: Build Plugin (5 minutes)

### 5.1 Create Plugin Files

```bash
# Create directory
mkdir -p server/plugin/plg_ai_semantic

# Create index.go (minimal version for quick start)
cat > server/plugin/plg_ai_semantic/index.go << 'EOF'
package plg_ai_semantic

import (
    . "github.com/mickael-kerjean/filestash/server/common"
)

func init() {
    Hooks.Register.Onload(func() {
        Log.Info("plg_ai_semantic::loaded (minimal version)")
        
        // TODO: Initialize clients
        // TODO: Register endpoints
        // TODO: Start workers
    })
}
EOF
```

### 5.2 Register Plugin

Add to `server/plugin/index.go`:
```go
import (
    // ... existing imports ...
    _ "github.com/mickael-kerjean/filestash/server/plugin/plg_ai_semantic"
)
```

### 5.3 Add Dependencies

Add to `go.mod`:
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

### 5.4 Build

```bash
make build_backend
```

If successful, you should see `./filestash` binary created.

## Step 6: Test Basic Setup (5 minutes)

### 6.1 Start Filestash

```bash
./filestash
```

Look for log message:
```
plg_ai_semantic::loaded (minimal version)
```

### 6.2 Test Connections

**Test Supabase:**
```bash
psql "postgresql://postgres:YOUR_PASSWORD@db.xxxxx.supabase.co:5432/postgres" \
  -c "SELECT COUNT(*) FROM ai_documents;"
```

**Test Azure OpenAI:**
```bash
curl -X POST "https://xxxxx.openai.azure.com/openai/deployments/text-embedding-3-large/embeddings?api-version=2024-02-01" \
  -H "Content-Type: application/json" \
  -H "api-key: YOUR_KEY" \
  -d '{"input":"test"}'
```

**Test Unstructured:**
```bash
echo "test" > test.txt
curl -X POST http://localhost:8000/general/v0/general \
  -F "files=@test.txt"
rm test.txt
```

All three should return successful responses!

## What's Next?

### For Complete Implementation

You've completed the infrastructure setup! Now proceed to:

1. **Full Plugin Development**
   - Review [PLUGIN_ARCHITECTURE.md](./PLUGIN_ARCHITECTURE.md)
   - Implement all plugin files
   - Add API endpoints

2. **Document Processing Pipeline**
   - Implement file watcher
   - Create indexing worker
   - Add embedding generation

3. **Search & RAG**
   - Add semantic search handler
   - Implement RAG query handler
   - Build frontend UI

### Quick Testing Script

Create `test-ai-stack.sh`:
```bash
#!/bin/bash

echo "Testing AI Stack..."

# Test Supabase
echo "1. Testing Supabase..."
psql "$SUPABASE_URL" -c "SELECT 1" && echo "✓ Supabase OK" || echo "✗ Supabase FAIL"

# Test Azure OpenAI
echo "2. Testing Azure OpenAI..."
curl -s -X POST "$AZURE_OPENAI_ENDPOINT/openai/deployments/text-embedding-3-large/embeddings?api-version=2024-02-01" \
  -H "api-key: $AZURE_OPENAI_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input":"test"}' > /dev/null && echo "✓ Azure OpenAI OK" || echo "✗ Azure OpenAI FAIL"

# Test Unstructured
echo "3. Testing Unstructured..."
curl -s http://localhost:8000/healthcheck > /dev/null && echo "✓ Unstructured OK" || echo "✗ Unstructured FAIL"

echo "Done!"
```

Run it:
```bash
chmod +x test-ai-stack.sh
./test-ai-stack.sh
```

## Troubleshooting

### Supabase Connection Fails
```bash
# Check network
ping db.xxxxx.supabase.co

# Test direct connection
psql "postgresql://postgres:PASSWORD@db.xxxxx.supabase.co:5432/postgres"

# Check Supabase dashboard for connection pooler issues
```

### Azure OpenAI 401 Error
```bash
# Verify key is correct
echo $AZURE_OPENAI_KEY

# Check deployment names in Azure OpenAI Studio
# They must exactly match your config
```

### Unstructured Container Won't Start
```bash
# Check logs
docker logs unstructured-api

# Check resources
docker stats unstructured-api

# Try with more memory
# Edit docker-compose.ai.yml and increase memory limit
```

### Filestash Won't Build
```bash
# Clean and retry
go clean -modcache
rm -rf vendor/
go mod tidy
go mod download
make build_backend
```

## Cost Estimate

Based on moderate usage (1,000 documents, 100 searches/day):

| Service | Tier | Cost/Month |
|---------|------|------------|
| Supabase | Free | $0 |
| Azure OpenAI | Pay-as-you-go | ~$20-50 |
| Unstructured.io | Self-hosted | $0 (compute only) |
| **Total** | | **~$20-50** |

## Next Steps

1. ✅ Infrastructure is ready
2. 📝 Read [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) for full plugin code
3. 🔨 Implement complete plugin functionality
4. 🧪 Test with real documents
5. 🚀 Deploy to production

**Questions?** Review the [README.md](./README.md) for full documentation index.

---

**Congratulations!** 🎉 Your AI infrastructure is ready. Time to build the plugin!
