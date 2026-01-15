You are working on a distributed crawler.

The parser_worker already parses HTML using goquery and extracts data via ExtractionSpec (Fields, ExtractorSpec, Transforms).

Task: add support for pagination based on user-defined CSS selectors.

The user provides pagination selectors (similar to field extractors). The parser must:

Find pagination elements in the current HTML using provided CSS selectors.

Extract URLs from them (usually href, resolving relative URLs using task.FinalURL or task.URL).

Deduplicate extracted URLs.

Filter URLs using existing allowed / deny patterns.

For each allowed URL, enqueue a new crawl task (same job, increased depth).

Pagination extraction must be independent from field extraction but reuse the same selector semantics (selector, attribute, multiple).

Do not follow links automatically — only those explicitly defined by pagination selectors.

The solution must fit into the existing parser_worker architecture and ExtractionSpec model.

Use the provided ExtractionSpec documentation as the source of truth. 

PARSING