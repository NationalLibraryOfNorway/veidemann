package script

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
)

type testdata struct {
	name   string
	script string
	qUri   *frontierV1.QueuedUri
	debug  bool
	want   *scopecheckerV1.ScopeCheckResponse
}

func init() {
	InitializeCanonicalizationProfiles(false)
}

func Test_isSameHost(t *testing.T) {
	tests := []testdata{
		{name: "isSameHost1",
			script: "isSameHost().then(Include, continueEvaluation=True)",
			qUri: &frontierV1.QueuedUri{
				Uri:     "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				SeedUri: "http://foo.bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isSameHost2",
			script: "isSameHost().then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:     "http://sub.foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				SeedUri: "http://foo.bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://sub.foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "sub.foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Error: &commonsV1.Error{
					Code:   -5001,
					Msg:    "Blocked",
					Detail: "No scope rules matched",
				},
				Console: "",
			}},
		{name: "isSameHostSub1",
			script: "isSameHost(True).then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:     "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				SeedUri: "http://foo.bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isSameHostSub2",
			script: "isSameHost(True).then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:     "http://sub.foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				SeedUri: "http://foo.bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://sub.foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "sub.foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Console: "",
			}},
		{name: "isSameHostSub3",
			script: "isSameHost(param('IncludeSubdomain')).then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:     "http://sub.foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				SeedUri: "http://foo.bar",
				Annotation: []*configV1.Annotation{
					{Key: "IncludeSubdomain", Value: "True"},
				},
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://sub.foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "sub.foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Console: "",
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunScopeScript(tt.name, tt.script, tt.qUri, tt.debug)
			verify(t, got, tt.want)
		})
	}
}

func Test_isScheme(t *testing.T) {
	tests := []testdata{
		{name: "isScheme1",
			script: "isScheme('http').then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isScheme2",
			script: "isScheme('https').then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Error: &commonsV1.Error{
					Code:   -5001,
					Msg:    "Blocked",
					Detail: "No scope rules matched",
				},
				Console: "",
			}},
		{name: "isScheme3",
			script: "isScheme(param('scheme')).then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				Annotation: []*configV1.Annotation{
					{Key: "scheme", Value: "http"},
				},
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isScheme4",
			script: "isScheme(param('scheme')).then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "HttP://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				Annotation: []*configV1.Annotation{
					{Key: "scheme", Value: "hTtp"},
				},
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isScheme5",
			script: "isScheme(param('scheme')).then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: RuntimeException.AsInt32(),
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
					Code: RuntimeException.AsInt32(),
					Msg:  "error executing scope script",
					Detail: `Traceback (most recent call last):
  isScheme5:1:15: in <toplevel>
Error in param: no value with name 'scheme'`,
				},
			}},
		{name: "isScheme6",
			script: "isScheme('http').then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "foo.bar/aa bb/cc?jsessionid=1&foo#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isScheme7",
			script: "isScheme(param('scheme')).then(Blocked)",
			qUri: &frontierV1.QueuedUri{
				Uri: "file:c|/foo/bar/aa bb/",
				Annotation: []*configV1.Annotation{
					{Key: "scheme", Value: "hTtp https file ftp"},
				},
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "file:///c:/foo/bar/aa%20bb/",
					Scheme: "file",
					Path:   "/c:/foo/bar/aa%20bb/",
				},
				Console: "",
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunScopeScript(tt.name, tt.script, tt.qUri, tt.debug)
			verify(t, got, tt.want)
		})
	}
}

func Test_isReferrer(t *testing.T) {
	tests := []testdata{
		{name: "isReferrer1",
			script: "isReferrer('http://foo.bar/sitemap.txt').then(Blocked).otherwise(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:      "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				Referrer: "http://foo.bar/",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
		{name: "isReferrer2",
			script: "isReferrer('http://foo.bar/sitemap.txt').then(Blocked).otherwise(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:      "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				Referrer: "http://foo.bar/sitemap.txt",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
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
		{name: "isReferrer3",
			script: "isReferrer('http://foo.bar/sitemap.txt http://foo.bar/aa').then(Blocked).otherwise(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:      "http://foo.bar/bb",
				Referrer: "http://foo.bar/aa",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/bb",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/bb",
				},
				Console: "",
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunScopeScript(tt.name, tt.script, tt.qUri, tt.debug)
			verify(t, got, tt.want)
		})
	}
}

func Test_isUrl(t *testing.T) {
	tests := []testdata{
		{name: "isUrl1",
			script: "isUrl('http://foo.bar/aa//bb/cc?jsessionid=1&foo#bar').then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa//bb/cc?jsessionid=1&foo#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa/bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa/bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Console: "",
			}},
		{name: "isUrl2",
			script: "isUrl('http://foo.bar/aa//bb/cc?foo&a=c&jsessionid=1&a=b').then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa//bb/cc?jsessionid=1&foo&a=c&a=b#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa/bb/cc?a=c&a=b&foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa/bb/cc",
					Query:  "a=c&a=b&foo&jsessionid=1",
				},
				Console: "",
			}},
		{name: "isUrl3",
			script: "isUrl('foo.bar/aa/ff/../bb/cc?foo&a=c&jsessionid=1&a=b').then(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa//bb/cc?jsessionid=1&foo&a=c&a=b#bar",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa/bb/cc?a=c&a=b&foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa/bb/cc",
					Query:  "a=c&a=b&foo&jsessionid=1",
				},
				Console: "",
			}},
		{name: "isUrl4",
			script: "isUrl('foo.bar/aa/ example.com').then(Blocked)",
			qUri: &frontierV1.QueuedUri{
				Uri: "http://foo.bar/aa/",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa/",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa/",
				},
				Console: "",
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunScopeScript(tt.name, tt.script, tt.qUri, tt.debug)
			verify(t, got, tt.want)
		})
	}
}

func Test_maxHopsFromSeed(t *testing.T) {
	tests := []testdata{
		{name: "maxHopsFromSeed1",
			script: "maxHopsFromSeed(2).then(TooManyHops)",
			qUri: &frontierV1.QueuedUri{
				Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				DiscoveryPath: "RLERLR",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: TooManyHops.AsInt32(),
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
		{name: "maxHopsFromSeed2",
			script: "maxHopsFromSeed(4).then(TooManyHops)",
			qUri: &frontierV1.QueuedUri{
				Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				DiscoveryPath: "RLERLR",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: Blocked.AsInt32(),
				IncludeCheckUri: &commonsV1.ParsedUri{
					Href:   "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
					Scheme: "http",
					Host:   "foo.bar",
					Port:   80,
					Path:   "/aa%20bb/cc",
					Query:  "foo&jsessionid=1",
				},
				Error: &commonsV1.Error{
					Code:   -5001,
					Msg:    "Blocked",
					Detail: "No scope rules matched",
				},
				Console: "",
			}},
		{name: "maxHopsFromSeed3",
			script: "maxHopsFromSeed(param('depth')).then(TooManyHops)",
			qUri: &frontierV1.QueuedUri{
				Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				DiscoveryPath: "RLERLR",
				Annotation: []*configV1.Annotation{
					{Key: "depth", Value: "2"},
				},
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: TooManyHops.AsInt32(),
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
		{name: "maxHopsFromSeed4",
			script: "maxHopsFromSeed(param('depth'), param('includeRedirects')).then(TooManyHops)",
			qUri: &frontierV1.QueuedUri{
				Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				DiscoveryPath: "RLERLR",
				Annotation: []*configV1.Annotation{
					{Key: "depth", Value: "3"},
					{Key: "includeRedirects", Value: "yeS"},
				},
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: TooManyHops.AsInt32(),
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
		{name: "maxHopsFromSeed5",
			script: "maxHopsFromSeed(0).then(TooManyHops)",
			qUri: &frontierV1.QueuedUri{
				Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				DiscoveryPath: "L",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason: TooManyHops.AsInt32(),
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
		{name: "maxHopsFromSeed6",
			script: "maxHopsFromSeed(1).otherwise(Include)",
			qUri: &frontierV1.QueuedUri{
				Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
				DiscoveryPath: "L",
			},
			debug: false,
			want: &scopecheckerV1.ScopeCheckResponse{
				Evaluation:    scopecheckerV1.ScopeCheckResponse_INCLUDE,
				ExcludeReason: Include.AsInt32(),
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunScopeScript(tt.name, tt.script, tt.qUri, tt.debug)
			verify(t, got, tt.want)
		})
	}
}

// Helper functions

func verify(t *testing.T, got, want *scopecheckerV1.ScopeCheckResponse) {
	if got.Evaluation != want.Evaluation {
		t.Errorf("RunScopeScript().Evaluation got = %v, want %v", got.Evaluation, want.Evaluation)
	}
	if got.ExcludeReason != want.ExcludeReason {
		t.Errorf("RunScopeScript().ExcludeReason got = %v, want %v", got.ExcludeReason, want.ExcludeReason)
	}
	if !reflect.DeepEqual(got.IncludeCheckUri, want.IncludeCheckUri) {
		t.Errorf("RunScopeScript().IncludeCheckUri got = %v, want %v", got.IncludeCheckUri, want.IncludeCheckUri)
	}
	if got.Console != want.Console {
		t.Errorf("RunScopeScript().Console \ngot:\n  %v\nwant:\n  %v",
			strings.ReplaceAll(got.Console, "\n", "\n  "),
			strings.ReplaceAll(want.Console, "\n", "\n  "))
	}
	if !reflect.DeepEqual(got.Error, want.Error) {
		t.Errorf("RunScopeScript().Error \nGot:\n%v\nWant:\n%v\n", formatError(got.Error), formatError(want.Error))
	}
}

func formatError(e *commonsV1.Error) string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("    code: %v\n     msg: %v\n  detail: %v",
		e.Code, e.Msg, strings.ReplaceAll(e.Detail, "\n", "\n          "))
}
