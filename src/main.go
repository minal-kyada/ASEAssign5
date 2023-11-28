package src

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Items []struct {
		Tags  []string `json:"tags"`
		Owner struct {
			AccountID    int    `json:"account_id"`
			Reputation   int    `json:"reputation"`
			UserID       int    `json:"user_id"`
			UserType     string `json:"user_type"`
			ProfileImage string `json:"profile_image"`
			DisplayName  string `json:"display_name"`
			Link         string `json:"link"`
		} `json:"owner"`
		IsAnswered       bool   `json:"is_answered"`
		ViewCount        int    `json:"view_count"`
		AnswerCount      int    `json:"answer_count"`
		Score            int    `json:"score"`
		LastActivityDate int    `json:"last_activity_date"`
		CreationDate     int    `json:"creation_date"`
		QuestionID       int    `json:"question_id"`
		ContentLicense   string `json:"content_license"`
		Link             string `json:"link"`
		Title            string `json:"title"`
	} `json:"items"`
	HasMore        bool `json:"has_more"`
	QuotaMax       int  `json:"quota_max"`
	QuotaRemaining int  `json:"quota_remaining"`
}

func main() {

	// Database connection settings
	connectionName := "aseassign5:us-central1:mypostgres"
	dbUser := "postgres"
	dbPass := "root"
	dbName := "StackoverflowDB"

	dbURI := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable",
		connectionName, dbName, dbUser, dbPass)

	// Initialize the SQL DB handle
	log.Println("Initializing database connection")
	db, err := sql.Open("cloudsqlpostgres", dbURI)
	if err != nil {
		log.Fatalf("Error on initializing database connection: %s", err.Error())
	}
	defer db.Close()

	//Test the database connection
	log.Println("Testing database connection")
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error on database connection: %s", err.Error())
	}
	log.Println("Database connection established")

	log.Println("Database query done!")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
	}()

	GetStackIssues(db, "golang")

}

func GetStackIssues(db *sql.DB, qTitle string) {
	dropTable := `drop table if exists stack_issues`
	_, err := db.Exec(dropTable)
	if err != nil {
		panic(err)
	}

	createTable := `CREATE TABLE IF NOT EXISTS "stack_issues" (
						"id" SERIAL,	
						"title" VARCHAR(255),
						"display_name" VARCHAR(255),
						"account_id" VARCHAR(255),
    					"user_id" VARCHAR(255),
    					"question_id" VARCHAR(255),
    					"creation_date" VARCHAR(255),
    					"query" VARCHAR(255),
						PRIMARY KEY ("id") 
					);`

	_, _err := db.Exec(createTable)
	if _err != nil {
		panic(_err)
	}
	var url = "https://api.stackexchange.com/2.3/search?order=desc&sort=activity&intitle=" + qTitle + "&site=stackoverflow"

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	body, _ := ioutil.ReadAll(res.Body)
	var response Response
	json.Unmarshal(body, &response)

	// Store items in PostgreSQL database
	for i := 0; i < len(response.Items); i++ {

		title := response.Items[i].Title
		displayName := response.Items[i].Owner.DisplayName
		accountId := response.Items[i].Owner.AccountID
		userID := response.Items[i].Owner.UserID
		questionId := response.Items[i].QuestionID
		creationDate := response.Items[i].CreationDate

		loc, _ := time.LoadLocation("America/Chicago")
		date := time.Unix(int64(creationDate), 0).In(loc)
		cd := date.Format("2006-01-02T15:04:05 -07:00:00")
		sql := `INSERT INTO stack_issues ("title", "display_name", "account_id", "user_id", "question_id", "creation_date", "query") values($1, $2, $3, $4, $5, $6, $7)`

		_, err = db.Exec(
			sql,
			title,
			displayName,
			accountId,
			userID,
			questionId,
			cd,
			qTitle)

		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Items fetched and stored successfully!")

}
