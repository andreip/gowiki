package main

import (
    "html/template"
    "io/ioutil"
    "log"
    "net/http"
    "path/filepath"
    "regexp"
    "strings"
)

var pageDir = "data/"
var templateDir = "tmpl/"
var templateFilenames = []string{
    filepath.Join(templateDir, "edit.gohtml"),
    filepath.Join(templateDir, "view.gohtml"),
    filepath.Join(templateDir, "view_frontpage.gohtml"),
}

type Page struct {
    Title string
    Body []byte
}

func FilenameToPagename(filename string) string {
    // remove .txt part to get the page name
    return strings.TrimSuffix(filename, ".txt")
}

func (p *Page) save() error {
    filename := filepath.Join(pageDir, p.Title + ".txt")
    return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
    filename := filepath.Join(pageDir, title + ".txt")
    body, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return &Page{Title: title, Body: body}, nil
}

var templates = template.Must(template.ParseFiles(templateFilenames...))

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
    err := templates.ExecuteTemplate(w, tmpl + ".gohtml", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        http.Redirect(w, r, "/edit/"+title, http.StatusFound) // 302
        return
    }
    renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
    if r.Method == http.MethodPost {
        body := r.FormValue("body")
        p := &Page{Title: title, Body: []byte(body)}
        err := p.save()
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        http.Redirect(w, r, "/view/" + title, 302)
    }
}

func viewFrontPageHandler(w http.ResponseWriter, r *http.Request) {
    files, err := ioutil.ReadDir(pageDir)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    pages := make([]string, len(files))
    for i, file := range files {
        pages[i] = FilenameToPagename(file.Name())
    }
    renderTemplate(w, "view_frontpage", pages)
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m := validPath.FindStringSubmatch(r.URL.Path)
        // cannot allow anyone to edit FrontPage, even if they're not
        // affecting it, it looks like a bug
        if m == nil || (m != nil && m[2] == "FrontPage") {
            http.NotFound(w, r)
            return
        }
        fn(w, r, m[2])
    }
}

func redirectHandler(path, redirect string, code int) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != path {
            http.NotFound(w, r)
            return
        }
        http.Redirect(w, r, redirect, code)
    }
}

func main() {
    http.HandleFunc("/view/FrontPage", viewFrontPageHandler)
    http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))

    // redirect to view/FrontPage
    http.HandleFunc("/", redirectHandler("/", "/view/FrontPage", http.StatusFound))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
