package search

import (
	"encoding/json"
	"fmt"

	"github.com/blevesearch/bleve/v2"
)

// SearchBM25 performs BM25 keyword search using Bleve.
func (i *Indexer) SearchBM25(query string, limit int) ([]SearchResult, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	// Build search query
	searchQuery := i.buildMatchQuery(query)

	// Create search request
	searchRequest := bleve.NewSearchRequestOptions(searchQuery, limit, 0, false)
	searchRequest.Fields = []string{"name", "description", "server", "inputSchema"}

	// Execute search
	results, err := i.bleveIndex.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	// Convert results
	return convertBleveResults(results), nil
}

// convertBleveResults converts Bleve search results to our SearchResult format.
func convertBleveResults(results *bleve.SearchResult) []SearchResult {
	searchResults := make([]SearchResult, 0, len(results.Hits))

	for _, hit := range results.Hits {
		// Extract fields from hit
		name, _ := hit.Fields["name"].(string)
		description, _ := hit.Fields["description"].(string)
		server, _ := hit.Fields["server"].(string)

		// Parse inputSchema
		var inputSchema interface{}
		if schemaRaw, ok := hit.Fields["inputSchema"]; ok {
			// Convert to JSON
			if schemaBytes, err := json.Marshal(schemaRaw); err == nil {
				json.Unmarshal(schemaBytes, &inputSchema)
			}
		}

		result := SearchResult{
			ToolName:    name,
			Description: description,
			InputSchema: inputSchema,
			ServerName:  server,
			Score:       hit.Score,
		}

		searchResults = append(searchResults, result)
	}

	return searchResults
}

// SearchByServer performs BM25 search scoped to a specific server.
func (i *Indexer) SearchByServer(query, serverName string, limit int) ([]SearchResult, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	// Create conjunction query: (match query) AND (server filter)
	matchQuery := i.buildMatchQuery(query)
	serverQuery := bleve.NewTermQuery(serverName)
	serverQuery.SetField("server")

	conjunctionQuery := bleve.NewConjunctionQuery(matchQuery, serverQuery)

	// Create search request
	searchRequest := bleve.NewSearchRequestOptions(conjunctionQuery, limit, 0, false)
	searchRequest.Fields = []string{"name", "description", "server", "inputSchema"}

	// Execute search
	results, err := i.bleveIndex.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	return convertBleveResults(results), nil
}

// GetAllTools retrieves all indexed tools (up to limit).
func (i *Indexer) GetAllTools(limit int) ([]SearchResult, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	// Match all documents
	query := bleve.NewMatchAllQuery()
	searchRequest := bleve.NewSearchRequestOptions(query, limit, 0, false)
	searchRequest.Fields = []string{"name", "description", "server", "inputSchema"}

	results, err := i.bleveIndex.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	return convertBleveResults(results), nil
}
