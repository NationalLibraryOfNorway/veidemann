package main

import (
	"log"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/reset/internal/file"
	"github.com/NationalLibraryOfNorway/veidemann/reset/internal/redis"
	"github.com/NationalLibraryOfNorway/veidemann/reset/internal/rethinkdb"
	"github.com/NationalLibraryOfNorway/veidemann/reset/internal/scylla"
	"github.com/NationalLibraryOfNorway/veidemann/reset/internal/version"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// configuration defaults

	rethinkdbUser := "admin"
	rethinkdbPassword := "rethinkdb"
	rethinkdbName := "veidemann"
	rethinkdbHost := "localhost"
	rethinkdbPort := 28015

	redisHost := "redis-master"
	redisPort := 6379

	scyllaHosts := []string{"scylla-client"}
	scyllaKeyspace := "v7n_v3_dev"

	directories := []string{
		"/warcs",
		"/validwarcs",
		"/invalidwarcs",
		"/backup/oos",
	}

	tables := map[string][]string{
		"veidemann": {
			"crawled_content",
			"events",
			"executions",
			"job_executions",
			"uri_queue",
		},
	}

	flag.StringVar(&rethinkdbHost, "rethinkdb-host", rethinkdbHost, "RethinkDb hostname")
	flag.IntVar(&rethinkdbPort, "rethinkdb-port", rethinkdbPort, "RethinkDb port")
	flag.StringVar(&rethinkdbName, "rethinkdb-name", rethinkdbName, "RethinkDb database name")
	flag.StringVar(&rethinkdbUser, "rethinkdb-user", rethinkdbUser, "RethinkDb user")
	flag.StringVar(&rethinkdbPassword, "rethinkdb-password", rethinkdbPassword, "RethinkDb password")

	flag.StringVar(&redisHost, "redis-host", redisHost, "Redis host")
	flag.IntVar(&redisPort, "redis-port", redisPort, "Redis port")

	flag.StringSliceVar(&scyllaHosts, "scylla-hosts", scyllaHosts, "List of db hosts")
	flag.StringVar(&scyllaKeyspace, "scylla-keyspace", scyllaKeyspace, "Name of keyspace")

	flag.StringSliceVar(&directories, "directories", directories, "directories to clean")

	flag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	err := viper.BindPFlags(flag.CommandLine)
	if err != nil {
		log.Fatal(err)
	}

	rethinkdbUser = viper.GetString("rethinkdb-user")
	rethinkdbPassword = viper.GetString("rethinkdb-password")
	rethinkdbName = viper.GetString("rethinkdb-name")
	rethinkdbHost = viper.GetString("rethinkdb-host")
	rethinkdbPort = viper.GetInt("rethinkdb-port")

	rethinkdbOptions := rethinkdb.Options{
		Name:     rethinkdbName,
		Host:     rethinkdbHost,
		Port:     rethinkdbPort,
		Username: rethinkdbUser,
		Password: rethinkdbPassword,
	}

	redisHost = viper.GetString("redis-host")
	redisPort = viper.GetInt("redis-port")

	scyllaHosts = viper.GetStringSlice("scylla-hosts")
	scyllaKeyspace = viper.GetString("scylla-keyspace")

	directories = viper.GetStringSlice("directories")

	log.Printf("Veidemann reset, version: %s\n", version.Version)

	// Delete files
	file.NewDirectoryCleaner(directories).RemoveFiles()

	// Empty RethinkDB tables
	rethinkClient := rethinkdb.NewClient(rethinkdbOptions)
	err = rethinkClient.Connect()
	if err != nil {
		panic(err)
	}
	defer rethinkClient.Disconnect()
	rethinkClient.Clean(tables)

	// Flush redis
	redisClient, err := redis.NewClient(redisHost, redisPort)
	if err != nil {
		panic(err)
	}
	defer redisClient.Close()
	redisClient.Flush()

	// Drop scylla keyspace
	scyllaClient := scylla.NewClient(scyllaHosts...)
	err = scyllaClient.Connect()
	if err != nil {
		panic(err)
	}
	defer scyllaClient.Disconnect()
	scyllaClient.Drop(scyllaKeyspace)

	log.Printf("Veidemann reset completed successfully")
}
