package spotify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxHashResolutionScripts = 60
	webPlayerScriptsURL      = "https://open.spotify.com/search"
)

type hashResolver struct {
	client *http.Client

	mu     sync.Mutex
	hashes map[string]string
}

func newHashResolver(client *http.Client) *hashResolver {
	return &hashResolver{client: client, hashes: map[string]string{}}
}

func (h *hashResolver) Hash(ctx context.Context, operation string) (string, error) {
	if strings.TrimSpace(operation) == "" {
		return "", errors.New("operation required")
	}
	h.mu.Lock()
	if hash := h.hashes[operation]; hash != "" {
		h.mu.Unlock()
		return hash, nil
	}
	h.mu.Unlock()

	if err := h.load(ctx, []string{operation}); err != nil {
		return "", err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	hash := h.hashes[operation]
	if hash == "" {
		return "", fmt.Errorf("hash for %s not found", operation)
	}
	return hash, nil
}

func (h *hashResolver) load(ctx context.Context, operations []string) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	h.mu.Lock()
	need := make([]string, 0, len(operations))
	for _, operation := range operations {
		if h.hashes[operation] == "" {
			need = append(need, operation)
		}
	}
	h.mu.Unlock()
	if len(need) == 0 {
		return nil
	}

	html, err := h.fetchText(deadlineCtx, webPlayerScriptsURL)
	if err != nil {
		html, err = h.fetchText(deadlineCtx, defaultTokenBaseURL)
		if err != nil {
			return err
		}
	}
	scriptURLs, err := extractWebPlayerScripts(html)
	if err != nil {
		return err
	}
	scriptURLs = h.expandFromManifests(deadlineCtx, scriptURLs)
	if len(scriptURLs) > maxHashResolutionScripts {
		scriptURLs = scriptURLs[:maxHashResolutionScripts]
	}
	for _, scriptURL := range scriptURLs {
		if err := deadlineCtx.Err(); err != nil {
			return err
		}
		body, err := h.fetchText(deadlineCtx, scriptURL)
		if err != nil {
			continue
		}
		found := findOperationHashes(body, need)
		if len(found) == 0 {
			continue
		}
		h.store(found)
		need = filterMissing(need, found)
		if len(need) == 0 {
			return nil
		}
	}
	return fmt.Errorf("missing hashes for %s", strings.Join(need, ", "))
}

func (h *hashResolver) expandFromManifests(ctx context.Context, scriptURLs []string) []string {
	if len(scriptURLs) == 0 {
		return scriptURLs
	}
	merged := make([]string, 0, len(scriptURLs))
	seen := make(map[string]struct{}, len(scriptURLs))
	for _, rawURL := range scriptURLs {
		if _, exists := seen[rawURL]; exists {
			continue
		}
		seen[rawURL] = struct{}{}
		merged = append(merged, rawURL)
	}

	for _, rawURL := range scriptURLs {
		if err := ctx.Err(); err != nil {
			break
		}
		if !strings.Contains(rawURL, "/generated/manifest-") {
			continue
		}
		body, err := h.fetchText(ctx, rawURL)
		if err != nil {
			continue
		}
		for _, extracted := range extractJSURLsFromText(body) {
			url := normalizeScriptURL(extracted)
			if _, exists := seen[url]; exists {
				continue
			}
			seen[url] = struct{}{}
			merged = append(merged, url)
		}
	}

	sort.SliceStable(merged, func(i, j int) bool {
		left := scriptPriority(merged[i])
		right := scriptPriority(merged[j])
		if left == right {
			return merged[i] < merged[j]
		}
		return left > right
	})
	return merged
}

func (h *hashResolver) store(found map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for operation, hash := range found {
		if h.hashes[operation] == "" {
			h.hashes[operation] = hash
		}
	}
}

func (h *hashResolver) fetchText(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	applyRequestHeaders(req, requestHeaders{})
	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apiError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func extractWebPlayerScripts(html string) ([]string, error) {
	matches := regexp.MustCompile(`<script[^>]+src="([^"]+)"`).FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, errors.New("web player scripts not found")
	}
	seen := map[string]struct{}{}
	scripts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(match[1])
		if raw == "" || !strings.HasSuffix(raw, ".js") {
			continue
		}
		if !strings.Contains(raw, "/web-player/") && !strings.Contains(raw, "/mobile-web-player/") {
			continue
		}
		url := normalizeScriptURL(raw)
		if _, exists := seen[url]; exists {
			continue
		}
		seen[url] = struct{}{}
		scripts = append(scripts, url)
	}
	if len(scripts) == 0 {
		return nil, errors.New("web player scripts not found")
	}
	sort.SliceStable(scripts, func(i, j int) bool {
		left := scriptPriority(scripts[i])
		right := scriptPriority(scripts[j])
		if left == right {
			return scripts[i] < scripts[j]
		}
		return left > right
	})
	return scripts, nil
}

func normalizeScriptURL(src string) string {
	switch {
	case strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://"):
		return src
	case strings.HasPrefix(src, "//"):
		return "https:" + src
	case strings.HasPrefix(src, "/"):
		return strings.TrimSuffix(defaultTokenBaseURL, "/") + src
	default:
		return strings.TrimSuffix(defaultTokenBaseURL, "/") + "/" + src
	}
}

func scriptPriority(rawURL string) int {
	priority := 0
	if strings.Contains(rawURL, "search") {
		priority += 12
	}
	if strings.Contains(rawURL, "xpui-routes") {
		priority += 8
	}
	if strings.Contains(rawURL, "/generated/manifest-") {
		priority += 2
	}
	if strings.Contains(rawURL, "web-player.") {
		priority += 2
	}
	if strings.Contains(rawURL, "/web-player/") {
		priority += 1
	}
	return priority
}

func extractJSURLsFromText(body string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`https://open\.spotifycdn\.com[^"']+\.js`),
		regexp.MustCompile(`/cdn/[^"']+\.js`),
		regexp.MustCompile(`https:\\/\\/open\.spotifycdn\.com[^"']+\.js`),
		regexp.MustCompile(`\\/cdn\\/[^"']+\.js`),
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, pattern := range patterns {
		matches := pattern.FindAllString(body, -1)
		for _, match := range matches {
			normalized := strings.ReplaceAll(match, `\\/`, `/`)
			normalized = strings.ReplaceAll(normalized, `\u002F`, `/`)
			normalized = strings.ReplaceAll(normalized, `\u002f`, `/`)
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
		}
	}
	return out
}

func findOperationHashes(body string, operations []string) map[string]string {
	found := map[string]string{}
	for _, operation := range operations {
		if operation == "" {
			continue
		}
		escaped := regexp.QuoteMeta(operation)
		primary := regexp.MustCompile(`(?s)` + escaped + `.{0,400}?sha256Hash\":\"([a-f0-9]{64})\"`)
		if match := primary.FindStringSubmatch(body); len(match) > 1 {
			found[operation] = match[1]
			continue
		}
		fallback := regexp.MustCompile(`"` + escaped + `","(?:query|mutation)","([a-f0-9]{64})"`)
		if match := fallback.FindStringSubmatch(body); len(match) > 1 {
			found[operation] = match[1]
		}
	}
	return found
}

func filterMissing(need []string, found map[string]string) []string {
	remaining := make([]string, 0, len(need))
	for _, operation := range need {
		if found[operation] == "" {
			remaining = append(remaining, operation)
		}
	}
	return remaining
}
