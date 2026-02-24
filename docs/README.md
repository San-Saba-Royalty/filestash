# AI Integration Documentation

Complete documentation for adding AI-powered semantic search and RAG capabilities to Filestash using Supabase PostgreSQL (pgvector), Azure OpenAI, and self-hosted Unstructured.io.

## 📚 Documentation Index

### Quick Start
- **[README.md](./README.md)** - This file
- **[QUICK_START.md](./QUICK_START.md)** - Get up and running in 30 minutes

### Architecture & Planning
- **[AI_INTEGRATION_OVERVIEW.md](./AI_INTEGRATION_OVERVIEW.md)** - High-level architecture and design decisions
- **[DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md)** - Supabase PostgreSQL schema with pgvector
- **[PLUGIN_ARCHITECTURE.md](./PLUGIN_ARCHITECTURE.md)** - Plugin structure and code organization

### Implementation
- **[IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)** - Step-by-step implementation guide
- **[UNSTRUCTURED_INTEGRATION.md](./UNSTRUCTURED_INTEGRATION.md)** - Self-hosted Unstructured.io setup
- **[API_DOCUMENTATION.md](./API_DOCUMENTATION.md)** - REST API specifications

## 🎯 What Gets Built

This integration adds the following capabilities to Filestash:

### 1. Semantic Search
- Natural language queries across all files
- Search by meaning, not just keywords
- "Find documents about quarterly financial performance" → Finds relevant docs even without exact keywords
- Works across all storage backends (S3, Azure Blob, IPFS, local, etc.)

### 2. RAG (Retrieval Augmented Generation)
- Ask questions about your documents
- "What were the key findings in Q3?" → AI answers based on actual document contents
- Context-aware responses with source citations
- Multi-document synthesis

### 3. Automatic Indexing
- Files automatically processed on upload
- Background worker for batch processing
- Incremental updates for modified files
- Support for 20+ file formats

## 🏗️ Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Vector Database | **Supabase PostgreSQL + pgvector** | Store embeddings & metadata |
| Embeddings | **Azure OpenAI text-embedding-3-large** | Convert text to vectors |
| LLM | **Azure OpenAI GPT-4** | Generate RAG responses |
| Document Processing | **Unstructured.io (self-hosted)** | Extract text from any file format |
| Backend | **Go** | Filestash plugin |
| Frontend | **JavaScript** | Search UI components |

## 📋 Prerequisites

Before you begin, ensure you have:

- ✅ Filestash installed and running
- ✅ Go 1.21+ installed
- ✅ Docker & Docker Compose installed
- ✅ Supabase account (free tier works)
- ✅ Azure OpenAI resource provisioned
- ✅ Basic knowledge of Go and PostgreSQL

## 🚀 Quick Start (30 Minutes)

### Step 1: Set Up Supabase (5 minutes)

```bash
# 1. Create Supabase project at https://supabase.com
# 2. Copy connection details from Settings → Database
# 3. Run the database schema in SQL Editor

# Open Supabase SQL Editor and run:
cat docs/DATABASE_SCHEMA.md
# Copy the complete migration script and execute
```

### Step 2: Set Up Azure OpenAI (5 minutes)

```bash
# 1. Create Azure OpenAI resource in Azure Portal
# 2. Deploy models:
#    - text-embedding-3-large
#    - gpt-4o
# 3. Copy endpoint and API key from Keys and Endpoint
```

### Step 3: Start Unstructured.io (5 minutes)

```bash
# Navigate to your Filestash directory
cd /Users/gqadonis/Projects/sansaba/filestash

# Create docker-compose.ai.yml (see UNSTRUCTURED_INTEGRATION.md)
# Then start:
docker-compose -f docker-compose.ai.yml up -d

# Verify it's running:
curl http://localhost:8000/healthcheck
```

### Step 4: Create Plugin Directory (2 minutes)

```bash
# Create plugin structure
mkdir -p server/plugin/plg_ai_semantic
cd server/plugin/plg_ai_semantic

# Copy plugin files from PLUGIN_ARCHITECTURE.md
# (See implementation guide for complete code)
```

### Step 5: Configure Filestash (3 minutes)

Add to your Filestash configuration:

```json
{
  "ai": {
    "enabled": true,
    "supabase": {
      "project_id": "your-project-id",
      "password": "your-db-password"
    },
    "azure_openai": {
      "endpoint": "https://your-resource.openai.azure.com/",
      "api_key": "your-api-key",
      "api_version": "2024-02-01",
      "embeddings_deployment": "text-embedding-3-large",
      "chat_deployment": "gpt-4o"
    },
    "unstructured": {
      "url": "http://localhost:8000"
    }
  }
}
```

### Step 6: Build & Run (10 minutes)

```bash
# Add plugin to index
echo '_ "github.com/mickael-kerjean/filestash/server/plugin/plg_ai_semantic"' \
  >> server/plugin/index.go

# Build
make build_backend

# Run Filestash
./filestash
```

### Step 7: Test It! (5 minutes)

```bash
# Upload a test document through Filestash UI

# Wait a few seconds for processing, then:
curl -X POST http://localhost:8334/api/ai/search \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{"query": "test document", "max_results": 5}'

# Try RAG:
curl -X POST http://localhost:8334/api/ai/chat \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=your-session-cookie" \
  -d '{"query": "What is this document about?"}'
```

## 📖 Detailed Documentation

### For Implementation
1. Start with **[IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)**
2. Set up database using **[DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md)**
3. Review code structure in **[PLUGIN_ARCHITECTURE.md](./PLUGIN_ARCHITECTURE.md)**
4. Deploy Unstructured.io per **[UNSTRUCTURED_INTEGRATION.md](./UNSTRUCTURED_INTEGRATION.md)**

### For API Integration
1. Review **[API_DOCUMENTATION.md](./API_DOCUMENTATION.md)** for endpoint specs
2. Use provided cURL examples to test
3. Build client applications using SDKs

### For Understanding
1. Read **[AI_INTEGRATION_OVERVIEW.md](./AI_INTEGRATION_OVERVIEW.md)** for architecture
2. Understand data flow and design decisions
3. Review security and performance considerations

## 💡 Key Features

### Multi-Storage Backend Support
Works seamlessly with all Filestash storage backends:
- ✅ AWS S3
- ✅ Azure Blob Storage
- ✅ IPFS
- ✅ Local Filesystem
- ✅ SFTP
- ✅ Google Drive
- ✅ Dropbox
- ✅ And 20+ more

### Privacy & Security
- 🔒 All data stays in your infrastructure
- 🔒 Self-hosted Unstructured.io (no external API calls)
- 🔒 Your own Supabase instance
- 🔒 Row-level security for user isolation
- 🔒 Respects existing Filestash permissions

### Performance
- ⚡ Vector search in <100ms (pgvector optimized)
- ⚡ Real-time indexing on file upload
- ⚡ Batch processing for existing files
- ⚡ Caching for frequently accessed data

## 📊 Architecture Overview

```
┌────────────────────────────────────────────────────────┐
│                   Filestash UI                          │
│      File Browser | AI Search | RAG Chat               │
└───────────────────────┬────────────────────────────────┘
                        │
                        ▼
┌────────────────────────────────────────────────────────┐
│              Filestash Backend (Go)                     │
│  ┌──────────────────────────────────────────────────┐  │
│  │        plg_ai_semantic Plugin                    │  │
│  │  • Document Processor                            │  │
│  │  • Indexing Worker                               │  │
│  │  • Search Handler                                │  │
│  │  • RAG Handler                                   │  │
│  └──────────────────────────────────────────────────┘  │
└────┬─────────────────────┬──────────────────┬─────────┘
     │                     │                  │
     ▼                     ▼                  ▼
┌─────────┐      ┌──────────────┐    ┌──────────────┐
│Supabase │      │Azure OpenAI  │    │Unstructured  │
│pgvector │      │• Embeddings  │    │(Self-hosted) │
│Database │      │• GPT-4       │    │Docker        │
└─────────┘      └──────────────┘    └──────────────┘
```

## 📈 Scalability

### Small Deployment (< 10,000 files)
- **Supabase**: Free tier
- **Azure OpenAI**: Pay-as-you-go
- **Unstructured**: Single Docker container (4GB RAM)
- **Cost**: ~$50/month

### Medium Deployment (10,000 - 100,000 files)
- **Supabase**: Pro tier ($25/month)
- **Azure OpenAI**: Reserved capacity
- **Unstructured**: Multiple containers + load balancer
- **Cost**: ~$200-500/month

### Large Deployment (> 100,000 files)
- **Supabase**: Team/Enterprise tier
- **Azure OpenAI**: Provisioned throughput
- **Unstructured**: Kubernetes cluster
- **Cost**: Custom pricing

## 🔧 Maintenance

### Daily Tasks
- Monitor indexing queue
- Check error logs
- Review search analytics

### Weekly Tasks
- Vacuum Supabase database
- Review Azure OpenAI usage
- Clean up failed jobs

### Monthly Tasks
- Reindex outdated documents
- Update Unstructured.io image
- Review and optimize performance

## 🐛 Troubleshooting

### Common Issues

**Problem**: Supabase connection fails
```bash
# Check credentials
psql "postgresql://postgres:password@db.project.supabase.co:5432/postgres"
```

**Problem**: Azure OpenAI rate limits
```bash
# Check quota in Azure Portal
# Increase tokens per minute or add retry logic
```

**Problem**: Unstructured.io slow
```bash
# Check container resources
docker stats unstructured-api

# Increase CPU/memory in docker-compose.yml
```

**Problem**: No search results
```bash
# Check if documents are indexed
curl -X GET http://localhost:8334/api/ai/index/status \
  -H "Cookie: auth=your-session"

# Verify embeddings in Supabase
SELECT COUNT(*) FROM ai_embeddings;
```

## 📞 Support & Resources

### Documentation
- This documentation set
- [Filestash Docs](https://www.filestash.app/docs/)
- [Supabase Docs](https://supabase.com/docs)
- [Azure OpenAI Docs](https://learn.microsoft.com/en-us/azure/ai-services/openai/)
- [Unstructured.io Docs](https://docs.unstructured.io/)

### Community
- [Filestash GitHub](https://github.com/mickael-kerjean/filestash)
- [Filestash Discord](https://discord.gg/filestash)

## 🎓 Learning Path

### Beginner
1. Read [AI_INTEGRATION_OVERVIEW.md](./AI_INTEGRATION_OVERVIEW.md)
2. Follow [QUICK_START.md](./QUICK_START.md)
3. Deploy test environment
4. Upload and search test documents

### Intermediate
1. Review [PLUGIN_ARCHITECTURE.md](./PLUGIN_ARCHITECTURE.md)
2. Understand database schema
3. Customize processing options
4. Build client applications

### Advanced
1. Optimize vector search performance
2. Implement custom chunking strategies
3. Add multi-language support
4. Scale to production workload

## 📝 License

This documentation is provided as-is for use with Filestash. 

Individual components:
- **Filestash**: AGPLv3
- **Supabase**: Apache 2.0
- **Azure OpenAI**: Microsoft Terms
- **Unstructured.io**: Apache 2.0

## 🙏 Acknowledgments

- Filestash team for the excellent file management platform
- Supabase for making PostgreSQL vector search accessible
- Azure OpenAI for powerful AI capabilities
- Unstructured.io for document processing tools

## 📅 Version History

- **v1.0.0** (2024-01-29): Initial documentation
  - Complete architecture design
  - Database schema
  - Plugin implementation guide
  - API documentation
  - Deployment guides

## 🗺️ Roadmap

### Phase 1: MVP (Weeks 1-6)
- [x] Documentation complete
- [ ] Core plugin implementation
- [ ] Basic semantic search
- [ ] Simple RAG queries
- [ ] Automatic indexing

### Phase 2: Enhancement (Weeks 7-10)
- [ ] Frontend UI components
- [ ] Advanced filtering
- [ ] Multi-turn conversations
- [ ] Performance optimizations
- [ ] Production deployment

### Phase 3: Scale (Weeks 11-12)
- [ ] Load testing
- [ ] Monitoring & alerting
- [ ] Multi-language support
- [ ] Advanced analytics
- [ ] Documentation improvements

## 🚦 Status

**Current Status**: Documentation Complete ✅

**Next Steps**:
1. Review all documentation files
2. Set up development environment
3. Begin Phase 1 implementation
4. Test with sample documents

---

**Ready to get started?** Head over to **[IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)** for step-by-step instructions!
