package isc

import (
	"fmt"
	stdUrl "net/url"
	"strconv"
	"strings"

	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/rs/zerolog/log"
)

type urlCrawler struct {
	baseURL      string
	sortBy       string
	currentPage  int
	sortOrder    string
	searchPhrase string
}

func (u *urlCrawler) constructURL() string {
	builder := strings.Builder{}
	builder.WriteString(u.baseURL)
	builder.WriteString("?")
	builder.WriteString(fmt.Sprintf("q=%s", u.searchPhrase))
	builder.WriteString(fmt.Sprintf("&page=%d", u.currentPage))
	builder.WriteString(fmt.Sprintf("&obf=%s", u.sortBy))
	builder.WriteString(fmt.Sprintf("&obm=%s", u.sortOrder))
	return builder.String()
}

func (u *urlCrawler) copy() urlCrawler {
	return urlCrawler{
		baseURL:      u.baseURL,
		sortBy:       u.sortBy,
		currentPage:  u.currentPage,
		sortOrder:    u.sortOrder,
		searchPhrase: u.searchPhrase,
	}
}

func newURLCrawler(baseUrl string, config crawler.IndonesiaSupremeCourtConfig, page int) (urlCrawler, error) {
	parsedUrl, err := stdUrl.Parse(baseUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing URL")
		return urlCrawler{}, err
	}

	base := fmt.Sprintf("%s://%s%s", parsedUrl.Scheme, parsedUrl.Host, parsedUrl.Path)

	sortBy := parsedUrl.Query().Get("obf")
	sortOrder := parsedUrl.Query().Get("obm")
	searchPhrase := parsedUrl.Query().Get("q")
	currentPage := parsedUrl.Query().Get("page")

	currentPageInt, err := strconv.Atoi(currentPage)
	if err != nil {
		log.Error().Err(err).Msg("Error converting currentPage to integer")
		return urlCrawler{}, err
	}

	return urlCrawler{
		baseURL:      base,
		sortBy:       sortBy,
		searchPhrase: searchPhrase,
		currentPage:  currentPageInt,
		sortOrder:    sortOrder,
	}, nil
}
func newStartURLCrawler(baseConfig crawler.BaseCrawlerConfig, config crawler.IndonesiaSupremeCourtConfig) (urlCrawler, error) {
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
