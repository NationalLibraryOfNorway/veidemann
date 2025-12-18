/*
 * Copyright 2021 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package flags

import (
	"strings"
	"time"

	"github.com/nlnwa/gowarc"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func ParseFlags() (Options, error) {
	flags := pflag.CommandLine

	flags.String("interface", "", "interface the contentwriter api listens to. No value means all interfaces.")
	flags.Int("port", 8080, "port the contentwriter listens to.")
	flags.String("hostname", "", "")
	flags.String("warc-dir", "", "")
	flags.String("warc-version", "1.1", "which WARC version to use for generated records. Allowed values: 1.0, 1.1")
	flags.Int("warc-writer-pool-size", 1, "")
	flags.Bool("flush-record", false, "if true, flush WARC-file to disk after each record.")
	flags.String("work-dir", "", "")
	flags.Int("termination-grace-period-seconds", 0, "")
	flags.Bool("strict", false, "if true, use strict record validation")

	flags.String("db-host", "rethinkdb-proxy", "DB host")
	flags.Int("db-port", 28015, "DB port")
	flags.String("db-name", "veidemann", "DB name")
	flags.String("db-user", "", "Database username")
	flags.String("db-password", "", "Database password")
	flags.Duration("db-query-timeout", 1*time.Minute, "Database query timeout")
	flags.Int("db-max-retries", 5, "Max retries when database query fails")
	flags.Int("db-max-open-conn", 10, "Max open database connections")
	flags.Bool("db-use-opentracing", false, "Use opentracing for database queries")
	flags.Duration("db-cache-ttl", 5*time.Minute, "How long to cache results from database")

	flags.String("redis-host", "redis", "Redis host")
	flags.String("redis-password", "", "Redis password")
	flags.Int("redis-port", 6379, "Redis port")
	flags.Int("redis-db", 1, "Redis database number")

	flags.String("metrics-interface", "", "Interface for exposing metrics. Empty means all interfaces")
	pflag.String("metrics-address", ":9153", "address to expose metrics on")
	pflag.String("metrics-path", "/metrics", "path to expose metrics on")

	flags.String("log-level", "info", "log level, available levels are panic, fatal, error, warn, info, debug and trace")
	flags.String("log-formatter", "logfmt", "log formatter, available values are logfmt and json")

	pflag.String("s3-address", "localhost:9000", "s3 endpoint (address:port)")
	pflag.String("s3-bucket-name", "", "name of bucket to upload files to")
	pflag.String("s3-access-key-id", "", "access key ID")
	pflag.String("s3-secret-access-key", "", "secret access key")
	pflag.String("s3-token", "", "token to use for s3 authentication (optional)")
	pflag.Bool("s3-secure", false, "use secure connection to S3 endpoint")

	pflag.Parse()

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("CONTENTWRITER")
	viper.AutomaticEnv()

	return Options{}, viper.BindPFlags(flags)
}

type Options struct {
}

func (o Options) HostName() string {
	return viper.GetString("hostname")
}

func (o Options) WarcDir() string {
	return viper.GetString("warc-dir")
}

func (o Options) WarcWriterPoolSize() int {
	return viper.GetInt("warc-writer-pool-size")
}

func (o Options) WorkDir() string {
	return viper.GetString("work-dir")
}

func (o Options) TerminationGracePeriodSeconds() int {
	return viper.GetInt("termination-grace-period-seconds")
}

func (o Options) WarcVersion() *gowarc.WarcVersion {
	v := viper.GetString("warc-version")
	switch v {
	case "1.0":
		return gowarc.V1_0
	case "1.1":
		return gowarc.V1_1
	default:
		panic("Unsupported WARC version: " + v)
	}
}

func (o Options) FlushRecord() bool {
	return viper.GetBool("flush-record")
}

func (o Options) UseStrictValidation() bool {
	return viper.GetBool("strict-validation")
}

func (o Options) LogLevel() string {
	return viper.GetString("log-level")
}

func (o Options) LogFormatter() string {
	return viper.GetString("log-formatter")
}

func (o Options) Interface() string {
	return viper.GetString("interface")
}

func (o Options) Port() int {
	return viper.GetInt("port")
}

func (o Options) DbHost() string {
	return viper.GetString("db-host")
}

func (o Options) DbPort() int {
	return viper.GetInt("db-port")
}

func (o Options) DbName() string {
	return viper.GetString("db-name")
}

func (o Options) DbUser() string {
	return viper.GetString("db-user")
}

func (o Options) DbPassword() string {
	return viper.GetString("db-password")
}

func (o Options) DbQueryTimeout() time.Duration {
	return viper.GetDuration("db-query-timeout")
}

func (o Options) DbMaxRetries() int {
	return viper.GetInt("db-max-retries")
}

func (o Options) DbMaxOpenConn() int {
	return viper.GetInt("db-max-open-conn")
}

func (o Options) DbUseOpenTracing() bool {
	return viper.GetBool("db-use-opentracing")
}

func (o Options) DbCacheTTL() time.Duration {
	return viper.GetDuration("db-cache-ttl")
}

func (o Options) MetricsInterface() string {
	return viper.GetString("metrics-interface")
}

func (o Options) MetricsPort() int {
	return viper.GetInt("metrics-port")
}

func (o Options) MetricsPath() string {
	return viper.GetString("metrics-path")
}

func (o Options) RedisHost() string {
	return viper.GetString("redis-host")
}

func (o Options) RedisPort() int {
	return viper.GetInt("redis-port")
}

func (o Options) RedisDb() int {
	return viper.GetInt("redis-db")
}

func (o Options) RedisPassword() string {
	return viper.GetString("redis-password")
}

func (o Options) S3Address() string {
	return viper.GetString("s3-address")
}

func (o Options) S3BucketName() string {
	return viper.GetString("s3-bucket-name")
}

func (o Options) S3AccessKeyID() string {
	return viper.GetString("s3-access-key-id")
}

func (o Options) S3SecretAccessKey() string {
	return viper.GetString("s3-secret-access-key")
}

func (o Options) S3Token() string {
	return viper.GetString("s3-token")
}

func (o Options) S3Secure() bool {
	return viper.GetBool("s3-secure")
}
