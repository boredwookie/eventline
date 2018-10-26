package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/boredwookie/eventline/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"log"
	"net/http"
)

var db *sql.DB
func main(){
	// Get a handle to the SQLite database, using mattn/go-sqlite3
	var err error
	db, err = sql.Open("sqlite3", "./eventline.db")
	if err != nil{
		panic(err)
	}

	// Configure SQLBoiler to use the sqlite database
	boil.SetDB(db)

	// Start the API server
	http.HandleFunc("/postRecord", postRecord)
	log.Fatal(http.ListenAndServe(":8421", nil))

	//// Need to set a context for purposes I don't understand yet
	//ctx := context.Background()     // Dark voodoo magic, https://golang.org/pkg/context/#Background
	//
	//// This pulls 'all' of the books from the books table in the sample database (see schema below!)
	//eventlines, _ := models.Eventlines().All(ctx, db)
	//for _, event := range eventlines {
	//	fmt.Println(event.Subject)
	//}
}

func postRecord(w http.ResponseWriter, r *http.Request){
	// Requires a post
	if r.Method != http.MethodPost{
		http.Error(w, `{ "status": "400 BAD REQUEST" }`, http.StatusBadRequest)
		return
	}

	// Parse the input
	var record models.Record
	json.NewDecoder(r.Body).Decode(&record)

	// Check if the record already exists (if it does we'll want to update rather than overwrite
	if record.SourceRecordId.String == ""{
		http.Error(w, `{ "status": "400 BAD REQUEST", "details": "invalid SourceRecordId" }`, http.StatusBadRequest)
		return
	}

	records, dbErr := models.Records(qm.Where("SourceRecordId=?", record.SourceRecordId)).All(context.Background(), db)
	if dbErr != nil{
		fmt.Println(dbErr)
	}
	if len(records) != 1{
		// Insert the record
		if err := record.Insert(context.Background(), db, boil.Infer()); err != nil{
			fmt.Println(err)
		}
	} else {
		// Update the record
		records[0].Notes = null.StringFrom(records[0].Notes.String + "\n" + record.Notes.String)
		records[0].Update(context.Background(), db, boil.Infer())
	}



	// Return an "OK" message if things went well
	w.Write([]byte(`{ "status": "200 OK" }`))
}