package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// CollectPage fetches all pages from a paginated API endpoint and returns all
// items. valuesKey is the JSON key containing items, lastPageKey is the boolean
// key indicating last page, params are initial query parameters.
//
// Pagination: if isLastPage is false, the next request adds start=nextPageStart.
func CollectPage(c *Client, path, valuesKey, lastPageKey string, params map[string]string) ([]any, error) {
	var allItems []any
	p := make(map[string]string)
	for k, v := range params {
		p[k] = v
	}

	for {
		body, status, err := c.Do(context.Background(), "GET", path, nil, p)
		if err != nil {
			return nil, fmt.Errorf("fetching page: %w", err)
		}
		if status >= 400 {
			return nil, fmt.Errorf("API returned %d", status)
		}

		var doc map[string]any
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		values, ok := doc[valuesKey].([]any)
		if !ok {
			return nil, fmt.Errorf("key %q not found or not an array", valuesKey)
		}
		allItems = append(allItems, values...)

		isLast, _ := doc[lastPageKey].(bool)
		if isLast {
			break
		}

		// Get nextPageStart for continuation.
		next, ok := doc["nextPageStart"]
		if !ok {
			break
		}
		p["start"] = strconv.Itoa(int(next.(float64)))
	}
	return allItems, nil
}
