package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/boredwookie/eventline/models"
	"github.com/boredwookie/eventline/setup"
	"github.com/boredwookie/eventline/slackbot"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var db *sql.DB
func main(){
	// Read in the user flags
	slackToken := flag.String("SlackUserToken", "", "Enter your 'xoxp-...' slack user token")
	fqdns := flag.String("FQDNs", "localhost", "Enter the comma-separated list of FQDNs the local certificate should be generated for. Ex: 'localhost' -or- 'mydomain.local,anotherdomain.tld")
	dbName := flag.String("DbName", "eventline.db", "What should the sqlite db name be? Defaults to 'eventline.db'")
	flag.Parse()

	// If the SQLite database doesn't exist, create it
	if _, err := os.Stat(*dbName); os.IsNotExist(err){
		setup.CreateSqliteDb(*dbName)
	}

	// If a self-signed certificate is not available, create it
	if _, err := os.Stat("cert.pem"); os.IsNotExist(err){
		setup.GenerateSelfSignedCert(*fqdns)
	}

	// Get a handle to the SQLite database, using mattn/go-sqlite3
	var err error
	db, err = sql.Open("sqlite3", "./eventline.db")
	if err != nil{
		panic(err)
	}

	// Configure SQLBoiler to use the sqlite database
	boil.SetDB(db)

	// Load the stars from Slack (every few seconds)
	interval := time.NewTicker(5 * time.Second)
	quitSlack := make(chan struct{})
	go func() {
		for {
			select {
				case <- interval.C:
					starredMessages := slackbot.LoadStars(*slackToken)
					for _, messageRecord := range starredMessages{
						storeRecord(messageRecord)
					}
				case <- quitSlack:
					interval.Stop()
					return
			}
		}
	}()


	// Start the API server (HTTP)
	//		Testing for slack since it won't likely trust my self signed certificate
	//go func(){
	//	log.Fatal(http.ListenAndServe(":8421", &SlackHandler{}))
	//}()

	// Start the API server (HTTPS)
	http.HandleFunc("/postRecord", postRecord)
	//log.Fatal(http.ListenAndServe(":8421", nil))	// http listener on 8421
	log.Fatal(http.ListenAndServeTLS(":8422", "cert.pem", "key.pem", nil))	// https listener on 8422

	//// Need to set a context for purposes I don't understand yet
	//ctx := context.Background()     // Dark voodoo magic, https://golang.org/pkg/context/#Background
	//
	//// This pulls 'all' of the books from the books table in the sample database (see schema below!)
	//eventlines, _ := models.Eventlines().All(ctx, db)
	//for _, event := range eventlines {
	//	fmt.Println(event.Subject)
	//}
}

type SlackHandler struct{

}

func (sh *SlackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sh.postSlack(w, r)
}

func (sh *SlackHandler) postSlack(w http.ResponseWriter, r *http.Request) {
	fmt.Println(ioutil.ReadAll(r.Body))
}

func postRecord(w http.ResponseWriter, r *http.Request){
	// Requires a post
	//		_SIGH_ or an 'OPTIONS' because apparently that's a thing from webclients...
	//		Incredibly helpful blog post: https://flaviocopes.com/golang-enable-cors/
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method != http.MethodPost{
		if r.Method == http.MethodOptions{
			return
		}
		http.Error(w, `{ "status": "400 BAD REQUEST" }`, http.StatusBadRequest)
		fmt.Println("Error 400: Did not submit a POST or OPTIONS request. Received: " + r.Method)
		fmt.Println()
		return
	}

	// Parse the input
	var record models.Record
	json.NewDecoder(r.Body).Decode(&record)

	// Check if the record already exists (if it does we'll want to update rather than overwrite
	if record.SourceRecordId == ""{
		http.Error(w, `{ "status": "400 BAD REQUEST", "details": "invalid SourceRecordId" }`, http.StatusBadRequest)
		fmt.Println("Error 400: bad source record id. Received nothing")
		return
	}


	if storeRecord(record) != nil{
		fmt.Println("Error storing record!")
	}


	// Return an "OK" message if things went well
	w.Write([]byte(`{ "status": "200 OK" }`))
}

func storeRecord(record models.Record) error{
	records, dbErr := models.Records(qm.Where("SourceRecordId=?", record.SourceRecordId)).All(context.Background(), db)
	if dbErr != nil{
		fmt.Println(dbErr)
		return errors.New("Unable to load db: "+ dbErr.Error())
	}
	if len(records) != 1{
		// Insert the record
		if err := record.Insert(context.Background(), db, boil.Infer()); err != nil{
			fmt.Println(err)
			return errors.New("Unable to insert record: "+ dbErr.Error())
		}
		fmt.Println("Record inserted: " + record.SourceRecordId)
	} else {
		// Update the record
		records[0].Notes = null.StringFrom(records[0].Notes.String + "\n" + record.Notes.String)
		records[0].Update(context.Background(), db, boil.Infer())
		fmt.Println("Record Updated: " + record.SourceRecordId)
	}

	return nil
}