package web

import (
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func (s *Server) handlePreviewURL(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	client := &http.Client{Timeout: 5 * time.Second} // 5 seconds
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	// Add a standard browser user-agent to avoid instant blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch url")
		return
	}
	defer resp.Body.Close()

	// We intentionally do NOT check for resp.StatusCode >= 400 here.
	// Many news sites (like WSJ, NYT) or Cloudflare return 401/403 to automated requests,
	// but still include the <meta og:title> tags in the "Access Denied" HTML body
	// so that social media link previews still work!


	title, desc, image := extractMetaTags(resp.Body)
	writeJSON(w, http.StatusOK, map[string]string{
		"title":       title,
		"description": desc,
		"image":       image,
		"url":         target,
	})
}

func extractMetaTags(r io.Reader) (title, desc, image string) {
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			t := z.Token()
			if t.Data == "title" {
				tt = z.Next()
				if tt == html.TextToken {
					if title == "" {
						title = strings.TrimSpace(z.Token().Data)
					}
				}
				continue
			}

			if t.Data == "meta" {
				var name, property, content string
				for _, a := range t.Attr {
					if a.Key == "name" {
						name = a.Val
					} else if a.Key == "property" {
						property = a.Val
					} else if a.Key == "content" {
						content = a.Val
					}
				}

				if property == "og:title" || name == "twitter:title" {
					title = content
				} else if property == "og:description" || name == "twitter:description" || name == "description" {
					desc = content
				} else if property == "og:image" || name == "twitter:image" {
					image = content
				}
			}
		}
	}
	return title, desc, image
}
