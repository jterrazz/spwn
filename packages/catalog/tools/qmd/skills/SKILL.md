# QMD - Query Markdown Documents

QMD is an on-device search engine for markdown notes, meeting transcripts, documentation, and knowledge bases.

## Usage
```bash
qmd search "your query"              # Search indexed documents
qmd index /path/to/docs              # Index a directory
qmd search --semantic "concept"      # Semantic search with embeddings
```

## Features
- BM25 full-text search for keyword matching
- Vector semantic search using local embeddings
- LLM re-ranking for contextual retrieval
- Hybrid approach ideal for agentic workflows

## When to Use
Use QMD when you need to search through large knowledge bases, documentation, or notes.
It runs entirely locally - no external API calls needed for search.
