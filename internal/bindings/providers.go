/*
  providers.go -- CRUD bindings for provider management.
  Exposes ListProviders, CreateProvider, UpdateProvider, DeleteProvider to the Wails frontend.
  Each provider stores its own connection parameters and runtime defaults (temperature, num_ctx).
  API keys are NEVER serialised to the frontend (json:"-"). Use api_key_set to detect presence.
*/

package bindings

import (
	"context"
	"fmt"
	"strings"
)

// ProviderRecord is a full provider row returned to the frontend.
// APIKey is intentionally excluded from JSON — keys are resolved Go-side via getProviderCredentials.
type ProviderRecord struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	URL         string  `json:"url"`
	APIKey      string  `json:"-"`
	APIKeySet   bool    `json:"api_key_set"`
	Description string  `json:"description"`
	UseInRAG    bool    `json:"use_in_rag"`
	Active      bool    `json:"active"`
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// ListProvidersPayload controls optional filters for provider listing.
type ListProvidersPayload struct {
	ActiveOnly bool `json:"active_only"`
}

// CreateProviderPayload is the payload for adding a new provider.
type CreateProviderPayload struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	URL         string  `json:"url"`
	APIKey      string  `json:"api_key"`
	Description string  `json:"description"`
	UseInRAG    bool    `json:"use_in_rag"`
	Active      bool    `json:"active"`
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
}

// UpdateProviderPayload is the payload for modifying an existing provider.
// If APIKey is empty, the stored key is preserved unchanged.
type UpdateProviderPayload struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	URL         string  `json:"url"`
	APIKey      string  `json:"api_key"`
	Description string  `json:"description"`
	UseInRAG    bool    `json:"use_in_rag"`
	Active      bool    `json:"active"`
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
}

// DeleteProviderPayload identifies a provider to remove.
type DeleteProviderPayload struct {
	ID int64 `json:"id"`
}

// ListProviders returns all providers ordered by name.
func (s *Service) ListProviders(ctx context.Context, payload ListProvidersPayload) ([]ProviderRecord, error) {
	query := `SELECT id, name, type, url, api_key, description, use_in_rag, active,
	                 temperature, num_ctx, created_at, updated_at
	          FROM providers`
	if payload.ActiveOnly {
		query += " WHERE active = 1"
	}
	query += " ORDER BY name ASC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ProviderRecord, 0)
	for rows.Next() {
		var p ProviderRecord
		var useInRAG, active int
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Type, &p.URL, &p.APIKey, &p.Description,
			&useInRAG, &active, &p.Temperature, &p.NumCtx, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.UseInRAG = useInRAG == 1
		p.Active = active == 1
		p.APIKeySet = p.APIKey != ""
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// CreateProvider inserts a new provider row and returns the created record.
func (s *Service) CreateProvider(ctx context.Context, payload CreateProviderPayload) (ProviderRecord, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return ProviderRecord{}, fmt.Errorf("provider name is required")
	}
	pType := strings.TrimSpace(payload.Type)
	if pType == "" {
		pType = "openai-compatible"
	}
	temp := payload.Temperature
	if temp <= 0 {
		temp = 0.7
	}
	numCtx := payload.NumCtx
	if numCtx <= 0 {
		numCtx = 1024
	}
	useInRAG := 0
	if payload.UseInRAG {
		useInRAG = 1
	}
	active := 1
	if !payload.Active {
		active = 0
	}

	url := strings.TrimSpace(payload.URL)
	apiKey := strings.TrimSpace(payload.APIKey)
	description := strings.TrimSpace(payload.Description)

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO providers (name, type, url, api_key, description, use_in_rag, active, temperature, num_ctx)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name, pType, url, apiKey, description,
		useInRAG, active, temp, numCtx,
	)
	if err != nil {
		return ProviderRecord{}, fmt.Errorf("create provider: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ProviderRecord{}, err
	}
	return s.getProviderByID(ctx, id)
}

// UpdateProvider modifies an existing provider and returns the updated record.
// If APIKey is empty in the payload, the stored key is preserved.
func (s *Service) UpdateProvider(ctx context.Context, payload UpdateProviderPayload) (ProviderRecord, error) {
	if payload.ID <= 0 {
		return ProviderRecord{}, fmt.Errorf("invalid provider id")
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return ProviderRecord{}, fmt.Errorf("provider name is required")
	}
	useInRAG := 0
	if payload.UseInRAG {
		useInRAG = 1
	}
	active := 0
	if payload.Active {
		active = 1
	}
	pType := strings.TrimSpace(payload.Type)
	if pType == "" {
		pType = "openai-compatible"
	}
	temp := payload.Temperature
	if temp <= 0 {
		temp = 0.7
	}
	numCtx := payload.NumCtx
	if numCtx <= 0 {
		numCtx = 1024
	}
	url := strings.TrimSpace(payload.URL)
	apiKey := strings.TrimSpace(payload.APIKey)
	description := strings.TrimSpace(payload.Description)

	var err error
	if apiKey == "" {
		// Preserve the existing API key — do not overwrite with empty.
		_, err = s.db.ExecContext(ctx,
			`UPDATE providers
			 SET name=?, type=?, url=?, description=?,
			     use_in_rag=?, active=?, temperature=?, num_ctx=?,
			     updated_at=datetime('now')
			 WHERE id=?`,
			name, pType, url, description,
			useInRAG, active, temp, numCtx, payload.ID,
		)
	} else {
		_, err = s.db.ExecContext(ctx,
			`UPDATE providers
			 SET name=?, type=?, url=?, api_key=?, description=?,
			     use_in_rag=?, active=?, temperature=?, num_ctx=?,
			     updated_at=datetime('now')
			 WHERE id=?`,
			name, pType, url, apiKey, description,
			useInRAG, active, temp, numCtx, payload.ID,
		)
	}
	if err != nil {
		return ProviderRecord{}, fmt.Errorf("update provider: %w", err)
	}
	return s.getProviderByID(ctx, payload.ID)
}

// DeleteProvider removes a provider by ID.
func (s *Service) DeleteProvider(ctx context.Context, payload DeleteProviderPayload) (map[string]any, error) {
	if payload.ID <= 0 {
		return map[string]any{"ok": false, "reason": "invalid-id"}, nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id=?`, payload.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// getProviderByID is an internal helper to fetch a single provider row (includes API key).
func (s *Service) getProviderByID(ctx context.Context, id int64) (ProviderRecord, error) {
	var p ProviderRecord
	var useInRAG, active int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, type, url, api_key, description, use_in_rag, active,
		        temperature, num_ctx, created_at, updated_at
		 FROM providers WHERE id=?`, id,
	).Scan(
		&p.ID, &p.Name, &p.Type, &p.URL, &p.APIKey, &p.Description,
		&useInRAG, &active, &p.Temperature, &p.NumCtx, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return ProviderRecord{}, err
	}
	p.UseInRAG = useInRAG == 1
	p.Active = active == 1
	p.APIKeySet = p.APIKey != ""
	return p, nil
}

// getProviderCredentials fetches a provider from DB and returns a ready-to-use openAIConfig.
// Falls back to environment-based config when id is 0.
func (s *Service) getProviderCredentials(ctx context.Context, id int64) (openAIConfig, error) {
	if id <= 0 {
		return loadOpenAIConfig(), nil
	}
	p, err := s.getProviderByID(ctx, id)
	if err != nil {
		return openAIConfig{}, fmt.Errorf("provider %d not found: %w", id, err)
	}
	cfg := loadOpenAIConfig()
	if url := strings.TrimSpace(p.URL); url != "" {
		cfg.BaseURL = strings.TrimRight(url, "/")
	}
	if key := strings.TrimSpace(p.APIKey); key != "" {
		cfg.APIKey = key
	}
	cfg.Type = p.Type
	return cfg, nil
}
