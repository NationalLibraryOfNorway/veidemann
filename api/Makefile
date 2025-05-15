PROTOC_VERSION := 31.0
PROTOC_GEN_GO_VERSION := v1.31.0
PROTOC_GEN_GRPC_VERSION := "v1.5.1"
GO_MODULE := github.com/NationalLibraryOfNorway/veidemann/api
PROTO_DIR := ../proto
PROTOC_DIR := .protoc

PROTOC := $(PROTOC_DIR)/bin/protoc
PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

.PHONY: all
.PHONY: clean
.PHONY: distclean
.PHONY: build

all:	build

clean:
	@find . -name '*.pb.go' -delete

distclean: clean
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
	"${PROTOC_DIR}/bin/protoc" \
		--proto_path=${PROTO_DIR} \
		-I"${PROTOC_DIR}/include" \
		--go_out="." \
		--go_opt=module=${GO_MODULE} \
		--go-grpc_out="." \
		--go-grpc_opt=module=${GO_MODULE} \
		$(shell find ${PROTO_DIR} -iname '*.proto')
