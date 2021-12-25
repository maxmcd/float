FROM golang:alpine

WORKDIR /opt

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build .

FROM alpine
RUN apk add ssh
COPY --from=0 /opt/float /bin/float
CMD ["/bin/float"]

