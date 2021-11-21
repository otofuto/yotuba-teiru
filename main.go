package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/otofuto/yotuba-teiru/pkg/database"
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

func main() {
	_ = godotenv.Load()
	port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", IndexHandle)
	http.HandleFunc("/favicon.ico", FaviconHandle)

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if r.Method == http.MethodGet {
		db := database.Connect()
		defer db.Close()
		sql := "select id, `name`, comment, `datetime`, replyto from `comments` order by `id` limit 30"
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
