FROM golang:1.8.0
MAINTAINER nsoushi

RUN mkdir /src

COPY ./lendo-polling-service /src/lendo-polling-service

ENTRYPOINT ["/src/lendo-polling-service"]