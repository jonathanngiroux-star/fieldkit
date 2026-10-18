FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /fieldkit ./cmd/fieldkit

FROM scratch
COPY --from=build /fieldkit /fieldkit
VOLUME /data
ENV FIELDKIT_DATA_DIR=/data
ENTRYPOINT ["/fieldkit"]
CMD ["version"]
