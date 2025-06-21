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
func UnmarshalIndonesiaSupremeCourtConfig(data []byte) (IndonesiaSupremeCourtConfig, error) {
	var r IndonesiaSupremeCourtConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *IndonesiaSupremeCourtConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type IndonesiaSupremeCourtConfig struct {
	ListPath       string                  `json:"list_path"`
	SortMethods    []string                `json:"sort_methods"`
	SortOptions    []string                `json:"sort_options"`
	ListQueryParam IndonesiaListQueryParam `json:"list_query_params"`
	MinDelayMs     int                     `json:"min_delay_ms"`
	MaxDelayMs     int                     `json:"max_delay_ms"`
}

func (c IndonesiaSupremeCourtConfig) Validate() error {
	if c.ListPath == "" {
		return errors.New("list path is required")
	}
	if c.ListQueryParam.Page == "" {
		return errors.New("page is required")
	}
	if c.ListQueryParam.Sort == "" {
		return errors.New("sort is required")
	}
	if c.ListQueryParam.Query == "" {
		return errors.New("query is required")
	}
	if c.ListQueryParam.SortBy == "" {
		return errors.New("sort by is required")
	}
	if c.ListQueryParam.Category == "" {
		return errors.New("category is required")
	}
	if c.ListQueryParam.SortOrder == "" {
		return errors.New("sort order is required")
	}
	if c.ListQueryParam.SortOrderBy == "" {
		return errors.New("sort order by is required")
	}
	if c.ListQueryParam.CategoryValue == "" {
		return errors.New("category value is required")
	}
	return nil
}

type IndonesiaListQueryParam struct {
	Page          string `json:"page"`
	Sort          string `json:"sort"`
	Query         string `json:"query"`
	SortBy        string `json:"sort_by"`
	Category      string `json:"category"`
	SortOrder     string `json:"sort_order"`
	SortOrderBy   string `json:"sort_order_by"`
	CategoryValue string `json:"category_value"`
}

func UnmarshalSingaporeSupremeCourtConfig(data []byte) (SingaporeSupremeCourtConfig, error) {
	var r SingaporeSupremeCourtConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *SingaporeSupremeCourtConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type SingaporeSupremeCourtConfig struct {
	ListPath       string                  `json:"list_path"`
	SortMethods    []string                `json:"sort_methods"`
	SortOptions    []string                `json:"sort_options"`
	ListQueryParam SingaporeListQueryParam `json:"list_query_param"`
	MinDelayMs     int                     `json:"min_delay_ms"`
	MaxDelayMs     int                     `json:"max_delay_ms"`
}

func (c SingaporeSupremeCourtConfig) Validate() error {
	if c.ListPath == "" {
		return errors.New("list path is required")
	}
	if c.ListQueryParam.Page == "" {
		return errors.New("page is required")
	}
	if c.ListQueryParam.Query == "" {
		return errors.New("query is required")
	}
	if c.ListQueryParam.Filter.Key == "" {
		return errors.New("filter key is required")
	}
	if c.ListQueryParam.Filter.Value == "" {
		return errors.New("filter value is required")
	}
	if c.ListQueryParam.Order.Key == "" {
		return errors.New("order key is required")
	}
	if c.ListQueryParam.Order.Value == "" {
		return errors.New("order value is required")
	}
	if c.ListQueryParam.Verbose.Key == "" {
		return errors.New("verbose key is required")
	}
	if c.ListQueryParam.Verbose.Value == "" {
		return errors.New("verbose value is required")
	}

	return nil
}

type SingaporeListQueryParam struct {
	Page    string   `json:"page"`
	Query   string   `json:"query"`
	Sort    KeyValue `json:"sort"`
	Order   KeyValue `json:"order"`
	Filter  KeyValue `json:"filter"`
	Verbose KeyValue `json:"verbose"`
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
	case "indonesia-supreme-court":
		var c IndonesiaSupremeCourtConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mahkamahagung config: %w", err)
		}
		return c, nil
	case "singapore-supreme-court":
		var c SingaporeSupremeCourtConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal elitigation config: %w", err)
		}
		return c, nil
	case "lkpp-blacklist":
		var c LKPPBlacklistConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lkpp-blacklist config: %w", err)
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
