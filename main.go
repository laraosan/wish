package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Wish structure to hold wish data
type Wish struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

const templatePath = "templates/index.html"

func main() {
	var err error
	// PostgreSQLに接続
	connStr := "host=127.0.0.1 user=tera password=password dbname=wish sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// データベースにテーブルがなければ作成
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wishes (
		id SERIAL PRIMARY KEY,
		content TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", homePage).Methods("GET")
	r.HandleFunc("/api/wishes", createWish).Methods("POST")
	r.HandleFunc("/api/wishes", getWishes).Methods("GET")
	r.HandleFunc("/api/wishes/{id}", updateWish).Methods("PUT")
	r.HandleFunc("/api/wishes/{id}", deleteWish).Methods("DELETE")
	http.Handle("/", r)

	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ルートページでテンプレートを返す
func homePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// POSTリクエストで願い事を追加
func createWish(w http.ResponseWriter, r *http.Request) {
	var wish Wish
	err := json.NewDecoder(r.Body).Decode(&wish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// PostgreSQLにデータを挿入
	err = insertWish(wish.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// PostgreSQLに願い事を挿入
func insertWish(content string) error {
	_, err := db.Exec("INSERT INTO wishes (content) VALUES ($1)", content)
	return err
}

// GETリクエストで願い事を一覧取得
// func getWishes(w http.ResponseWriter, r *http.Request) {
// 	wishes, err := fetchWishes()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(wishes)
// }

// PostgreSQLから願い事を取得
func fetchWishes() ([]Wish, error) {
	rows, err := db.Query("SELECT id, content FROM wishes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wishes []Wish
	for rows.Next() {
		var wish Wish
		err := rows.Scan(&wish.ID, &wish.Content)
		if err != nil {
			return nil, err
		}
		wishes = append(wishes, wish)
	}
	return wishes, nil
}

// PUTリクエストで願い事を更新
func updateWish(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	var wish Wish
	err := json.NewDecoder(r.Body).Decode(&wish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = updateWishContent(id, wish.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PostgreSQLで願い事を更新
func updateWishContent(id string, content string) error {
	_, err := db.Exec("UPDATE wishes SET content = $1 WHERE id = $2", content, id)
	return err
}

// DELETEリクエストで願い事を削除
func deleteWish(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	err := deleteWishByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PostgreSQLで願い事を削除
func deleteWishByID(id string) error {
	_, err := db.Exec("DELETE FROM wishes WHERE id = $1", id)
	return err
}

// GETリクエストで願い事を検索
func getWishes(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	var wishes []Wish
	var err error

	// searchパラメータが存在すればあいまい検索を実行
	if search != "" {
		wishes, err = searchWishes(search)
	} else {
		wishes, err = fetchWishes()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wishes)
}

// あいまい検索を実行
func searchWishes(search string) ([]Wish, error) {
	// PostgreSQLのLIKE演算子を使用してあいまい検索を行う
	query := `SELECT id, content FROM wishes WHERE content ILIKE '%' || $1 || '%'`
	rows, err := db.Query(query, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wishes []Wish
	for rows.Next() {
		var wish Wish
		err := rows.Scan(&wish.ID, &wish.Content)
		if err != nil {
			return nil, err
		}
		wishes = append(wishes, wish)
	}
	return wishes, nil
}
