package ssc

import (
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

func getTitle(e *rod.Element) (string, error) {
	title, err := e.Element("a.h5.gd-heardertext")
	if err != nil {
		log.Error().Err(err).Msg("Error getting title")
		return "", err
	}

	titleText := title.MustText()
	return strings.TrimSpace(titleText), nil
}

func getCaseNumbers(e *rod.Element) ([]string, error) {
	caseNumbers, err := e.Elements("a.case-num-link")
	if err != nil {
		log.Error().Err(err).Msg("Error getting case numbers")
		return []string{}, err
	}

	if len(caseNumbers) == 0 {
		return []string{}, nil
	}

	numbers := make([]string, len(caseNumbers))
	for i, number := range caseNumbers {
		numbers[i] = strings.TrimSpace(number.MustText())
	}

	return numbers, nil
}

func getCitationNumber(e *rod.Element) (string, error) {
	citationNumber, err := e.Element("a.citation-num-link")
	if err != nil {
		log.Error().Err(err).Msg("Error getting citation number")
		return "", err
	}

	return strings.TrimSpace(strings.ReplaceAll(citationNumber.MustText(), "|", "")), nil
}

func getDecisionDate(e *rod.Element) (string, error) {
	stringDate := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(e.MustElement("a.decision-date-link").MustText(), "Decision Date:", ""), "|", ""))
	decisionDate, err := time.Parse("2 Jan 2006", stringDate)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing decision date")
		return "", err
	}
	return decisionDate.Format(time.RFC3339), nil
}

func getCategories(e *rod.Element) ([]string, error) {
	categories, err := e.Elements("div.gd-catchword-container > a")
	if err != nil {
		log.Error().Err(err).Msg("Error getting categories")
		return []string{}, err
	}

	if len(categories) == 0 {
		return []string{}, nil
	}

	names := make([]string, len(categories))
	for i, category := range categories {
		names[i] = strings.TrimSpace(removeBrackets(category.MustText()))
	}

	return names, nil
}

func removeBrackets(input string) string {
	return strings.ReplaceAll(strings.ReplaceAll(input, "[", ""), "]", "")
}
