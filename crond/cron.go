package main

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/arifmahmudrana/csv-serve/cassandra"
	"github.com/arifmahmudrana/csv-serve/csv"
	"github.com/robfig/cron/v3"
)

var (
	workers = runtime.NumCPU()
)

const version = "1.0.0"

type application struct {
	infoLog, errorLog *log.Logger
	version           string
	db                cassandra.CassandraRepository
}

func (app *application) ConnectCassandra() {
	cassandraMaxRetryConnect, err := strconv.Atoi(os.Getenv("CASSANDRA_MAX_RETRY_CONNECT"))
	if err != nil {
		app.errorLog.Fatal(err)
	}

	db, err := cassandra.NewCassandraRepository(
		os.Getenv("CASSANDRA_USER"),
		os.Getenv("CASSANDRA_PASSWORD"),
		cassandraMaxRetryConnect,
		strings.Split(os.Getenv("CASSANDRA_DB_HOST"), ",")...)
	if err != nil {
		app.errorLog.Fatal(err)
	}
	app.db = db
}

func (app *application) ProcessCSV() {
	app.infoLog.Println("Running CRON to process CSV file...")
	if err := app.db.Truncate(cassandra.TBL_PROMOTIONS); err != nil {
		app.errorLog.Fatal(err)
	}

	r, err := os.Open(os.Getenv("DATA_FILE_PATH"))
	if err != nil {
		app.errorLog.Fatal(err)
	}
	defer r.Close()

	csvRepository := csv.NewCSVRepository(r, workers, app.db.InsertPromotions)
	err = csvRepository.Process()
	if err == nil {
		app.infoLog.Println("CRON run successfully")
		return
	}

	if os.Getenv("CRON_EXIT_ON_ERROR") == "true" {
		app.errorLog.Fatal(err)
	}
	app.errorLog.Println(err)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	app := &application{
		infoLog:  log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		errorLog: log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
		version:  version,
	}

	app.ConnectCassandra()
	defer app.db.Close()

	c := cron.New()
	c.AddFunc(os.Getenv("CRON_SCHEDULE"), app.ProcessCSV)
	c.Run()
}
