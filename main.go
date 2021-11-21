package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/otofuto/yotuba-teiru/pkg/database"
	"golang.org/x/net/html"
)

var port string

type TempContext struct {
	Message string `json:"message"`
}

type Comment struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Datetime string `json:"datetime"`
	Comment  string `json:"comment"`
	Email    string
	Ip       string
	ReplyTo  int `json:"replyto"`
}

type Ogp struct {
	Image       string `json:"image"`
	ImageToOgp  string `json:"image_to_ogp"`
	Url         string `json:"url"`
	Title       string `json:"title"`
	SiteName    string `json:"site_name"`
	Description string `json:"description"`
	Valid       bool   `json:"valid"`
}

func main() {
	_ = godotenv.Load()
	port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", IndexHandle)
	http.HandleFunc("/favicon.ico", FaviconHandle)
	http.HandleFunc("/ogpimg/", OgpImgHandle)
	http.HandleFunc("/ogp/", OgpHandle)

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if r.Method == http.MethodGet {
		db := database.Connect()
		defer db.Close()
		sql := "select id, names, comment, dt, replyto from comments order by id limit 30"
		rows, err := db.Query(sql)
		msg := ""
		if err != nil {
			log.Println(err)
		} else {
			defer rows.Close()
			comments := make([]Comment, 0)
			for rows.Next() {
				var com Comment
				err = rows.Scan(&com.Id, &com.Name, &com.Comment, &com.Datetime, &com.ReplyTo)
				if err == nil {
					comments = append(comments, com)
				}
			}
			bytes, _ := json.Marshal(comments)
			msg = string(bytes)
		}
		temp := template.Must(template.ParseFiles("template/index.html"))
		if err := temp.Execute(w, TempContext{
			Message: msg,
		}); err != nil {
			log.Println(err)
			http.Error(w, "HTTP 500 Internal server error", 500)
			return
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func FaviconHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/ico")
	file, err := os.Open("./static/favicon.ico")
	if err != nil {
		http.Error(w, "failed to open the favicon", 500)
		return
	}
	defer file.Close()
	io.Copy(w, file)
}

func OgpImgHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))

	if r.Method == http.MethodGet {
		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "Parameter 'url' does not allow empty.", 400)
		} else {
			res, err := http.Get(url)
			if err != nil {
				log.Println(err)
				http.Error(w, "Access denied.", 500)
				return
			}
			defer res.Body.Close()
			io.Copy(w, res.Body)
		}
	} else {
		http.Error(w, "Supported method is only 'GET'.", http.StatusMethodNotAllowed)
	}
}

func OgpHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "Parameter 'url' does not allow empty.", 400)
		} else {
			ogp := GetOgp(url)
			bytes, _ := json.Marshal(ogp)
			fmt.Fprintf(w, string(bytes))
		}
	} else {
		http.Error(w, "Supported method is only 'GET'.", http.StatusMethodNotAllowed)
	}
}

func GetOgp(url string) Ogp {
	ret := Ogp{
		Valid: false,
	}
	if strings.TrimSpace(url) == "" {
		return ret
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Println("main.go GetOgp(url string)")
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	node, err := html.Parse(strings.NewReader(bytes.NewBuffer(body).String()))
	meta := findNodesByTagName(node, "meta")
	for _, m := range meta {
		if prop := getAttribute(m, "property"); strings.HasPrefix(prop, "og:") {
			if prop == "og:image" {
				ret.Image = getAttribute(m, "content")
				ret.ImageToOgp = getAttribute(m, "content")
				if strings.HasPrefix(ret.Image, "http://") {
					ret.Image = "/ogpimg?url=" + ret.Image
					ret.ImageToOgp = "https://yotuba-teiru.herokuapp.com/ogpimg?url=" + ret.ImageToOgp
				}
			} else if prop == "og:url" {
				ret.Url = getAttribute(m, "content")
			} else if prop == "og:title" {
				ret.Title = getAttribute(m, "content")
			} else if prop == "og:site_name" {
				ret.SiteName = getAttribute(m, "content")
			} else if prop == "og:description" {
				ret.Description = getAttribute(m, "content")
			}
		}
	}
	if ret.Image != "" || ret.Url != "" || ret.Title != "" || ret.SiteName != "" || ret.Description != "" {
		ret.Valid = true
	}

	return ret
}

func findNodesByTagName(parent *html.Node, tagname string) []*html.Node {
	nodes := make([]*html.Node, 0)
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Data == tagname {
			nodes = append(nodes, child)
		}
		ns := findNodesByTagName(child, tagname)
		for _, n := range ns {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func getAttribute(node *html.Node, attrname string) string {
	for _, attr := range node.Attr {
		if attr.Key == attrname {
			return attr.Val
		}
	}
	return ""
}

//GETでは使えない
func isset(r *http.Request, keys []string) bool {
	for _, v := range keys {
		exist := false
		for k, _ := range r.MultipartForm.Value {
			if v == k {
				exist = true
			}
		}
		if !exist {
			return false
		}
	}
	return true
}
