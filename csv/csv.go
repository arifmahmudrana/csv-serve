package csv

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/arifmahmudrana/csv-serve/cassandra"
)

type CSVRepository interface {
	ReadLines(chan<- error) <-chan []string
	ProcessLines(<-chan []string, chan<- error) <-chan cassandra.Promotion
	SavePromotions(<-chan cassandra.Promotion, chan<- struct{}, chan<- error)
}

type csvRepository struct {
	r       io.Reader
	workers int
	cb      func([]cassandra.Promotion) error
}

func NewCSVRepository(r io.Reader, workers int, cb func([]cassandra.Promotion) error) CSVRepository {
	return &csvRepository{
		r:       r,
		workers: workers,
		cb:      cb,
	}
}

func (c *csvRepository) ReadLines(errChan chan<- error) <-chan []string {
	lines := make(chan []string, c.workers*4*4)

	go func() {
		csvReader := csv.NewReader(c.r)
		for {
			row, err := csvReader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}

				errChan <- err
			}

			lines <- row
		}

		close(lines)
	}()

	return lines
}

func (c *csvRepository) ProcessLines(lines <-chan []string, errChan chan<- error) <-chan cassandra.Promotion {
	promotions := make(chan cassandra.Promotion, c.workers*4)
	done := make(chan struct{}, c.workers*4)

	for i := 0; i < c.workers; i++ {
		go func() {
			for line := range lines {
				price, err := strconv.ParseFloat(line[1], 64)
				if err != nil {
					errChan <- err
				}

				expirationDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", line[2])
				if err != nil {
					errChan <- err
				}

				promotions <- cassandra.Promotion{
					ID:             line[0],
					Price:          price,
					ExpirationDate: expirationDate,
				}
			}

			done <- struct{}{}
		}()
	}

	go func() {
		for i := 0; i < c.workers; i++ {
			<-done
		}

		close(promotions)
	}()

	return promotions
}

func (c *csvRepository) SavePromotions(promotions <-chan cassandra.Promotion, done chan<- struct{}, errChan chan<- error) {
	p := make([]cassandra.Promotion, 0, 16)
	for promotion := range promotions {
		if len(p) == 16 {
			if err := c.cb(p); err != nil {
				errChan <- err
			}

			p = make([]cassandra.Promotion, 0, 16)
		}

		p = append(p, promotion)
	}

	if len(p) > 0 {
		if err := c.cb(p); err != nil {
			errChan <- err
		}
	}

	done <- struct{}{}
}
