FROM golang:latest

WORKDIR /application
COPY . .
USER root
RUN apt-get update \
  && apt-get install -y sudo
RUN make generate && (GOOS=linux GOARCH=amd64 make build)
CMD ["./bin/library"]
