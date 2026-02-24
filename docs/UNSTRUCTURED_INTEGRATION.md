# Unstructured.io Self-Hosted Integration

## Overview

This guide explains how to deploy and integrate the open-source version of Unstructured.io for document processing in your Filestash AI setup.

## Why Self-Hosted Unstructured.io?

**Benefits**:
- ✅ **Data Privacy**: Documents never leave your infrastructure
- ✅ **Cost Control**: No per-document API charges
- ✅ **Customization**: Modify processing pipelines as needed
- ✅ **Performance**: Optimize for your specific use case
- ✅ **No External Dependencies**: Works in air-gapped environments

**Trade-offs**:
- ⚠️ Requires Docker infrastructure
- ⚠️ Higher resource consumption
- ⚠️ You manage updates and maintenance

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Filestash Backend                       │
│                                                          │
│  ┌────────────────────────────────────────────┐         │
│  │  plg_ai_semantic Plugin                    │         │
│  │  (Document Processing Logic)               │         │
│  └──────────────────┬─────────────────────────┘         │
│                     │ HTTP POST /general/v0/general     │
└─────────────────────┼─────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│         Unstructured.io Docker Container                 │
│                                                          │
│  ┌──────────────────────────────────────────────┐       │
│  │  FastAPI Service (Port 8000)                 │       │
│  │  • /general/v0/general (main endpoint)       │       │
│  │  • /healthcheck                              │       │
│  └──────────────────────────────────────────────┘       │
│                                                          │
│  ┌──────────────────────────────────────────────┐       │
│  │  Document Partitioners                        │       │
│  │  • PDF: pdfplumber, pdfminer, tesseract     │       │
│  │  • DOCX: python-docx                         │       │
│  │  • Images: PIL, tesseract OCR                │       │
│  │  • HTML: BeautifulSoup                       │       │
│  └──────────────────────────────────────────────┘       │
│                                                          │
│  ┌──────────────────────────────────────────────┐       │
│  │  ML Models                                    │       │
│  │  • Layout detection (yolox)                  │       │
│  │  • Table extraction (table-transformer)      │       │
│  │  • OCR (tesseract)                          │       │
│  └──────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────┘
```

## Installation

### Option 1: Docker Compose (Recommended)

Create `docker-compose.ai.yml` in your Filestash root directory:

```yaml
version: '3.8'

services:
  unstructured-api:
    image: quay.io/unstructured-io/unstructured-api:latest
    container_name: unstructured-api
    ports:
      - "8000:8000"
    environment:
      # API Configuration
      - UNSTRUCTURED_API_KEY=${UNSTRUCTURED_API_KEY:-}  # Optional: set for auth
      
      # Processing Configuration
      - UNSTRUCTURED_PARALLEL_MODE=true
      - UNSTRUCTURED_PARALLEL_NUM_THREADS=4
      
      # Memory Configuration
      - UNSTRUCTURED_MEMORY_FREE_MINIMUM_MB=512
      
      # Allowed MIME types (customize as needed)
      - UNSTRUCTURED_ALLOWED_MIMETYPES=application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-powerpoint,application/vnd.openxmlformats-officedocument.presentationml.presentation,text/plain,text/html,text/csv,image/jpeg,image/png,image/tiff
      
      # OCR Configuration
      - OCR_AGENT=tesseract  # Options: tesseract, paddle
      - TESSERACT_LANG=eng  # Language for OCR
      
    volumes:
      # Cache directory for models
      - ./unstructured-cache:/cache
      
      # Optional: Mount local documents for testing
      # - ./test-documents:/test-documents:ro
    
    # Resource limits
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
    
    restart: unless-stopped
    
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    
    # Logging
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  # Optional: Redis cache for better performance
  unstructured-redis:
    image: redis:7-alpine
    container_name: unstructured-redis
    ports:
      - "6379:6379"
    volumes:
      - unstructured-redis-data:/data
    restart: unless-stopped
    command: redis-server --appendonly yes --maxmemory 1gb --maxmemory-policy allkeys-lru

volumes:
  unstructured-redis-data:
```

### Start the Services

```bash
# Navigate to Filestash directory
cd /Users/gqadonis/Projects/sansaba/filestash

# Create cache directory
mkdir -p unstructured-cache

# Start services
docker-compose -f docker-compose.ai.yml up -d

# View logs
docker-compose -f docker-compose.ai.yml logs -f unstructured-api

# Check health
curl http://localhost:8000/healthcheck
```

Expected health check response:
```json
{
  "healthcheck": "OK",
  "version": "0.13.0"
}
```

### Option 2: Docker Run (Simple)

For quick testing:

```bash
docker run -d \
  --name unstructured-api \
  -p 8000:8000 \
  -e UNSTRUCTURED_PARALLEL_MODE=true \
  -v $(pwd)/unstructured-cache:/cache \
  --memory=8g \
  --cpus=4 \
  quay.io/unstructured-io/unstructured-api:latest
```

### Option 3: Kubernetes Deployment

Create `k8s-unstructured-deployment.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: unstructured-config
data:
  UNSTRUCTURED_PARALLEL_MODE: "true"
  UNSTRUCTURED_PARALLEL_NUM_THREADS: "4"
  OCR_AGENT: "tesseract"
  TESSERACT_LANG: "eng"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: unstructured-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: unstructured-api
  template:
    metadata:
      labels:
        app: unstructured-api
    spec:
      containers:
      - name: unstructured-api
        image: quay.io/unstructured-io/unstructured-api:latest
        ports:
        - containerPort: 8000
        envFrom:
        - configMapRef:
            name: unstructured-config
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
        volumeMounts:
        - name: cache
          mountPath: /cache
        livenessProbe:
          httpGet:
            path: /healthcheck
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthcheck
            port: 8000
          initialDelaySeconds: 20
          periodSeconds: 5
      volumes:
      - name: cache
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: unstructured-api
spec:
  selector:
    app: unstructured-api
  ports:
  - protocol: TCP
    port: 8000
    targetPort: 8000
  type: ClusterIP
```

Deploy:
```bash
kubectl apply -f k8s-unstructured-deployment.yaml
```

## API Usage

### Basic Document Processing

**Endpoint**: `POST http://localhost:8000/general/v0/general`

**Request** (multipart/form-data):
```bash
curl -X POST http://localhost:8000/general/v0/general \
  -F "files=@/path/to/document.pdf" \
  -F "strategy=hi_res" \
  -F "chunking_strategy=by_title" \
  -F "max_characters=500" \
  -F "new_after_n_chars=400" \
  -F "overlap=50"
```

**Parameters**:
- `files`: File to process (required)
- `strategy`: Processing strategy (default: `auto`)
  - `fast`: Fast processing, lower accuracy
  - `hi_res`: High accuracy, slower processing
  - `auto`: Automatic selection based on file type
- `chunking_strategy`: How to chunk text (default: `by_title`)
  - `by_title`: Chunk by document titles/sections
  - `by_page`: One chunk per page
  - `basic`: Simple character-based chunking
- `max_characters`: Maximum chunk size (default: 500)
- `new_after_n_chars`: Start new chunk after N chars (default: same as max)
- `overlap`: Character overlap between chunks (default: 0)
- `ocr_languages`: Languages for OCR (default: `eng`)
- `encoding`: Output encoding (default: `utf-8`)

**Response**:
```json
[
  {
    "type": "Title",
    "element_id": "8d8e4b24f21e3a7e6e8b23c6f03dc8e1",
    "text": "Executive Summary",
    "metadata": {
      "filename": "document.pdf",
      "file_directory": "/tmp",
      "page_number": 1,
      "coordinates": {
        "points": [[100, 200], [500, 200], [500, 250], [100, 250]],
        "system": "PixelSpace"
      },
      "languages": ["eng"]
    }
  },
  {
    "type": "NarrativeText",
    "element_id": "7c7d3a13e10c2a6d5d7a12b5e02cb7d0",
    "text": "This report presents the quarterly financial results...",
    "metadata": {
      "filename": "document.pdf",
      "page_number": 1
    }
  }
]
```

### Supported Element Types

Unstructured.io classifies content into element types:

| Type | Description | Use Case |
|------|-------------|----------|
| `Title` | Document/section titles | Chunking boundaries |
| `NarrativeText` | Body paragraphs | Main content |
| `ListItem` | List items | Structured content |
| `Table` | Tables (with structure) | Structured data |
| `Image` | Images (with captions) | Visual content |
| `Header` | Page headers | Metadata |
| `Footer` | Page footers | Metadata |
| `PageBreak` | Page boundaries | Document structure |
| `FigureCaption` | Figure captions | Image context |
| `Formula` | Mathematical formulas | Technical content |

### Advanced Options

#### PDF with OCR

For scanned PDFs:
```bash
curl -X POST http://localhost:8000/general/v0/general \
  -F "files=@scanned.pdf" \
  -F "strategy=hi_res" \
  -F "ocr_languages=eng+fra" \
  -F "skip_infer_table_types=[]"
```

#### Extract Tables

```bash
curl -X POST http://localhost:8000/general/v0/general \
  -F "files=@document.xlsx" \
  -F "extract_tables_as_html=true"
```

#### Process Images

```bash
curl -X POST http://localhost:8000/general/v0/general \
  -F "files=@diagram.png" \
  -F "strategy=hi_res" \
  -F "ocr_languages=eng"
```

## Integration with Filestash Plugin

### Client Implementation

In `server/plugin/plg_ai_semantic/unstructured_client.go`:

```go
package plg_ai_semantic

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "time"
)

type UnstructuredClient struct {
    BaseURL    string
    APIKey     string
    HTTPClient *http.Client
}

func initUnstructuredClient() error {
    if !IsAIEnabled() {
        return nil
    }
    
    unstructuredClient = &UnstructuredClient{
        BaseURL: aiConfig.Unstructured.URL,
        APIKey:  aiConfig.Unstructured.APIKey,
        HTTPClient: &http.Client{
            Timeout: 5 * time.Minute, // Long timeout for large files
        },
    }
    
    // Test connection
    if err := unstructuredClient.HealthCheck(); err != nil {
        return fmt.Errorf("unstructured health check failed: %w", err)
    }
    
    Log.Info("plg_ai_semantic::unstructured client initialized")
    return nil
}

func (c *UnstructuredClient) HealthCheck() error {
    url := fmt.Sprintf("%s/healthcheck", c.BaseURL)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }
    
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("health check failed: status %d", resp.StatusCode)
    }
    
    return nil
}

func (c *UnstructuredClient) ProcessDocument(
    fileContent []byte,
    fileName string,
    options ProcessingOptions,
) ([]UnstructuredElement, error) {
    url := fmt.Sprintf("%s/general/v0/general", c.BaseURL)
    
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
    
    // Add processing parameters
    writer.WriteField("strategy", options.Strategy)
    writer.WriteField("chunking_strategy", options.ChunkingStrategy)
    writer.WriteField("max_characters", fmt.Sprintf("%d", options.MaxCharacters))
    writer.WriteField("new_after_n_chars", fmt.Sprintf("%d", options.NewAfterNChars))
    writer.WriteField("overlap", fmt.Sprintf("%d", options.Overlap))
    
    if options.OCRLanguages != "" {
        writer.WriteField("ocr_languages", options.OCRLanguages)
    }
    
    writer.Close()
    
    // Create request
    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", writer.FormDataContentType())
    if c.APIKey != "" {
        req.Header.Set("unstructured-api-key", c.APIKey)
    }
    
    // Send request
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf(
            "unstructured API error (status %d): %s",
            resp.StatusCode,
            string(bodyBytes),
        )
    }
    
    // Parse response
    var elements []UnstructuredElement
    if err := json.NewDecoder(resp.Body).Decode(&elements); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    return elements, nil
}

type ProcessingOptions struct {
    Strategy          string // "fast", "hi_res", "auto"
    ChunkingStrategy  string // "by_title", "by_page", "basic"
    MaxCharacters     int
    NewAfterNChars    int
    Overlap           int
    OCRLanguages      string // e.g., "eng+fra"
}

func DefaultProcessingOptions() ProcessingOptions {
    return ProcessingOptions{
        Strategy:         "hi_res",
        ChunkingStrategy: "by_title",
        MaxCharacters:    500,
        NewAfterNChars:   400,
        Overlap:          50,
        OCRLanguages:     "eng",
    }
}
```

### Usage in Document Processor

```go
func ProcessFileForIndexing(filePath string, fileContent []byte) error {
    // Process with Unstructured.io
    options := DefaultProcessingOptions()
    elements, err := unstructuredClient.ProcessDocument(
        fileContent,
        filepath.Base(filePath),
        options,
    )
    if err != nil {
        return fmt.Errorf("unstructured processing failed: %w", err)
    }
    
    // Group elements into chunks
    chunks := groupElementsIntoChunks(elements, options.MaxCharacters)
    
    // Generate embeddings for each chunk
    for i, chunk := range chunks {
        embedding, err := GenerateEmbedding(chunk.Text)
        if err != nil {
            return fmt.Errorf("embedding generation failed: %w", err)
        }
        
        // Store embedding
        emb := &Embedding{
            DocumentID:   documentID,
            ChunkIndex:   i,
            ChunkText:    chunk.Text,
            ChunkType:    chunk.Type,
            PageNumber:   chunk.PageNumber,
            SectionTitle: chunk.SectionTitle,
            Vector:       embedding,
            TokenCount:   len(chunk.Text) / 4, // Rough estimate
        }
        
        if err := StoreEmbedding(ctx, emb); err != nil {
            return fmt.Errorf("failed to store embedding: %w", err)
        }
    }
    
    return nil
}
```

## Performance Optimization

### 1. Processing Strategy Selection

Choose strategy based on document type:

```go
func selectStrategy(fileType string) string {
    switch fileType {
    case "text/plain", "text/html", "text/csv":
        return "fast"  // Plain text doesn't need hi_res
    case "application/pdf":
        return "hi_res"  // PDFs benefit from layout detection
    case "image/jpeg", "image/png":
        return "hi_res"  // Images need OCR
    default:
        return "auto"
    }
}
```

### 2. Parallel Processing

Process multiple documents concurrently:

```go
func processBatch(files []FileInfo) error {
    const maxConcurrent = 4
    sem := make(chan struct{}, maxConcurrent)
    errChan := make(chan error, len(files))
    
    for _, file := range files {
        sem <- struct{}{} // Acquire
        go func(f FileInfo) {
            defer func() { <-sem }() // Release
            
            if err := ProcessFileForIndexing(f.Path, f.Content); err != nil {
                errChan <- err
            }
        }(file)
    }
    
    // Wait for all to complete
    for i := 0; i < maxConcurrent; i++ {
        sem <- struct{}{}
    }
    
    close(errChan)
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### 3. Caching Results

Cache processed results to avoid reprocessing:

```go
func ProcessDocumentWithCache(filePath string, fileHash string) error {
    // Check if already processed
    cacheKey := fmt.Sprintf("unstructured:%s", fileHash)
    if cached, err := getFromCache(cacheKey); err == nil {
        return useCache(cached)
    }
    
    // Process document
    elements, err := unstructuredClient.ProcessDocument(...)
    if err != nil {
        return err
    }
    
    // Cache results
    saveToCache(cacheKey, elements, 24*time.Hour)
    
    return nil
}
```

## Troubleshooting

### Issue: Container Out of Memory

**Symptoms**: Container crashes or processing fails
**Solution**: Increase memory limits

```yaml
deploy:
  resources:
    limits:
      memory: 16G  # Increase from 8G
```

### Issue: Slow Processing

**Symptoms**: Documents take minutes to process
**Solutions**:
1. Use `fast` strategy for simple documents
2. Reduce `max_characters` for chunking
3. Add more CPU cores
4. Enable parallel mode

```yaml
environment:
  - UNSTRUCTURED_PARALLEL_MODE=true
  - UNSTRUCTURED_PARALLEL_NUM_THREADS=8
```

### Issue: OCR Not Working

**Symptoms**: Scanned PDFs return empty content
**Solutions**:
1. Verify tesseract is installed
2. Check OCR languages are available
3. Use `hi_res` strategy

```bash
# Check available languages
docker exec unstructured-api tesseract --list-langs
```

### Issue: Connection Timeout

**Symptoms**: API requests timeout
**Solutions**:
1. Increase client timeout
2. Process smaller files
3. Check network connectivity

```go
HTTPClient: &http.Client{
    Timeout: 10 * time.Minute,  // Increase timeout
}
```

## Monitoring

### Health Check Endpoint

```bash
curl http://localhost:8000/healthcheck
```

### Prometheus Metrics

Add metrics collection:

```yaml
services:
  unstructured-api:
    environment:
      - ENABLE_METRICS=true
    ports:
      - "9090:9090"  # Metrics port
```

Query metrics:
```bash
curl http://localhost:9090/metrics
```

### Logging

View logs:
```bash
docker-compose -f docker-compose.ai.yml logs -f unstructured-api
```

Configure log level:
```yaml
environment:
  - LOG_LEVEL=DEBUG  # Options: DEBUG, INFO, WARNING, ERROR
```

## Maintenance

### Updates

Pull latest image:
```bash
docker pull quay.io/unstructured-io/unstructured-api:latest
docker-compose -f docker-compose.ai.yml up -d
```

### Backup

Backup cache directory:
```bash
tar -czf unstructured-cache-backup.tar.gz unstructured-cache/
```

### Cleanup

Remove old containers:
```bash
docker-compose -f docker-compose.ai.yml down
docker volume prune
```

## Next Steps

1. Review [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) for integration
2. Review [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for API details
3. Test with sample documents
4. Monitor performance and adjust resources
