package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// Issue represents the structure of the GitHub issue.
type Issues []struct {
	Title      string `json:"title"`
	State      string `json:"state"`
	Created_At string `json:"created_at"`
	Body       string `json:"body"`
	Issue_id   string `json:"id"`
}

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

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests made",
		},
		[]string{"api"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of HTTP request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api"},
	)

	responseSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "Summary of HTTP response sizes in bytes",
		},
		[]string{"api"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(responseSize)
}

func main() {

	connectionName := "aseassign5:us-central1:mypostgres"
	dbUser := "postgres"
	dbPass := "root"
	dbName := "GitHubDB"

	dbURI := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable",
		connectionName, dbName, dbUser, dbPass)

	// Initialize the SQL DB handle
	log.Println("Initializing database connection")
	db, err := sql.Open("cloudsqlpostgres", dbURI)
	if err != nil {
		log.Fatalf("Error on initializing database connection: %s", err.Error())
	}

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

	GetGitIssues(db, "prometheus", "prometheus")
	GetGitIssues(db, "SeleniumHQ", "selenium")
	GetGitIssues(db, "openai", "openai-python")
	GetGitIssues(db, "docker", "compose")
	GetGitIssues(db, "milvus-io", "milvus")
	GetGitIssues(db, "golang", "go")
	db.Close()

	dbName = "StackoverflowDB"
	dbURI = fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable",
		connectionName, dbName, dbUser, dbPass)

	// Initialize the SQL DB handle
	log.Println("Initializing database connection")

	db, err = sql.Open("cloudsqlpostgres", dbURI)
	if err != nil {
		log.Fatalf("Error on initializing database connection: %s", err.Error())
	}
	GetStackIssues(db, "prometheus")
	GetStackIssues(db, "selenium")
	GetStackIssues(db, "openai")
	GetStackIssues(db, "docker")
	GetStackIssues(db, "milvus")
	GetStackIssues(db, "golang")
	db.Close()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9091", nil))

}

func GetGitIssues(db *sql.DB, owner string, repo string) {

	drop_table := `drop table if exists git_issues`
	_, err := db.Exec(drop_table)
	if err != nil {
		panic(err)
	}

	create_table := `CREATE TABLE IF NOT EXISTS "git_issues" (
						"id"   SERIAL , 
						"title" VARCHAR(255), 
						"state" VARCHAR(255), 
						"created_at" TIMESTAMP WITH TIME ZONE,
						"repo" VARCHAR(255),
						"body" VARCHAR(2048),
						"issue_id" int,
						PRIMARY KEY ("id") 
					);`

	_, _err := db.Exec(create_table)
	if _err != nil {
		panic(_err)
	}
	var url = "https://api.github.com/repos/" + owner + "/" + repo + "/issues?state=all"
	start := time.Now()
	res, err := http.Get(url)
	if err != nil {
		requestsTotal.WithLabelValues("GitError").Inc()
		requestDuration.WithLabelValues("GitError").Observe(time.Since(start).Seconds())
		panic(err)
	}
	duration := time.Since(start).Seconds()

	// Increment total requests metric
	requestsTotal.WithLabelValues("GitSuccess").Inc()

	// Observe request duration metric
	requestDuration.WithLabelValues("GitSuccess").Observe(duration)
	body, _ := ioutil.ReadAll(res.Body)
	responseSize.WithLabelValues("GitSuccess").Observe(float64(len(body)))
	var issuesList Issues
	json.Unmarshal(body, &issuesList)

	// Store issues in PostgreSQL database
	for i := 0; i < len(issuesList); i++ {
		title := issuesList[i].Title
		state := issuesList[i].State
		created_at := issuesList[i].Created_At
		body1 := issuesList[i].Body
		issue_id := issuesList[i].Issue_id

		sql := `INSERT INTO git_issues ("title", "state", "created_at","repo","body","issue_id") values($1, $2, $3, $4, $5, $6)`

		_, err = db.Exec(
			sql,
			title,
			state,
			created_at,
			repo,
			body1,
			issue_id)

		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Issues fetched and stored successfully!")

}

func GetStackIssues(db *sql.DB, qTitle string) {
	start := time.Now()
	dropTable := `drop table if exists stack_issues`
	_, err := db.Exec(dropTable)
	if err != nil {
		panic(err)
	}

	createTable := `CREATE TABLE IF NOT EXISTS "stack_issues" (
						"id" SERIAL,	
						"question" VARCHAR(255),
						"answer" bool,
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
		requestsTotal.WithLabelValues("StackoverflowError").Inc()
		requestDuration.WithLabelValues("StackoverflowError").Observe(time.Since(start).Seconds())
		panic(err)
	}

	body, _ := ioutil.ReadAll(res.Body)
	responseSize.WithLabelValues("StackoverflowSuccess").Observe(float64(len(body)))

	var response Response
	json.Unmarshal(body, &response)

	// Store items in PostgreSQL database
	for i := 0; i < len(response.Items); i++ {

		question := response.Items[i].Title
		answer := response.Items[i].IsAnswered
		displayName := response.Items[i].Owner.DisplayName
		accountId := response.Items[i].Owner.AccountID
		userID := response.Items[i].Owner.UserID
		questionId := response.Items[i].QuestionID
		creationDate := response.Items[i].CreationDate

		loc, _ := time.LoadLocation("America/Chicago")
		date := time.Unix(int64(creationDate), 0).In(loc)
		cd := date.Format("2006-01-02T15:04:05 -07:00:00")
		sql := `INSERT INTO stack_issues ("question", "answer", "display_name", "account_id", "user_id", "question_id", "creation_date", "query") values($1, $2, $3, $4, $5, $6, $7)`

		_, err = db.Exec(
			sql,
			question,
			answer,
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
	duration := time.Since(start).Seconds()

	// Increment total requests metric
	requestsTotal.WithLabelValues("StackoverflowSuccess").Inc()

	// Observe request duration metric
	requestDuration.WithLabelValues("StackoverflowSuccess").Observe(duration)

	fmt.Println("Items fetched and stored successfully!")

}
