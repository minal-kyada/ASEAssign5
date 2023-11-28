# syntax=docker/dockerfile:1
FROM golang:1.19-alpine
ENV PORT 8080
ENV HOSTDIR 0.0.0.0

EXPOSE 8080
WORKDIR /app
RUN go mod init main
RUN go mod tidy
RUN go get github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres
RUN go get github.com/lib/pq
COPY . ./
RUN go build -o /main
CMD [ "/main" ]