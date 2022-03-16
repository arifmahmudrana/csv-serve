package main

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/arifmahmudrana/csv-serve/cassandra"
	"github.com/arifmahmudrana/csv-serve/csv"
	"github.com/arifmahmudrana/csv-serve/memcached"
	"github.com/arifmahmudrana/csv-serve/redis"
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
	m                 memcached.MemcachedRepository
	r                 redis.RedisRepository
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
		go func() {
			app.infoLog.Println("CRON deleting all memcached!!")
			if err := app.m.DeleteAll(); err != nil {
				app.errorLog.Println("Memcache: ", err)
				return
			}

			app.infoLog.Println("CRON deleting all memcached successfully!!")
		}()
		go func() {
			app.infoLog.Println("CRON deleting all redis!!")
			if err := app.r.DeleteAll(); err != nil {
				app.errorLog.Println("Redis: ", err)
				return
			}

			app.infoLog.Println("CRON deleting all redis successfully!!")
		}()

		return
	}

	if os.Getenv("CRON_EXIT_ON_ERROR") == "true" {
		app.errorLog.Fatal(err)
	}
	app.errorLog.Println(err)
}

func (app *application) ConnectMemcached() {
	m, err := memcached.NewMemcachedRepository(
		strings.Split(os.Getenv("MEMCACHED_SERVER"), ",")...)
	if err != nil {
		app.errorLog.Fatal(err)
	}

	app.m = m
}

func (app *application) ConnectRedis() {
	r, err := redis.NewRedisRepository(
		os.Getenv("REDIS_SERVER"), os.Getenv("REDIS_PREFIX"))
	if err != nil {
		app.errorLog.Fatal(err)
	}

	app.r = r
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
	app.ConnectMemcached()
	app.ConnectRedis()
	defer app.r.Close()

	c := cron.New()
	c.AddFunc(os.Getenv("CRON_SCHEDULE"), app.ProcessCSV)
	c.Run()
}
