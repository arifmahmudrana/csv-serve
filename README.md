CSV serve API
================

CSV serve API is an API that serves a promotion API end point cached backed using redis on top of memcache with persistent store backed by Apache cassandra. Components/services
1. API
2. CRON

## Tech
1. Golang 1.6
2. Apache cassandra
3. Memcached
4. Redis

## Running the project
1. Copy .env.example as .env
```sh
cp .env.example .env
```
2. Fill the `.env` missing values see comments in `.env.example`
3. Run
```sh
docker-compose up # or docker-compose up -d for detached mode
```
4. After CRON is successful from browser run API `BASE_URL/api/promotions/PROMOTION_ID` see `data/promotions.csv` for any promotion ID

## Todos

- [x] Development workflow
- [x] Caching
- [x] Dockerized
- [ ] Production Docker
- [ ] CRON independent from local file storage
- [ ] Write tests
- [ ] CI/CD

# Contributors

- [arifmahmudrana](https://github.com/arifmahmudrana)

## License
[MIT](LICENSE)
