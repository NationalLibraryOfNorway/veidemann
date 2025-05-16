module github.com/NationalLibraryOfNorway/veidemann/reset

go 1.14

require (
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.15.0
	gopkg.in/rethinkdb/rethinkdb-go.v6 v6.2.2
)

require (
	github.com/gocql/gocql v1.3.1
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.7.3
