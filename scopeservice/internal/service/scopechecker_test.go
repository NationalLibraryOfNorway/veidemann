package service

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/script"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func init() {
	script.InitializeCanonicalizationProfiles(false)
}

const defaultScopeScript = `
isScheme(param('scope_allowedSchemes')).otherwise(Blocked)
isSameHost(param('scope_includeSubdomains'), altSeeds=param('scope_altSeeds')).then(Include, continueEvaluation=True).otherwise(Blocked, continueEvaluation=False)
maxHopsFromSeed(param('scope_maxHopsFromSeed'), param('scope_hopsIncludeRedirects')).then(TooManyHops)
isUrl(param('scope_excludedUris')).then(Blocked)`

const defaultScopeScriptWithPathPrefix = `
isScheme(param('scope_allowedSchemes')).otherwise(Blocked)
isSameHost(param('scope_includeSubdomains'), altSeeds=param('scope_altSeeds')).then(Include, continueEvaluation=True).otherwise(Blocked, continueEvaluation=False)
maxHopsFromSeed(param('scope_maxHopsFromSeed'), param('scope_hopsIncludeRedirects')).then(TooManyHops)
isPathPrefix(param('scope_excludedPathPrefixes')).then(Blocked)
isUrl(param('scope_excludedUris')).then(Blocked)`

func TestScopeCheckerServer_ScopeCheck(t *testing.T) {
	server := &ScopeChecker{}
	qUri := newQUri("http://foo.bar/aa bb/cc?jsessionid=1&foo#bar", "http://foo.bar/", "RL")
	badQUri := newQUri("http://%00foo.bar/aa bb/cc?jsessionid=1&foo#bar", "http://foo.bar/", "RL")

	tests := []struct {
		name   string
		script string
		qUri   *frontierV1.QueuedUri
		debug  bool
		want   *scopecheckerV1.ScopeCheckResponse
	}{
		{"exclude", "test(True).then(ChaffDetection)", qUri, false, &scopecheckerV1.ScopeCheckResponse{
			Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
			ExcludeReason: script.ChaffDetection.AsInt32(),
			IncludeCheckUri: &commonsV1.ParsedUri{
				Href:   "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
				Scheme: "http",
				Host:   "foo.bar",
				Port:   80,
				Path:   "/aa%20bb/cc",
				Query:  "foo&jsessionid=1",
			},
			Console: "",
		}},
		{"missingParam",
			"test(param(\"foo\"))", qUri, false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.RuntimeException.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Console: "",
				Error: &commonsV1.Error{
					Code:   -5,
					Msg:    "error executing scope script",
					Detail: "Traceback (most recent call last):\n  scope_script:1:11: in <toplevel>\nError in param: no value with name 'foo'",
				},
			}},
		{"badScript",
			"test(", qUri, false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.RuntimeException.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Console: "",
				Error: &commonsV1.Error{
					Code:   -5,
					Msg:    "error parsing scope script",
					Detail: "scope_script:1:6: got end of file, want ')'",
				},
			}},
		{"withDebug",
			"test(param(\"testValue\")).then(ChaffDetection)", qUri, true,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.ChaffDetection.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Console: "scope_script:1:5 test(\"True\") match=True\nscope_script:1:30 match.then(ChaffDetection) status=ChaffDetection\n",
			}},
		{"badUri",
			"test(True).then(ChaffDetection)", badQUri, true,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.IllegalUri.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href: "http://%00foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				},
				Console: "",
				Error: &commonsV1.Error{
					Code:   -7,
					Msg:    "error parsing uri",
					Detail: "Error: The host contains a forbidden domain code point: '\x00'. Url: 'http://%00foo.bar/aa bb/cc?jsessionid=1&foo#bar'",
				},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &scopecheckerV1.ScopeCheckRequest{
				QueuedUri:       tt.qUri,
				ScopeScriptName: "scope_script",
				ScopeScript:     tt.script,
				Debug:           tt.debug,
			}

			got, err := server.ScopeCheck(context.TODO(), request)
			if err != nil {
				t.Errorf("ScopeCheck() error = %v", err)
				return
			}
			if got.Evaluation != tt.want.Evaluation {
				t.Errorf("ScopeCheck() evaluation got = %v, want %v", got.Evaluation, tt.want.Evaluation)
			}
			if got.ExcludeReason != tt.want.ExcludeReason {
				t.Errorf("ScopeCheck() excludeReason got = %v, want %v", got.ExcludeReason, tt.want.ExcludeReason)
			}
			if !reflect.DeepEqual(got.IncludeCheckUri, tt.want.IncludeCheckUri) {
				t.Errorf("ScopeCheck() includeCheckUri got = %v, want %v", got.IncludeCheckUri, tt.want.IncludeCheckUri)
			}
			if got.Console != tt.want.Console {
				t.Errorf("ScopeCheck() consoleLog \ngot:\n  %v\nwant:\n  %v",
					strings.ReplaceAll(got.Console, "\n", "\n  "),
					strings.ReplaceAll(tt.want.Console, "\n", "\n  "))
			}
			if !reflect.DeepEqual(got.Error, tt.want.Error) {
				t.Errorf("ScopeCheck() error \nGot:\n%v\nWant:\n%v\n", formatError(got.Error), formatError(tt.want.Error))
			}
		})
	}
}

func TestFullScript(t *testing.T) {
	server := &ScopeChecker{}

	tests := []struct {
		name   string
		script string
		qUri   *frontierV1.QueuedUri
		debug  bool
		want   *scopecheckerV1.ScopeCheckResponse
	}{
		{"include",
			defaultScopeScript,
			newQUri("http://foo.bar/aa", "http://foo.bar/", "RL"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation: scopecheckerV1.ScopeCheckResponse_INCLUDE,
			}},
		{"wrongScheme",
			defaultScopeScript,
			newQUri("ftp://foo.bar/aa", "http://foo.bar/", "RL"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.Blocked.AsInt32(),
			}},
		{"tooManyHops",
			defaultScopeScript,
			newQUri("http://foo.bar/aa", "http://foo.bar/", "RLLL"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.TooManyHops.AsInt32(),
			}},
		{"offHost",
			defaultScopeScript,
			newQUri("http://foo2.bar/aa", "http://foo.bar/", "RL"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.Blocked.AsInt32(),
			}},
		{"altHost",
			defaultScopeScript,
			newQUri("http://alt.host.com/aa", "http://foo.bar/", "RL"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation: scopecheckerV1.ScopeCheckResponse_INCLUDE,
			}},
		{"excludedUrl",
			defaultScopeScript,
			setAnnotation(newQUri("http://foo.bar/aa", "http://foo.bar/", "RL"), "scope_excludedUris", "http://foo.bar/aa"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.Blocked.AsInt32(),
			}},
		{"blockedByPathPrefix",
			defaultScopeScriptWithPathPrefix,
			setAnnotation(newQUri("http://foo.bar/aa/bb", "http://foo.bar/", "RL"), "scope_excludedPathPrefixes", "/aa"),
			false,
			&scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: script.Blocked.AsInt32(),
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &scopecheckerV1.ScopeCheckRequest{
				QueuedUri:       tt.qUri,
				ScopeScriptName: "scope_script",
				ScopeScript:     tt.script,
				Debug:           tt.debug,
			}

			got, err := server.ScopeCheck(context.TODO(), request)
			if err != nil {
				t.Errorf("ScopeCheck() error = %v", err)
				return
			}
			if got.Evaluation != tt.want.Evaluation {
				t.Errorf("ScopeCheck() evaluation got = %v, want %v", got.Evaluation, tt.want.Evaluation)
			}
			if got.ExcludeReason != tt.want.ExcludeReason {
				t.Errorf("ScopeCheck() excludeReason got = %v, want %v", got.ExcludeReason, tt.want.ExcludeReason)
			}
			if got.Console != tt.want.Console {
				t.Errorf("ScopeCheck() consoleLog \ngot:\n  %v\nwant:\n  %v",
					strings.ReplaceAll(got.Console, "\n", "\n  "),
					strings.ReplaceAll(tt.want.Console, "\n", "\n  "))
			}
			if !reflect.DeepEqual(got.Error, tt.want.Error) {
				t.Errorf("ScopeCheck() error \nGot:\n%v\nWant:\n%v\n", formatError(got.Error), formatError(tt.want.Error))
			}
		})
	}
}

var result *scopecheckerV1.ScopeCheckResponse

func BenchmarkParse(b *testing.B) {
	server := &ScopeChecker{}
	qUri := &frontierV1.QueuedUri{
		Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
		SeedUri:       "http://foo.bar",
		Ip:            "127.0.0.1",
		DiscoveryPath: "RL",
		Referrer:      "http://foo.bar/",
		Annotation: []*configV1.Annotation{
			{Key: "testValue", Value: "True"},
		},
	}

	tests := []struct {
		name   string
		script string
		qUri   *frontierV1.QueuedUri
	}{
		{"1", "test(True).then(ChaffDetection)", qUri},
		{"2", "test(param(\"testValue\")).then(ChaffDetection)", qUri},
		{"3", `
isSameHost().then(ChaffDetection, continueEvaluation=True)
isScheme('ftp').then(Blocked)
maxHopsFromSeed(1).then(Include)
`, qUri},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			request := &scopecheckerV1.ScopeCheckRequest{
				QueuedUri:       tt.qUri,
				ScopeScriptName: "scope_script",
				ScopeScript:     tt.script,
			}

			for i := 0; i < b.N; i++ {
				got, err := server.ScopeCheck(context.TODO(), request)
				if err == nil {
					result = got
					if got.Error != nil {
						b.Error(got)
					}
				}
			}
		})
	}
}

func newQUri(uri, seed, discoveryPath string) *frontierV1.QueuedUri {
	return &frontierV1.QueuedUri{
		Id:                  "id1",
		ExecutionId:         "eid1",
		DiscoveredTimeStamp: timestamppb.Now(),
		Sequence:            2,
		Uri:                 uri,
		Ip:                  "127.0.0.1",
		DiscoveryPath:       discoveryPath,
		SeedUri:             seed,
		Referrer:            "http://foo.bar/",
		Cookies:             nil,
		Retries:             0,
		Annotation: []*configV1.Annotation{
			{Key: "testValue", Value: "True"},
			{Key: "scope_includeSubdomains", Value: "True"},
			{Key: "scope_maxHopsFromSeed", Value: "2"},
			{Key: "scope_hopsIncludeRedirects", Value: "True"},
			{Key: "scope_excludedPathPrefixes", Value: ""},
			{Key: "scope_excludedUris", Value: ""},
			{Key: "scope_allowedSchemes", Value: "http https"},
			{Key: "scope_altSeeds", Value: "alt.host.com"},
		},
	}
}

func setAnnotation(qUri *frontierV1.QueuedUri, key, value string) *frontierV1.QueuedUri {
	for _, annotation := range qUri.Annotation {
		if annotation.Key == key {
			annotation.Value = value
			return qUri
		}
	}
	qUri.Annotation = append(qUri.Annotation, &configV1.Annotation{Key: key, Value: value})
	return qUri
}

func formatError(e *commonsV1.Error) string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("    code: %v\n     msg: %v\n  detail: %v",
		e.Code, e.Msg, strings.ReplaceAll(e.Detail, "\n", "\n          "))
}
