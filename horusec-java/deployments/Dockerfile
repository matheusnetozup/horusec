FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

ADD . /go/src/github.com/ZupIT/horusec
WORKDIR /go/src/github.com/ZupIT/horusec
COPY . .

RUN env GOOS=linux GOARCH=amd64 go build -o /bin/horusec-java ./horusec-java/cmd/app/main.go

FROM golang:alpine

COPY --from=builder /bin/horusec-java /bin/horusec-java
RUN chmod +x /bin/horusec-java

