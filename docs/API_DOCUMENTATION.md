# API Documentation

## Overview

The AI Semantic plugin exposes RESTful API endpoints for semantic search and RAG (Retrieval Augmented Generation) capabilities.

All endpoints require authentication via Filestash's existing session management.

## Base URL

```
/api/ai
```

## Authentication

All requests must include a valid Filestash session cookie. Use the existing Filestash authentication flow.

## Endpoints

### 1. Semantic Search

Perform semantic search across indexed documents.

**Endpoint**: `POST /api/ai/search`

**Request Body**:
```json
{
  "query": "quarterly financial reports",
  "max_results": 20,
  "similarity_threshold": 0.7,
  "filter": {
    "file_types": ["pdf", "docx"],
    "storage_backends": ["s3", "azure"],
    "date_from": "2024-01-01",
    "date_to": "2024-12-31"
  }
}
```

**Parameters**:
- `query` (string, required): Natural language search query
- `max_results` (integer, optional): Maximum results to return (default: 20, max: 100)
- `similarity_threshold` (float, optional): Minimum similarity score 0.0-1.0 (default: 0.7)
- `filter` (object, optional): Filter criteria
  - `file_types` (array, optional): Filter by file extensions
  - `storage_backends` (array, optional): Filter by storage backend
  - `date_from` (string, optional): ISO 8601 date
  - `date_to` (string, optional): ISO 8601 date

**Response** (200 OK):
```json
{
  "success": true,
  "results": [
    {
      "document_id": "uuid-1234",
      "chunk_id": "uuid-5678",
      "file_path": "/reports/Q3-2024.pdf",
      "file_name": "Q3-2024.pdf",
      "chunk_text": "Revenue increased by 25% in Q3...",
      "page_number": 5,
      "similarity": 0.89,
      "section_title": "Financial Summary",
      "storage_backend": "s3"
    }
  ],
  "total": 15,
  "query_time_ms": 45
}
```

**Response** (400 Bad Request):
```json
{
  "success": false,
  "error": "Query parameter is required"
}
```

**Response** (401 Unauthorized):
```json
{
  "success": false,
  "error": "Authentication required"
}
```

**Response** (500 Internal Server Error):
```json
{
  "success": false,
  "error": "Failed to generate embedding: connection timeout"
}
```

**Example cURL**:
```bash
curl -X POST http://localhost:8334/api/ai/search \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{
    "query": "quarterly financial reports",
    "max_results": 10
  }'
```

---

### 2. RAG Chat Query

Ask questions and get AI-generated answers based on your documents.

**Endpoint**: `POST /api/ai/chat`

**Request Body**:
```json
{
  "query": "What were the key findings in Q3 financial report?",
  "conversation_id": "optional-conversation-id",
  "max_context_chunks": 5,
  "temperature": 0.7,
  "stream": false
}
```

**Parameters**:
- `query` (string, required): User question
- `conversation_id` (string, optional): For multi-turn conversations
- `max_context_chunks` (integer, optional): Number of document chunks to use as context (default: 5)
- `temperature` (float, optional): LLM temperature 0.0-1.0 (default: 0.7)
- `stream` (boolean, optional): Enable streaming response (default: false)

**Response** (200 OK):
```json
{
  "success": true,
  "answer": "Based on the Q3 financial report, the key findings include:\n\n1. Revenue increased by 25% year-over-year\n2. Net profit margin improved to 18%\n3. Customer acquisition costs decreased by 15%\n\nThese findings are detailed in the Financial Summary section.",
  "sources": [
    {
      "file_name": "Q3-2024.pdf",
      "file_path": "/reports/Q3-2024.pdf",
      "page_number": 5,
      "chunk_text": "Revenue increased by 25%...",
      "similarity": 0.92
    }
  ],
  "conversation_id": "conv-uuid-1234",
  "query_time_ms": 78,
  "llm_time_ms": 2340
}
```

**Streaming Response** (when `stream: true`):

Server-Sent Events (SSE) format:

```
data: {"type":"chunk","content":"Based on"}
data: {"type":"chunk","content":" the Q3"}
data: {"type":"chunk","content":" financial report"}
...
data: {"type":"sources","sources":[...]}
data: {"type":"done"}
```

**Response** (400 Bad Request):
```json
{
  "success": false,
  "error": "Query parameter is required"
}
```

**Example cURL**:
```bash
curl -X POST http://localhost:8334/api/ai/chat \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{
    "query": "What were the key findings in Q3?",
    "max_context_chunks": 5
  }'
```

**Example with Streaming**:
```bash
curl -X POST http://localhost:8334/api/ai/chat \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{
    "query": "Summarize all quarterly reports",
    "stream": true
  }' \
  --no-buffer
```

---

### 3. Index Status

Get the current status of document indexing.

**Endpoint**: `GET /api/ai/index/status`

**Query Parameters**:
- `limit` (integer, optional): Number of recent documents to return (default: 50)
- `status` (string, optional): Filter by status: `pending`, `processing`, `completed`, `failed`

**Response** (200 OK):
```json
{
  "success": true,
  "summary": {
    "total_documents": 1250,
    "indexed": 1180,
    "pending": 45,
    "processing": 15,
    "failed": 10
  },
  "recent_documents": [
    {
      "id": "uuid-1234",
      "file_name": "Report.pdf",
      "file_path": "/reports/Report.pdf",
      "status": "completed",
      "chunk_count": 25,
      "indexed_at": "2024-01-28T14:30:00Z",
      "processing_time_ms": 5240
    }
  ],
  "queue_depth": 60
}
```

**Example cURL**:
```bash
curl -X GET "http://localhost:8334/api/ai/index/status?status=failed&limit=10" \
  -H "Cookie: auth=your-session-cookie"
```

---

### 4. Trigger Indexing

Manually trigger indexing for specific files or re-index existing documents.

**Endpoint**: `POST /api/ai/index/trigger`

**Request Body**:
```json
{
  "action": "index",
  "files": [
    {
      "path": "/documents/report.pdf",
      "storage_backend": "s3"
    }
  ],
  "priority": 5,
  "force_reindex": false
}
```

**Parameters**:
- `action` (string, required): Action to perform: `index`, `reindex_all`, `reindex_failed`
- `files` (array, optional): Specific files to index (required if action is `index`)
- `priority` (integer, optional): Priority 1-10 (default: 5, higher = higher priority)
- `force_reindex` (boolean, optional): Reindex even if already indexed (default: false)

**Response** (200 OK):
```json
{
  "success": true,
  "queued": 5,
  "message": "5 documents queued for indexing"
}
```

**Example cURL - Index specific files**:
```bash
curl -X POST http://localhost:8334/api/ai/index/trigger \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{
    "action": "index",
    "files": [
      {"path": "/reports/Q4-2024.pdf", "storage_backend": "s3"}
    ],
    "priority": 8
  }'
```

**Example cURL - Reindex all failed**:
```bash
curl -X POST http://localhost:8334/api/ai/index/trigger \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{
    "action": "reindex_failed",
    "priority": 7
  }'
```

---

### 5. Get Document Status

Get detailed status for a specific document.

**Endpoint**: `GET /api/ai/index/document/{id}`

**Path Parameters**:
- `id` (string, required): Document UUID

**Response** (200 OK):
```json
{
  "success": true,
  "document": {
    "id": "uuid-1234",
    "file_path": "/reports/Q3-2024.pdf",
    "file_name": "Q3-2024.pdf",
    "file_type": "application/pdf",
    "file_size": 2458624,
    "storage_backend": "s3",
    "status": "completed",
    "chunk_count": 47,
    "total_tokens": 12450,
    "indexed_at": "2024-01-28T14:30:00Z",
    "last_updated": "2024-01-28T14:35:00Z",
    "processing_time_ms": 5240,
    "chunks": [
      {
        "chunk_index": 0,
        "chunk_text": "Executive Summary...",
        "page_number": 1,
        "section_title": "Executive Summary",
        "token_count": 245
      }
    ]
  }
}
```

**Response** (404 Not Found):
```json
{
  "success": false,
  "error": "Document not found"
}
```

**Example cURL**:
```bash
curl -X GET http://localhost:8334/api/ai/index/document/uuid-1234 \
  -H "Cookie: auth=your-session-cookie"
```

---

### 6. Get Statistics

Get AI usage statistics and analytics.

**Endpoint**: `GET /api/ai/stats`

**Query Parameters**:
- `period` (string, optional): Time period: `day`, `week`, `month`, `year` (default: `week`)

**Response** (200 OK):
```json
{
  "success": true,
  "period": "week",
  "stats": {
    "total_documents": 1250,
    "total_embeddings": 58420,
    "total_searches": 3420,
    "total_rag_queries": 856,
    "avg_search_time_ms": 67,
    "avg_llm_time_ms": 2340,
    "storage_size_mb": 342,
    "popular_queries": [
      {"query": "financial reports", "count": 145},
      {"query": "project updates", "count": 98}
    ],
    "search_by_day": [
      {"date": "2024-01-22", "searches": 420, "rag_queries": 95},
      {"date": "2024-01-23", "searches": 385, "rag_queries": 102}
    ],
    "top_documents": [
      {
        "file_name": "Q3-Report.pdf",
        "file_path": "/reports/Q3-Report.pdf",
        "access_count": 234
      }
    ]
  }
}
```

**Example cURL**:
```bash
curl -X GET "http://localhost:8334/api/ai/stats?period=month" \
  -H "Cookie: auth=your-session-cookie"
```

---

## Error Responses

All endpoints may return the following error codes:

### 400 Bad Request
Invalid request parameters or body.

```json
{
  "success": false,
  "error": "Invalid parameter: max_results must be between 1 and 100"
}
```

### 401 Unauthorized
No valid session or authentication required.

```json
{
  "success": false,
  "error": "Authentication required"
}
```

### 403 Forbidden
User doesn't have permission to access resource.

```json
{
  "success": false,
  "error": "Access denied to this document"
}
```

### 404 Not Found
Resource not found.

```json
{
  "success": false,
  "error": "Document not found"
}
```

### 429 Too Many Requests
Rate limit exceeded.

```json
{
  "success": false,
  "error": "Rate limit exceeded. Please try again in 60 seconds."
}
```

### 500 Internal Server Error
Server error during processing.

```json
{
  "success": false,
  "error": "Internal server error: failed to connect to database"
}
```

### 503 Service Unavailable
External service (Azure OpenAI, Unstructured.io, Supabase) unavailable.

```json
{
  "success": false,
  "error": "Azure OpenAI service temporarily unavailable"
}
```

---

## Rate Limiting

API endpoints are rate-limited to prevent abuse:

- **Search endpoint**: 100 requests per minute per user
- **Chat endpoint**: 20 requests per minute per user
- **Index trigger**: 10 requests per minute per user
- **Other endpoints**: 200 requests per minute per user

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1706453400
```

---

## Webhooks

Configure webhooks to receive notifications about indexing events.

**Configuration** (in Filestash config):
```json
{
  "ai": {
    "webhooks": {
      "enabled": true,
      "url": "https://your-server.com/webhook",
      "events": ["document.indexed", "document.failed", "batch.completed"]
    }
  }
}
```

**Webhook Payload**:
```json
{
  "event": "document.indexed",
  "timestamp": "2024-01-28T14:35:00Z",
  "data": {
    "document_id": "uuid-1234",
    "file_path": "/reports/Q3-2024.pdf",
    "status": "completed",
    "chunk_count": 47
  }
}
```

---

## SDK Examples

### JavaScript/Node.js

```javascript
const FilestashAI = require('filestash-ai-client');

const client = new FilestashAI({
  baseURL: 'http://localhost:8334',
  sessionCookie: 'your-session-cookie'
});

// Semantic search
const results = await client.search({
  query: 'quarterly financial reports',
  maxResults: 10
});

console.log(`Found ${results.total} results`);
results.results.forEach(r => {
  console.log(`${r.file_name} (similarity: ${r.similarity})`);
});

// RAG chat
const answer = await client.chat({
  query: 'What were the key findings?',
  maxContextChunks: 5
});

console.log(`Answer: ${answer.answer}`);
console.log(`Sources: ${answer.sources.length}`);
```

### Python

```python
from filestash_ai import FilestashAI

client = FilestashAI(
    base_url='http://localhost:8334',
    session_cookie='your-session-cookie'
)

# Semantic search
results = client.search(
    query='quarterly financial reports',
    max_results=10
)

print(f"Found {results['total']} results")
for r in results['results']:
    print(f"{r['file_name']} (similarity: {r['similarity']})")

# RAG chat
answer = client.chat(
    query='What were the key findings?',
    max_context_chunks=5
)

print(f"Answer: {answer['answer']}")
print(f"Sources: {len(answer['sources'])}")
```

### Go

```go
package main

import (
    "fmt"
    "github.com/yourusername/filestash-ai-go"
)

func main() {
    client := filestashai.NewClient(&filestashai.Config{
        BaseURL:       "http://localhost:8334",
        SessionCookie: "your-session-cookie",
    })

    // Semantic search
    results, err := client.Search(&filestashai.SearchRequest{
        Query:      "quarterly financial reports",
        MaxResults: 10,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d results\n", results.Total)
    for _, r := range results.Results {
        fmt.Printf("%s (similarity: %.2f)\n", r.FileName, r.Similarity)
    }

    // RAG chat
    answer, err := client.Chat(&filestashai.ChatRequest{
        Query:            "What were the key findings?",
        MaxContextChunks: 5,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Answer: %s\n", answer.Answer)
    fmt.Printf("Sources: %d\n", len(answer.Sources))
}
```

---

## Performance Optimization

### Caching

The plugin implements caching for:
- **Embeddings**: Cached per file hash (avoid regenerating for unchanged files)
- **Search results**: Cached for 5 minutes per query
- **RAG responses**: Cached for 1 hour per query+context combination

### Batch Operations

For bulk operations, use batch endpoints:

**Batch Index**:
```bash
POST /api/ai/index/batch
{
  "files": [
    {"path": "/file1.pdf", "storage_backend": "s3"},
    {"path": "/file2.pdf", "storage_backend": "s3"}
  ]
}
```

### Pagination

For large result sets, use pagination:

```bash
GET /api/ai/search?query=reports&page=2&per_page=20
```

---

## Best Practices

1. **Query Formulation**: Use natural language, be specific
2. **Context Chunks**: Start with 5, increase if needed
3. **Similarity Threshold**: 0.7 is good default, lower for broader results
4. **Rate Limiting**: Implement client-side throttling
5. **Error Handling**: Always handle 503 errors with retry logic
6. **Caching**: Cache results client-side when appropriate
7. **Streaming**: Use streaming for long-running RAG queries

---

## Migration Guide

### From Previous Versions

If upgrading from a previous version, follow these steps:

1. **Database Migration**: Run migration script
2. **Config Update**: Add new config parameters
3. **API Changes**: Update client code for new response formats
4. **Testing**: Test all endpoints before production deployment

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/mickael-kerjean/filestash/issues
- Documentation: https://filestash.app/docs
- Community: https://discord.gg/filestash
