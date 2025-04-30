package indonesia_supreme_court

import (
	"fmt"
	stdUrl "net/url"
	"strconv"

	"github.com/rs/zerolog/log"
)

type urlCrawler struct {
	BaseUrl      string
	SortBy       string
	CurrentPage  int
	SortOrder    string
	SearchPhrase string
}

func (u *urlCrawler) constructUrl() string {
	return fmt.Sprintf("%s?q=%s&page=%d&obf=%s&obm=%s", u.BaseUrl, u.SearchPhrase, u.CurrentPage, u.SortBy, u.SortOrder)
}

func (u *urlCrawler) copy() urlCrawler {
	return urlCrawler{
		BaseUrl:      u.BaseUrl,
		SortBy:       u.SortBy,
		CurrentPage:  u.CurrentPage,
		SortOrder:    u.SortOrder,
		SearchPhrase: u.SearchPhrase,
	}

}

func newUrlCrawler(baseUrl string) (urlCrawler, error) {
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
		BaseUrl:      base,
		SortBy:       sortBy,
		SearchPhrase: searchPhrase,
		CurrentPage:  currentPageInt,
		SortOrder:    sortOrder,
	}, nil

}
