package main

import (
	"crypto/rand"
	"flag"
	"io"
	"log"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//go:generate go get github.com/PuerkitoBio/goquery
//go:generate go run dl.go

var wordsMap = make(map[string][]string, 100)
var langNames = map[string]string{"hu": "hungarian", "en": "english"}

func main() {
	lng := os.Getenv("LANG")
	if i := strings.IndexByte(lng, '.'); i >= 0 {
		lng = lng[:i]
	}
	if lng == "" {
		lng = "en_US"
	}
	flagLang := flag.String("lang", lng, "language")
	flagFile := flag.String("file", "words.json.gz", "words file - will be created if not exist")
	flag.Parse()

	n := 4
	if s := flag.Arg(0); s != "" {
		if i, err := strconv.Atoi(s); err != nil {
			log.Printf("first arg must be the number of words to return!")
		} else {
			n = i
		}
	}
	if err := Main(*flagFile, *flagLang, n); err != nil {
		log.Fatal(err)
	}
}

func Main(fn, lang string, n int) error {
	words := wordsMap[lang]
	if len(words) == 0 {
		s := lang
		if i := strings.IndexByte(s, '_'); i > 0 {
			s = s[:i]
		}
		if k := langNames[s]; k != "" {
			words = wordsMap[k]
		} else {
			for k, v := range wordsMap {
				if strings.HasPrefix(s, k) {
					words = v
					break
				}
			}
		}
		if len(words) == 0 {
			keys := make([]string, 0, len(wordsMap))
			for k := range wordsMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return errors.Errorf("No words for %q. Has %q.", lang, keys)
		}
	}
	max := big.NewInt(int64(len(words)))
	chosen := make([]string, n)
	for i := 0; i < n; i++ {
		I, err := rand.Int(rand.Reader, max)
		if err != nil {
			return errors.Wrapf(err, "%d. rand", i)
		}
		chosen[i] = words[int(I.Int64())]
	}
	sort.Sort(byLenR(chosen))
	io.WriteString(os.Stdout, strings.Join(chosen, " "))
	_, err := os.Stdout.Write([]byte{'\n'})
	return err
}

var _ = sort.Interface(byLenR(nil))

type byLenR []string

func (b byLenR) Len() int           { return len(b) }
func (b byLenR) Less(i, j int) bool { return len(b[i]) > len(b[j]) }
func (b byLenR) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
