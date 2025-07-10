package ssc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	mdp "github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	gq "github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

func scrapeOldTemplate(ctx context.Context, e *rod.Element, extraction *repository.Extraction, urlFrontier *repository.UrlFrontier) error {

	select {
	case <-ctx.Done():
		return errors.New("context canceled")
	default:
	}
	log.Info().Msgf("Scraping old template for url: %s", urlFrontier.Url)

	var urlFrontierMetadata UrlFrontierMetadata = UrlFrontierMetadata{}
	err := json.Unmarshal(urlFrontier.Metadata, &urlFrontierMetadata)
	if err != nil {
		log.Error().Err(err).Msg("Error unmarshalling metadata")
		return err
	}

	var metadata Metadata
	if len(extraction.Metadata) > 0 {
		if err := json.Unmarshal(extraction.Metadata, &metadata); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal extraction metadata")
		}
	}

	metadata.CitationNumber = urlFrontierMetadata.CitationNumber
	metadata.Numbers = urlFrontierMetadata.CaseNumbers
	metadata.Classifications = urlFrontierMetadata.Categories
	year, err := time.Parse(time.RFC3339, urlFrontierMetadata.DecisionDate)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing year")
		return err
	}
	metadata.Year = year.Format("2006")
	metadata.DecisionDate = urlFrontierMetadata.DecisionDate
	metadata.Title = urlFrontierMetadata.Title

	infoTable, err := e.Element("#info-table")
	if err != nil {
		log.Error().Err(err).Msg("Error finding info table")
		return err
	}

	row := infoTable.MustElements("tr.info-row")

	for _, r := range row {
		key := r.MustElement("td.txt-label").MustText()
		value := r.MustElement("td.txt-body").MustText()

		if strings.Contains(key, "Tribunal/Court") {
			metadata.JudicalInstitution = value
		}
		if strings.Contains(key, "Coram") {
			metadata.Judges = value
		}
		if strings.Contains(key, "Counsel Name") {
			metadata.Counsel = value
		}
		if strings.Contains(key, "Parties") {
			extractedParties := strings.Split(value, "—")
			var defendant string
			for _, party := range extractedParties {
				if strings.Contains(strings.ToLower(party), "public prosecutor") {
					continue
				}
				defendant = strings.TrimSpace(party)
			}
			metadata.Defendant = defendant
		}

	}

	var rawVerdict []string
	var markdownVerdict []string
	converter := md.NewConverter("", true, nil)
	converter.Use(mdp.GitHubFlavored())

	converter.AddRules(
		md.Rule{
			Filter: []string{"p.Judg-Author"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("** " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judg-Heading-1"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("# " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judg-Heading-2"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("## " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judg-1", "p.Judg-2", "p.Judg-List-1-Item", "p.Judg-List-2-Item"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String(content)
			},
		},

		md.Rule{
			Filter: []string{"p.Judge-Quote-1"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("> " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judge-Quote-2"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String(">> " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judg-QuoteList-2"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("- " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judg-QuoteList-3"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("  - " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Footnote"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("^" + content)

			},
		},
		md.Rule{
			Filter: []string{"p.Judg-List-1-No"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("1. " + content)
			},
		},
		md.Rule{
			Filter: []string{"p.Judg-List-2-No"},
			Replacement: func(content string, selec *gq.Selection, options *md.Options) *string {

				content = strings.TrimSpace(content)
				return md.String("a. " + content)
			},
		},
	)
	content, err := e.Elements("div > p")
	if err != nil {
		log.Error().Err(err).Msg("Error getting content")
		return err
	}
	for _, c := range content {
		rawVerdict = append(rawVerdict, c.MustText())
		html, err := c.HTML()
		if err != nil {
			log.Error().Err(err).Msg("Error getting html")
			continue
		}
		md, err := converter.ConvertString(html)
		if err != nil {
			log.Error().Err(err).Msg("Error converting verdict")
			continue
		}

		markdownVerdict = append(markdownVerdict, md)

	}

	metadata.Verdict = strings.Join(rawVerdict, "\n")
	metadata.VerdictMarkdown = strings.Join(markdownVerdict, "\n")

	extraction.Metadata, err = json.Marshal(metadata)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling metadata")
		return err
	}

	log.Info().Msgf("Scraped old template for url: %s", urlFrontier.Url)

	return nil
}
