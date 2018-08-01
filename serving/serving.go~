package serving

import (
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
	Content   []string
	Image     []string
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

func splitTextBlob(s string) []string {
	return strings.Split(s, `\n`)
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

	var histServe HistoryServing
	for key := range m {
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

func EntryToServing(e db.Entry) EntryServing {
	t := time.Unix(int64(e.Entry_id), 0)
	nextStr, prevStr := "", ""
	if e.Next != 0 {
		nextStr = filepath.Join("/entry", strconv.Itoa(e.Next))
	}
	if e.Previous != 0 {
		prevStr = filepath.Join("/entry", strconv.Itoa(e.Previous))
	}

	return EntryServing{
		Title: e.Title,
		NextPath: nextStr,
		PrevPath: prevStr,
		Month: t.Month().String(),
		Day: strconv.Itoa(t.Day()),
		Year: strconv.Itoa(t.Year()),
		Content: splitTextBlob(e.Content),
		Image: splitTextBlob(e.Image),
	}
}

func OneoffToServing(o db.Oneoff) EntryServing {
	return EntryServing{
		Title: o.Uid,
		Content: splitTextBlob(o.Paragraph),
		Image: splitTextBlob(o.Image),
	}
}
