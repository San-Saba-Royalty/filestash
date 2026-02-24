# AI Integration Overview for Filestash

## Executive Summary

This document outlines the architecture for integrating AI-powered semantic search and RAG (Retrieval Augmented Generation) capabilities directly into the Filestash codebase. The integration will:

- Use **Supabase PostgreSQL with pgvector** for vector storage (instead of Pinecone)
- Use **Azure OpenAI** for embeddings and LLM completions
- Use **self-hosted Unstructured.io** for document processing
- Integrate seamlessly with existing Filestash plugin architecture
- Support all storage backends (S3, Azure Blob, IPFS, SFTP, etc.)

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Filestash Frontend UI                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ File Browser │  │ AI Search UI │  │  Chat Panel  │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Filestash Backend (Go)                          │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              NEW: plg_ai_semantic Plugin                  │  │
│  │                                                            │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │  │
│  │  │ AI Search    │  │  RAG Query   │  │ File Indexer │   │  │
│  │  │ API Handler  │  │   Handler    │  │   Worker     │   │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │  │
│  │                                                            │  │
│  │  ┌──────────────────────────────────────────────────┐    │  │
│  │  │        Document Processing Pipeline              │    │  │
│  │  │  1. File Watch → 2. Extract → 3. Chunk →        │    │  │
│  │  │  4. Embed → 5. Store in Supabase                │    │  │
│  │  └──────────────────────────────────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         Existing Filestash Components                     │  │
│  │   Storage Backends | Auth | Session | Middleware          │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────┬─────────────────────────────────────┬─────────────────┘
         │                                     │
         ▼                                     ▼
┌──────────────────────┐            ┌──────────────────────┐
│  Storage Backends    │            │   External Services  │
│  • S3                │            │                      │
│  • Azure Blob        │            │  ┌────────────────┐  │
│  • IPFS              │            │  │   Supabase     │  │
│  • Local FS          │            │  │  PostgreSQL +  │  │
│  • SFTP              │            │  │   pgvector     │  │
│  • etc.              │            │  └────────────────┘  │
└──────────────────────┘            │                      │
                                    │  ┌────────────────┐  │
                                    │  │  Azure OpenAI  │  │
                                    │  │  • Embeddings  │  │
                                    │  │  • Chat GPT-4  │  │
                                    │  └────────────────┘  │
                                    │                      │
                                    │  ┌────────────────┐  │
                                    │  │ Unstructured   │  │
                                    │  │ (Self-hosted)  │  │
                                    │  │ Docker Service │  │
                                    │  └────────────────┘  │
                                    └──────────────────────┘
```

## Core Components

### 1. Plugin: `plg_ai_semantic`

A new Filestash plugin that provides:

- **File Indexing Worker**: Background process that monitors file changes
- **Document Processor**: Integrates with self-hosted Unstructured.io
- **Embedding Generator**: Uses Azure OpenAI to create vector embeddings
- **Vector Store Client**: Manages Supabase pgvector operations
- **Search API Handler**: Provides semantic search endpoints
- **RAG Query Handler**: Generates AI responses based on file contents

### 2. Supabase PostgreSQL with pgvector

Vector database for storing:
- File embeddings (1536-dimensional vectors from Azure OpenAI)
- File metadata (path, name, type, storage backend, etc.)
- Chunk information (for long documents)
- User permissions (for access control)

### 3. Azure OpenAI Integration

- **text-embedding-3-large**: For generating embeddings (3072 dimensions, can be reduced to 1536)
- **gpt-4o** or **gpt-4**: For RAG completions
- **Deployment Names**: Configured via environment variables

### 4. Self-Hosted Unstructured.io

Docker container providing:
- Document parsing (PDF, DOCX, PPT, images, etc.)
- Text extraction with layout preservation
- Table and image processing
- OCR capabilities for scanned documents

## Key Features

### Semantic Search
- Natural language queries: "Find documents about quarterly financial reports"
- Search across all storage backends
- Results ranked by relevance
- Respects user permissions

### RAG (Retrieval Augmented Generation)
- Ask questions about your files: "What were the key findings in Q3?"
- Context-aware responses based on actual file contents
- Citations with direct links to source files
- Multi-file synthesis

### Automatic Indexing
- Files automatically processed on upload
- Background re-indexing for modified files
- Configurable batch processing for existing files
- Incremental updates (only new/changed files)

### Multi-Backend Support
- Works with ALL Filestash storage backends
- Unified search across multiple storage systems
- Storage-agnostic vector embeddings

## Integration Points with Existing Filestash

### 1. Plugin Registration
Registers via standard Filestash plugin hooks:
- `HttpEndpoint`: New API routes for AI features
- `SearchEngine`: Enhanced search implementation
- `Onload`: Initialize background workers
- `Middleware`: File upload/modify event hooks

### 2. File Operations Hooks
Intercepts file operations:
- **Upload**: Trigger document processing
- **Save/Update**: Re-index modified files
- **Delete**: Remove from vector store
- **Move/Rename**: Update metadata

### 3. Session & Authentication
Uses existing Filestash auth:
- Leverages `App.Session` for user context
- Respects storage backend permissions
- Per-user vector store isolation

### 4. Storage Backend Abstraction
Works through `App.Backend` interface:
- No modifications to storage backends needed
- Files accessed via standard `Cat()`, `Ls()` methods
- Storage-agnostic implementation

## Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | Go | Filestash plugin implementation |
| Vector DB | Supabase PostgreSQL + pgvector | Embedding storage & semantic search |
| Embeddings | Azure OpenAI text-embedding-3-large | Vector generation |
| LLM | Azure OpenAI GPT-4 | RAG completions |
| Doc Processing | Unstructured.io (self-hosted) | Extract text from any file format |
| HTTP Client | Go stdlib `net/http` | API communications |
| Database Client | `pgx` Go library | PostgreSQL operations |

## Security Considerations

### 1. Data Isolation
- Vector embeddings tagged with user/session IDs
- RLS (Row Level Security) in Supabase
- Only search within user's accessible files

### 2. API Key Management
- Azure OpenAI keys stored in Filestash config
- Supabase credentials in environment variables
- Never exposed to frontend

### 3. Content Privacy
- Option to process files locally (no external API calls for sensitive data)
- Embeddings stored in your own Supabase instance
- Self-hosted Unstructured.io (no data sent to external services)

## Performance Characteristics

### Initial Indexing
- **Small Library** (< 1,000 files): 10-30 minutes
- **Medium Library** (1,000-10,000 files): 1-4 hours
- **Large Library** (> 10,000 files): 4-24 hours

### Query Performance
- **Semantic Search**: < 100ms (pgvector optimized)
- **RAG Response**: 2-5 seconds (depends on GPT-4 latency)
- **Incremental Index**: Real-time on file upload

### Resource Requirements
- **CPU**: 2-4 cores recommended
- **RAM**: 4-8GB minimum
- **Supabase**: Free tier supports up to 500MB vectors
- **Azure OpenAI**: Pay-per-token pricing

## Scalability

### Horizontal Scaling
- Multiple Filestash instances can share same Supabase DB
- Worker processes can be distributed
- Stateless API handlers

### Vertical Scaling
- pgvector scales to millions of vectors
- HNSW indexing for sub-linear search performance
- Configurable embedding dimensions (trade-off: accuracy vs. storage)

## Configuration Model

All AI features configured via Filestash's existing config system:

```json
{
  "ai": {
    "enabled": true,
    "supabase": {
      "url": "https://your-project.supabase.co",
      "api_key": "your-service-role-key"
    },
    "azure_openai": {
      "endpoint": "https://your-resource.openai.azure.com",
      "api_key": "your-azure-key",
      "api_version": "2024-02-01",
      "embeddings_deployment": "text-embedding-3-large",
      "chat_deployment": "gpt-4o"
    },
    "unstructured": {
      "url": "http://localhost:8000",
      "api_key": "optional-if-secured"
    },
    "indexing": {
      "auto_index_on_upload": true,
      "batch_size": 10,
      "max_file_size_mb": 100
    }
  }
}
```

## Next Steps

1. Review [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) for Supabase setup
2. Review [PLUGIN_ARCHITECTURE.md](./PLUGIN_ARCHITECTURE.md) for code structure
3. Review [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) for step-by-step instructions
4. Review [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for endpoint specifications

## Timeline Estimate

- **Week 1**: Database schema + basic plugin structure
- **Week 2**: Unstructured.io integration + embedding generation
- **Week 3**: Vector storage + semantic search API
- **Week 4**: RAG query handler + frontend UI
- **Week 5**: Testing, optimization, documentation
- **Week 6**: Production deployment + monitoring

**Total: 6 weeks for production-ready implementation**
