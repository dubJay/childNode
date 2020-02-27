package db

import (
	"database/sql"
	"fmt"
	
	_ "github.com/mattn/go-sqlite3"
)

// Queries for db actions.
var (
	entryQuery       = `SELECT timestamp, title, next, previous, paragraph, image FROM entry WHERE timestamp = ?`
	historyQuery     = `SELECT timestamp, title FROM entry ORDER BY timestamp DESC`
	landingQuery     = `SELECT timestamp, title, next, previous, paragraph, image FROM entry ORDER BY timestamp DESC LIMIT ?`
	oneoffQuery      = `SELECT uid, paragraph, image from oneoff WHERE uid = ?`
	articleMetaQuery = `SELECT timestamp, title, organization, hyperlink FROM articlemeta ORDER BY timestamp DESC`
	articleQuery     = `SELECT pdf FROM articlemeta where timestamp = ?`
)

var globalDB *sql.DB

type Entry struct {
	// Timestamp, UID.
	Entry_id  int
	Title     string
	Next      int
	Previous  int
	Content   string
	Image     string
}

type Oneoff struct {
	Uid       string
	Paragraph string
	Image     string
}

type History struct {
	Entry_id int
	Title    string
}

type ArticleMeta struct {
	EntryId int
	Title string
	Organization string
	Hyperlink string
}

func Init(dbPath string) error {
	var err error
	globalDB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		err = fmt.Errorf("Failed to init db (path: %s): %v", dbPath, err)
	}
	return err
}

func GetArticleMeta() ([]ArticleMeta, error) {
	rows, err := globalDB.Query(articleMetaQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []ArticleMeta
	for rows.Next() {
		article := ArticleMeta{}
		err := rows.Scan(&article.EntryId, &article.Title, &article.Organization, &article.Hyperlink)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, nil
}

func GetArticle(id int) (string, error) {
	if id == 0 {
		return "", fmt.Errorf("%d is not a valid id", id)
	}

	var pdf string
	err := globalDB.QueryRow(articleQuery, id).Scan(&pdf)
	return pdf, err
}

func GetRecentEntries(limit int) ([]Entry, error) {
	rows, err := globalDB.Query(landingQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		entry := Entry{}
		err := rows.Scan(&entry.Entry_id, &entry.Title, &entry.Next, &entry.Previous, &entry.Content, &entry.Image)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func GetOneOff(id string) (Oneoff, error) {
	oneoff := Oneoff{}
	err := globalDB.QueryRow(oneoffQuery, id).Scan(&oneoff.Uid, &oneoff.Paragraph, &oneoff.Image)
	return oneoff, err
}

func GetEntry(id int) (Entry, error) {
	// Get entry at id. If id is empty get most recent entry.
	page := Entry{}
	if id == 0 {
		rows, err := globalDB.Query(landingQuery, 1)
		if err != nil {
			return page, err
		}
		defer rows.Close()

		for rows.Next() {
			err := rows.Scan(
				&page.Entry_id, &page.Title, &page.Next, &page.Previous, &page.Content, &page.Image)
			if err != nil {
				return page, err
			}
			break
		}
	} else {
		err := globalDB.QueryRow(entryQuery, id).Scan(
			&page.Entry_id, &page.Title, &page.Next, &page.Previous, &page.Content, &page.Image)
		if err != nil {
			return page, err
		}
	}
	return page, nil
}

func GetHistory() ([]History, error) {
	rows, err := globalDB.Query(historyQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []History
	for rows.Next() {
		entry := History{}
		err := rows.Scan(&entry.Entry_id, &entry.Title)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
