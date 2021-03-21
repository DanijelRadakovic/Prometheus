FROM golang:1.16.2-alpine3.13 AS build
LABEL maintainer="danijelradakovic@uns.ac.rs"

WORKDIR /app
COPY ./ ./
RUN go mod download && \
    go build -o main .

FROM alpine:3.13 as runtime
WORKDIR /app
COPY --from=build /app/main ./
CMD /app/main
