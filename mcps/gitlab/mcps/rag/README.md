# RAG MCP Server

A comprehensive Model Context Protocol (MCP) server for Retrieval Augmented Generation (RAG), providing document processing, vector database operations, knowledge graph management, and intelligent semantic search.

## Features

### Document Processing Tools
- **scan_folder**: Recursive file discovery with filtering and size limits
- **extract_text**: Multi-format text extraction (PDF, DOCX, XLSX, PPTX, TXT, MD, JSON, YAML)
- **chunk_documents**: Intelligent document chunking with overlap control
- **generate_embeddings**: Vector embeddings using SentenceTransformers
- **detect_duplicates**: Content similarity-based duplicate detection

### Vector Database Tools
- **build_vector_index**: ChromaDB vector index creation with metadata
- **semantic_search**: Similarity search with filtering and ranking
- **update_index**: Incremental index updates with duplicate prevention
- **manage_collections**: Collection management and statistics
- **vector_similarity_analysis**: Advanced similarity analysis

### Knowledge Graph Tools
- **extract_entities**: Named Entity Recognition (NER) using spaCy
- **build_knowledge_graph**: Neo4j knowledge graph construction
- **find_relationships**: Entity relationship discovery and analysis
- **graph_query**: Custom Cypher query execution
- **concept_clustering**: Automatic concept categorization

### RAG Query Interface
- **rag_query**: Unified search interface combining vector and graph search
- **contextual_retrieval**: Context-aware semantic search with query expansion
- **summarize_knowledge**: Topic-based knowledge synthesis
- **explain_concepts**: Detailed concept explanation with multiple depth levels
- **knowledge_gap_identification**: Identify missing information areas

### Resources
- **rag://collections**: Real-time collection information and statistics
- **rag://status**: System status and configuration details

### Prompts
- **document_processing_prompt**: Workflow guidance for document processing
- **semantic_search_prompt**: Search optimization strategies by domain
- **knowledge_graph_analysis_prompt**: Graph analysis methodologies

## Setup

### Prerequisites
- Python 3.8+
- ChromaDB for vector storage
- Neo4j for knowledge graphs (optional)
- spaCy with English language model

### Installation

1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Install spaCy language model:
```bash
python -m spacy download en_core_web_sm
```

3. Set up environment variables:
```bash
# Required
export RAG_CHROMA_DIR="./chroma_db"
export RAG_EMBEDDING_MODEL="all-MiniLM-L6-v2"

# Optional - Neo4j for knowledge graphs
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="your-password"
```

### Configuration Options

#### ChromaDB Configuration
- `RAG_CHROMA_DIR`: Directory for persistent vector storage (default: "./chroma_db")

#### Embedding Model Options
- `all-MiniLM-L6-v2`: Fast, good quality (384 dimensions)
- `all-mpnet-base-v2`: Higher quality (768 dimensions)
- `all-distilroberta-v1`: Balanced performance (768 dimensions)

#### Neo4j Configuration (Optional)
- `NEO4J_URI`: Database connection URI
- `NEO4J_USER`: Database username
- `NEO4J_PASSWORD`: Database password

## Usage

### Running the Server

```bash
python server.py
```

### Testing

Run the comprehensive test suite:

```bash
python test_rag_server.py
```

### Basic Workflow

1. **Document Discovery**
```python
# Scan a folder for documents
result = await client.call_tool("scan_folder", {
    "folder_path": "/path/to/documents",
    "extensions": [".txt", ".pdf", ".docx"],
    "max_file_size_mb": 50
})
```

2. **Text Extraction**
```python
# Extract text from documents
result = await client.call_tool("extract_text", {
    "file_path": "/path/to/document.pdf"
})
```

3. **Document Processing**
```python
# Chunk documents for embedding
result = await client.call_tool("chunk_documents", {
    "documents_json": json.dumps(documents),
    "chunk_size": 1000,
    "chunk_overlap": 200
})
```

4. **Vector Index Creation**
```python
# Build searchable vector index
result = await client.call_tool("build_vector_index", {
    "collection_name": "my_documents",
    "chunks_json": json.dumps(chunks)
})
```

5. **Semantic Search**
```python
# Search for relevant information
result = await client.call_tool("semantic_search", {
    "collection_name": "my_documents",
    "query": "machine learning algorithms",
    "n_results": 10
})
```

6. **Knowledge Graph (Optional)**
```python
# Build knowledge graph
result = await client.call_tool("build_knowledge_graph", {
    "documents_json": json.dumps(documents)
})

# Find entity relationships
result = await client.call_tool("find_relationships", {
    "entity1": "Python",
    "entity2": "machine learning"
})
```

7. **Advanced RAG Queries**
```python
# Unified RAG search
result = await client.call_tool("rag_query", {
    "collection_name": "my_documents",
    "query": "explain neural networks",
    "n_results": 5,
    "include_entities": True
})
```

## Supported File Formats

### Text Files
- `.txt` - Plain text
- `.md` - Markdown
- `.json` - JSON files
- `.yaml/.yml` - YAML files
- `.py/.js` - Source code files

### Document Formats
- `.pdf` - PDF documents (via PyMuPDF)
- `.docx` - Microsoft Word documents
- `.xlsx` - Microsoft Excel spreadsheets
- `.pptx` - Microsoft PowerPoint presentations

## Advanced Features

### Intelligent Chunking
- Recursive character text splitting
- Configurable chunk size and overlap
- Preserves document structure and context
- Metadata preservation across chunks

### Duplicate Detection
- Content-based similarity using embeddings
- Configurable similarity thresholds
- Hash-based file deduplication
- Handles near-duplicates and variations

### Knowledge Graph Integration
- Automatic entity extraction using spaCy NER
- Document-entity relationship mapping
- Entity co-occurrence analysis
- Custom Cypher query support

### Context-Aware Search
- Query expansion using context
- Multi-modal search combining vectors and graphs
- Relevance scoring and ranking
- Metadata filtering and faceted search

## Performance Optimization

### Batch Processing
- Efficient batch embedding generation
- Parallel document processing
- Incremental index updates
- Memory-efficient chunking

### Caching
- Persistent vector storage with ChromaDB
- File hash-based deduplication
- Embedding cache for repeated content
- Graph relationship caching

### Scalability
- Collection-based index organization
- Configurable batch sizes
- Memory usage optimization
- Large file handling with size limits

## Error Handling

### Robust Processing
- Graceful handling of corrupted files
- Encoding detection and fallback
- Partial processing continuation
- Detailed error reporting

### Validation
- Input parameter validation
- File format verification
- Collection existence checking
- Query syntax validation

## Integration Examples

### With Claude Code
```python
# Process a codebase
await client.call_tool("scan_folder", {
    "folder_path": "./src",
    "extensions": [".py", ".js", ".md"],
    "exclude_patterns": ["__pycache__", "node_modules"]
})
```

### With Documentation
```python
# Process documentation
await client.call_tool("rag_query", {
    "collection_name": "docs",
    "query": "API authentication methods",
    "n_results": 5
})
```

### With Research Papers
```python
# Academic research processing
await client.call_tool("contextual_retrieval", {
    "collection_name": "papers",
    "context": "machine learning research",
    "query": "transformer architectures",
    "n_results": 10
})
```

## Troubleshooting

### Common Issues

1. **spaCy Model Not Found**
   - Install: `python -m spacy download en_core_web_sm`
   - Verify: Check model availability in spaCy

2. **ChromaDB Permission Issues**
   - Check directory permissions for `RAG_CHROMA_DIR`
   - Ensure write access to the database directory

3. **Neo4j Connection Failed**
   - Verify Neo4j is running and accessible
   - Check connection URI and credentials
   - Ensure proper network configuration

4. **Large File Processing**
   - Adjust `max_file_size_mb` parameter
   - Consider chunking large documents manually
   - Monitor memory usage during processing

5. **Embedding Generation Slow**
   - Use faster embedding models (MiniLM vs MPNet)
   - Reduce batch sizes for memory-constrained systems
   - Consider GPU acceleration for large datasets

### Debug Mode

Enable detailed logging:

```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

### Performance Monitoring

Monitor key metrics:
- Document processing speed
- Embedding generation time
- Search query latency
- Memory usage patterns
- Index size growth

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run the test suite: `python test_rag_server.py`
5. Submit a pull request

## License

MIT License - see LICENSE file for details