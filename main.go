package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"thumbnail/fetch"
	"time"
)

var mux map[string]func(http.ResponseWriter, *http.Request)
var tplcache map[string]*template.Template

const (
	TEMPLATE_DIR string = `./static/html/`
	STATIIC_DIR  string = `./static/`
)

type myHandler struct {
}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := mux[r.URL.Path]; ok {
		handler(w, r)
	} else {
		if result, _ := regexp.MatchString(`/static/*`, r.URL.Path); result {
			staticPage(w, r)
		}
		io.WriteString(w, "no page found")
	}
}

func init() {
	tplcache = make(map[string]*template.Template)
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	//缓存模板
	fileInfoList, err := ioutil.ReadDir(TEMPLATE_DIR)
	if err != nil {
		panic(err)
		return
	}
	var tplName, tplPath string
	for _, fileInfo := range fileInfoList {
		tplName = fileInfo.Name()
		if ext := path.Ext(tplName); ext != ".html" {
			continue
		}
		tplPath = TEMPLATE_DIR + tplName
		t := template.Must(template.ParseFiles(tplPath))
		tplcache[tplName] = t
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if err := renderPage(w, "home", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	if r.Method == "POST" {
		http.Error(w, "home page not found", http.StatusInternalServerError)
		return
	}
}
func searchPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		key := r.FormValue("key")
		if key == "" {
			http.Redirect(w, r, "/", 301)
			return
		}
		from := r.FormValue("from")
		if from == "" {
			from = "baidu"
		}

		ItemList, err := fetch.Fetch(key, from)
		if err != nil {
			fmt.Println("searchPage error", err)
		}

		time.Sleep(1 * time.Second)
		dataMap := make(map[string]interface{})
		dataMap["Keyword"] = key
		dataMap["Items"] = *ItemList
		dataMap["From"] = from
		renderPage(w, "home", dataMap)
	} else {
		io.WriteString(w, "No post method...")
	}
}
func viewPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		imgid := r.FormValue("imgID")
		imgpath := fetch.IMAGE_DIR + imgid
		if exist := isExists(imgpath); !exist {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image")
		http.ServeFile(w, r, imgpath)
	} else {
		io.WriteString(w, "No post method...")
	}
}
func swfPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}
func staticPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}

func domainPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, `./crossdomain.xml`)
}

func renderPage(w http.ResponseWriter, tpl string, locals map[string]interface{}) (err error) {
	t := tplcache[tpl+".html"]
	if t != nil {
		return t.Execute(w, locals)
	}
	return errors.New("find no template")
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	server := http.Server{
		Addr:           ":8888",
		Handler:        &myHandler{},
		ReadTimeout:    5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/"] = homePage
	mux["/s"] = searchPage
	mux["/view"] = viewPage
	mux["/static"] = staticPage
	mux["/crossdomain.xml"] = domainPage
	mux["/clipboard.swf"] = swfPage
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
