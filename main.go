package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

//go:embed templates/*.html friends/friends.txt
var content embed.FS

const friendsPerPage = 3

type PageData struct {
	Title        string
	Description  string
	CanonicalURL string
	Friends      []template.HTML
	PrevPage     int
	NextPage     int
	CurrentPage  int
	PageRanges   []string
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/friends", friendsHandler)
	mux.HandleFunc("/", redirectHandler)
	mux.HandleFunc("/404", notFoundHandler)
	mux.HandleFunc("/500", internalServerErrorHandler)

	secureMux := addSecurityHeaders(mux)

	log.Println("Server starting on 6055")
	log.Fatal(http.ListenAndServe(":6055", secureMux))
}

func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		//w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/friends", http.StatusMovedPermanently)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "templates/404.html", nil, http.StatusNotFound)
}

func internalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "templates/500.html", nil, http.StatusInternalServerError)
}

func friendsHandler(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	friends, err := getFriendsFromTextFile("friends/friends.txt")
	if err != nil {
		log.Printf("Error reading friends: %v", err)
		internalServerErrorHandler(w, r)
		return
	}

	totalPages := (len(friends) + friendsPerPage - 1) / friendsPerPage
	if page > totalPages {
		//http.Redirect(w, r, "/404", http.StatusSeeOther)
		renderTemplate(w, "templates/404.html", nil, http.StatusSeeOther)
		return
	}

	start := (page - 1) * friendsPerPage
	end := start + friendsPerPage
	if end > len(friends) {
		end = len(friends)
	}

	prevPage := 0
	if page > 1 {
		prevPage = page - 1
	}

	nextPage := 0
	if page < totalPages {
		nextPage = page + 1
	}

	pageRanges := getPageRanges(page, totalPages)

	canonicalURL := constructCanonicalURL(r)

	data := PageData{
		Title:        "Happy Friendship Day Quotes - Page " + strconv.Itoa(page),
		Description:  "A collection of Friendship Quotes - Page " + strconv.Itoa(page),
		CanonicalURL: canonicalURL,
		Friends:      friends[start:end],
		PrevPage:     prevPage,
		NextPage:     nextPage,
		CurrentPage:  page,
		PageRanges:   pageRanges,
	}

	renderTemplate(w, "templates/index.html", data, http.StatusOK)
}

func getFriendsFromTextFile(filePath string) ([]template.HTML, error) {
	content, err := content.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	blocks := strings.Split(string(content), "\n\n")
	friends := make([]template.HTML, 0, len(blocks))

	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed != "" {
			htmlBlock := strings.ReplaceAll(trimmed, "\n", "<br>")
			friends = append(friends, template.HTML(htmlBlock))
		}
	}

	return friends, nil
}

func getPageRanges(currentPage, totalPages int) []string {
	pageRanges := make([]string, 0)
	pageStart := (currentPage-1)/3*3 + 1
	pageEnd := pageStart + 2
	if pageEnd > totalPages {
		pageEnd = totalPages
	}
	for i := pageStart; i <= pageEnd; i++ {
		pageRanges = append(pageRanges, strconv.Itoa(i))
	}
	return pageRanges
}

func renderTemplate(w http.ResponseWriter, tmplPath string, data interface{}, statusCode int) {
	tmplContent, err := content.ReadFile(tmplPath)
	if err != nil {
		log.Printf("Error loading template: %v", err)
		internalServerErrorHandler(w, nil)
		return
	}

	tmpl, err := template.New("tmpl").Parse(string(tmplContent))
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		internalServerErrorHandler(w, nil)
		return
	}

	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering template: %v", err)
		internalServerErrorHandler(w, nil)
	}
}

func constructCanonicalURL(r *http.Request) string {
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	requestURL := r.RequestURI
	return proto + "://" + r.Host + requestURL
}
