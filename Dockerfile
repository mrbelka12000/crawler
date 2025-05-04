FROM golang:1.24.2-alpine3.20 AS race-build
WORKDIR /app

# install gcc + musl-dev so cgo can work
RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# enable cgo
ENV CGO_ENABLED=1

# build with -race
RUN go build -race -o main-race .

FROM alpine AS final-race
WORKDIR /
COPY --from=race-build /app/main-race /main
EXPOSE 8081
ENTRYPOINT ["/main"]