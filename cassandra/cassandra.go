package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

type CassandraRepository interface {
	Close()
	Truncate(TBL_NAME) error
	InsertPromotions([]Promotion) error
	GetPromotionByID(id string) (*Promotion, error)
}

type cassandraRepository struct {
	s *gocql.Session
}

type Promotion struct {
	ID             string
	Price          float64
	ExpirationDate time.Time
}

type TBL_NAME string

const (
	TBL_PROMOTIONS TBL_NAME = "promotions"

	keySpace            = "csv_serve"
	createKeyspaceQuery = `CREATE KEYSPACE IF NOT EXISTS %s
  WITH REPLICATION = { 
   'class' : 'SimpleStrategy', 
   'replication_factor' : 1 
  }`
	createTablePromotionsQuery = `CREATE TABLE IF NOT EXISTS %s
	(
			id            		VARCHAR PRIMARY KEY,
			price      				double,
			expiration_date   timestamp
	);`
	truncateTableQuery    = `TRUNCATE %s;`
	insertQuery           = `INSERT INTO %s (%s) VALUES (%s)`
	promotionGetByIDQuery = `SELECT price, expiration_date FROM %s WHERE id = ? LIMIT 1`
)

func NewCassandraRepository(userName, password string, cassandraMaxRetryConnect int, hosts ...string) (CassandraRepository, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: userName,
		Password: password,
	}
	cluster.Consistency = gocql.Quorum

	var (
		session *gocql.Session
		err     error
	)

	for i := 1; i <= cassandraMaxRetryConnect; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			break
		}

		<-time.After(time.Second * time.Duration(i))
	}
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := session.Query(fmt.Sprintf(createKeyspaceQuery, keySpace)).WithContext(ctx).Exec(); err != nil {
		session.Close()
		return nil, err
	}

	session.Close()
	cluster.Keyspace = keySpace
	session, err = cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	ctx = context.Background()
	if err := session.Query(fmt.Sprintf(createTablePromotionsQuery, TBL_PROMOTIONS)).WithContext(ctx).Exec(); err != nil {
		session.Close()
		return nil, err
	}

	return &cassandraRepository{
		s: session,
	}, nil
}

func (c *cassandraRepository) Close() {
	c.s.Close()
}

func (c *cassandraRepository) InsertPromotions(promotions []Promotion) error {
	ctx := context.Background()
	b := c.s.NewBatch(gocql.UnloggedBatch).WithContext(ctx)

	for _, promotion := range promotions {
		b.Entries = append(b.Entries, gocql.BatchEntry{
			Stmt: fmt.Sprintf(
				insertQuery, TBL_PROMOTIONS, "id, price, expiration_date", "?, ?, ?"),
			Args: []interface{}{
				promotion.ID,
				promotion.Price,
				promotion.ExpirationDate,
			},
			Idempotent: true,
		})
	}

	return c.s.ExecuteBatch(b)
}

func (c *cassandraRepository) GetPromotionByID(id string) (*Promotion, error) {
	var promotion Promotion
	promotion.ID = id
	ctx := context.Background()
	err := c.s.
		Query(fmt.Sprintf(promotionGetByIDQuery, TBL_PROMOTIONS), promotion.ID).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&promotion.Price, &promotion.ExpirationDate)

	if err == gocql.ErrNotFound {
		return nil, nil
	}

	return &promotion, err
}

func (c *cassandraRepository) Truncate(tblName TBL_NAME) error {
	ctx := context.Background()
	return c.s.
		Query(fmt.Sprintf(truncateTableQuery, tblName)).
		WithContext(ctx).
		Exec()
}
