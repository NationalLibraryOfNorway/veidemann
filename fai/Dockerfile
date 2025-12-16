FROM golang:1.25 AS build

WORKDIR /go/src/app

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" .


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /go/src/app/fai /fai

EXPOSE 8081

ENTRYPOINT ["/fai"]
