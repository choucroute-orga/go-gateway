# Gateway

This is the API Gateway for the recipes management system. It is a REST API that allows you to manage the recipes. Handles all the requests and responses from the clients.


### Start the server

```bash
cp .env.example .env
export $(cat .env | xargs)
docker-compose up
go run main.go
```
