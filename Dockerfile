# syntax=docker/dockerfile:1
FROM golang:1.21.0
ENV PORT 8080
ENV HOSTNAME 0.0.0.0

EXPOSE 8080
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod tidy
COPY . ./
RUN go build -o /main
CMD [ "/main" ]
