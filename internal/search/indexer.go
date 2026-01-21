package search

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

// Indexer manages the search index for all tools.
type Indexer struct {
	bleveIndex bleve.Index
	mu         sync.RWMutex
	indexPath  string
}

// NewIndexer creates a new search indexer with in-memory Bleve index.
func NewIndexer() (*Indexer, error) {
	// Use scorch (modern, fast index) with in-memory storage
	indexMapping := buildIndexMapping()

	// Create in-memory index for fast startup
	index, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to create bleve index: %w", err)
	}

	return &Indexer{
		bleveIndex: index,
		indexPath:  "",
	}, nil
}

// NewIndexerWithPath creates a new indexer with persistent disk storage.
func NewIndexerWithPath(indexPath string) (*Indexer, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	indexMapping := buildIndexMapping()

	// Open or create index with Scorch backend
	index, err := bleve.NewUsing(indexPath, indexMapping, scorch.Name, scorch.Name, nil)
	if err != nil {
		// If index exists, open it
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open/create index: %w", err)
		}
	}

	return &Indexer{
		bleveIndex: index,
		indexPath:  indexPath,
	}, nil
}

// buildIndexMapping creates the Bleve index mapping.
func buildIndexMapping() mapping.IndexMapping {
	// Create a mapping for tool documents
	toolMapping := bleve.NewDocumentMapping()

	// Name field: searchable text
	nameFieldMapping := bleve.NewTextFieldMapping()
	toolMapping.AddFieldMappingsAt("name", nameFieldMapping)

	// Description field: searchable text
	descFieldMapping := bleve.NewTextFieldMapping()
	toolMapping.AddFieldMappingsAt("description", descFieldMapping)

	// Server field: searchable text for filtering
	serverFieldMapping := bleve.NewTextFieldMapping()
	toolMapping.AddFieldMappingsAt("server", serverFieldMapping)

	// InputSchema: stored but not indexed (for retrieval)
	inputSchemaMapping := bleve.NewTextFieldMapping()
	inputSchemaMapping.Index = false
	inputSchemaMapping.IncludeInAll = false
	toolMapping.AddFieldMappingsAt("inputSchema", inputSchemaMapping)

	// Create index mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("_default", toolMapping)

	return indexMapping
}

// IndexServer indexes all tools from a server.
func (i *Indexer) IndexServer(serverName string, tools []spawner.Tool) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	batch := i.bleveIndex.NewBatch()

	for _, tool := range tools {
		doc := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"server":      serverName,
			"inputSchema": tool.InputSchema,
		}

		// Use serverName/toolName as document ID
		docID := fmt.Sprintf("%s/%s", serverName, tool.Name)

		if err := batch.Index(docID, doc); err != nil {
			log.Printf("Warning: failed to index tool %s: %v", docID, err)
		}
	}

	if err := i.bleveIndex.Batch(batch); err != nil {
		return fmt.Errorf("failed to batch index tools: %w", err)
	}

	return nil
}

// RemoveServer removes all tools from a server (for reindexing).
func (i *Indexer) RemoveServer(serverName string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Delete all documents with this server prefix
	query := bleve.NewWildcardQuery(fmt.Sprintf("%s/*", serverName))
	searchRequest := bleve.NewSearchRequestOptions(query, 1000, 0, false)

	results, err := i.bleveIndex.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("failed to find server docs: %w", err)
	}

	batch := i.bleveIndex.NewBatch()
	for _, hit := range results.Hits {
		batch.Delete(hit.ID)
	}

	if err := i.bleveIndex.Batch(batch); err != nil {
		return fmt.Errorf("failed to batch delete: %w", err)
	}

	return nil
}

// Count returns the total number of indexed tools.
func (i *Indexer) Count() (uint64, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	docCount, err := i.bleveIndex.DocCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get doc count: %w", err)
	}

	return docCount, nil
}

// Close closes the index and releases resources.
func (i *Indexer) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.bleveIndex != nil {
		return i.bleveIndex.Close()
	}

	return nil
}

// buildMatchQuery creates a match query for BM25 search.
func (i *Indexer) buildMatchQuery(searchText string) query.Query {
	// Use match query with fuzzy matching
	return bleve.NewMatchQuery(searchText)
}
