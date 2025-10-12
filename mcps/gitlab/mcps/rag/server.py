"""
RAG MCP Server for document processing, vector search, and knowledge graph operations.

This server provides comprehensive RAG (Retrieval Augmented Generation) capabilities including:
- Document processing and text extraction from multiple formats
- Vector database operations with ChromaDB
- Knowledge graph management with Neo4j
- Semantic search and contextual retrieval
- Entity extraction and relationship discovery
- Intelligent document chunking and duplicate detection
"""

import os
import sys
import json
import logging
import asyncio
from typing import Any, Dict, List, Optional
from datetime import datetime

# Add the project root to the Python path
project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
sys.path.insert(0, project_root)

from src.infrastructure.mcp.server_base import MCPServerBase, mcp_tool, mcp_resource, mcp_prompt
from rag_client import RAGClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class RAGMCPServer(MCPServerBase):
    """MCP Server for RAG operations and knowledge management."""

    def __init__(self):
        super().__init__("rag-mcp-server", "1.0.0")
        self.client: Optional[RAGClient] = None

    async def setup(self) -> None:
        """Initialize RAG client and validate connections."""
        try:
            # Get configuration from environment variables
            chroma_dir = os.getenv("RAG_CHROMA_DIR", "./chroma_db")
            embedding_model = os.getenv("RAG_EMBEDDING_MODEL", "all-MiniLM-L6-v2")
            neo4j_uri = os.getenv("NEO4J_URI")
            neo4j_user = os.getenv("NEO4J_USER")
            neo4j_password = os.getenv("NEO4J_PASSWORD")

            self.client = RAGClient(
                chroma_persist_directory=chroma_dir,
                embedding_model=embedding_model,
                neo4j_uri=neo4j_uri,
                neo4j_user=neo4j_user,
                neo4j_password=neo4j_password
            )

            logger.info(f"RAG MCP Server initialized with embedding model: {embedding_model}")

        except Exception as e:
            logger.error(f"Failed to initialize RAG client: {e}")
            raise

    # Document Processing Tools

    @mcp_tool
    async def scan_folder(self, folder_path: str,
                         extensions: Optional[List[str]] = None,
                         exclude_patterns: Optional[List[str]] = None,
                         max_file_size_mb: int = 50) -> str:
        """Recursively scan folder for supported documents.

        Args:
            folder_path: Path to folder to scan
            extensions: List of file extensions to include (e.g., ['.txt', '.pdf'])
            exclude_patterns: List of regex patterns to exclude files/folders
            max_file_size_mb: Maximum file size in MB (default: 50)
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            files = self.client.scan_folder(
                folder_path=folder_path,
                extensions=extensions,
                exclude_patterns=exclude_patterns,
                max_file_size_mb=max_file_size_mb
            )

            result = f"Folder Scan Results: {folder_path}\n"
            result += "=" * 60 + "\n\n"

            result += f"Total Files Found: {len(files)}\n\n"

            # Group by extension
            by_extension = {}
            total_size = 0

            for file_info in files:
                ext = file_info['extension']
                if ext not in by_extension:
                    by_extension[ext] = []
                by_extension[ext].append(file_info)
                total_size += file_info['size_bytes']

            result += "Files by Extension:\n"
            for ext, file_list in sorted(by_extension.items()):
                size_mb = sum(f['size_bytes'] for f in file_list) / (1024 * 1024)
                result += f"  {ext}: {len(file_list)} files ({size_mb:.2f} MB)\n"

            result += f"\nTotal Size: {total_size / (1024 * 1024):.2f} MB\n\n"

            # Show recent files
            if files:
                result += "Most Recently Modified Files:\n"
                sorted_files = sorted(files, key=lambda x: x['modified_time'], reverse=True)
                for file_info in sorted_files[:10]:
                    result += f"  • {file_info['relative_path']}\n"
                    result += f"    Size: {file_info['size_bytes'] / 1024:.1f} KB, "
                    result += f"Modified: {file_info['modified_time'].strftime('%Y-%m-%d %H:%M')}\n"

            return result

        except Exception as e:
            logger.error(f"Error scanning folder: {e}")
            return f"Error scanning folder: {str(e)}"

    @mcp_tool
    async def extract_text(self, file_path: str) -> str:
        """Extract text content from a file.

        Args:
            file_path: Path to the file to process
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            extracted = self.client.extract_text(file_path)

            result = f"Text Extraction Results: {extracted['file_name']}\n"
            result += "=" * 60 + "\n\n"

            result += f"File Path: {extracted['file_path']}\n"
            result += f"File Extension: {extracted['file_extension']}\n"
            result += f"File Size: {extracted['file_size'] / 1024:.1f} KB\n"
            result += f"Text Length: {extracted['text_length']} characters\n"
            result += f"File Hash: {extracted['file_hash'][:16]}...\n"
            result += f"Extraction Time: {extracted['extraction_time'].strftime('%Y-%m-%d %H:%M:%S')}\n\n"

            # Show text preview
            text_preview = extracted['text_content'][:1000]
            if len(extracted['text_content']) > 1000:
                text_preview += "...\n\n[Text truncated - showing first 1000 characters]"

            result += "Text Content Preview:\n"
            result += "-" * 30 + "\n"
            result += text_preview

            return result

        except Exception as e:
            logger.error(f"Error extracting text: {e}")
            return f"Error extracting text: {str(e)}"

    @mcp_tool
    async def chunk_documents(self, documents_json: str,
                            chunk_size: int = 1000,
                            chunk_overlap: int = 200) -> str:
        """Split documents into chunks for embedding.

        Args:
            documents_json: JSON string containing list of document dictionaries
            chunk_size: Target size for each chunk (default: 1000)
            chunk_overlap: Overlap between chunks (default: 200)
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            documents = json.loads(documents_json)
            chunks = self.client.chunk_documents(
                documents=documents,
                chunk_size=chunk_size,
                chunk_overlap=chunk_overlap
            )

            result = f"Document Chunking Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Input Documents: {len(documents)}\n"
            result += f"Generated Chunks: {len(chunks)}\n"
            result += f"Chunk Size: {chunk_size} characters\n"
            result += f"Chunk Overlap: {chunk_overlap} characters\n\n"

            # Show chunk distribution by source
            by_source = {}
            total_chunk_chars = 0

            for chunk in chunks:
                source = chunk['source_file_name']
                if source not in by_source:
                    by_source[source] = []
                by_source[source].append(chunk)
                total_chunk_chars += chunk['chunk_length']

            result += "Chunks by Source File:\n"
            for source, chunk_list in by_source.items():
                avg_chunk_size = sum(c['chunk_length'] for c in chunk_list) / len(chunk_list)
                result += f"  • {source}: {len(chunk_list)} chunks (avg {avg_chunk_size:.0f} chars)\n"

            result += f"\nTotal Characters in Chunks: {total_chunk_chars:,}\n"
            result += f"Average Chunk Size: {total_chunk_chars / len(chunks):.0f} characters\n"

            # Show sample chunks
            if chunks:
                result += "\nSample Chunks:\n"
                for i, chunk in enumerate(chunks[:3]):
                    result += f"\nChunk {i+1} (from {chunk['source_file_name']}):\n"
                    result += f"  ID: {chunk['chunk_id']}\n"
                    result += f"  Length: {chunk['chunk_length']} chars\n"
                    preview = chunk['chunk_text'][:200]
                    if len(chunk['chunk_text']) > 200:
                        preview += "..."
                    result += f"  Preview: {preview}\n"

            return result

        except Exception as e:
            logger.error(f"Error chunking documents: {e}")
            return f"Error chunking documents: {str(e)}"

    @mcp_tool
    async def generate_embeddings(self, texts_json: str) -> str:
        """Generate embeddings for text chunks.

        Args:
            texts_json: JSON string containing list of text strings
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            texts = json.loads(texts_json)
            embeddings = self.client.generate_embeddings(texts)

            result = f"Embedding Generation Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Input Texts: {len(texts)}\n"
            result += f"Generated Embeddings: {len(embeddings)}\n"

            if embeddings:
                result += f"Embedding Dimension: {len(embeddings[0])}\n"
                result += f"Model: {self.client.embedding_model_name}\n\n"

                # Show statistics
                total_chars = sum(len(text) for text in texts)
                result += f"Total Characters Processed: {total_chars:,}\n"
                result += f"Average Text Length: {total_chars / len(texts):.0f} characters\n\n"

                # Show sample embedding info
                result += "Sample Embeddings:\n"
                for i, (text, embedding) in enumerate(zip(texts[:3], embeddings[:3])):
                    text_preview = text[:100] + ("..." if len(text) > 100 else "")
                    result += f"\nText {i+1}: {text_preview}\n"
                    result += f"  Embedding: [{embedding[0]:.4f}, {embedding[1]:.4f}, ..., {embedding[-1]:.4f}]\n"
                    result += f"  Vector Length: {len(embedding)}\n"

            return result

        except Exception as e:
            logger.error(f"Error generating embeddings: {e}")
            return f"Error generating embeddings: {str(e)}"

    @mcp_tool
    async def detect_duplicates(self, documents_json: str,
                              similarity_threshold: float = 0.95) -> str:
        """Detect duplicate documents based on content similarity.

        Args:
            documents_json: JSON string containing list of document dictionaries
            similarity_threshold: Similarity threshold for duplicate detection (0.0-1.0)
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            documents = json.loads(documents_json)
            duplicates = self.client.detect_duplicates(
                documents=documents,
                similarity_threshold=similarity_threshold
            )

            result = f"Duplicate Detection Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Documents Analyzed: {len(documents)}\n"
            result += f"Similarity Threshold: {similarity_threshold}\n"
            result += f"Duplicate Groups Found: {len(duplicates)}\n\n"

            if duplicates:
                total_duplicates = sum(len(group) for group in duplicates)
                result += f"Total Documents in Duplicates: {total_duplicates}\n"
                result += f"Unique Documents After Deduplication: {len(documents) - total_duplicates + len(duplicates)}\n\n"

                result += "Duplicate Groups:\n"
                for i, group in enumerate(duplicates):
                    result += f"\nGroup {i+1} ({len(group)} documents):\n"
                    for doc_idx in group:
                        doc = documents[doc_idx]
                        result += f"  • {doc.get('file_name', f'Document {doc_idx}')}\n"
                        result += f"    Path: {doc.get('file_path', 'Unknown')}\n"
                        result += f"    Size: {doc.get('file_size', 0) / 1024:.1f} KB\n"
                        if doc.get('modified_time'):
                            result += f"    Modified: {doc['modified_time']}\n"

            else:
                result += "No duplicates found with the specified threshold.\n"

            return result

        except Exception as e:
            logger.error(f"Error detecting duplicates: {e}")
            return f"Error detecting duplicates: {str(e)}"

    # Vector Database Tools

    @mcp_tool
    async def build_vector_index(self, collection_name: str,
                               chunks_json: str,
                               metadata_fields: Optional[List[str]] = None) -> str:
        """Build vector index from document chunks.

        Args:
            collection_name: Name for the vector collection
            chunks_json: JSON string containing list of chunk dictionaries
            metadata_fields: List of metadata fields to include in index
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            chunks = json.loads(chunks_json)
            collection_name = self.client.build_vector_index(
                collection_name=collection_name,
                chunks=chunks,
                metadata_fields=metadata_fields
            )

            result = f"Vector Index Creation Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Collection Name: {collection_name}\n"
            result += f"Chunks Indexed: {len(chunks)}\n"
            result += f"Metadata Fields: {metadata_fields or 'default'}\n"
            result += f"Embedding Model: {self.client.embedding_model_name}\n"
            result += f"Created: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"

            # Show indexing statistics
            total_chars = sum(chunk['chunk_length'] for chunk in chunks)
            unique_sources = len(set(chunk['source_file'] for chunk in chunks))

            result += "Indexing Statistics:\n"
            result += f"  • Total Characters: {total_chars:,}\n"
            result += f"  • Average Chunk Size: {total_chars / len(chunks):.0f} characters\n"
            result += f"  • Unique Source Files: {unique_sources}\n"
            result += f"  • Vector Dimension: 384 (MiniLM-L6-v2)\n\n"

            result += f"Vector index '{collection_name}' created successfully!\n"
            result += "You can now use semantic_search to query this collection.\n"

            return result

        except Exception as e:
            logger.error(f"Error building vector index: {e}")
            return f"Error building vector index: {str(e)}"

    @mcp_tool
    async def semantic_search(self, collection_name: str,
                            query: str,
                            n_results: int = 10,
                            filter_metadata_json: Optional[str] = None) -> str:
        """Perform semantic search on vector collection.

        Args:
            collection_name: Name of the collection to search
            query: Search query text
            n_results: Number of results to return (default: 10)
            filter_metadata_json: Optional JSON string with metadata filters
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            filter_metadata = None
            if filter_metadata_json:
                filter_metadata = json.loads(filter_metadata_json)

            results = self.client.semantic_search(
                collection_name=collection_name,
                query=query,
                n_results=n_results,
                filter_metadata=filter_metadata
            )

            result = f"Semantic Search Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Collection: {collection_name}\n"
            result += f"Query: \"{query}\"\n"
            result += f"Results Found: {len(results)}\n"
            if filter_metadata:
                result += f"Filters Applied: {filter_metadata}\n"
            result += "\n"

            if results:
                result += "Search Results (ordered by relevance):\n\n"

                for i, search_result in enumerate(results):
                    result += f"Result #{i+1} (Score: {search_result['score']:.4f})\n"
                    result += f"Source: {search_result['metadata'].get('source_file_name', 'Unknown')}\n"
                    result += f"Chunk ID: {search_result['id']}\n"

                    # Show text preview
                    text_preview = search_result['text'][:300]
                    if len(search_result['text']) > 300:
                        text_preview += "..."
                    result += f"Text: {text_preview}\n"

                    # Show metadata
                    metadata = search_result['metadata']
                    if metadata:
                        result += "Metadata: "
                        metadata_items = []
                        for key, value in metadata.items():
                            if key != 'source_file_name':  # Already shown above
                                metadata_items.append(f"{key}={value}")
                        result += ", ".join(metadata_items) + "\n"

                    result += "\n" + "-" * 40 + "\n\n"

            else:
                result += "No results found for the given query.\n"
                result += "\nTips:\n"
                result += "- Try using different keywords\n"
                result += "- Check if the collection exists and has data\n"
                result += "- Consider using broader search terms\n"

            return result

        except Exception as e:
            logger.error(f"Error performing semantic search: {e}")
            return f"Error performing semantic search: {str(e)}"

    @mcp_tool
    async def update_index(self, collection_name: str, chunks_json: str) -> str:
        """Update vector index with new chunks.

        Args:
            collection_name: Name of the collection to update
            chunks_json: JSON string containing list of new chunk dictionaries
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            chunks = json.loads(chunks_json)
            added_count = self.client.update_index(
                collection_name=collection_name,
                chunks=chunks
            )

            result = f"Vector Index Update Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Collection: {collection_name}\n"
            result += f"Chunks Provided: {len(chunks)}\n"
            result += f"New Chunks Added: {added_count}\n"
            result += f"Duplicates Skipped: {len(chunks) - added_count}\n"
            result += f"Updated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"

            if added_count > 0:
                result += f"Successfully added {added_count} new chunks to the index.\n"
            else:
                result += "No new chunks were added - all provided chunks already exist in the index.\n"

            return result

        except Exception as e:
            logger.error(f"Error updating index: {e}")
            return f"Error updating index: {str(e)}"

    @mcp_tool
    async def manage_collections(self) -> str:
        """Get information about all vector collections."""
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            collections_info = self.client.manage_collections()

            result = f"Vector Collections Management\n"
            result += "=" * 60 + "\n\n"

            if not collections_info:
                result += "No collections found.\n"
                result += "\nTo create a collection, use the build_vector_index tool.\n"
                return result

            result += f"Total Collections: {len(collections_info)}\n\n"

            total_documents = 0
            for collection_name, info in collections_info.items():
                result += f"Collection: {collection_name}\n"
                result += f"  Document Count: {info.get('count', 0)}\n"

                metadata = info.get('metadata', {})
                if metadata:
                    result += f"  Metadata: {metadata}\n"

                if 'error' in info:
                    result += f"  Error: {info['error']}\n"

                result += "\n"
                total_documents += info.get('count', 0)

            result += f"Total Documents Across All Collections: {total_documents}\n"

            if total_documents > 0:
                result += "\nYou can use semantic_search to query any of these collections.\n"

            return result

        except Exception as e:
            logger.error(f"Error managing collections: {e}")
            return f"Error managing collections: {str(e)}"

    # Knowledge Graph Tools

    @mcp_tool
    async def extract_entities(self, text: str) -> str:
        """Extract named entities from text using spaCy NER.

        Args:
            text: Text to process for entity extraction
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            entities = self.client.extract_entities(text)

            result = f"Named Entity Extraction Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Text Length: {len(text)} characters\n"
            result += f"Entities Found: {len(entities)}\n\n"

            if entities:
                # Group entities by type
                by_type = {}
                for entity in entities:
                    entity_type = entity['label']
                    if entity_type not in by_type:
                        by_type[entity_type] = []
                    by_type[entity_type].append(entity)

                result += "Entities by Type:\n"
                for entity_type, entity_list in sorted(by_type.items()):
                    result += f"\n{entity_type} ({entity_list[0]['description']}):\n"

                    # Show unique entities of this type
                    unique_entities = {}
                    for entity in entity_list:
                        text_key = entity['text'].lower()
                        if text_key not in unique_entities:
                            unique_entities[text_key] = entity

                    for entity in list(unique_entities.values())[:10]:  # Limit to 10 per type
                        result += f"  • {entity['text']} (confidence: {entity['confidence']:.3f})\n"

                    if len(unique_entities) > 10:
                        result += f"  ... and {len(unique_entities) - 10} more\n"

                # Show text preview with entities highlighted
                result += f"\nText Preview with Entities:\n"
                result += "-" * 30 + "\n"

                text_preview = text[:500]
                if len(text) > 500:
                    text_preview += "..."

                # Highlight entities in preview (simple approach)
                for entity in entities:
                    if entity['start_char'] < 500:  # Only highlight entities in preview
                        entity_text = entity['text']
                        text_preview = text_preview.replace(
                            entity_text,
                            f"[{entity_text}:{entity['label']}]",
                            1  # Replace only first occurrence
                        )

                result += text_preview

            else:
                result += "No entities found in the provided text.\n"
                result += "\nNote: This tool requires the spaCy 'en_core_web_sm' model.\n"
                result += "Install with: python -m spacy download en_core_web_sm\n"

            return result

        except Exception as e:
            logger.error(f"Error extracting entities: {e}")
            return f"Error extracting entities: {str(e)}"

    @mcp_tool
    async def build_knowledge_graph(self, documents_json: str) -> str:
        """Build knowledge graph from documents using Neo4j.

        Args:
            documents_json: JSON string containing list of document dictionaries
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            documents = json.loads(documents_json)
            graph_stats = self.client.build_knowledge_graph(documents)

            result = f"Knowledge Graph Construction Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Documents Processed: {graph_stats['documents_processed']}\n"
            result += f"Total Entities Extracted: {graph_stats['total_entities']}\n"
            result += f"Graph Nodes Created: {graph_stats['nodes_created']}\n"
            result += f"Relationships Created: {graph_stats['relationships_created']}\n"
            result += f"Construction Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"

            if graph_stats['total_entities'] > 0:
                result += "Knowledge Graph Statistics:\n"
                result += f"  • Average Entities per Document: {graph_stats['total_entities'] / graph_stats['documents_processed']:.1f}\n"
                result += f"  • Average Relationships per Document: {graph_stats['relationships_created'] / graph_stats['documents_processed']:.1f}\n\n"

                result += "Graph Structure Created:\n"
                result += "  • Document nodes with metadata (name, path, size, modified date)\n"
                result += "  • Entity nodes with labels and descriptions\n"
                result += "  • CONTAINS relationships between documents and entities\n\n"

                result += "You can now use:\n"
                result += "  • find_relationships - to discover entity co-occurrences\n"
                result += "  • graph_query - to run custom Cypher queries\n"

            else:
                result += "No entities were extracted from the documents.\n"
                result += "This could be due to:\n"
                result += "  • Empty or very short documents\n"
                result += "  • Missing spaCy language model\n"
                result += "  • Documents in unsupported language\n"

            return result

        except Exception as e:
            logger.error(f"Error building knowledge graph: {e}")
            return f"Error building knowledge graph: {str(e)}"

    @mcp_tool
    async def find_relationships(self, entity1: str, entity2: str) -> str:
        """Find relationships between entities in the knowledge graph.

        Args:
            entity1: First entity text
            entity2: Second entity text
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            relationships = self.client.find_relationships(entity1, entity2)

            result = f"Entity Relationship Analysis\n"
            result += "=" * 60 + "\n\n"

            result += f"Entity 1: {entity1}\n"
            result += f"Entity 2: {entity2}\n"
            result += f"Relationships Found: {len(relationships)}\n\n"

            if relationships:
                result += "Relationship Details:\n\n"

                for i, rel in enumerate(relationships):
                    result += f"Relationship #{i+1}:\n"
                    result += f"  Document: {rel['document']}\n"
                    result += f"  Path: {rel['document_path']}\n"
                    result += f"  Entity 1: {rel['entity1']} ({rel['entity1_label']})\n"
                    result += f"  Entity 2: {rel['entity2']} ({rel['entity2_label']})\n"
                    result += f"  Relationship Type: {rel['relationship_type']}\n"
                    result += f"  Confidence Score: {rel['confidence_score']:.3f}\n\n"

                # Show summary
                documents = set(rel['document'] for rel in relationships)
                result += f"Summary:\n"
                result += f"  • Found in {len(documents)} documents\n"
                result += f"  • Average confidence: {sum(r['confidence_score'] for r in relationships) / len(relationships):.3f}\n"

            else:
                result += f"No relationships found between '{entity1}' and '{entity2}'.\n\n"
                result += "This could mean:\n"
                result += "  • The entities don't co-occur in any documents\n"
                result += "  • The entities haven't been extracted (try extract_entities first)\n"
                result += "  • The knowledge graph hasn't been built yet\n"
                result += "  • The entity names don't match exactly (case sensitive)\n"

            return result

        except Exception as e:
            logger.error(f"Error finding relationships: {e}")
            return f"Error finding relationships: {str(e)}"

    @mcp_tool
    async def graph_query(self, cypher_query: str) -> str:
        """Execute custom Cypher query on knowledge graph.

        Args:
            cypher_query: Cypher query string to execute
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            query_results = self.client.graph_query(cypher_query)

            result = f"Graph Query Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Query: {cypher_query}\n"
            result += f"Results: {len(query_results)}\n\n"

            if query_results:
                result += "Query Results:\n\n"

                for i, record in enumerate(query_results[:20]):  # Limit to 20 results
                    result += f"Result #{i+1}:\n"
                    for key, value in record.items():
                        result += f"  {key}: {value}\n"
                    result += "\n"

                if len(query_results) > 20:
                    result += f"... and {len(query_results) - 20} more results\n"

            else:
                result += "No results returned by the query.\n\n"
                result += "Common Cypher queries:\n"
                result += "  • MATCH (d:Document) RETURN d.name, d.path LIMIT 10\n"
                result += "  • MATCH (e:Entity) RETURN e.label, count(*) GROUP BY e.label\n"
                result += "  • MATCH (d:Document)-[:CONTAINS]->(e:Entity) RETURN d.name, e.text, e.label LIMIT 10\n"

            return result

        except Exception as e:
            logger.error(f"Error executing graph query: {e}")
            return f"Error executing graph query: {str(e)}"

    # RAG Query Interface Tools

    @mcp_tool
    async def rag_query(self, collection_name: str,
                       query: str,
                       n_results: int = 5,
                       include_entities: bool = False) -> str:
        """Unified RAG search interface combining vector search and knowledge graph.

        Args:
            collection_name: Name of the vector collection to search
            query: Search query text
            n_results: Number of vector search results to return
            include_entities: Whether to extract entities from query and results
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            result = f"RAG Query Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Query: \"{query}\"\n"
            result += f"Collection: {collection_name}\n"
            result += f"Results Requested: {n_results}\n\n"

            # 1. Perform semantic search
            search_results = self.client.semantic_search(
                collection_name=collection_name,
                query=query,
                n_results=n_results
            )

            result += f"Vector Search Results: {len(search_results)} found\n\n"

            # 2. Extract entities from query if requested
            query_entities = []
            if include_entities:
                try:
                    query_entities = self.client.extract_entities(query)
                    result += f"Query Entities Detected: {len(query_entities)}\n"
                    if query_entities:
                        entity_texts = [e['text'] for e in query_entities[:5]]
                        result += f"  Key Entities: {', '.join(entity_texts)}\n"
                    result += "\n"
                except Exception as e:
                    result += f"Entity extraction failed: {e}\n\n"

            # 3. Show search results with context
            if search_results:
                result += "Relevant Documents:\n\n"

                for i, search_result in enumerate(search_results):
                    result += f"Document #{i+1} (Relevance: {1 - search_result['score']:.3f})\n"
                    result += f"Source: {search_result['metadata'].get('source_file_name', 'Unknown')}\n"

                    # Show text with highlighting
                    text_content = search_result['text']
                    result += f"Content: {text_content}\n"

                    # Extract entities from this result if requested
                    if include_entities and query_entities:
                        try:
                            result_entities = self.client.extract_entities(text_content)

                            # Find common entities between query and result
                            query_entity_texts = {e['text'].lower() for e in query_entities}
                            common_entities = [e for e in result_entities
                                             if e['text'].lower() in query_entity_texts]

                            if common_entities:
                                result += f"  Common Entities: "
                                result += ", ".join([f"{e['text']} ({e['label']})" for e in common_entities])
                                result += "\n"

                        except Exception as e:
                            result += f"  Entity analysis failed: {e}\n"

                    result += "\n" + "-" * 40 + "\n\n"

                # 4. Provide contextual summary
                result += "Query Summary:\n"
                avg_relevance = sum(1 - r['score'] for r in search_results) / len(search_results)
                result += f"  • Average relevance score: {avg_relevance:.3f}\n"
                result += f"  • Most relevant source: {search_results[0]['metadata'].get('source_file_name', 'Unknown')}\n"

                unique_sources = len(set(r['metadata'].get('source_file_name', '') for r in search_results))
                result += f"  • Sources covered: {unique_sources}\n"

            else:
                result += "No relevant documents found.\n\n"
                result += "Suggestions:\n"
                result += "  • Try broader search terms\n"
                result += "  • Check if the collection has been populated\n"
                result += "  • Consider using different keywords\n"

            return result

        except Exception as e:
            logger.error(f"Error in RAG query: {e}")
            return f"Error in RAG query: {str(e)}"

    @mcp_tool
    async def contextual_retrieval(self, collection_name: str,
                                 context: str,
                                 query: str,
                                 n_results: int = 5) -> str:
        """Context-aware semantic search with query expansion.

        Args:
            collection_name: Name of the vector collection to search
            context: Context information to enhance the search
            query: Search query text
            n_results: Number of results to return
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            # Combine context and query for enhanced search
            enhanced_query = f"{context} {query}"

            result = f"Contextual Retrieval Results\n"
            result += "=" * 60 + "\n\n"

            result += f"Original Query: \"{query}\"\n"
            result += f"Context: \"{context}\"\n"
            result += f"Enhanced Query: \"{enhanced_query}\"\n"
            result += f"Collection: {collection_name}\n\n"

            # Perform enhanced semantic search
            search_results = self.client.semantic_search(
                collection_name=collection_name,
                query=enhanced_query,
                n_results=n_results
            )

            # Also perform original query for comparison
            original_results = self.client.semantic_search(
                collection_name=collection_name,
                query=query,
                n_results=n_results
            )

            result += f"Enhanced Search Results: {len(search_results)} found\n"
            result += f"Original Search Results: {len(original_results)} found\n\n"

            if search_results:
                result += "Context-Enhanced Results:\n\n"

                for i, search_result in enumerate(search_results):
                    result += f"Result #{i+1} (Score: {search_result['score']:.4f})\n"
                    result += f"Source: {search_result['metadata'].get('source_file_name', 'Unknown')}\n"

                    text_preview = search_result['text'][:300]
                    if len(search_result['text']) > 300:
                        text_preview += "..."
                    result += f"Content: {text_preview}\n"

                    # Check if this result was also in original search
                    original_ids = [r['id'] for r in original_results]
                    if search_result['id'] in original_ids:
                        original_idx = original_ids.index(search_result['id'])
                        original_score = original_results[original_idx]['score']
                        improvement = original_score - search_result['score']
                        result += f"  Context Improvement: {improvement:+.4f} (was rank #{original_idx + 1})\n"
                    else:
                        result += f"  New Result: Not found in original query\n"

                    result += "\n" + "-" * 40 + "\n\n"

                # Show analysis
                result += "Contextual Analysis:\n"
                enhanced_scores = [r['score'] for r in search_results]
                original_scores = [r['score'] for r in original_results]

                result += f"  • Enhanced average score: {sum(enhanced_scores) / len(enhanced_scores):.4f}\n"
                result += f"  • Original average score: {sum(original_scores) / len(original_scores):.4f}\n"

                # Count new vs improved results
                enhanced_ids = set(r['id'] for r in search_results)
                original_ids = set(r['id'] for r in original_results)
                new_results = len(enhanced_ids - original_ids)
                improved_results = len(enhanced_ids & original_ids)

                result += f"  • New results found: {new_results}\n"
                result += f"  • Results improved: {improved_results}\n"

            else:
                result += "No results found with contextual enhancement.\n"

            return result

        except Exception as e:
            logger.error(f"Error in contextual retrieval: {e}")
            return f"Error in contextual retrieval: {str(e)}"

    @mcp_tool
    async def summarize_knowledge(self, collection_name: str,
                                topic: str,
                                max_chunks: int = 20) -> str:
        """Summarize knowledge about a specific topic from the knowledge base.

        Args:
            collection_name: Name of the vector collection to search
            topic: Topic to summarize knowledge about
            max_chunks: Maximum number of chunks to analyze
        """
        if not self.client:
            return "Error: RAG client not initialized"

        try:
            result = f"Knowledge Summary: {topic}\n"
            result += "=" * 60 + "\n\n"

            # Search for relevant chunks
            search_results = self.client.semantic_search(
                collection_name=collection_name,
                query=topic,
                n_results=max_chunks
            )

            if not search_results:
                result += f"No information found about '{topic}' in the knowledge base.\n"
                return result

            result += f"Found {len(search_results)} relevant documents\n"
            result += f"Topic: {topic}\n\n"

            # Group results by source
            by_source = {}
            total_chars = 0

            for search_result in search_results:
                source = search_result['metadata'].get('source_file_name', 'Unknown')
                if source not in by_source:
                    by_source[source] = []
                by_source[source].append(search_result)
                total_chars += len(search_result['text'])

            result += "Knowledge Sources:\n"
            for source, chunks in by_source.items():
                avg_relevance = sum(1 - c['score'] for c in chunks) / len(chunks)
                result += f"  • {source}: {len(chunks)} chunks (avg relevance: {avg_relevance:.3f})\n"

            result += f"\nTotal Content: {total_chars:,} characters\n\n"

            # Extract key concepts and entities
            all_text = " ".join([r['text'] for r in search_results[:10]])  # Use top 10 for entity extraction

            try:
                entities = self.client.extract_entities(all_text)
                if entities:
                    # Group entities by type
                    entity_by_type = {}
                    for entity in entities:
                        entity_type = entity['label']
                        if entity_type not in entity_by_type:
                            entity_by_type[entity_type] = set()
                        entity_by_type[entity_type].add(entity['text'])

                    result += "Key Concepts Identified:\n"
                    for entity_type, entity_set in sorted(entity_by_type.items()):
                        if len(entity_set) > 0:
                            entities_list = list(entity_set)[:10]  # Limit to 10 per type
                            result += f"  {entity_type}: {', '.join(entities_list)}\n"

                    result += "\n"
            except Exception as e:
                result += f"Entity extraction failed: {e}\n\n"

            # Show content summary
            result += "Content Summary:\n\n"

            # Show top chunks with context
            for i, search_result in enumerate(search_results[:5]):
                relevance = 1 - search_result['score']
                result += f"Key Point #{i+1} (Relevance: {relevance:.3f}):\n"
                result += f"Source: {search_result['metadata'].get('source_file_name', 'Unknown')}\n"

                content = search_result['text']
                # Highlight the topic in content (simple approach)
                highlighted_content = content.replace(topic, f"**{topic}**")
                result += f"Content: {highlighted_content}\n\n"

            # Provide insights
            result += "Knowledge Insights:\n"
            result += f"  • Information spans {len(by_source)} different sources\n"
            result += f"  • Most comprehensive source: {max(by_source.keys(), key=lambda k: len(by_source[k]))}\n"

            high_relevance = sum(1 for r in search_results if (1 - r['score']) > 0.7)
            result += f"  • High-relevance chunks: {high_relevance} out of {len(search_results)}\n"

            return result

        except Exception as e:
            logger.error(f"Error summarizing knowledge: {e}")
            return f"Error summarizing knowledge: {str(e)}"

    @mcp_tool
    async def explain_concepts(self, collection_name: str,
                             concept: str,
                             depth: str = "medium") -> str:
        """Generate detailed explanation of concepts from knowledge base.

        Args:
            collection_name: Name of the vector collection to search
            concept: Concept to explain
            depth: Explanation depth ("basic", "medium", "detailed")
        """
        if not self.client:
            return "Error: RAG client not initialized"

        # Set search parameters based on depth
        depth_params = {
            "basic": {"n_results": 5, "min_relevance": 0.6},
            "medium": {"n_results": 10, "min_relevance": 0.5},
            "detailed": {"n_results": 20, "min_relevance": 0.4}
        }

        params = depth_params.get(depth, depth_params["medium"])

        try:
            result = f"Concept Explanation: {concept}\n"
            result += "=" * 60 + "\n\n"

            result += f"Concept: {concept}\n"
            result += f"Explanation Depth: {depth}\n"
            result += f"Collection: {collection_name}\n\n"

            # Search for concept information
            search_results = self.client.semantic_search(
                collection_name=collection_name,
                query=concept,
                n_results=params["n_results"]
            )

            if not search_results:
                result += f"No information found about '{concept}' in the knowledge base.\n"
                return result

            # Filter by minimum relevance
            relevant_results = [r for r in search_results if (1 - r['score']) >= params["min_relevance"]]

            if not relevant_results:
                result += f"No sufficiently relevant information found about '{concept}'.\n"
                result += f"Try using more basic terms or check the spelling.\n"
                return result

            result += f"Found {len(relevant_results)} relevant sources\n\n"

            # Extract and analyze entities
            all_text = " ".join([r['text'] for r in relevant_results])

            try:
                entities = self.client.extract_entities(all_text)

                # Find related concepts (entities that appear with our concept)
                related_concepts = []
                for entity in entities:
                    if entity['text'].lower() != concept.lower() and entity['label'] in ['ORG', 'PERSON', 'GPE', 'PRODUCT', 'TECH']:
                        related_concepts.append(entity['text'])

                if related_concepts:
                    unique_related = list(set(related_concepts))[:10]
                    result += f"Related Concepts: {', '.join(unique_related)}\n\n"

            except Exception as e:
                result += f"Related concept analysis failed: {e}\n\n"

            # Generate explanation based on depth
            if depth == "basic":
                result += "Basic Explanation:\n\n"
                # Use top 2 most relevant chunks
                for i, search_result in enumerate(relevant_results[:2]):
                    relevance = 1 - search_result['score']
                    result += f"Definition {i+1} (Confidence: {relevance:.2f}):\n"

                    content = search_result['text'][:400]  # Shorter for basic
                    if len(search_result['text']) > 400:
                        content += "..."

                    result += f"{content}\n\n"
                    result += f"Source: {search_result['metadata'].get('source_file_name', 'Unknown')}\n\n"

            elif depth == "medium":
                result += "Detailed Explanation:\n\n"

                # Group by themes/aspects
                result += "Key Aspects:\n\n"
                for i, search_result in enumerate(relevant_results[:5]):
                    relevance = 1 - search_result['score']
                    result += f"Aspect #{i+1} - Relevance: {relevance:.3f}\n"
                    result += f"From: {search_result['metadata'].get('source_file_name', 'Unknown')}\n"

                    content = search_result['text'][:600]
                    if len(search_result['text']) > 600:
                        content += "..."

                    result += f"Content: {content}\n\n"
                    result += "-" * 40 + "\n\n"

            else:  # detailed
                result += "Comprehensive Explanation:\n\n"

                # Show all relevant information organized by source
                by_source = {}
                for search_result in relevant_results:
                    source = search_result['metadata'].get('source_file_name', 'Unknown')
                    if source not in by_source:
                        by_source[source] = []
                    by_source[source].append(search_result)

                for source, chunks in by_source.items():
                    result += f"Information from: {source}\n"
                    result += "=" * 40 + "\n"

                    for j, chunk in enumerate(chunks):
                        relevance = 1 - chunk['score']
                        result += f"\nSection {j+1} (Relevance: {relevance:.3f}):\n"
                        result += f"{chunk['text']}\n"

                    result += "\n" + "=" * 40 + "\n\n"

            # Add summary insights
            result += "Summary Insights:\n"
            avg_relevance = sum(1 - r['score'] for r in relevant_results) / len(relevant_results)
            result += f"  • Average information confidence: {avg_relevance:.3f}\n"
            result += f"  • Information sources: {len(set(r['metadata'].get('source_file_name', '') for r in relevant_results))}\n"
            result += f"  • Total content analyzed: {sum(len(r['text']) for r in relevant_results):,} characters\n"

            return result

        except Exception as e:
            logger.error(f"Error explaining concept: {e}")
            return f"Error explaining concept: {str(e)}"

    # Resources

    @mcp_resource("rag://collections")
    async def get_collections_resource(self) -> str:
        """Get information about all vector collections."""
        if not self.client:
            return "RAG client not initialized"

        collections_info = self.client.manage_collections()

        resource_data = {
            "collections": collections_info,
            "total_collections": len(collections_info),
            "total_documents": sum(info.get("count", 0) for info in collections_info.values()),
            "timestamp": datetime.now().isoformat()
        }

        return json.dumps(resource_data, indent=2)

    @mcp_resource("rag://status")
    async def get_status_resource(self) -> str:
        """Get RAG system status and configuration."""
        if not self.client:
            return json.dumps({"status": "not_initialized"})

        status = {
            "status": "initialized",
            "embedding_model": self.client.embedding_model_name,
            "chroma_directory": self.client.persist_directory,
            "neo4j_connected": self.client.neo4j_driver is not None,
            "spacy_model_available": self.client.nlp is not None,
            "supported_extensions": list(self.client.supported_extensions),
            "timestamp": datetime.now().isoformat()
        }

        return json.dumps(status, indent=2)

    # Prompts

    @mcp_prompt("document_processing_prompt")
    async def document_processing_prompt(self, folder_path: str, file_types: Optional[str] = None) -> str:
        """Generate prompts for document processing workflows.

        Args:
            folder_path: Path to the folder to process
            file_types: Comma-separated list of file extensions (optional)
        """
        return f"""You are tasked with processing documents from: {folder_path}

Processing Workflow:
1. Use scan_folder to discover all supported documents
   - Include file types: {file_types or 'all supported types'}
   - Apply appropriate filters to exclude unwanted files

2. For each discovered file:
   - Use extract_text to get the content
   - Store the extracted data for chunking

3. Process the extracted documents:
   - Use chunk_documents to split into embedding-ready chunks
   - Use detect_duplicates to identify and handle duplicate content

4. Build the knowledge base:
   - Use build_vector_index to create searchable embeddings
   - Use build_knowledge_graph to create entity relationships

5. Validate the results:
   - Use manage_collections to verify index creation
   - Test with semantic_search queries

Focus on:
- Handling errors gracefully for corrupted or inaccessible files
- Optimizing chunk sizes for your specific content type
- Maintaining metadata links between chunks and source files
- Creating meaningful collection names for easy identification

Remember to process files incrementally if dealing with large document sets."""

    @mcp_prompt("semantic_search_prompt")
    async def semantic_search_prompt(self, search_type: str = "general") -> str:
        """Generate prompts for semantic search optimization.

        Args:
            search_type: Type of search (general, technical, academic, legal, etc.)
        """
        search_strategies = {
            "general": "broad keywords and natural language queries",
            "technical": "specific technical terms, APIs, and implementation details",
            "academic": "research concepts, methodologies, and theoretical frameworks",
            "legal": "legal terms, regulations, and case-specific language",
            "business": "business processes, metrics, and strategic terminology"
        }

        strategy = search_strategies.get(search_type, search_strategies["general"])

        return f"""You are optimizing semantic search for {search_type} content.

Search Strategy for {search_type.title()} Content:
- Use {strategy}
- Consider context and domain-specific vocabulary
- Combine multiple search approaches for comprehensive results

Best Practices:
1. Query Formulation:
   - Start with specific terms, then broaden if needed
   - Use domain-specific vocabulary relevant to {search_type}
   - Include synonyms and alternative phrasings

2. Result Analysis:
   - Examine relevance scores to gauge result quality
   - Look for consistent themes across top results
   - Identify gaps that might require query refinement

3. Advanced Techniques:
   - Use contextual_retrieval for complex queries
   - Combine vector search with knowledge graph queries
   - Apply metadata filters to narrow results by source, date, or type

4. Knowledge Integration:
   - Use rag_query for comprehensive information synthesis
   - Apply explain_concepts for detailed understanding
   - Use summarize_knowledge for topic overviews

Iterative Improvement:
- Analyze which queries return the most relevant results
- Refine search terms based on result patterns
- Build query templates for common information needs
- Document effective search patterns for reuse"""

    @mcp_prompt("knowledge_graph_analysis_prompt")
    async def knowledge_graph_analysis_prompt(self, analysis_focus: str = "relationships") -> str:
        """Generate prompts for knowledge graph analysis workflows.

        Args:
            analysis_focus: Focus area (relationships, entities, patterns, insights)
        """
        return f"""You are analyzing knowledge graphs with focus on: {analysis_focus}

Knowledge Graph Analysis Framework:

1. Entity Discovery:
   - Use extract_entities to identify key concepts, people, organizations
   - Analyze entity frequency and distribution across documents
   - Identify domain-specific entity types and patterns

2. Relationship Mapping:
   - Use find_relationships to discover entity co-occurrences
   - Build relationship networks between key concepts
   - Identify strong vs. weak entity associations

3. Graph Querying:
   - Use graph_query with Cypher to explore graph structure
   - Find shortest paths between related concepts
   - Discover clusters and communities in the knowledge network

4. Pattern Analysis:
   - Identify recurring entity combinations
   - Discover knowledge gaps (isolated entities)
   - Find central/hub entities that connect many concepts

Advanced Analysis Queries:
```cypher
// Find most connected entities
MATCH (e:Entity)-[r:CONTAINS]-(d:Document)
RETURN e.text, e.label, count(r) as connections
ORDER BY connections DESC LIMIT 10

// Find document clusters by shared entities
MATCH (d1:Document)-[:CONTAINS]->(e:Entity)<-[:CONTAINS]-(d2:Document)
WHERE d1 <> d2
RETURN d1.name, d2.name, count(e) as shared_entities
ORDER BY shared_entities DESC

// Identify knowledge domains
MATCH (e:Entity)
RETURN e.label, count(*) as frequency
ORDER BY frequency DESC
```

Focus Areas for {analysis_focus.title()}:
- Examine cross-document entity patterns
- Identify knowledge clusters and themes
- Discover implicit relationships through shared contexts
- Map information flow and knowledge dependencies

Insights to Generate:
- Key knowledge domains in your corpus
- Most important connecting concepts
- Information gaps and underrepresented topics
- Potential areas for knowledge base expansion"""


async def main():
    """Main function to run the RAG MCP server."""
    server = RAGMCPServer()
    await server.run()


if __name__ == "__main__":
    asyncio.run(main())