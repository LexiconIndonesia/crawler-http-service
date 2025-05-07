package crawler

import (
	"encoding/json"
	"errors"
	"fmt"
)

// DataSourceConfig is the interface that all data source configs must implement
type DataSourceConfig interface {
	// Validate validates the configuration
	Validate() error
}

// BaseConfig contains common configuration fields for all data sources
type BaseConfig struct {
	PaginationSelector string `json:"pagination_selector"`
	DetailLinkSelector string `json:"detail_link_selector"`
	MaxPages           int    `json:"max_pages"`
	Delay              int    `json:"delay_ms"`
}

// MahkamahAgungConfig represents configuration for Indonesia Supreme Court
type MahkamahAgungConfig struct {
	BaseConfig
	SearchFormSelector string `json:"search_form_selector"`
}

// Validate validates the MahkamahAgungConfig
func (c MahkamahAgungConfig) Validate() error {
	if c.PaginationSelector == "" {
		return errors.New("missing pagination selector")
	}
	if c.DetailLinkSelector == "" {
		return errors.New("missing detail link selector")
	}
	return nil
}

// ElitigationSGConfig represents configuration for Singapore Courts
type ElitigationSGConfig struct {
	BaseConfig
	APIToken string `json:"api_token"`
}

// Validate validates the ElitigationSGConfig
func (c ElitigationSGConfig) Validate() error {
	if c.APIToken == "" {
		return errors.New("api token required")
	}
	if c.DetailLinkSelector == "" {
		return errors.New("missing detail link selector")
	}
	return nil
}

// LKPPBlacklistConfig represents configuration for LKPP Blacklist
type LKPPBlacklistConfig struct {
	BaseConfig
	BaseURL         string `json:"base_url"`
	SearchFormURL   string `json:"search_form_url"`
	CompanySelector string `json:"company_selector"`
}

// Validate validates the LKPPBlacklistConfig
func (c LKPPBlacklistConfig) Validate() error {
	if c.BaseURL == "" {
		return errors.New("base URL is required")
	}
	if c.CompanySelector == "" {
		return errors.New("company selector is required")
	}
	return nil
}

// LoadDataSourceConfig loads a data source config based on the config type
func LoadDataSourceConfig(raw json.RawMessage, configType string) (DataSourceConfig, error) {
	switch configType {
	case "mahkamahagung":
		var c MahkamahAgungConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mahkamahagung config: %w", err)
		}
		return c, nil
	case "elitigation":
		var c ElitigationSGConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal elitigation config: %w", err)
		}
		return c, nil
	case "lkpp_blacklist":
		var c LKPPBlacklistConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lkpp_blacklist config: %w", err)
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}
