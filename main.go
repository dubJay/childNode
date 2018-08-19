package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dubJay/db"
	"github.com/dubJay/serving"
	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
)

var (
	tmpls map[string]*template.Template

	dbPath    = flag.String("dbPath", "db/testDB.db", "Datafile to use")
	logDir   = flag.String("logDir", "logs", "Directory to log to. This path with be joined with rootDir")
	port      = flag.String("port", ":8080", "Port for server to listen on")
	rootDir   = flag.String("rootDir", "", "Path to webdir structure")
	templates = flag.String("templates", "templates", "Templates directory")
	resources = flag.String("resources", "resources", "Images directory")
	static    = flag.String("static", "static" , "CSS, HTML, JS, etc...")
)

const (
	entryPage   = "entry.html"
	historyPage = "history.html"
	landingPage = "index.html"
)

func initDeps() {
	flag.Parse()	
	parseTemplates()
        if err := db.Init(filepath.Join(*rootDir, *dbPath)); err != nil {
		log.Fatalf("could not open database: %v", err)
	}
}

func parseTemplates() {
	var err error
	tmpls = make(map[string]*template.Template)
	tmpls[landingPage], err = template.New(
		landingPage).ParseFiles(filepath.Join(*rootDir, *templates, landingPage))
	if err != nil {
		log.Fatalf("error parsing template %s: %v", landingPage, err)
	}
	tmpls[entryPage], err = template.New(
		entryPage).ParseFiles(filepath.Join(*rootDir, *templates, entryPage))
	if err != nil {
		log.Fatalf("error parsing template %s: %v", entryPage, err)
	}
	tmpls[historyPage], err = template.New(
		historyPage).ParseFiles(filepath.Join(*rootDir, *templates, historyPage))
	if err != nil {
		log.Fatalf("error parsing template %s: %v", historyPage, err)
	}
	log.Print("Templates successfully initialized");
}


func setupLogging() (*os.File, error) {
	if *logDir == "" {
		return nil, errors.New("logDir flag must be set")
	}

	logFile := filepath.Join(*rootDir, *logDir, "general-logs.txt")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return os.Create(logFile)
	}
	return os.Open(logFile);
}

func buildOneOff(w http.ResponseWriter, uid string) error {
	oneoff, err := db.GetOneOff(uid)
	if err != nil {
		return fmt.Errorf("unable to find oneoff entry: %v", err)
	}
	serving := serving.OneoffToServing(oneoff)
	
	return tmpls[entryPage].Execute(w, serving)
}

func buildLandingPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars["id"] != "" {
		if err := buildOneOff(w, vars["id"]); err != nil {
			// This will fall through to landing page.
			log.Printf("failed to build oneoff page: %v", err)
		} else {
			// If oneoff build was successful we don't need the landing page.
			// StatusOk is written to headers implicitly.
			return
		}
	} 
	entry, err := db.GetEntry(0)
	if err != nil {
		log.Printf("failed to get entry: %v", err)
		http.Error(w, "failed to retrieve langing page content from db", http.StatusInternalServerError)
		return
	}
	serving := serving.EntryToServing(entry)

	if err := tmpls[landingPage].Execute(w, serving); err != nil {
		log.Printf("error executing template %s: %v", landingPage, err)
		http.Error(w, "failed to build landing page", http.StatusInternalServerError)
	}
}

func buildPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if len(vars["id"]) == 0 {
		buildLandingPage(w, r)
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("invalid id: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	entry, err := db.GetEntry(id)
	if err != nil {
		log.Printf("failed to get entry: %v", err)
		http.Error(w, "failed to retrieve content from database", http.StatusInternalServerError)
		return
	}

	if err := tmpls[entryPage].Execute(w, serving.EntryToServing(entry)); err != nil {
		log.Printf("error executing template %s: %v", entryPage, err)
		http.Error(w, "failed to build page", http.StatusInternalServerError)
	}
}

func buildNavPage(w http.ResponseWriter, r *http.Request) {
	entries, err := db.GetHistory()
	if err != nil {
		log.Printf("unable to retrieve history entries: %v", err)
		http.Error(w, "failed to retrieve records from archive", http.StatusInternalServerError)
		return
	}
	serving := serving.HistoryToServing(entries)

	if err := tmpls[historyPage].Execute(w, serving); err != nil {
		log.Printf("error executing template %s: %v", historyPage, err)
		http.Error(w, "failed to build navigation from historical records", http.StatusInternalServerError)
	}
}

func buildFeedPage(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
	if len(vars["type"]) == 0 {
		log.Print("no type requested by user.")
		http.Error(w, "no feed type specified by user.", http.StatusPreconditionRequired)
		return
	}
	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	}
	if !contains([]string{"atom.xml", "rss.xml", "jsonfeed.json"}, vars["type"]) {
		log.Print("invalid type requested by user.")
		http.Error(w, "invalid type requested by user.", http.StatusPreconditionFailed)
		return
	}
	
	entries, err := db.GetRecentEntries(15)
	if err != nil {
		log.Printf("unable to retrieve history entries: %v", err)
		http.Error(w, "failed to retrieve recent entries.", http.StatusInternalServerError)
		return
	}

	feed := &feeds.Feed{
		Title:       "Christopher Cawdrey's Blog",
		Link:        &feeds.Link{Href: "https://christopher.cawdrey.name"},
		Description: "Chris' musings, projects, and dispositions.",
		Author:      &feeds.Author{Name: "Christopher Cawdrey", Email: "chris@cawdrey.name"},
		Created:     time.Unix(1489554739, 0),
	}
	for _, entry := range entries {
		feed.Items = append(feed.Items,
			&feeds.Item{
				Title:       entry.Title,
				Id:          strconv.Itoa(entry.Entry_id),
				Link:        &feeds.Link{Href: strings.Join([]string{"https://christopher.cawdrey.name/entry/", strconv.Itoa(entry.Entry_id)}, "")},
				Description: entry.Content,
				Created:     time.Unix(int64(entry.Entry_id), 0),
			})
	}

	switch vars["type"] {
	case "atom.xml":
		atom, err := feed.ToAtom()
		if err != nil {
			log.Printf("failed to create atom feed %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(atom))
	case "rss.xml":
		rss, err := feed.ToRss()
		if err != nil {
			log.Printf("failed to create rss feed %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return

		}
		w.Write([]byte(rss))
	default:
		json, err := feed.ToJSON()
		if err != nil {
			log.Printf("failed to create json feed %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(json))
	}
}

// func loggingMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// Do stuff here
// 		fmt.Println(r.RequestURI)
// 		// Call the next handler, which can be another middleware in the chain, or the final handler.
// 		next.ServeHTTP(w, r)
// 	})
// }

func main() {
	initDeps()
	
	file, err := setupLogging()
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)
	// TODO:
	// 1) DONE -- Build converter package from DB to serving structs
	// 2) DONE -- Init DB package and use here.
	// 3) DEPRECATE -- Build map helper package and compine geocoding and lat lng read functions
	// 4) DONE -- Add apache logging middle ware from gorilla
	// 5) DONE -- Instead of breaking out of http handlers with return empty use http package for return values.
	// 6) Clean up http handlers and history sorter.
	// 7) DONE -- DB Driver does this for me -- Check to see if I need to sanitize my URL vars before querying DB.
	// 8) Backup all SD cards
	// 9) Minimize all JPGs in shared folder.
	// 9.5) Maybe cache to disk actually. Use imageproxy. Tier the cache. 100mb memory by 2hrs first. Disk cache next. Convert to png and 200px on the fly.
	// 10) Conglomerate html files. They can have a common base.
	// 11) I should probably write unit tests...
	// 12) All nodes should bring servers up on startup. Head node should restart /mnt/usb sharing server on startup also.
	// 13) Implement logging and debugging middleware and make it not terrible.

	router := mux.NewRouter()
	router.HandleFunc("/", buildLandingPage).Methods("GET")
	router.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(*rootDir, "robots.txt"))})
	router.HandleFunc("/history", buildNavPage).Methods("GET")
	router.HandleFunc("/entry/{id}", buildPage).Methods("GET")
	router.HandleFunc("/feeds/{type}", buildFeedPage).Methods("GET")
	router.Handle("/static/{item}", http.StripPrefix("/static", http.FileServer(http.Dir(filepath.Join(*rootDir, *static))))).Methods("GET")
	router.Handle("/images/{item}", http.StripPrefix("/images", http.FileServer(http.Dir(filepath.Join(*rootDir, *resources))))).Methods("GET")
	router.Handle("/images/{dir}/{item}", http.StripPrefix("/images", http.FileServer(http.Dir(filepath.Join(*rootDir, *resources))))).Methods("GET")
	router.HandleFunc("/{id}", buildLandingPage).Methods("GET")
	// router.Use(loggingMiddleware) 
	log.Fatal(http.ListenAndServe(*port, router))
}
