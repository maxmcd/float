FROM golang:alpine

WORKDIR /opt

RUN apk add binutils

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build . && strip float

FROM alpine
RUN apk add openssh ncurses
COPY --from=0 /opt/float /bin/float
CMD ["/bin/float"]

