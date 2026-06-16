/*
 * Copyright 2020 National Library of Norway.
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

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/script"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/opentracing/opentracing-go"
)

type sessionScripts struct {
	scripts   map[configV1.BrowserScript_BrowserScriptType][]*configV1.ConfigObject
	blacklist []configV1.BrowserScript_BrowserScriptType
}

func newSessionScripts() *sessionScripts {
	return &sessionScripts{
		scripts: make(map[configV1.BrowserScript_BrowserScriptType][]*configV1.ConfigObject),
		blacklist: []configV1.BrowserScript_BrowserScriptType{
			configV1.BrowserScript_SCOPE_CHECK,
		},
	}
}

func (s *sessionScripts) IsBlacklisted(scriptType configV1.BrowserScript_BrowserScriptType) bool {
	for _, b := range s.blacklist {
		if scriptType == b {
			return true
		}
	}
	return false
}

func (s *sessionScripts) Get(scriptType configV1.BrowserScript_BrowserScriptType) []*configV1.ConfigObject {
	return s.scripts[scriptType]
}

func scriptPriority(configObject *configV1.ConfigObject) int {
	if configObject == nil || configObject.GetMeta() == nil {
		return 0
	}

	for _, label := range configObject.GetMeta().GetLabel() {
		if label.GetKey() != "priority" {
			continue
		}

		priority, err := strconv.Atoi(label.GetValue())
		if err != nil {
			return 0
		}
		return priority
	}

	return 0
}

func orderOnLoadScripts(scripts []*configV1.ConfigObject) {
	if len(scripts) < 2 {
		return
	}

	priorities := make(map[string]int, len(scripts))
	for _, configObject := range scripts {
		priorities[configObject.GetId()] = scriptPriority(configObject)
	}

	sort.SliceStable(scripts, func(i, j int) bool {
		return priorities[scripts[i].GetId()] > priorities[scripts[j].GetId()]
	})
}

func (sess *Session) loadScripts(ctx context.Context) (*sessionScripts, error) {
	bs := newSessionScripts()

	scripts, err := sess.configAdapter.GetScripts(ctx, sess.browserConfig)
	if err != nil {
		return nil, err
	}
	for _, s := range scripts {
		scriptType := s.GetBrowserScript().GetBrowserScriptType()
		if bs.IsBlacklisted(scriptType) {
			continue
		}
		if !match(s.GetBrowserScript().GetUrlRegexp(), sess.RequestedUrl.Uri) {
			continue
		}
		bs.scripts[scriptType] = append(bs.scripts[scriptType], s)
	}
	orderOnLoadScripts(bs.scripts[configV1.BrowserScript_ON_LOAD])
	return bs, nil
}

func (sess *Session) GetReplacementScript(uri string) *configV1.BrowserScript {
	log := sess.logger

	replacements := sess.scripts.Get(configV1.BrowserScript_REPLACEMENT)
	if len(replacements) == 0 {
		return nil
	}
	normalizedUri := url.Normalize(uri)
	longestMatch := 0
	var currentBestMatch *configV1.BrowserScript
	for _, bc := range replacements {
		for _, urlRegexp := range bc.GetBrowserScript().UrlRegexp {
			if re, err := regexp.Compile(urlRegexp); err == nil {
				re.Longest()
				l := len(re.FindString(normalizedUri))
				if l > 0 && l > longestMatch {
					longestMatch = l
					currentBestMatch = bc.GetBrowserScript()
				}
			} else {
				log.Warn("Could not match url for replacement script", "error", err)
			}
		}
	}
	return currentBestMatch
}

func buildNewDocumentScriptSource(functionDeclaration string, arguments json.RawMessage) string {
	args := string(arguments)
	if len(args) == 0 {
		args = "{}"
	}

	return fmt.Sprintf(";(() => { const __veidemannFn = (%s); void __veidemannFn(%s); })();", functionDeclaration, args)
}

func (sess *Session) registerNewDocumentScripts(ctx context.Context) error {
	for _, configObject := range sess.scripts.Get(configV1.BrowserScript_ON_NEW_DOCUMENT) {
		arguments, err := script.CompileArguments(configObject, sess.RequestedUrl.Annotation, nil)
		if err != nil {
			return fmt.Errorf("failed to prepare init-script arguments for script %s (%s): %w", configObject.GetMeta().GetName(), configObject.GetId(), err)
		}

		source := buildNewDocumentScriptSource(configObject.GetBrowserScript().GetScript(), arguments)
		if _, err := page.AddScriptToEvaluateOnNewDocument(source).Do(ctx); err != nil {
			return fmt.Errorf("failed to register init script %s (%s): %w", configObject.GetMeta().GetName(), configObject.GetId(), err)
		}

		sess.logger.Info("Registered script",
			"scriptType", configV1.BrowserScript_ON_NEW_DOCUMENT.String(),
			"scriptName", configObject.GetMeta().GetName(),
			"scriptId", configObject.GetId())
	}

	return nil
}

// executeScripts executes runtime behavior scripts after navigation.
func (sess *Session) executeScripts(ctx context.Context, scriptType configV1.BrowserScript_BrowserScriptType) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "execute-scripts")
	defer span.Finish()
	span.SetTag("scriptType", scriptType)

	log := sess.logger.With("scriptType", scriptType)

	// wait is executed depending on value returned from script (WaitForData)
	wait := func() {
		waitStart := time.Now()
		waitErr := sess.waitForNetworkIdle(ctx, time.Duration(sess.browserConfig.MaxInactivityTimeMs))
		if waitErr != nil {
			log.Debug("Timed out while waiting for network activity to settle", "error", waitErr)
			return
		}
		log.Debug("Waited for network activity to settle", "duration", time.Since(waitStart))
	}

	if scriptType != configV1.BrowserScript_ON_LOAD {
		return fmt.Errorf("script execution for type %v is not implemented", scriptType)
	}

	// Reuse one isolated world for the whole ON_LOAD phase so script chaining and
	// stateful page interaction happen in a stable crawler-owned context.
	executionContextID, err := getExecutionContextID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get isolated execution context id: %w", err)
	}
	resolveExecutionContextId := func() (runtime.ExecutionContextID, error) {
		return executionContextID, nil
	}

	execute := func(configObject *configV1.ConfigObject, arguments json.RawMessage) (json.RawMessage, error) {
		name := configObject.GetMeta().GetName()
		id := configObject.GetId()
		eci, err := resolveExecutionContextId()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve execution context id for script %s (%s): %w", name, id, err)
		}

		log := log.With(
			"scriptName", name,
			"scriptId", id,
			"scriptEci", int64(eci),
		)

		log.Info("Executing script", "arguments", string(arguments))
		res, err := callScript(ctx, eci, configObject.GetBrowserScript().GetScript(), arguments)
		if err != nil {
			log.Warn("Script execution failed", "error", err)
		} else {
			log.Info("Script returned", "result", string(res))
		}

		return res, err
	}
	scripts := make(map[string]*configV1.ConfigObject)
	for _, s := range sess.scripts.Get(configV1.BrowserScript_UNDEFINED) {
		scripts[s.Id] = s
	}
	for _, s := range sess.scripts.Get(scriptType) {
		// add initial script to map
		scripts[s.Id] = s
		err := script.Run(s.Id, scripts, sess.RequestedUrl.Annotation, execute, wait)
		if err != nil {
			return fmt.Errorf("failed to run script %s (%s): %w", s.Meta.Name, s.Id, err)
		}
	}
	return nil
}

// callScript runs a script function in the given execution context using the
// provided arguments via chrome debug protocol.
//
// The result value from the debug protocol action is unmarshalled into a
// ReturnValue struct. Promise results are awaited before the value is returned
// so scripts may resolve asynchronously without changing their JSON shape.
//
// Returns an error: if the debug protocol action fails, if script execution
// caused an exception, or if unmarshalling of result value fails.
func callScript(ctx context.Context, eci runtime.ExecutionContextID, functionDeclaration string, arguments json.RawMessage) (json.RawMessage, error) {
	var res *runtime.RemoteObject
	var exceptionDetails *runtime.ExceptionDetails
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			res, exceptionDetails, err = runtime.
				CallFunctionOn(functionDeclaration).
				WithArguments([]*runtime.CallArgument{{Value: jsontext.Value(arguments)}}).
				WithExecutionContextID(eci).
				WithReturnByValue(true).
				WithAwaitPromise(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}
	if exceptionDetails != nil {
		return nil, exceptionDetails
	}

	return json.RawMessage(res.Value), nil
}

// evaluateScript evaluates a script expression and awaits Promise results
// before returning the resolved value.
func evaluateScript(ctx context.Context, expression string) (json.RawMessage, error) {
	var res *runtime.RemoteObject
	var exceptionDetails *runtime.ExceptionDetails
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			res, exceptionDetails, err = runtime.Evaluate(expression).
				WithReturnByValue(true).
				WithAwaitPromise(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}
	if exceptionDetails != nil {
		return nil, exceptionDetails
	}
	if res == nil || res.Value == nil {
		return nil, nil
	}

	return json.RawMessage(res.Value), nil
}

// getExecutionContextID creates an isolated world from the root frame and returns
// an execution context id.
func getExecutionContextID(ctx context.Context) (runtime.ExecutionContextID, error) {
	frameTree, err := getFrameTree(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get frameTree: %w", err)
	}

	var eci runtime.ExecutionContextID
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		eci, err = page.CreateIsolatedWorld(frameTree.Frame.ID).Do(ctx)
		return err
	}))
	if err != nil {
		return 0, fmt.Errorf("failed to create isolated world: %w", err)
	}
	return eci, nil
}

// getFrameTree returns the frame tree of the current page or an error if it fails.
func getFrameTree(ctx context.Context) (*page.FrameTree, error) {
	var frameTree *page.FrameTree
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		frameTree, err = page.GetFrameTree().Do(ctx)
		return err
	}))
	if err != nil {
		return nil, err
	}
	return frameTree, nil
}

// match takes an array of regular expressions and a URI. It returns true if
// a normalized version of the URI matches any of the regular expressions or
// if the array of regular expressions is empty, and false otherwise.
func match(regExps []string, uri string) bool {
	if len(regExps) == 0 {
		return true
	}
	normalizedUri := url.Normalize(uri)
	for _, urlRegexp := range regExps {
		re, err := regexp.Compile(urlRegexp)
		if err != nil {
			slog.Error("Failed to compile regular expression", "regexp", urlRegexp, "error", err)
			continue
		}
		if match := re.MatchString(normalizedUri); match {
			return true
		}
	}
	return false
}
