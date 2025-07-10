package isc

// func extractBreadCrumbs(p *rod.Element) ([]string, error) {
// 	breadcrumbs := []string{}
// 	breadcrumbsElement, err := p.Elements("div:nth-child(1) > a")
// 	if err != nil {
// 		return []string{}, err
// 	}

// 	for _, breadcrumb := range breadcrumbsElement {
// 		breadcrumbText, err := breadcrumb.Text()
// 		if err != nil {
// 			return []string{}, err
// 		}
// 		breadcrumbs = append(breadcrumbs, breadcrumbText)
// 	}
// 	return breadcrumbs, nil
// }

// func extractDecisionDate(p *rod.Element) (time.Time, error) {
// 	dateDiv, err := p.Element("div:nth-child(2)")
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to find date div: %w", err)
// 	}

// 	text, err := dateDiv.Text()
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to get text from date div: %w", err)
// 	}

// 	// For debugging
// 	// fmt.Printf("Date div text: %s\n", text)

// 	// Extract the date part after "Putus :"
// 	putusIndex := strings.Index(text, "Putus :")
// 	if putusIndex == -1 {
// 		// Putus date not found, which is acceptable
// 		return time.Time{}, nil
// 	}

// 	// Extract the date string
// 	dateStr := strings.TrimSpace(text[putusIndex+8:])
// 	dateStr = strings.Split(dateStr, " —")[0]

// 	// Split by newline if present and take only the first line
// 	if strings.Contains(dateStr, "\n") {
// 		dateStr = strings.Split(dateStr, "\n")[0]
// 	}
// 	dateStr = strings.TrimSpace(dateStr)

// 	// For debugging
// 	// fmt.Printf("Extracted decision date string: %s\n", dateStr)

// 	// Parse the date in format DD-MM-YYYY
// 	parsedTime, err := time.Parse("02-01-2006", dateStr)
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to parse decision date '%s': %w", dateStr, err)
// 	}

// 	return parsedTime, nil
// }

// func extractUploadDate(p *rod.Element) (time.Time, error) {
// 	dateDiv, err := p.Element("div:nth-child(2)")
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to find date div: %w", err)
// 	}

// 	text, err := dateDiv.Text()
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to get text from date div: %w", err)
// 	}

// 	// For debugging
// 	// fmt.Printf("Date div text: %s\n", text)

// 	// Extract the date part after "Upload :"
// 	uploadIndex := strings.Index(text, "Upload :")
// 	if uploadIndex == -1 {
// 		// Upload date not found, which is acceptable
// 		return time.Time{}, nil
// 	}

// 	// Extract the date string
// 	dateStr := strings.TrimSpace(text[uploadIndex+9:])
// 	// Split by dash if present (but usually upload date is the last one)
// 	if strings.Contains(dateStr, " —") {
// 		dateStr = strings.Split(dateStr, " —")[0]
// 	}

// 	// Split by newline if present and take only the first line
// 	if strings.Contains(dateStr, "\n") {
// 		dateStr = strings.Split(dateStr, "\n")[0]
// 	}
// 	dateStr = strings.TrimSpace(dateStr)

// 	// For debugging
// 	// fmt.Printf("Extracted upload date string: %s\n", dateStr)

// 	// Parse the date in format DD-MM-YYYY
// 	parsedTime, err := time.Parse("02-01-2006", dateStr)
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to parse upload date '%s': %w", dateStr, err)
// 	}

// 	return parsedTime, nil
// }

// func extractRegistrationDate(p *rod.Element) (time.Time, error) {
// 	dateDiv, err := p.Element("div:nth-child(2)")
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to find date div: %w", err)
// 	}

// 	text, err := dateDiv.Text()
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to get text from date div: %w", err)
// 	}

// 	// For debugging
// 	// fmt.Printf("Date div text: %s\n", text)

// 	// Extract the date part after "Register :"
// 	registerIndex := strings.Index(text, "Register :")
// 	if registerIndex == -1 {
// 		// Register date not found, which is acceptable
// 		return time.Time{}, nil
// 	}

// 	// Extract the date string
// 	dateStr := strings.TrimSpace(text[registerIndex+11:])
// 	dateStr = strings.Split(dateStr, " —")[0]

// 	// Split by newline if present and take only the first line
// 	if strings.Contains(dateStr, "\n") {
// 		dateStr = strings.Split(dateStr, "\n")[0]
// 	}
// 	dateStr = strings.TrimSpace(dateStr)

// 	// For debugging
// 	// fmt.Printf("Extracted registration date string: %s\n", dateStr)

// 	// Parse the date in format DD-MM-YYYY
// 	parsedTime, err := time.Parse("02-01-2006", dateStr)
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("failed to parse registration date '%s': %w", dateStr, err)
// 	}

// 	return parsedTime, nil
// }

// func extractTitle(p *rod.Element) (string, error) {
// 	titleElement, err := p.Element("strong > a")
// 	if err != nil {
// 		return "", err
// 	}

// 	title, err := titleElement.Text()
// 	if err != nil {
// 		return "", err
// 	}
// 	return title, nil
// }

// func extractUrl(p *rod.Element) (string, error) {
// 	urlElement, err := p.Element("strong > a")
// 	if err != nil {
// 		return "", err
// 	}

// 	url, err := urlElement.Attribute("href")
// 	if err != nil {
// 		return "", err
// 	}
// 	return *url, nil
// }

// func removeViewDownloadHTML(text string) string {
// 	// Remove content with i tags related to view/download icons
// 	re := regexp.MustCompile(`<i\s+class=["']icon-(eye|download).*?</i>`)
// 	text = re.ReplaceAllString(text, "")

// 	// Remove strong tags with view/download counts
// 	re = regexp.MustCompile(`<strong\s+title=["']Jumlah (view|download)["']>.*?</strong>`)
// 	text = re.ReplaceAllString(text, "")

// 	// Clean up any remaining dash patterns that might be left after removing the HTML
// 	text = strings.ReplaceAll(text, " — ", "")
// 	text = strings.ReplaceAll(text, "—", "")

// 	// Remove any multiple spaces that might be created
// 	re = regexp.MustCompile(`\s+`)
// 	text = re.ReplaceAllString(text, " ")

// 	return strings.TrimSpace(text)
// }

// func stripHTML(html string) string {
// 	// Remove HTML tags with regex for better handling
// 	re := regexp.MustCompile(`<[^>]*?>`)
// 	html = re.ReplaceAllString(html, "")
// 	return html
// }

// func isOnlyViewDownloadCount(text string) bool {
// 	// Trim spaces
// 	text = strings.TrimSpace(text)

// 	// Check with regex for more precise pattern matching
// 	re := regexp.MustCompile(`^\s*\d+\s*[—-]\s*\d+\s*$`)
// 	return re.MatchString(text)
// }

// func removeViewDownloadCounts(text string) string {
// 	// Remove view/download counts with more precise regex
// 	re := regexp.MustCompile(`\s*\d+\s*[—-]\s*\d+\s*$`)
// 	return strings.TrimSpace(re.ReplaceAllString(text, ""))
// }
