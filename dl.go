//go:build nobuild
// +build nobuild

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/renameio"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := Main(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func Main() error {
	fh, err := renameio.TempFile("", "generated.go")
	if err != nil {
		return err
	}
	defer fh.Cleanup()
	if err = Download(fh); err != nil {
		return err
	}
	return fh.CloseAtomicallyReplace()
}

const concurrency = 8

type urlSel struct {
	Lang     string
	URL      string
	Selector string
}

var (
	URL   = urlSel{URL: "http://www.1000mostcommonwords.com/", Selector: ".entry-content > table > tbody > tr > td:nth-child(2)"}
	enURL = urlSel{Lang: "english", URL: "https://www.ef.com/wwen/english-resources/english-vocabulary/top-3000-words/", Selector: ".col-md-12 > div:nth-child(2) > p:nth-child(2)"}
	huURL = urlSel{Lang: "hungarian", URL: "https://en.wiktionary.org/wiki/Wiktionary:Frequency_lists/Hungarian_frequency_list_1-10000", Selector: ".mw-parser-output > ol:nth-child(1) > li > a:nth-child(1)"}
)

var httpClient = http.DefaultClient

func Download(w io.Writer) error {
	resp, err := httpClient.Get(URL.URL)
	if err != nil {
		return fmt.Errorf("%s: %w", URL, err)
	}
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	urls := make([]urlSel, 2, 128)
	urls[0] = enURL
	urls[1] = huURL
	var rURL = regexp.MustCompile(
		`href="(` +
			strings.Replace(URL.URL, "://www.", "://(?:www[.])?", 1) +
			`(?:words/)?1000-(?:most-)?common-([^-]+)-words)/*"`,
	)

	for _, loc := range rURL.FindAllSubmatchIndex(b, 1001) {
		k, v := string(b[loc[2*2]:loc[2*2+1]]), b[loc[2*1]:loc[2*1+1]]
		v = bytes.Replace(v, []byte("/www."), []byte("/"), 1)
		v = bytes.Replace(v, []byte("/words/"), []byte("/"), 1)
		if bytes.Contains(v, []byte("-english-")) || bytes.Contains(v, []byte("-hungarian-")) {
			continue
		}
		urls = append(urls, urlSel{Lang: string(k), URL: string(v), Selector: URL.Selector})
	}

	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, `// Generated by dl.go, DO NOT EDIT!

package main

func init() {
	wordsMap = generatedWordsMap
}

var generatedWordsMap = map[string]stringWithLengths{
`)
	limit := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var grp errgroup.Group
	var token struct{}
	for _, u := range urls {
		u := u
		limit <- token
		grp.Go(func() error {
			defer func() { <-limit }()

			log.Println(u)
			resp, err := httpClient.Get(u.URL)
			if err != nil {
				return fmt.Errorf("%s: %w", u, err)
			}
			defer resp.Body.Close()
			doc, qErr := goquery.NewDocumentFromResponse(resp)
			if qErr != nil {
				return fmt.Errorf("%s: %w", u, qErr)
			}
			var buf, lengths strings.Builder
			var nth int
			Add := func(text string) {
				text = strings.TrimSpace(text)
				if text == "" || strings.EqualFold(text, u.Lang) {
					return
				}
				buf.WriteString(text)
				if nth != 0 {
					lengths.WriteByte(',')
				}
				fmt.Fprintf(&lengths, "%d", len(text))
				nth++
			}
			var f func(n *html.Node) int
			f = func(n *html.Node) int {
				if n == nil {
					return 0
				}
				if n.Type == html.TextNode {
					Add(n.Data)
					return 1
				}
				var num int
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					num += f(c)
				}
				return num
			}

			doc.Find(u.Selector).
				Each(func(i int, s *goquery.Selection) {
					if i == 1 {
						return
					}
					var num int
					for _, n := range s.Nodes {
						num += f(n)
					}
					if num == 0 {
						Add(s.Text())
					}
				})
			if buf.Len() <= 1 {
				log.Printf("WARN: no words for %q in %q!", u.Lang, u.URL)
				return nil
			}
			mu.Lock()
			fmt.Fprintf(bw, "\t%q: stringWithLengths{\n\t\tSource: %q,\n\t\tWords: %q,\n\t\tLengths: []uint8{%s},\n\t},\n",
				u.Lang, u.URL, buf.String(), lengths.String())
			mu.Unlock()
			return nil
		})
	}

	if err != nil {
		err = fmt.Errorf("%s: %w", URL, err)
	}
	if wErr := grp.Wait(); wErr != nil && err == nil {
		err = wErr
	}
	bw.WriteString("\n}\n")
	if wErr := bw.Flush(); wErr != nil && err == nil {
		err = wErr
	}
	return err
}
