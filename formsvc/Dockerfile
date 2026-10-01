FROM golang:1-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go vet ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /formsvc .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /formsvc /formsvc
USER nonroot:nonroot
EXPOSE 8081
ENTRYPOINT ["/formsvc"]
