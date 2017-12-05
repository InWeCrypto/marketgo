FROM golang:1.9.2-stretch

LABEL maintainer="support@inwecrypto.com"

COPY . /go/src/github.com/inwecrypto/marketgo

RUN go install github.com/inwecrypto/marketgo/cmd/marketgo && rm -rf /go/src

VOLUME ["/etc/inwecrypto/marketgo"]

WORKDIR /etc/inwecrypto/marketgo

EXPOSE 8000

CMD ["/go/bin/marketgo","--conf","/etc/inwecrypto/marketgo/marketgo.json"]