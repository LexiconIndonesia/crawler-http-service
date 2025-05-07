# Rod Browser Automation Guide

This guide provides detailed instructions and best practices for using the [Rod](https://github.com/go-rod/rod) browser automation library in your crawler and scraper implementations.

## Table of Contents

- [Rod Browser Automation Guide](#rod-browser-automation-guide)
  - [Table of Contents](#table-of-contents)
  - [Introduction to Rod](#introduction-to-rod)
  - [Basic Setup](#basic-setup)
  - [Navigation](#navigation)
  - [Element Selection](#element-selection)
  - [Extracting Data](#extracting-data)
  - [Handling Forms](#handling-forms)
  - [Handling Pagination](#handling-pagination)
  - [Dealing with JavaScript](#dealing-with-javascript)
  - [Error Handling](#error-handling)
  - [Performance Optimization](#performance-optimization)
  - [Debugging](#debugging)
  - [Conclusion](#conclusion)

## Introduction to Rod

Rod is a high-level driver library for [Devtools Protocol](https://chromedevtools.github.io/devtools-protocol/). It's designed to be simple, fluent, and chainable.

Key benefits:
- No external dependencies (like Selenium)
- Supports headless and visible browser modes
- Provides a fluent, chainable API
- Supports concurrent operations

## Basic Setup

In our crawler framework, Rod is typically set up in the `Setup` method of your crawler/scraper:

```go
// Example of setting up Rod in a crawler
func (c *YourCrawler) Setup(ctx context.Context) error {
    // Create a launcher
    launcher := launcher.New().
        Headless(true).
        Set("disable-gpu").
        Set("no-sandbox").
        Set("disable-setuid-sandbox")

    if c.Config.ProxyURL != "" {
        launcher.Set("proxy-server", c.Config.ProxyURL)
    }

    // Launch the browser
    browser, err := rod.New().
        ControlURL(launcher.MustLaunch()).
        Context(ctx).
        Timeout(c.Config.RequestTimeout).
        Connect()
    if err != nil {
        return fmt.Errorf("failed to connect to browser: %w", err)
    }

    c.Browser = browser
    return nil
}
```

Don't forget to clean up in the `Teardown` method:

```go
func (c *YourCrawler) Teardown(ctx context.Context) error {
    if c.Browser != nil {
        return c.Browser.Close()
    }
    return nil
}
```

## Navigation

For navigating to pages:

```go
func (c *YourCrawler) Navigate(ctx context.Context, url string) (*rod.Page, error) {
    if c.Browser == nil {
        return nil, errors.New("browser not initialized")
    }

    // Create a new page
    page, err := c.Browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
    if err != nil {
        return nil, fmt.Errorf("failed to create page: %w", err)
    }

    // Set user agent if needed
    if c.Config.UserAgent != "" {
        err = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
            UserAgent: c.Config.UserAgent,
        })
        if err != nil {
            return nil, fmt.Errorf("failed to set user agent: %w", err)
        }
    }

    // Navigate to the URL
    err = page.Context(ctx).Navigate(url)
    if err != nil {
        return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
    }

    // Wait for the page to load
    err = page.WaitLoad()
    if err != nil {
        return nil, fmt.Errorf("failed to wait for page load: %w", err)
    }

    return page, nil
}
```

## Element Selection

Rod provides several methods to select elements:

```go
// Select by CSS selector
element, err := page.Element("div.main-content")

// Select multiple elements
elements, err := page.Elements("a.result-link")

// Select by XPath
element, err := page.ElementX("//div[@class='result-item']")

// Chained selection
element, err := page.Element("div.results").Element("a.first-result")
```

Always check if the element exists before using it:

```go
element, err := page.Element(".pagination")
if err != nil {
    return fmt.Errorf("pagination element not found: %w", err)
}
```

## Extracting Data

Once you have elements, you can extract various data:

```go
// Extract text content
text, err := element.Text()

// Extract attribute value
href, err := element.Attribute("href")

// Convert href to absolute URL if needed
if href != nil {
    absoluteURL, err := page.Info().URL.Parse(*href)
    if err == nil {
        urlStr := absoluteURL.String()
        // Use urlStr
    }
}

// Extract HTML content
html, err := element.HTML()

// Get property
value, err := element.Property("value")
```

## Handling Forms

For interacting with forms:

```go
// Fill input fields
err := page.Element("#search-input").Input("your search term")

// Click buttons
err := page.Element("#search-button").Click()

// Select dropdown options
err := page.Element("select#sort-by").Select([]string{"date-desc"})

// Check/uncheck checkboxes
err := page.Element("input[type=checkbox]").Click()
```

## Handling Pagination

A common pattern for handling pagination:

```go
func (c *YourCrawler) handlePagination(ctx context.Context, page *rod.Page) error {
    var frontiers []repository.UrlFrontier

    // Process first page
    pageResults, err := c.ExtractElements(ctx, page)
    if err != nil {
        return err
    }
    frontiers = append(frontiers, pageResults...)

    // Get pagination information
    paginationElement, err := page.Element(c.Config.PaginationSelector)
    if err != nil {
        // No pagination or pagination element not found
        return nil
    }

    // Extract number of pages
    pageLinks, err := paginationElement.Elements("a")
    if err != nil {
        return err
    }

    var maxPage int = 1
    for _, link := range pageLinks {
        text, err := link.Text()
        if err != nil {
            continue
        }

        pageNum, err := strconv.Atoi(text)
        if err != nil {
            continue
        }

        if pageNum > maxPage {
            maxPage = pageNum
        }
    }

    // Limit to config max pages
    if c.Config.MaxPages > 0 && maxPage > c.Config.MaxPages {
        maxPage = c.Config.MaxPages
    }

    // Process remaining pages
    for i := 2; i <= maxPage; i++ {
        // Construct URL for next page
        nextPageURL := fmt.Sprintf("%s?page=%d", baseURL, i)

        // Navigate to next page
        err = page.Navigate(nextPageURL)
        if err != nil {
            return err
        }

        // Wait for the page to load
        err = page.WaitLoad()
        if err != nil {
            return err
        }

        // Wait for a bit to avoid rate limiting
        if c.Config.Delay > 0 {
            time.Sleep(time.Duration(c.Config.Delay) * time.Millisecond)
        }

        // Extract results from this page
        pageResults, err := c.ExtractElements(ctx, page)
        if err != nil {
            return err
        }
        frontiers = append(frontiers, pageResults...)
    }

    // Save results to database
    // ...

    return nil
}
```

## Dealing with JavaScript

For websites with heavy JavaScript:

```go
// Wait for specific element to appear
element, err := page.Timeout(30 * time.Second).Element("#dynamic-content")

// Wait for network to be idle
err := page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)

// Wait for specific condition
err := page.WaitFunction(`() => document.querySelectorAll('.result-item').length > 0`)

// Execute JavaScript
result, err := page.Eval(`() => {
    // Custom JavaScript to extract data
    const items = [];
    document.querySelectorAll('.result-item').forEach(el => {
        items.push({
            title: el.querySelector('.title').textContent,
            url: el.querySelector('a').href
        });
    });
    return items;
}`)
```

## Error Handling

Implement robust error handling:

```go
// Retry mechanism for flaky elements
func (c *YourCrawler) retryGetElement(page *rod.Page, selector string, maxRetries int) (*rod.Element, error) {
    var element *rod.Element
    var err error

    for i := 0; i < maxRetries; i++ {
        element, err = page.Element(selector)
        if err == nil {
            return element, nil
        }

        // Wait before retry
        time.Sleep(500 * time.Millisecond)
    }

    return nil, fmt.Errorf("failed to get element %s after %d retries: %w", selector, maxRetries, err)
}

// Handle timeouts
func (c *YourCrawler) safeNavigate(ctx context.Context, url string) (*rod.Page, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, c.Config.RequestTimeout)
    defer cancel()

    var page *rod.Page
    var err error

    done := make(chan struct{})
    go func() {
        page, err = c.Navigate(timeoutCtx, url)
        close(done)
    }()

    select {
    case <-done:
        return page, err
    case <-timeoutCtx.Done():
        return nil, fmt.Errorf("navigation to %s timed out after %v", url, c.Config.RequestTimeout)
    }
}
```

## Performance Optimization

Tips for better performance:

```go
// Disable images and CSS for faster loading
browser := rod.New().
    MustConnect().
    MustIncognito().
    MustPage()

router := browser.HijackRequests()
defer router.MustStop()

router.MustAdd("*", func(ctx *rod.Hijack) {
    // Skip loading images, CSS, fonts, etc.
    if ctx.Request.Type() == proto.NetworkResourceTypeImage ||
       ctx.Request.Type() == proto.NetworkResourceTypeStylesheet ||
       ctx.Request.Type() == proto.NetworkResourceTypeFont {
        ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
        return
    }
    ctx.ContinueRequest(&proto.FetchContinueRequest{})
})

go router.Run()
```

## Debugging

Rod provides excellent debugging capabilities:

```go
// Show the browser window for debugging
launcher := launcher.New().
    Headless(false).
    Devtools(true)

// Trace browser actions
page := rod.New().
    MustConnect().
    Trace(true).
    MustPage()

// Take screenshots
err := page.Screenshot("screenshot.png")

// Print page as PDF
err := page.PDF("page.pdf")

// Log page console messages
go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
    log.Println("Console:", e)
})
```

For detailed debugging, consider using the Rod DevTools:

```go
// Enable the Rod DevTools UI
page.MustNavigate("https://example.org")
page.MustWaitStable()

s := page.MustScreenshot()
rod.Try(func() {
    // This will launch a browser window with a debugging interface
    rod.NewPage().MustElement("body").MustWaitLoad()
})
```

## Conclusion

Rod is a powerful library for browser automation that provides all the tools needed for effective web crawling and scraping. By following the patterns and practices in this guide, you can create robust, efficient crawlers and scrapers for any website.

Remember, when developing with our crawler framework, you only need to implement the stubs with "not implemented" errors as per the requirements. The actual implementation would follow these patterns when needed.
