package main

import (
	"compress/gzip"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	htmlDir     = "./html"
	staticDir   = "./static"
	tmplFileExt = ".tmpl.html"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (grw gzipResponseWriter) Write(data []byte) (int, error) {
	return grw.Writer.Write(data)
}

func gzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()
		gzippedResponseWriter := gzipResponseWriter{Writer: gzipWriter, ResponseWriter: w}
		next.ServeHTTP(gzippedResponseWriter, r)
	})
}

func redirectWWW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Host, "www.") {
			http.Redirect(w, r, scheme+"://"+strings.TrimPrefix(r.Host, "www.")+r.RequestURI, 302)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func doesFileExist(pathToFile string) bool {
	info, err := os.Stat(filepath.Clean(pathToFile))
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

func bindTMPL(files ...string) (*template.Template, error) {
	for _, f := range files {
		if !doesFileExist(f) {
			return nil, errors.New("Template file missing " + f)
		}
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	path := strings.Split(r.URL.Path, "/")
	page := path[1]
	if r.URL.Path == "/" {
		page = "index"
	}

	if !doesFileExist(filepath.Join(htmlDir, "pages", page+tmplFileExt)) {
		http.Error(w, "page not found", 404)
		return
	}

	if len(path) > 2 && path[2] == "" {
		http.Redirect(w, r, scheme+"://"+r.Host+"/"+page, 302)
		return
	}

	if len(path) > 2 {
		http.Error(w, "page not found", 404)
		return
	}

	tmpl, err := bindTMPL(
		filepath.Join(htmlDir, "base"+tmplFileExt),
		filepath.Join(htmlDir, "pages", page+tmplFileExt),
	)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "internal server errors", 500)
		return
	}

	data := make(map[string]interface{})
	data["Path"] = r.URL.Path
	tmpl.ExecuteTemplate(w, "base", data)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	path := strings.Split(r.URL.Path, "/")
	post := path[2]

	if r.URL.Path == "/posts/" {
		http.Redirect(w, r, scheme+"://"+r.Host+"/", 302)
		return
	}

	if !doesFileExist(filepath.Join(htmlDir, "posts", post+tmplFileExt)) {
		http.Error(w, "post not found", 404)
		return
	}

	if len(path) > 3 && path[3] == "" {
		http.Redirect(w, r, scheme+"://"+r.Host+"/posts/"+post, 302)
		return
	}

	if len(path) > 3 {
		http.Error(w, "post not found", 404)
		return
	}

	tmpl, err := bindTMPL(
		filepath.Join(htmlDir, "base"+tmplFileExt),
		filepath.Join(htmlDir, "posts", post+tmplFileExt),
	)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "internal server errors", 500)
		return
	}

	data := make(map[string]interface{})
	data["Path"] = r.URL.Path
	tmpl.ExecuteTemplate(w, "base", data)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(staticDir, "favicon.ico"))
}
