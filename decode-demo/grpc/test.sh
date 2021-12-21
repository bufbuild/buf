#!/bin/bash

# Run the server in the background, but decode its output.
go run server/main.go | buf decode --debug | jq &

# Call the client so that the server writes a response.
go run client/main.go

wait
