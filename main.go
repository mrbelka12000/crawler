package main

import (
	"context"
)

func main() {

	ctx := context.Background()

	db, err := connect(ctx)
	if err != nil {
		panic(err)
	}

	cache, err := ConnectToRedis()
	if err != nil {
		panic(err)
	}

	parser := NewParser(db, cache)

	parser.Walk(ctx, "https://docs.docker.com/reference/cli/docker/compose/down/")
}
