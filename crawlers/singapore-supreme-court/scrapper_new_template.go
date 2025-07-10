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

func scrapeNewTemplate(ctx context.Context, e *rod.Element, extraction *repository.Extraction, urlFrontier *repository.UrlFrontier) error {

	select {
	case <-ctx.Done():
		return errors.New("context canceled")
	default:
	}
	log.Info().Msgf("Scraping new template for url: %s", urlFrontier.Url)

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
	splittedTitle := strings.Split(metadata.Title, " v ")

	for _, part := range splittedTitle {
		if strings.Contains(strings.ToLower(part), "public prosecutor") {
			continue
		}

		metadata.Defendant = strings.TrimSpace(part)
	}
	corams, err := e.Elements("div.HN-Coram")
	if err != nil {
		log.Error().Err(err).Msg("Error getting coram")
		return err
	}

	for _, coram := range corams {
		splittedCoram := strings.Split(coram.MustText(), "\n")

		for i, part := range splittedCoram {
			if i == 0 {
				courtSplit := strings.Split(part, "—")
				metadata.JudicalInstitution = strings.TrimSpace(courtSplit[0])
				continue
			}
			if i == 1 {
				metadata.Judges = strings.TrimSpace(part)
				continue
			}

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

	verdicts, err := e.Elements("div.col.col-md-12.align-self-center")
	if err != nil {
		log.Error().Err(err).Msg("Error getting verdicts")
		return err
	}

	for _, verdict := range verdicts {

		rawVerdict = append(rawVerdict, verdict.MustText())
		html, err := verdict.HTML()
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
	log.Info().Msgf("Scraped new template for url: %s", urlFrontier.Url)

	return nil
}
