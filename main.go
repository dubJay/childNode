package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
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
	logTime time.Time

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
	kCawdPage   = "kcawd.html"
	wizardProgrammingPage = "christhewizardprogrammer.html"

	scpBasePage = "base.html"
	scpLanding = "landing"
	scpAnnouncements = "announcements"
	scpFaqs = "faqs"
	scpMap = "map"
	scpConditions = "conditions"
	scpWeather = "weather"
	scpCam = "cam"

	scpConst = "scp"
	htmlSuffix = ".html"
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
	tmpls[kCawdPage], err = template.New(
		kCawdPage).ParseFiles(filepath.Join(*rootDir, *templates, kCawdPage))
	if err != nil {
		log.Fatalf("error parsing template %s: %v", kCawdPage, err)
	}
	tmpls[scpBasePage], err = template.New(
		scpBasePage).ParseFiles(filepath.Join(*rootDir, *templates, scpConst, scpBasePage))
	if err != nil {
		log.Fatalf("error parsing template %s: %v", kCawdPage, err)
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

func buildSCPHome(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadFile(filepath.Join(*rootDir, *templates, scpConst, scpLanding + htmlSuffix))
	if err != nil {
		log.Printf("error reading file %s: %v", scpLanding, err)
		http.Error(w, "failed to build page", http.StatusInternalServerError)
		return
	}

	if err := tmpls[scpBasePage].Execute(w, serving.SCPToServing(content)); err != nil {
		log.Printf("error executing template %s: %v", scpLanding, err)
		http.Error(w, "failed to build page", http.StatusInternalServerError)
	}

}

func buildSCP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if len(vars["optional"]) != 0 {
		content, err := ioutil.ReadFile(filepath.Join(*rootDir, *templates, scpConst, vars["optional"] + htmlSuffix))
		if err != nil {
			log.Printf("error reading file %s: %v", vars["optional"], err)
			buildSCPHome(w, r)
			return
		}

		if err := tmpls[scpBasePage].Execute(w, serving.SCPToServing(content)); err != nil {
			log.Printf("error executing template %s: %v", vars["optional"], err)
			http.Error(w, "failed to build page", http.StatusInternalServerError)
		}
	}
}

func buildOneOff(w http.ResponseWriter, uid string) error {
	oneoff, err := db.GetOneOff(uid)
	if err != nil {
		return fmt.Errorf("unable to find oneoff entry: %v", err)
	}
	serving, err := serving.OneoffToServing(oneoff)
	if err != nil {
		return fmt.Errorf("failed to generate HTML content: %v", err)
	}
	
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
	serving, err := serving.EntryToServing(entry)
	if err != nil {
		log.Printf("failed to generate HTML content: %v", err)
		http.Error(w, "failed to generate content", http.StatusInternalServerError)
		return
	}

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

	serving, err := serving.EntryToServing(entry)
	if err != nil {
		log.Printf("failed to generate HTML content: %v", err)
		http.Error(w, "failed to generate content", http.StatusInternalServerError)
		return
	}
	
	if err := tmpls[entryPage].Execute(w, serving); err != nil {
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

func buildKCawdPage(w http.ResponseWriter, r *http.Request) {
	articles, err := db.GetArticleMeta()
	if err != nil {
		log.Printf("unable to retrieve kcawd article metadata: %v", err)
		http.Error(w, "failed to retrieve katy's articles from archive", http.StatusInternalServerError)
		return
	}

	if err := tmpls[kCawdPage].Execute(w, articles); err != nil {
		log.Printf("error executing template %s: %v", kCawdPage, err)
		http.Error(w, "failed to build katy's landing page", http.StatusInternalServerError)
	}
}

func serveKCawdPDF(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	idNumeric, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("invalid id for serveKatyCawd %s", id)
		http.Error(w, "invalid id for article: " + id, http.StatusNotFound)
	}
	
	article, err := db.GetArticle(idNumeric)
	if err != nil {
		log.Printf("unable to locate pdf for article: %d", id)
		http.Error(w, "no PDF found for article: " + id, http.StatusNotFound)
	}

	http.ServeContent(w, r, id + ".pdf", time.Unix(0, 0), serving.StringToPDF(article))
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
	
	entries, err := db.GetRecentEntries(1000)
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
		serving, err := serving.EntryToServing(entry)
		if err != nil {
			log.Printf("failed to generate HTML content for feed: %v", err)
			http.Error(w, "failed to generate content for feed", http.StatusInternalServerError)
			return
		}

		feed.Items = append(feed.Items,
			&feeds.Item{
				Title:       entry.Title,
				Id:          strconv.Itoa(entry.Entry_id),
				Link:        &feeds.Link{Href: strings.Join([]string{"https://christopher.cawdrey.name/entry/", strconv.Itoa(entry.Entry_id)}, "")},
				Description: string(serving.HTML),
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

// This is packed into the middleware so we crash instead of returning on failure.
func setLoggingLocation(now time.Time) {
	if *logDir == "" {
		log.Fatal("logDir flag must be set")
	}

	nowString := fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())
	logFile := filepath.Join(*rootDir, *logDir, fmt.Sprintf("debug-logs_%s.txt", nowString))

	var file *os.File
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		file, err = os.Create(logFile)
		if err != nil {
			log.Fatalf("Failed to create file with name %s: %v", logFile, err)
		}
	} else {
		file, err = os.Open(logFile)
		if err != nil {
			log.Fatalf("Failed to open file with name  %s: %v", logFile, err)
		}
	}
	// Reset logtime.
	logTime = now
	log.SetOutput(file)
} 

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC()
		if now.Day() != logTime.Day() || now.Month() != logTime.Month() || now.Year() != logTime.Year() {
			setLoggingLocation(now)
			logTime = now
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {
	initDeps()
	
	// TODO:
	// 1) DONE -- Build converter package from DB to serving structs
	// 2) DONE -- Init DB package and use here.
	// 3) DEPRECATE -- Build map helper package and compine geocoding and lat lng read functions
	// 4) DONE -- Add apache logging middle ware from gorilla
	// 5) DONE -- Instead of breaking out of http handlers with return empty use http package for return values.
	// 6) Clean up http handlers and history sorter.
	// 6.5) Clean up and cache feeds.
	// 7) DONE -- DB Driver does this for me -- Check to see if I need to sanitize my URL vars before querying DB.
	// 8) Backup all SD cards
	// 9) Minimize all JPGs in shared folder.
	// 9.5) Maybe cache to disk actually. Use imageproxy. Tier the cache. 100mb memory by 2hrs first. Disk cache next. Convert to png and 200px on the fly.
	// 10) Conglomerate html files. They can have a common base.
	// 11) I should probably write unit tests...
	// 12) All nodes should bring servers up on startup. Head node should restart /mnt/usb sharing server on startup also.
	// 13) Implement logging and debugging middleware and make it not terrible. This is halfway done. I'd like debug logs to be in combined logging format however.

	router := mux.NewRouter()
	router.HandleFunc("/", buildLandingPage).Methods("GET")
	router.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(*rootDir, "robots.txt"))})

	// ChrisTheWizardProgrammer route.
	router.HandleFunc("/wizardprogramming",func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(*rootDir, *templates, wizardProgrammingPage))})


	// Kcawd route.
	router.HandleFunc("/kcawd", buildKCawdPage).Methods("GET")
	router.HandleFunc("/kcawd/{id}", serveKCawdPDF).Methods("GET")

	// SCP route.
	scp := router.PathPrefix("/scp").Subrouter()
	scp.Handle("/static/{item}", http.StripPrefix("/scp/static", http.FileServer(http.Dir(filepath.Join(*rootDir, *static))))).Methods("GET")
	scp.Handle("/images/{dir}/{item}", http.StripPrefix("/scp/images", http.FileServer(http.Dir(filepath.Join(*rootDir, *resources))))).Methods("GET")
	scp.HandleFunc("", buildSCPHome).Methods("GET")
	scp.HandleFunc("/{optional}", buildSCP).Methods("GET")
	
	// Christopher.cawdrey.name route.
	router.HandleFunc("/history", buildNavPage).Methods("GET")
	router.HandleFunc("/entry/{id}", buildPage).Methods("GET")
	router.HandleFunc("/feeds/{type}", buildFeedPage).Methods("GET")
	router.Handle("/static/{item}", http.StripPrefix("/static", http.FileServer(http.Dir(filepath.Join(*rootDir, *static))))).Methods("GET")
	router.Handle("/images/{item}", http.StripPrefix("/images", http.FileServer(http.Dir(filepath.Join(*rootDir, *resources))))).Methods("GET")
	router.Handle("/images/{dir}/{item}", http.StripPrefix("/images", http.FileServer(http.Dir(filepath.Join(*rootDir, *resources))))).Methods("GET")
	router.HandleFunc("/{id}", buildLandingPage).Methods("GET")
	router.Use(logger) 
	log.Fatal(http.ListenAndServe(*port, router))
}
