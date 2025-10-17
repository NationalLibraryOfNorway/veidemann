module github.com/NationalLibraryOfNorway/veidemann/browser-controller

go 1.23.0

toolchain go1.24.3

require (
	github.com/NationalLibraryOfNorway/veidemann/api v1.0.2
	github.com/NationalLibraryOfNorway/veidemann/log-service v0.4.1
	github.com/NationalLibraryOfNorway/veidemann/recorderproxy v0.4.1
	github.com/chromedp/cdproto v0.0.0-20201009231348-1c6a710e77de
	github.com/chromedp/chromedp v0.5.5
	github.com/docker/go-connections v0.4.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/mailru/easyjson v0.7.7
	github.com/nlnwa/whatwg-url v0.6.2
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/common v0.42.0
	github.com/rs/zerolog v1.34.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/testcontainers/testcontainers-go v0.19.0
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.10
	gopkg.in/rethinkdb/rethinkdb-go.v6 v6.2.2
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/containerd/containerd v1.6.19 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v23.0.1+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/getlantern/byteexec v0.0.0-20170405023437-4cfb26ec74f4 // indirect
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520 // indirect
	github.com/getlantern/elevate v0.0.0-20180207094634-c2e2e4901072 // indirect
	github.com/getlantern/errors v0.0.0-20190325191628-abdb3e3e36f7 // indirect
	github.com/getlantern/filepersist v0.0.0-20160317154340-c5f0cd24e799 // indirect
	github.com/getlantern/go-cache v0.0.0-20141028142048-88b53914f467 // indirect
	github.com/getlantern/golog v0.0.0-20190830074920-4ef2e798c2d7 // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55 // indirect
	github.com/getlantern/keyman v0.0.0-20180207174507-f55e7280e93a // indirect
	github.com/getlantern/mitm v0.0.0-20180205214248-4ce456bae650 // indirect
	github.com/getlantern/netx v0.0.0-20190110220209-9912de6f94fd // indirect
	github.com/getlantern/ops v0.0.0-20190325191751-d70cb0d6f85f // indirect
	github.com/getlantern/preconn v0.0.0-20180328114929-0b5766010efe // indirect
	github.com/getlantern/proxy v0.0.0-20190225163220-31d1cc06ed3d // indirect
	github.com/getlantern/reconn v0.0.0-20161128113912-7053d017511c // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/httphead v0.0.0-20200921212729-da3d93bc3c58 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.0.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/moby/patternmatcher v0.5.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/term v0.0.0-20221128092401-c43b287e0e0f // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/opencontainers/runc v1.1.3 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250512202823-5a2f75b736a9 // indirect
	gopkg.in/cenkalti/backoff.v2 v2.2.1 // indirect
)

replace github.com/getlantern/proxy => ../getlantern-proxy
