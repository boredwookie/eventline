package setup

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func CreateSqliteDb(dbName string){
	dbCreateStatement := `CREATE TABLE "record" (
	"Id"	INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	"Identity"	TEXT,
	"Subject"	TEXT,
	"Notes"	TEXT,
	"Actions"	TEXT,
	"Tags"	TEXT,
	"SourceType"	TEXT,
	"SourceUserIdentity"	TEXT,
	"SourceRecordId"	TEXT NOT NULL UNIQUE,
	"SourceRaw"	TEXT,
	"DateAdded"	TEXT,
	"DateLastModified"	TEXT
);`

	// Get a handle to the SQLite database
	var err error
	db, err := sql.Open("sqlite3", dbName)
	if err != nil{
		panic(err)
	}
	defer db.Close()

	// Create the table
	_, err = db.Exec(dbCreateStatement)
	if err != nil {
		log.Println("Unable to create db: " + dbName)
		panic(err)
	}
}