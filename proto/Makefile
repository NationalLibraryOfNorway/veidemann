PROTOC_VERSION := 31.0
PROTOC_GEN_GO_VERSION := v1.31.0
PROTOC_GEN_GRPC_VERSION := "v1.5.1"

API_DIR := ../api
PROTOC_DIR := ./.protoc

PROTOC := $(PROTOC_DIR)/bin/protoc
PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

.PHONY: all
.PHONY: clean
.PHONY: distclean
.PHONY: build

all:	build

clean:
	@cd $(API_DIR) && find . -not -path './go.*' -delete

distclean:
	@rm -rf "${PROTOC_DIR}" && \
	rm $(PROTOC_GEN_GO) && \
	rm $(PROTOC_GEN_GO_GRPC)

$(PROTOC):
	@echo "Installing protoc ${PROTOC_VERSION}..." && \
	mkdir -p "${PROTOC_DIR}" && \
	curl -s -L -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
	unzip protoc.zip -d "${PROTOC_DIR}" > /dev/null && \
	rm protoc.zip; \

$(PROTOC_GEN_GO):
	@echo "Installing protoc-gen-go ${PROTOC_GEN_GO_VERSION}..." && \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}

$(PROTOC_GEN_GO_GRPC):
	@echo "Installing protoc-gen-go-grpc ${PROTOC_GEN_GRPC_VERSION}..." && \
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GRPC_VERSION}

build: $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	@echo "Generating golang code from proto files..." && \
	find . -name '*.proto' -not -path './.*' \
	-exec \
		"${PROTOC_DIR}/bin/protoc" \
		-I. \
		-I"${PROTOC_DIR}/include" \
		--go_out="${API_DIR}" \
		--go_opt=paths=source_relative \
		--go-grpc_out="${API_DIR}" \
		--go-grpc_opt=paths=source_relative \
		'{}' \;
