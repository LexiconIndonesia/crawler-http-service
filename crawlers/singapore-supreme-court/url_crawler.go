package ssc

import (
	"fmt"
	stdUrl "net/url"
	"strings"

	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/rs/zerolog/log"
)

type urlCrawler struct {
	baseURL        string
	config         crawler.SingaporeSupremeCourtConfig
	filter         string
	yearOfDecision string
	sortBy         string
	currentPage    int
	sortAscending  string
	searchPhrase   string
	verbose        string
}

func (u *urlCrawler) constructURL() string {
	builder := strings.Builder{}
	filterKey := u.config.ListQueryParam.Filter.Key
	sortKey := u.config.ListQueryParam.Sort.Key
	orderKey := u.config.ListQueryParam.Order.Key
	pageKey := u.config.ListQueryParam.Page
	queryKey := u.config.ListQueryParam.Query
	verboseKey := u.config.ListQueryParam.Verbose.Key

	builder.WriteString(u.baseURL)
	builder.WriteString("?")
	builder.WriteString(fmt.Sprintf("%s=%s", filterKey, u.filter))
	builder.WriteString(fmt.Sprintf("&%s=%s", sortKey, u.sortBy))
	builder.WriteString(fmt.Sprintf("&%s=%d", pageKey, u.currentPage))
	builder.WriteString(fmt.Sprintf("&%s=%s", orderKey, u.sortAscending))
	builder.WriteString(fmt.Sprintf("&%s=%s", queryKey, u.searchPhrase))
	builder.WriteString(fmt.Sprintf("&%s=%s", verboseKey, u.verbose))

	return builder.String()
}

func (u *urlCrawler) copy() urlCrawler {
	return urlCrawler{
		config:         u.config,
		baseURL:        u.baseURL,
		filter:         u.filter,
		yearOfDecision: u.yearOfDecision,
		sortBy:         u.sortBy,
		sortAscending:  u.sortAscending,
		searchPhrase:   u.searchPhrase,
		verbose:        u.verbose,
		currentPage:    u.currentPage,
	}
}

func getSearchPhrase(url string) (string, error) {
	parsedURL, err := stdUrl.Parse(url)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing URL")
		return "", err
	}

	queryParams := parsedURL.Query()
	searchPhrase := queryParams.Get("searchPhrase")
	return searchPhrase, nil
}

func newURLCrawler(baseURL string, config crawler.SingaporeSupremeCourtConfig, page int) (urlCrawler, error) {
	parsedURL, err := stdUrl.Parse(baseURL)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing URL")
		return urlCrawler{}, err
	}

	base := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)

	filter := config.ListQueryParam.Filter.Value
	sortBy := config.ListQueryParam.Sort.Value
	order := config.ListQueryParam.Order.Value
	yearOfDecision := config.ListQueryParam.Filter.Value
	searchPhrase := config.ListQueryParam.Query
	verbose := config.ListQueryParam.Verbose.Value

	return urlCrawler{
		config:         config,
		baseURL:        base,
		filter:         filter,
		yearOfDecision: yearOfDecision,
		sortBy:         sortBy,
		sortAscending:  order,
		searchPhrase:   searchPhrase,
		verbose:        verbose,
		currentPage:    page,
	}, nil
}

func newStartURLCrawler(baseConfig crawler.BaseCrawlerConfig, config crawler.SingaporeSupremeCourtConfig) (urlCrawler, error) {
	firstURL := fmt.Sprintf("%s%s", baseConfig.DataSource.BaseUrl.String, config.ListPath)

	return newURLCrawler(firstURL, config, 1)
}

func generateUrls(url urlCrawler, startPage int, endPage int) []urlCrawler {
	urls := []urlCrawler{}
	for i := startPage; i <= endPage; i++ {
		newURLCrawler := url.copy()
		newURLCrawler.currentPage = i
		urls = append(urls, newURLCrawler)
	}
	return urls
}
