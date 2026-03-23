FROM golang:1.23-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o envious-server ./cmd/server

FROM alpine:3.19
WORKDIR /srv
COPY --from=build /app/envious-server /usr/local/bin/envious-server
EXPOSE 8080
ENV PORT=8080
CMD ["envious-server"]

