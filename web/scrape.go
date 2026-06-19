package web

import (
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func (s *Server) handlePreviewURL(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	client := &http.Client{Timeout: 10 * 1000000000} // 10 seconds
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	// Add user-agent to avoid simple blocks
	req.Header.Set("User-Agent", "OpenShineBot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch url")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "url returned error status")
		return
	}

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
