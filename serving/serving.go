package serving

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"math/rand"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	
	"github.com/dubJay/db"
)

type EntryServing struct {
	Title     string
	NextPath  string
	PrevPath  string
	Month     string
	Day       string
	Year      string
	HTML      template.HTML
}

type entryHTMLRaw struct {
	Content []string
	Image []string
}

type historyMeta struct {
	Title string
	Path  string
}

type historyEntry struct {
	Year     int
	Metadata []historyMeta
}

type HistoryServing []historyEntry

type SCPServing struct {
	Content template.HTML
	Quip string
}

var quips  = []string{
	"\"Best website on the Internet!\" -- My Mother",
	"It's like the offical SCPLOA webpage, only updated regularly!",
	"This site is proudly hosted on pastries.",
	"Read only? But I'm pissed off and want to tell somebody about it!",
	"Do you think the webmaster knows this dialogue changes when I refresh the page?",
	"You know, I'm starting to suspect this Snipes fella is based on Bill...",
	"\"SLOW DOWN ON ASSOCIATION ROADS!\" --A Landowner probably",
	"SCP -- It's like Wyoming but without the tax incentives",
	"Packrats, cows, and hunters...oh my!"}

func splitTextBlob(s string) []string {
	return strings.Split(s, `\n`)
}

func SCPToServing(in []byte) SCPServing {
	return SCPServing{
		Content: template.HTML(string(in)),
		Quip: quips[rand.Intn(len(quips))],
	}
}

func HistoryToServing(h []db.History) HistoryServing {
	m := make(map[int]map[int]string)
	sk := make(map[int][]int)
	for _, entry := range h {
		t := time.Unix(int64(entry.Entry_id), 0)
		if _, ok := m[t.Year()]; !ok {
			m[t.Year()] = make(map[int]string)
			sk[t.Year()] = []int{}
		}
		m[t.Year()][entry.Entry_id] = entry.Title
		sk[t.Year()] = append(sk[t.Year()], entry.Entry_id)
	}

	mapKeys := make([]int, 0, len(m))
	for k := range m {
		mapKeys = append(mapKeys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(mapKeys)));

	var histServe HistoryServing
	for _, key := range mapKeys {
		history := historyEntry{}
		history.Year = key
		sort.Sort(sort.Reverse(sort.IntSlice(sk[key])))
		for _, entryId := range sk[key] {
			history.Metadata = append(history.Metadata,
				historyMeta{Title: m[key][entryId], Path: filepath.Join("/entry", strconv.Itoa(entryId))})
		}
		histServe = append(histServe, history)
	}
	return histServe

}

func EntryToServing(e db.Entry) (EntryServing, error) {
	t := time.Unix(int64(e.Entry_id), 0)
	nextStr, prevStr := "", ""
	if e.Next != 0 {
		nextStr = filepath.Join("/entry", strconv.Itoa(e.Next))
	}
	if e.Previous != 0 {
		prevStr = filepath.Join("/entry", strconv.Itoa(e.Previous))
	}

	rawHTML, err := entryHTMLFrom(entryHTMLRaw{
		Content: splitTextBlob(e.Content),
		Image: splitTextBlob(e.Image),
	})
	if err != nil {
		return EntryServing{}, err
	}

	return EntryServing{
		Title: e.Title,
		NextPath: nextStr,
		PrevPath: prevStr,
		Month: t.Month().String(),
		Day: strconv.Itoa(t.Day()),
		Year: strconv.Itoa(t.Year()),
		HTML: rawHTML,
	}, nil
}

func OneoffToServing(o db.Oneoff) (EntryServing, error) {
	rawHTML, err := entryHTMLFrom(entryHTMLRaw{
		Content: splitTextBlob(o.Paragraph),
		Image: splitTextBlob(o.Image),
	})
	if err != nil {
		return EntryServing{}, err
	}

	return EntryServing{
		Title: o.Uid,
		HTML: rawHTML,
	}, nil
}

func StringToPDF(pdf string) io.ReadSeeker {
	return bytes.NewReader([]byte(pdf))
}

func entryHTMLFrom(raw entryHTMLRaw) (template.HTML, error) {
	if len(raw.Content) > len(raw.Image) {
		return "", errors.New("Image and Content arrays are mismatched")
	}
	var htmlBuf bytes.Buffer
	for i, item := range raw.Content {
		htmlBuf.WriteString(strings.Join([]string{"<p>", item, "</p>"}, ""))
		if len(raw.Image[i]) != 0 {
			htmlBuf.WriteString(
				strings.Join(
					[]string{"<a href=",
						raw.Image[i],
						"><img class=image src=",
						raw.Image[i],
						"></a>"}, ""))
		}
	}
	return template.HTML(htmlBuf.String()), nil	
}
