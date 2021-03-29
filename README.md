# Node Status API

A monitoring API which stores node statuses for payment purposes.

Reports are stored in a SQLite database at `~/.nym/mixmining.db`

## Dependencies

* Go 1.15 or later

## Building and running


`go build` will build the binary. 

## Usage

The server exposes an HTTP interface which can be queried. To see documentation 
of the server's capabilities, go to http://<deployment-host>:8081/swagger/index.html in
your browser once you've run the server. You'll be presented with an overview
of functionality. All methods are runnable through the Swagger docs interface, 
so you can poke at the server to see what it does. 

## Developing

`go test ./...` will run the test suite.

From the top-level `node-status-api` directory, `swag init -g main.go --output docs/` rebuilds the Swagger docs.

