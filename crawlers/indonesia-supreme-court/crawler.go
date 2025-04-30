package indonesia_supreme_court

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdUrl "net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	crawler "github.com/adryanev/go-http-service-template/crawlers"
	"github.com/adryanev/go-http-service-template/repository"
	"github.com/go-rod/rod"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type Crawler interface {
	Setup()
	Teardown()
	CrawlAll(ctx context.Context) error
	Crawl(ctx context.Context, url string) error
	Consume(msg jetstream.Msg) error
}

func NewCrawler() Crawler {
	return &CrawlerImpl{}
}

type CrawlerImpl struct {
	browser    *rod.Browser
	service    *crawler.CrawlerService
	dataSource *repository.DataSource
}

func (c *CrawlerImpl) Setup() {
	c.browser = rod.New().MustConnect()
}

func (c *CrawlerImpl) Teardown() {
	c.browser.MustClose()
}
func (c *CrawlerImpl) CrawlAll(ctx context.Context) error {

	keyword := "korupsi"
	startPage := 1
	sortBy := "TANGGAL_PUTUS"
	sortOrder := "desc"

	startUrl := fmt.Sprintf("https://putusan3.mahkamahagung.go.id/search.html?q=%s&page=%d&obf=%s&obm=%s", keyword, startPage, sortBy, sortOrder)

	pagePool := rod.NewPagePool(10)
	defer pagePool.Cleanup(func(p *rod.Page) {
		err := p.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing page")
		}
	})

	create := func() (*rod.Page, error) {

		incognito, err := c.browser.Incognito()
		if err != nil {
			log.Error().Err(err).Msg("Error creating incognito page")
			return nil, err
		}
		return incognito.MustPage(), nil

	}

	createListOfUrl := func(urlCrawler urlCrawler, startPage int, endPage int) []string {
		urls := []string{}
		for i := startPage; i <= endPage; i++ {
			newUrlCrawler := urlCrawler.copy()
			newUrlCrawler.CurrentPage = i
			urls = append(urls, newUrlCrawler.constructUrl())
		}
		return urls
	}

	job := func(urlPage string) error {

		page, err := pagePool.Get(create)
		if err != nil {
			log.Error().Err(err).Msg("Error getting page")
			return err
		}
		defer pagePool.Put(page)
		return c.crawlUrl(ctx, page, urlPage)

	}
	parsedUrl, err := stdUrl.Parse(startUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing URL")
		return err
	}
	currentPage := parsedUrl.Query().Get("currentPage")
	if currentPage == "" {
		currentPage = "1"
	}
	currentPageInt, err := strconv.Atoi(currentPage)
	if err != nil {
		log.Error().Err(err).Msg("Error converting currentPage to integer")
		return err
	}

	rpLast, err := create()
	if err != nil {
		return fmt.Errorf("failed to create page for last page check: %w", err)
	}
	defer rpLast.Close()

	lastPage, err := getLastPage(ctx, rpLast, startUrl)
	if err != nil {
		return fmt.Errorf("failed to get last page: %w", err)
	}

	lastPageInt, totalResult := lastPage.Unpack()

	log.Info().Msg("Total result: " + strconv.Itoa(totalResult))

	urlCrawler, err := newUrlCrawler(startUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error creating url crawler")
		return err
	}

	urlList := createListOfUrl(urlCrawler, currentPageInt, lastPageInt)

	chunks := lo.Chunk(urlList, 1)

	var crawlErrors []error
	errChan := make(chan error, len(urlList)) // Channel to collect errors

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all resources are cleaned up

	for _, urls := range chunks {
		// Check if context is cancelled before starting new chunk
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg := sync.WaitGroup{}

		// Process all URLs in the chunk concurrently
		for _, url := range urls {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()

				// Create a new context for this goroutine
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				default:
					if err := job(url); err != nil {
						errChan <- fmt.Errorf("error crawling %s: %w", url, err)
						// Optional: cancel other goroutines if you want to stop on first error
						// cancel()
					}
				}
			}(url)
		}

		wg.Wait()

	}

	// Collect any errors that occurred
	for err := range errChan {
		if err != nil {
			if err == context.Canceled {
				return fmt.Errorf("crawling was cancelled: %w", err)
			}
			crawlErrors = append(crawlErrors, err)
		}
	}

	// If there were any errors, return them combined
	if len(crawlErrors) > 0 {
		return fmt.Errorf("encountered %d errors during crawling: %v", len(crawlErrors), crawlErrors)
	}

	close(errChan)

	return nil
}

func (c *CrawlerImpl) Crawl(ctx context.Context, url string) error {
	page := c.browser.MustPage(url)

	defer page.Close()
	return c.crawlUrl(ctx, page, url)
}

func (c *CrawlerImpl) Consume(msg jetstream.Msg) error {
	return nil
}

func getLastPage(ctx context.Context, rp *rod.Page, url string) (lo.Tuple2[int, int], error) {
	lastPage := 0

	if err := rp.Context(ctx).Navigate(url); err != nil {
		if err == context.Canceled {
			return lo.Tuple2[int, int]{}, err
		}
		log.Error().Err(err).Msg("Error navigating to url")
		return lo.Tuple2[int, int]{}, err
	}
	totalResult := 0
	totalResultElement, err := rp.Element("#content > div > div.postcontent.nobottommargin.col_three_fifth.col_last > div:nth-child(1) > div.col-md-7 > div > h2")
	if err != nil {
		log.Error().Err(err).Msg("Error getting total result element")
		return lo.Tuple2[int, int]{}, err
	}
	res, err := totalResultElement.HTML()
	if err != nil {
		log.Error().Err(err).Msg("Error getting total result element")
		return lo.Tuple2[int, int]{}, err
	}
	totalResult, err = strconv.Atoi(regexp.MustCompile(`\d+`).FindString(res))
	if err != nil {
		log.Error().Err(err).Msg("Error converting total result to integer")
		return lo.Tuple2[int, int]{}, err
	}

	elements, err := rp.Elements("#content > div > div.postcontent.nobottommargin.col_three_fifth.col_last > strong > div.pagging.text-center > nav > ul > li:nth-child(6) > a.page-link")
	if err != nil {
		log.Error().Err(err).Msg("Error getting elements")
		return lo.Tuple2[int, int]{}, err
	}
	for _, element := range elements {
		if element.MustHTML() == "" {
			continue
		}
		href := element.MustAttribute("href")
		if href == nil {
			continue
		}
		u, err := stdUrl.Parse(*href)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing URL")
			continue
		}
		lp := u.Query().Get("page")
		lpInt, err := strconv.Atoi(lp)
		if err != nil {
			log.Error().Err(err).Msg("Error converting LP to integer")
			continue
		}
		if lastPage < lpInt {
			lastPage = lpInt
		}
	}

	return lo.Tuple2[int, int]{
		A: lastPage,
		B: totalResult,
	}, nil

}

func (c *CrawlerImpl) crawlUrl(ctx context.Context, rp *rod.Page, url string) error {
	log.Info().Msg("Crawling URL: " + url)

	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rpCtx := rp.Context(ctx)
	wait := rpCtx.MustWaitNavigation()
	err := rpCtx.Navigate(url)
	if err != nil {
		log.Error().Err(err).Msg("Error navigating to url")
		return err
	}
	wait()

	// Check context after navigation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	x := rpCtx.MustWaitStable()

	allDetails := []repository.UrlFrontier{}
	// find current page links
	elements, err := x.Elements("#content > div > div.postcontent.nobottommargin.col_three_fifth.col_last > div.spost.clearfix")
	if err != nil {
		log.Error().Err(err).Msg("Error getting elements")
		return err
	}
	log.Info().Msgf("Url %s found %d elements", url, len(elements))

	log.Info().Msgf("Start Extracting Data from url %s", url)
	for _, element := range elements {
		detailElement := element.MustElement("div.entry-c")
		detailUrl, err := extractUrl(detailElement)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting url")
		}
		breadcrumbs, err := extractBreadCrumbs(element)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting breadcrumbs")
		}
		title, err := extractTitle(element)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting title")
		}
		decisionDate, err := extractDecisionDate(element)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting decision date")
		}
		uploadDate, err := extractUploadDate(element)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting upload date")
		}
		registrationDate, err := extractRegistrationDate(element)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting registration date")
		}

		id := sha256.Sum256([]byte(url))
		currentTime := time.Now()
		metadata := UrlFrontierMetadata{
			Title:            title,
			DecisionDate:     decisionDate.Format(time.RFC3339),
			UploadDate:       uploadDate.Format(time.RFC3339),
			RegistrationDate: registrationDate.Format(time.RFC3339),
			Breadcrumbs:      breadcrumbs,
		}

		// json marshal metadata
		metadataJson, err := json.Marshal(metadata)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling metadata")
		}

		allDetails = append(allDetails, repository.UrlFrontier{
			ID:  hex.EncodeToString(id[:]),
			Url: detailUrl,

			Domain:    c.dataSource.BaseUrl.String,
			Status:    int16(crawler.URL_FRONTIER_STATUS_NEW),
			Metadata:  metadataJson,
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
		})
	}
	log.Info().Msgf("Extracted %d urls", len(allDetails))

	log.Info().Msgf("Upserting %d urls", len(allDetails))
	// err = services.UpsertUrl(ctx, allDetails)

	if err != nil {
		log.Error().Err(err).Msg("Error upserting url")
	}

	log.Info().Msgf("Crawling url: %s done!", url)
	return nil
}
