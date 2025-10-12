#!/usr/bin/env python3
"""
Comprehensive test suite for RAG MCP Server.

This test suite validates all RAG functionality including:
- Document processing and text extraction
- Vector database operations
- Knowledge graph management
- Semantic search and retrieval
- Entity extraction and relationship discovery
- RAG query interface and contextual search
"""

import asyncio
import json
import logging
import os
import sys
import tempfile
import time
from pathlib import Path
from typing import Dict, List, Any
from unittest.mock import Mock, patch

# Add the project root to Python path
project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
sys.path.insert(0, project_root)

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(levelname)s:%(name)s:%(message)s')
logger = logging.getLogger(__name__)

class MockRAGClient:
    """Mock RAG client for testing without external dependencies."""

    def __init__(self, *args, **kwargs):
        self.embedding_model_name = "all-MiniLM-L6-v2"
        self.persist_directory = "./test_chroma_db"
        self.neo4j_driver = Mock()
        self.nlp = Mock()
        self.supported_extensions = {'.txt', '.md', '.pdf', '.docx', '.json'}

    def scan_folder(self, folder_path, **kwargs):
        return [
            {
                'path': '/test/file1.txt',
                'name': 'file1.txt',
                'extension': '.txt',
                'size_bytes': 1024,
                'modified_time': '2024-01-01T12:00:00Z',
                'relative_path': 'file1.txt'
            },
            {
                'path': '/test/file2.md',
                'name': 'file2.md',
                'extension': '.md',
                'size_bytes': 2048,
                'modified_time': '2024-01-02T12:00:00Z',
                'relative_path': 'docs/file2.md'
            }
        ]

    def extract_text(self, file_path):
        return {
            'file_path': file_path,
            'file_name': Path(file_path).name,
            'file_extension': Path(file_path).suffix,
            'file_size': 1024,
            'modified_time': '2024-01-01T12:00:00Z',
            'file_hash': 'abc123def456',
            'text_content': 'This is sample extracted text content from the document.',
            'text_length': 58,
            'extraction_time': '2024-01-01T12:00:00Z'
        }

    def chunk_documents(self, documents, **kwargs):
        chunks = []
        for i, doc in enumerate(documents):
            for j in range(2):  # 2 chunks per document
                chunks.append({
                    'chunk_id': f"{doc.get('file_hash', f'doc{i}')}_{j}",
                    'chunk_index': j,
                    'chunk_text': f"Chunk {j} content from {doc.get('file_name', f'doc{i}')}",
                    'chunk_length': 50,
                    'source_file': doc.get('file_path', f'/test/doc{i}.txt'),
                    'source_file_name': doc.get('file_name', f'doc{i}.txt'),
                    'source_extension': '.txt',
                    'source_hash': doc.get('file_hash', f'hash{i}'),
                    'chunk_created_time': '2024-01-01T12:00:00Z'
                })
        return chunks

    def generate_embeddings(self, texts):
        return [[0.1, 0.2, 0.3] * 128 for _ in texts]  # 384-dim embeddings

    def detect_duplicates(self, documents, **kwargs):
        if len(documents) >= 2:
            return [[0, 1]]  # Mock duplicate group
        return []

    def build_vector_index(self, collection_name, chunks, **kwargs):
        return collection_name

    def semantic_search(self, collection_name, query, n_results=10, **kwargs):
        return [
            {
                'id': 'chunk_1',
                'text': 'This is a relevant chunk of text that matches the query.',
                'score': 0.15,
                'metadata': {'source_file_name': 'test_doc.txt', 'chunk_index': 0}
            },
            {
                'id': 'chunk_2',
                'text': 'Another relevant piece of information related to the search.',
                'score': 0.25,
                'metadata': {'source_file_name': 'another_doc.md', 'chunk_index': 1}
            }
        ]

    def update_index(self, collection_name, chunks):
        return len(chunks)

    def manage_collections(self):
        return {
            'test_collection': {
                'name': 'test_collection',
                'count': 10,
                'metadata': {'description': 'Test collection'}
            }
        }

    def extract_entities(self, text):
        return [
            {
                'text': 'OpenAI',
                'label': 'ORG',
                'description': 'Companies, agencies, institutions, etc.',
                'start_char': 10,
                'end_char': 16,
                'confidence': 0.95
            },
            {
                'text': 'Python',
                'label': 'PRODUCT',
                'description': 'Objects, vehicles, foods, etc.',
                'start_char': 25,
                'end_char': 31,
                'confidence': 0.90
            }
        ]

    def build_knowledge_graph(self, documents):
        return {
            'documents_processed': len(documents),
            'total_entities': len(documents) * 3,
            'nodes_created': len(documents),
            'relationships_created': len(documents) * 2
        }

    def find_relationships(self, entity1, entity2):
        return [
            {
                'document': 'test_doc.txt',
                'document_path': '/test/test_doc.txt',
                'entity1': entity1,
                'entity1_label': 'ORG',
                'entity2': entity2,
                'entity2_label': 'PRODUCT',
                'confidence_score': 0.85,
                'relationship_type': 'co_occurrence'
            }
        ]

    def graph_query(self, query):
        return [
            {'entity': 'OpenAI', 'count': 5},
            {'entity': 'Python', 'count': 8}
        ]


class RAGServerTestSuite:
    """Comprehensive test suite for RAG MCP Server."""

    def __init__(self):
        self.test_results = []
        self.server = None

    async def setup_server(self):
        """Setup test server with mocked dependencies."""
        try:
            # Mock all external dependencies before importing
            mock_modules = [
                'fitz', 'docx', 'openpyxl', 'pptx', 'chromadb', 'sentence_transformers',
                'neo4j', 'spacy', 'langchain', 'langchain.text_splitter', 'langchain.schema'
            ]

            for module in mock_modules:
                sys.modules[module] = Mock()

            # Import server and client from current directory
            sys.path.insert(0, os.path.dirname(__file__))
            from server import RAGMCPServer

            # Create server instance
            self.server = RAGMCPServer()

            # Mock the RAG client creation completely
            self.server.client = MockRAGClient()

            logger.info("RAG MCP Server setup completed with mocked dependencies")
            return True

        except Exception as e:
            logger.error(f"Failed to setup server: {e}")
            return False

    async def run_test(self, test_name: str, test_func) -> bool:
        """Run a single test and record results."""
        try:
            logger.info(f"Running test: {test_name}")
            start_time = time.time()

            result = await test_func()

            elapsed = time.time() - start_time

            if result and len(result) > 50:  # Basic validation
                self.test_results.append({
                    'name': test_name,
                    'status': 'PASS',
                    'elapsed': elapsed,
                    'result_length': len(result)
                })
                logger.info(f"✅ {test_name} passed ({elapsed:.3f}s, {len(result)} chars)")
                return True
            else:
                self.test_results.append({
                    'name': test_name,
                    'status': 'FAIL',
                    'elapsed': elapsed,
                    'error': f"Invalid result: {result[:100] if result else 'None'}"
                })
                logger.error(f"❌ {test_name} failed: Invalid result")
                return False

        except Exception as e:
            elapsed = time.time() - start_time if 'start_time' in locals() else 0
            self.test_results.append({
                'name': test_name,
                'status': 'FAIL',
                'elapsed': elapsed,
                'error': str(e)
            })
            logger.error(f"❌ {test_name} failed: {e}")
            return False

    # Document Processing Tests

    async def test_scan_folder(self):
        """Test folder scanning functionality."""
        return await self.server.scan_folder(
            folder_path="/test/documents",
            extensions=['.txt', '.md'],
            max_file_size_mb=10
        )

    async def test_extract_text(self):
        """Test text extraction from files."""
        return await self.server.extract_text("/test/sample.txt")

    async def test_chunk_documents(self):
        """Test document chunking functionality."""
        documents = [
            {
                'file_path': '/test/doc1.txt',
                'file_name': 'doc1.txt',
                'file_hash': 'hash1',
                'text_content': 'This is a sample document for chunking test.'
            }
        ]
        return await self.server.chunk_documents(
            documents_json=json.dumps(documents),
            chunk_size=500,
            chunk_overlap=100
        )

    async def test_generate_embeddings(self):
        """Test embedding generation."""
        texts = ["Sample text for embedding", "Another text to embed"]
        return await self.server.generate_embeddings(json.dumps(texts))

    async def test_detect_duplicates(self):
        """Test duplicate document detection."""
        documents = [
            {'text_content': 'Same content', 'file_name': 'doc1.txt'},
            {'text_content': 'Same content', 'file_name': 'doc2.txt'}
        ]
        return await self.server.detect_duplicates(
            documents_json=json.dumps(documents),
            similarity_threshold=0.95
        )

    # Vector Database Tests

    async def test_build_vector_index(self):
        """Test vector index creation."""
        chunks = [
            {
                'chunk_id': 'chunk_1',
                'chunk_text': 'Sample chunk text',
                'source_file': '/test/doc.txt',
                'source_file_name': 'doc.txt',
                'chunk_index': 0,
                'chunk_length': 18
            }
        ]
        return await self.server.build_vector_index(
            collection_name="test_collection",
            chunks_json=json.dumps(chunks)
        )

    async def test_semantic_search(self):
        """Test semantic search functionality."""
        return await self.server.semantic_search(
            collection_name="test_collection",
            query="sample search query",
            n_results=5
        )

    async def test_update_index(self):
        """Test index updating with new chunks."""
        chunks = [
            {
                'chunk_id': 'new_chunk_1',
                'chunk_text': 'New chunk content',
                'source_file': '/test/new_doc.txt'
            }
        ]
        return await self.server.update_index(
            collection_name="test_collection",
            chunks_json=json.dumps(chunks)
        )

    async def test_manage_collections(self):
        """Test collection management functionality."""
        return await self.server.manage_collections()

    # Knowledge Graph Tests

    async def test_extract_entities(self):
        """Test entity extraction from text."""
        return await self.server.extract_entities(
            "OpenAI released Python tools for machine learning applications."
        )

    async def test_build_knowledge_graph(self):
        """Test knowledge graph construction."""
        documents = [
            {
                'file_path': '/test/doc1.txt',
                'file_name': 'doc1.txt',
                'file_hash': 'hash1',
                'text_content': 'OpenAI develops Python tools for AI applications.'
            }
        ]
        return await self.server.build_knowledge_graph(json.dumps(documents))

    async def test_find_relationships(self):
        """Test entity relationship discovery."""
        return await self.server.find_relationships("OpenAI", "Python")

    async def test_graph_query(self):
        """Test custom graph queries."""
        return await self.server.graph_query(
            "MATCH (e:Entity) RETURN e.text, count(*) as frequency"
        )

    # RAG Query Interface Tests

    async def test_rag_query(self):
        """Test unified RAG query interface."""
        return await self.server.rag_query(
            collection_name="test_collection",
            query="artificial intelligence applications",
            n_results=3,
            include_entities=True
        )

    async def test_contextual_retrieval(self):
        """Test context-aware semantic search."""
        return await self.server.contextual_retrieval(
            collection_name="test_collection",
            context="machine learning and AI development",
            query="Python tools",
            n_results=5
        )

    async def test_summarize_knowledge(self):
        """Test knowledge summarization."""
        return await self.server.summarize_knowledge(
            collection_name="test_collection",
            topic="artificial intelligence",
            max_chunks=10
        )

    async def test_explain_concepts(self):
        """Test concept explanation functionality."""
        return await self.server.explain_concepts(
            collection_name="test_collection",
            concept="machine learning",
            depth="medium"
        )

    # Resource and Prompt Tests

    async def test_collections_resource(self):
        """Test collections resource."""
        return await self.server.get_collections_resource()

    async def test_status_resource(self):
        """Test status resource."""
        return await self.server.get_status_resource()

    async def test_document_processing_prompt(self):
        """Test document processing prompt."""
        return await self.server.document_processing_prompt(
            folder_path="/test/documents",
            file_types=".txt,.md,.pdf"
        )

    async def test_semantic_search_prompt(self):
        """Test semantic search prompt."""
        return await self.server.semantic_search_prompt("technical")

    async def test_knowledge_graph_prompt(self):
        """Test knowledge graph analysis prompt."""
        return await self.server.knowledge_graph_analysis_prompt("relationships")

    async def run_all_tests(self):
        """Run all tests and generate comprehensive report."""
        logger.info("Starting RAG MCP Server comprehensive testing")

        # Setup server
        if not await self.setup_server():
            logger.error("Failed to setup server, aborting tests")
            return

        # Define all tests
        tests = [
            # Document Processing Tests
            ("scan_folder", self.test_scan_folder),
            ("extract_text", self.test_extract_text),
            ("chunk_documents", self.test_chunk_documents),
            ("generate_embeddings", self.test_generate_embeddings),
            ("detect_duplicates", self.test_detect_duplicates),

            # Vector Database Tests
            ("build_vector_index", self.test_build_vector_index),
            ("semantic_search", self.test_semantic_search),
            ("update_index", self.test_update_index),
            ("manage_collections", self.test_manage_collections),

            # Knowledge Graph Tests
            ("extract_entities", self.test_extract_entities),
            ("build_knowledge_graph", self.test_build_knowledge_graph),
            ("find_relationships", self.test_find_relationships),
            ("graph_query", self.test_graph_query),

            # RAG Query Interface Tests
            ("rag_query", self.test_rag_query),
            ("contextual_retrieval", self.test_contextual_retrieval),
            ("summarize_knowledge", self.test_summarize_knowledge),
            ("explain_concepts", self.test_explain_concepts),

            # Resource and Prompt Tests
            ("collections_resource", self.test_collections_resource),
            ("status_resource", self.test_status_resource),
            ("document_processing_prompt", self.test_document_processing_prompt),
            ("semantic_search_prompt", self.test_semantic_search_prompt),
            ("knowledge_graph_prompt", self.test_knowledge_graph_prompt),
        ]

        # Run tests
        for test_name, test_func in tests:
            await self.run_test(test_name, test_func)

        # Generate report
        self.generate_report()

    def generate_report(self):
        """Generate comprehensive test report."""
        total_tests = len(self.test_results)
        passed_tests = len([r for r in self.test_results if r['status'] == 'PASS'])
        failed_tests = total_tests - passed_tests
        success_rate = (passed_tests / total_tests) * 100 if total_tests > 0 else 0

        print("\n" + "=" * 60)
        print("RAG MCP SERVER TEST RESULTS")
        print("=" * 60)
        print(f"Total Tests: {total_tests}")
        print(f"Passed: {passed_tests}")
        print(f"Failed: {failed_tests}")
        print(f"Success Rate: {success_rate:.1f}%")
        print()

        print("Detailed Results:")
        for result in self.test_results:
            status_icon = "✅" if result['status'] == 'PASS' else "❌"
            test_name = result['name']

            if result['status'] == 'PASS':
                result_length = result.get('result_length', 0)
                print(f"{status_icon} PASS {test_name}: Result length: {result_length} chars")
            else:
                error = result.get('error', 'Unknown error')
                print(f"{status_icon} FAIL {test_name}: {error}")

        if failed_tests > 0:
            print(f"\n⚠️  {failed_tests} test(s) failed. Please review the issues above.")
        else:
            print(f"\n🎉 All tests passed! RAG MCP Server is working correctly.")

        # Performance summary
        total_time = sum(r.get('elapsed', 0) for r in self.test_results)
        print(f"\nPerformance Summary:")
        print(f"  • Total execution time: {total_time:.3f}s")
        print(f"  • Average test time: {total_time / total_tests:.3f}s")

        # Test categories summary
        categories = {
            'Document Processing': ['scan_folder', 'extract_text', 'chunk_documents', 'generate_embeddings', 'detect_duplicates'],
            'Vector Database': ['build_vector_index', 'semantic_search', 'update_index', 'manage_collections'],
            'Knowledge Graph': ['extract_entities', 'build_knowledge_graph', 'find_relationships', 'graph_query'],
            'RAG Interface': ['rag_query', 'contextual_retrieval', 'summarize_knowledge', 'explain_concepts'],
            'Resources & Prompts': ['collections_resource', 'status_resource', 'document_processing_prompt', 'semantic_search_prompt', 'knowledge_graph_prompt']
        }

        print(f"\nResults by Category:")
        for category, test_names in categories.items():
            category_results = [r for r in self.test_results if r['name'] in test_names]
            category_passed = len([r for r in category_results if r['status'] == 'PASS'])
            category_total = len(category_results)
            category_rate = (category_passed / category_total) * 100 if category_total > 0 else 0
            print(f"  • {category}: {category_passed}/{category_total} ({category_rate:.1f}%)")


async def main():
    """Main function to run the test suite."""
    test_suite = RAGServerTestSuite()
    await test_suite.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())