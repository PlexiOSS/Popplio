FROM golang:1.25.5-trixie AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum go.work go.work.sum ./
COPY cmd/kitehelper/go.mod cmd/kitehelper/go.sum ./cmd/kitehelper/
RUN go mod download
COPY . .
RUN mkdir /out \
 && go build -trimpath -ldflags="-s -w" -o /out/popplio . \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build --chown=nonroot:nonroot /out /app
EXPOSE 8081 3010 3012
ENTRYPOINT ["/app/popplio"]
