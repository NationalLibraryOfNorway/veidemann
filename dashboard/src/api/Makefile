PROTOC_VERSION := 31.0
PROTOC_GEN_JS_VERSION := 3.21.4
PROTOC_GEN_GRPC_WEB_VERSION := 1.5.0
PROTO_DIR := "../../../proto"

PROTOC_DIR := .protoc
PROTOC := $(PROTOC_DIR)/bin/protoc
PROTOC_GEN_GRPC_WEB := $(PROTOC_DIR)/bin/protoc-gen-grpc-web
PROTOC_GEN_JS := $(PROTOC_DIR)/bin/protoc-gen-js

DIRS := browsercontroller commons config contentwriter controller dnsresolver eventhandler frontier log ooshandler report robotsevaluator scopechecker uricanonicalizer
OUT := "."

.PHONY: all
.PHONY: clean
.PHONY: distclean
.PHONY: build

all:    $(DIRS)

clean:
	@echo "Cleaning up..." && \
  for dir in $(DIRS); do rm -rf $$dir; done

distclean: clean
	@rm -rf "${PROTOC_DIR}";

$(PROTOC):
	@echo "Installing $(PROTOC) ${PROTOC_VERSION}..." && \
	mkdir -p "${PROTOC_DIR}" && \
	curl -s -L -o protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" && \
	unzip protoc.zip -d "${PROTOC_DIR}" > /dev/null && \
	rm protoc.zip

$(PROTOC_GEN_JS):
	@echo "Installing ${PROTOC_GEN_JS} ${PROTOC_GEN_JS_VERSION}..." && \
	mkdir -p "${PROTOC_DIR}" && \
	curl -s -L -o protoc-gen-js.zip "https://github.com/protocolbuffers/protobuf-javascript/releases/download/v${PROTOC_GEN_JS_VERSION}/protobuf-javascript-${PROTOC_GEN_JS_VERSION}-linux-x86_64.zip" && \
	unzip protoc-gen-js.zip -d "${PROTOC_DIR}" > /dev/null && \
	rm protoc-gen-js.zip

$(PROTOC_GEN_GRPC_WEB):
	@echo "Installing ${PROTOC_GEN_GRPC_WEB} ${PROTOC_GEN_GRPC_WEB_VERSION}..." && \
	mkdir -p "${PROTOC_DIR}/bin" && \
	curl -s -L -o "${PROTOC_DIR}/bin/protoc-gen-grpc-web" "https://github.com/grpc/grpc-web/releases/download/${PROTOC_GEN_GRPC_WEB_VERSION}/protoc-gen-grpc-web-${PROTOC_GEN_GRPC_WEB_VERSION}-linux-x86_64" && \
	chmod +x "${PROTOC_DIR}/bin/protoc-gen-grpc-web";

$(DIRS): $(PROTOC) $(PROTOC_GEN_GRPC_WEB) $(PROTOC_GEN_JS)
	@echo "Generating javascript code from proto files..." && \
	"${PROTOC}" \
	--proto_path=${PROTO_DIR} \
	-I"${PROTOC_DIR}/include" \
	--plugin=protoc-gen-grpc-web=${PROTOC_GEN_GRPC_WEB} \
  --plugin=protoc-gen-js=${PROTOC_GEN_JS} \
	--js_out=import_style=commonjs:${OUT} \
	--grpc-web_out=import_style=commonjs+dts,mode=grpcwebtext:${OUT} \
	$(shell find ${PROTO_DIR} -iname '*.proto');
