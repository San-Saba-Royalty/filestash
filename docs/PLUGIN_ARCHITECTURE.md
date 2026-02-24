# Plugin Architecture - plg_ai_semantic

## Overview

This document details the structure of the `plg_ai_semantic` plugin that integrates AI capabilities into Filestash.

## Directory Structure

```
server/plugin/plg_ai_semantic/
├── index.go                  # Plugin entry point and initialization
├── config.go                 # Configuration management
├── types.go                  # Data structures and interfaces
├── supabase_client.go        # Supabase/pgvector client
├── azure_openai_client.go    # Azure OpenAI client
├── unstructured_client.go    # Unstructured.io client
├── document_processor.go     # Document processing pipeline
├── indexer.go                # File indexing worker
├── search_handler.go         # Semantic search API
├── rag_handler.go            # RAG query API
├── middleware.go             # File operation hooks
├── utils.go                  # Helper functions
└── static/                   # Frontend UI components
    ├── ai-search.js          # Search UI
    ├── ai-chat.js            # Chat interface
    └── ai-search.css         # Styles
```

## Core Files

### 1. `index.go` - Plugin Registration

```go
package plg_ai_semantic

import (
    "github.com/gorilla/mux"
    . "github.com/mickael-kerjean/filestash/server/common"
)

func init() {
    // Load configuration
    loadAIConfig()
    
    // Initialize clients
    initSupabaseClient()
    initAzureOpenAIClient()
    initUnstructuredClient()
    
    // Register HTTP endpoints
    Hooks.Register.HttpEndpoint(func(r *mux.Router) error {
        // AI Search endpoint
        r.HandleFunc(
            WithBase("/api/ai/search"),
            aiSearchHandler,
        ).Methods("GET", "POST")
        
        // RAG Query endpoint
        r.HandleFunc(
            WithBase("/api/ai/chat"),
            ragQueryHandler,
        ).Methods("POST")
        
        // Document indexing status
        r.HandleFunc(
            WithBase("/api/ai/index/status"),
            indexStatusHandler,
        ).Methods("GET")
        
        // Trigger manual indexing
        r.HandleFunc(
            WithBase("/api/ai/index/trigger"),
            triggerIndexingHandler,
        ).Methods("POST")
        
        return nil
    })
    
    // Register middleware for file operations
    Hooks.Register.Middleware(fileOperationMiddleware)
    
    // Register static frontend assets
    registerStaticAssets()
    
    // Start background indexer worker
    Hooks.Register.Onload(func() {
        Log.Info("AI Semantic Plugin: Starting background indexer...")
        go startIndexerWorker()
    })
    
    Log.Info("AI Semantic Plugin loaded successfully")
}
```

### 2. `config.go` - Configuration Management

```go
package plg_ai_semantic

import (
    . "github.com/mickael-kerjean/filestash/server/common"
)

type AIConfig struct {
    Enabled bool `json:"enabled"`
    
    Supabase struct {
        URL    string `json:"url"`
        APIKey string `json:"api_key"`
    } `json:"supabase"`
    
    AzureOpenAI struct {
        Endpoint            string `json:"endpoint"`
        APIKey              string `json:"api_key"`
        APIVersion          string `json:"api_version"`
        EmbeddingsDeployment string `json:"embeddings_deployment"`
        ChatDeployment      string `json:"chat_deployment"`
        EmbeddingDimensions int    `json:"embedding_dimensions"` // Default: 1536
    } `json:"azure_openai"`
    
    Unstructured struct {
        URL    string `json:"url"`
        APIKey string `json:"api_key"` // Optional
    } `json:"unstructured"`
    
    Indexing struct {
        AutoIndexOnUpload bool `json:"auto_index_on_upload"`
        BatchSize         int  `json:"batch_size"`
        MaxFileSizeMB     int  `json:"max_file_size_mb"`
        SupportedMimeTypes []string `json:"supported_mime_types"`
    } `json:"indexing"`
    
    Search struct {
        MaxResults          int     `json:"max_results"`
        SimilarityThreshold float64 `json:"similarity_threshold"`
    } `json:"search"`
    
    RAG struct {
        MaxContextChunks int     `json:"max_context_chunks"`
        Temperature      float64 `json:"temperature"`
        MaxTokens        int     `json:"max_tokens"`
    } `json:"rag"`
}

var aiConfig AIConfig

func loadAIConfig() {
    // Load from Filestash config
    config := Config.Get()
    if aiConfigData, ok := config.Get("ai").(map[string]interface{}); ok {
        // Parse configuration
        // Implementation details...
    }
    
    // Set defaults
    if aiConfig.Indexing.BatchSize == 0 {
        aiConfig.Indexing.BatchSize = 10
    }
    if aiConfig.Indexing.MaxFileSizeMB == 0 {
        aiConfig.Indexing.MaxFileSizeMB = 100
    }
    if aiConfig.AzureOpenAI.EmbeddingDimensions == 0 {
        aiConfig.AzureOpenAI.EmbeddingDimensions = 1536
    }
    if aiConfig.Search.MaxResults == 0 {
        aiConfig.Search.MaxResults = 20
    }
    if aiConfig.Search.SimilarityThreshold == 0 {
        aiConfig.Search.SimilarityThreshold = 0.7
    }
    if aiConfig.RAG.MaxContextChunks == 0 {
        aiConfig.RAG.MaxContextChunks = 5
    }
    if aiConfig.RAG.Temperature == 0 {
        aiConfig.RAG.Temperature = 0.7
    }
    if aiConfig.RAG.MaxTokens == 0 {
        aiConfig.RAG.MaxTokens = 1000
    }
}

func IsAIEnabled() bool {
    return aiConfig.Enabled
}
```

### 3. `types.go` - Data Structures

```go
package plg_ai_semantic

import (
    "time"
)

// Document represents a file in the vector database
type Document struct {
    ID              string    `json:"id"`
    FilePath        string    `json:"file_path"`
    FileName        string    `json:"file_name"`
    FileType        string    `json:"file_type"`
    FileSize        int64     `json:"file_size"`
    FileHash        string    `json:"file_hash"`
    StorageBackend  string    `json:"storage_backend"`
    StoragePath     string    `json:"storage_path"`
    UserID          string    `json:"user_id"`
    SessionID       string    `json:"session_id"`
    Status          string    `json:"status"`
    IndexedAt       time.Time `json:"indexed_at"`
    ChunkCount      int       `json:"chunk_count"`
}

// Embedding represents a vector embedding chunk
type Embedding struct {
    ID         string    `json:"id"`
    DocumentID string    `json:"document_id"`
    ChunkIndex int       `json:"chunk_index"`
    ChunkText  string    `json:"chunk_text"`
    ChunkType  string    `json:"chunk_type"`
    PageNumber int       `json:"page_number"`
    Vector     []float32 `json:"embedding"`
    CreatedAt  time.Time `json:"created_at"`
}

// SearchResult represents a semantic search result
type SearchResult struct {
    DocumentID    string  `json:"document_id"`
    ChunkID       string  `json:"chunk_id"`
    FilePath      string  `json:"file_path"`
    FileName      string  `json:"file_name"`
    ChunkText     string  `json:"chunk_text"`
    PageNumber    int     `json:"page_number"`
    Similarity    float64 `json:"similarity"`
    SectionTitle  string  `json:"section_title,omitempty"`
}

// RAGContext represents context for RAG generation
type RAGContext struct {
    Query    string         `json:"query"`
    Chunks   []SearchResult `json:"chunks"`
    TotalDocs int           `json:"total_docs"`
}

// RAGResponse represents a RAG query response
type RAGResponse struct {
    Answer    string         `json:"answer"`
    Sources   []SearchResult `json:"sources"`
    QueryTime int            `json:"query_time_ms"`
    LLMTime   int            `json:"llm_time_ms"`
}

// UnstructuredElement from Unstructured.io API
type UnstructuredElement struct {
    Type     string                 `json:"type"`
    Text     string                 `json:"text"`
    Metadata map[string]interface{} `json:"metadata"`
}

// IndexingJob represents a document waiting to be indexed
type IndexingJob struct {
    DocumentID string
    FilePath   string
    Priority   int
    RetryCount int
}
```

### 4. `supabase_client.go` - Database Client

```go
package plg_ai_semantic

import (
    "context"
    "fmt"
    
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

var supabasePool *pgxpool.Pool

func initSupabaseClient() error {
    if !IsAIEnabled() {
        return nil
    }
    
    connString := fmt.Sprintf(
        "postgres://postgres:[YOUR-PASSWORD]@%s:5432/postgres",
        aiConfig.Supabase.URL,
    )
    
    config, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return fmt.Errorf("unable to parse connection string: %w", err)
    }
    
    // Connection pool configuration
    config.MaxConns = 50
    config.MinConns = 10
    config.MaxConnLifetime = 3600
    
    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return fmt.Errorf("unable to create connection pool: %w", err)
    }
    
    supabasePool = pool
    Log.Info("Supabase connection pool established")
    return nil
}

// StoreDocument inserts or updates a document
func StoreDocument(doc *Document) error {
    query := `
        INSERT INTO ai_documents (
            file_path, file_name, file_type, file_size, file_hash,
            storage_backend, storage_path, user_id, session_id, status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (file_path, user_id, storage_backend) 
        DO UPDATE SET
            file_name = EXCLUDED.file_name,
            file_hash = EXCLUDED.file_hash,
            file_size = EXCLUDED.file_size,
            last_updated = NOW(),
            status = EXCLUDED.status
        RETURNING id
    `
    
    err := supabasePool.QueryRow(
        context.Background(),
        query,
        doc.FilePath, doc.FileName, doc.FileType, doc.FileSize, doc.FileHash,
        doc.StorageBackend, doc.StoragePath, doc.UserID, doc.SessionID, doc.Status,
    ).Scan(&doc.ID)
    
    return err
}

// StoreEmbedding inserts a vector embedding
func StoreEmbedding(emb *Embedding) error {
    query := `
        INSERT INTO ai_embeddings (
            document_id, chunk_index, chunk_text, chunk_type, 
            page_number, embedding, token_count
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (document_id, chunk_index) 
        DO UPDATE SET
            chunk_text = EXCLUDED.chunk_text,
            embedding = EXCLUDED.embedding
        RETURNING id
    `
    
    _, err := supabasePool.Exec(
        context.Background(),
        query,
        emb.DocumentID, emb.ChunkIndex, emb.ChunkText, emb.ChunkType,
        emb.PageNumber, emb.Vector, len(emb.ChunkText)/4, // rough token estimate
    )
    
    return err
}

// SemanticSearch performs vector similarity search
func SemanticSearch(
    queryEmbedding []float32,
    userID string,
    maxResults int,
    threshold float64,
) ([]SearchResult, error) {
    query := `
        SELECT * FROM semantic_search($1, $2, $3, $4)
    `
    
    rows, err := supabasePool.Query(
        context.Background(),
        query,
        queryEmbedding,
        userID,
        maxResults,
        threshold,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        err := rows.Scan(
            &r.DocumentID,
            &r.ChunkID,
            &r.ChunkText,
            &r.FilePath,
            &r.FileName,
            &r.Similarity,
            &r.PageNumber,
            &r.SectionTitle,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, r)
    }
    
    return results, nil
}

// GetRAGContext retrieves context chunks for RAG
func GetRAGContext(
    queryEmbedding []float32,
    userID string,
    maxChunks int,
) ([]SearchResult, error) {
    query := `
        SELECT * FROM get_rag_context($1, $2, $3)
    `
    
    rows, err := supabasePool.Query(
        context.Background(),
        query,
        queryEmbedding,
        userID,
        maxChunks,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        err := rows.Scan(
            &r.ChunkText,
            &r.FileName,
            &r.FilePath,
            &r.PageNumber,
            &r.Similarity,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, r)
    }
    
    return results, nil
}

// DeleteDocument removes document and embeddings
func DeleteDocument(documentID string) error {
    // CASCADE delete will handle embeddings
    query := `
        UPDATE ai_documents 
        SET deleted_at = NOW() 
        WHERE id = $1
    `
    
    _, err := supabasePool.Exec(context.Background(), query, documentID)
    return err
}
```

### 5. `azure_openai_client.go` - Azure OpenAI Integration

```go
package plg_ai_semantic

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

// GenerateEmbedding creates a vector embedding for text
func GenerateEmbedding(text string) ([]float32, error) {
    url := fmt.Sprintf(
        "%s/openai/deployments/%s/embeddings?api-version=%s",
        aiConfig.AzureOpenAI.Endpoint,
        aiConfig.AzureOpenAI.EmbeddingsDeployment,
        aiConfig.AzureOpenAI.APIVersion,
    )
    
    requestBody := map[string]interface{}{
        "input": text,
        "dimensions": aiConfig.AzureOpenAI.EmbeddingDimensions,
    }
    
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("api-key", aiConfig.AzureOpenAI.APIKey)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Azure OpenAI API error: %s", string(body))
    }
    
    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if len(result.Data) == 0 {
        return nil, fmt.Errorf("no embedding returned")
    }
    
    return result.Data[0].Embedding, nil
}

// GenerateRAGResponse generates an answer using GPT-4
func GenerateRAGResponse(query string, context []SearchResult) (string, error) {
    // Build context string from chunks
    contextText := ""
    for i, chunk := range context {
        contextText += fmt.Sprintf(
            "[Source %d: %s, Page %d]\n%s\n\n",
            i+1,
            chunk.FileName,
            chunk.PageNumber,
            chunk.ChunkText,
        )
    }
    
    systemPrompt := `You are a helpful AI assistant that answers questions based on the provided document excerpts. 
Always cite your sources by mentioning the source number and file name. 
If the answer cannot be found in the provided context, say so clearly.`
    
    userPrompt := fmt.Sprintf(
        "Context from documents:\n\n%s\n\nQuestion: %s",
        contextText,
        query,
    )
    
    url := fmt.Sprintf(
        "%s/openai/deployments/%s/chat/completions?api-version=%s",
        aiConfig.AzureOpenAI.Endpoint,
        aiConfig.AzureOpenAI.ChatDeployment,
        aiConfig.AzureOpenAI.APIVersion,
    )
    
    requestBody := map[string]interface{}{
        "messages": []map[string]string{
            {"role": "system", "content": systemPrompt},
            {"role": "user", "content": userPrompt},
        },
        "temperature": aiConfig.RAG.Temperature,
        "max_tokens":  aiConfig.RAG.MaxTokens,
    }
    
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return "", err
    }
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("api-key", aiConfig.AzureOpenAI.APIKey)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("Azure OpenAI API error: %s", string(body))
    }
    
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("no response from GPT-4")
    }
    
    return result.Choices[0].Message.Content, nil
}
```

### 6. `unstructured_client.go` - Document Processing

```go
package plg_ai_semantic

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
)

// ProcessDocument sends a file to Unstructured.io for processing
func ProcessDocument(fileContent []byte, fileName string) ([]UnstructuredElement, error) {
    url := fmt.Sprintf("%s/general/v0/general", aiConfig.Unstructured.URL)
    
    // Create multipart form
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    // Add file
    part, err := writer.CreateFormFile("files", fileName)
    if err != nil {
        return nil, err
    }
    if _, err := part.Write(fileContent); err != nil {
        return nil, err
    }
    
    // Add strategy parameter
    writer.WriteField("strategy", "hi_res") // Best for complex documents
    writer.WriteField("chunking_strategy", "by_title")
    writer.WriteField("max_characters", "500")
    writer.WriteField("new_after_n_chars", "400")
    writer.WriteField("overlap", "50")
    
    writer.Close()
    
    // Create request
    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", writer.FormDataContentType())
    if aiConfig.Unstructured.APIKey != "" {
        req.Header.Set("unstructured-api-key", aiConfig.Unstructured.APIKey)
    }
    
    // Send request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Unstructured.io error: %s", string(body))
    }
    
    // Parse response
    var elements []UnstructuredElement
    if err := json.NewDecoder(resp.Body).Decode(&elements); err != nil {
        return nil, err
    }
    
    return elements, nil
}

// ChunkElements groups elements into meaningful chunks
func ChunkElements(elements []UnstructuredElement) []string {
    chunks := []string{}
    currentChunk := ""
    maxChunkSize := 500 // characters
    
    for _, elem := range elements {
        // Skip empty elements
        if len(elem.Text) == 0 {
            continue
        }
        
        // If adding this would exceed max size, start new chunk
        if len(currentChunk)+len(elem.Text) > maxChunkSize && len(currentChunk) > 0 {
            chunks = append(chunks, currentChunk)
            currentChunk = elem.Text
        } else {
            if len(currentChunk) > 0 {
                currentChunk += "\n\n"
            }
            currentChunk += elem.Text
        }
    }
    
    // Add final chunk
    if len(currentChunk) > 0 {
        chunks = append(chunks, currentChunk)
    }
    
    return chunks
}
```

## Integration with Filestash

### Middleware Hook

```go
func fileOperationMiddleware(next HandlerFunc) HandlerFunc {
    return func(app *App, res http.ResponseWriter, req *http.Request) error {
        // Call next handler first
        err := next(app, res, req)
        if err != nil {
            return err
        }
        
        // After successful file operation, trigger indexing
        if IsAIEnabled() && aiConfig.Indexing.AutoIndexOnUpload {
            switch req.Method {
            case "POST", "PUT", "PATCH":
                // File upload or modification
                go queueDocumentForIndexing(app, req)
            case "DELETE":
                // File deletion
                go removeDocumentFromIndex(app, req)
            }
        }
        
        return nil
    }
}
```

## Next Steps

1. Review [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) for build instructions
2. Review [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for endpoint details
3. Review [UNSTRUCTURED_INTEGRATION.md](./UNSTRUCTURED_INTEGRATION.md) for Docker setup
