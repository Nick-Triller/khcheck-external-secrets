FROM golang:1.17-buster as build
ENV CGO_ENABLED=0
WORKDIR /workdir
# Add and install dependencies first (improves layer caching)
COPY go.mod .
COPY go.sum .
RUN go mod download
# Build binary
COPY . .
RUN make build


FROM gcr.io/distroless/static:latest
USER 1000
# Copy binary
COPY --from=build /workdir/khcheck-external-secrets /
ENTRYPOINT ["/khcheck-external-secrets"]
