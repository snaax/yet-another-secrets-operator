# Build the manager binary
FROM golang:1.22 as builder

WORKDIR /workspace

# Install controller-gen
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.2

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
RUN go mod download

# Copy the source code
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Create boilerplate header
RUN mkdir -p hack && echo '/*\nCopyright 2023.\n\nLicensed under the MIT License.\n*/' > hack/boilerplate.go.txt

# Generate DeepCopy code
RUN controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]