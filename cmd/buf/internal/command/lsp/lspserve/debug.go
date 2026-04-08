// Copyright 2020-2026 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lspserve

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/bufbuild/buf/private/pkg/transport/http/httpserver"
)

// debugServer is an HTTP server that serves debug information about the
// running LSP server.
type debugServer struct {
	version   string
	startTime time.Time

	listener net.Listener
	server   *http.Server
}

// newDebugServer creates and starts a debug HTTP server on the given address.
// The address format is "host:port", e.g. "localhost:6060" or ":0" for an
// OS-assigned port.
func newDebugServer(addr string, version string) (*debugServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("could not start debug server: %w", err)
	}

	// Sample 1-in-1000 blocking events and 1-in-10 mutex contention events.
	// Rate 1 would trace every event and add measurable overhead to the LSP.
	runtime.SetBlockProfileRate(1000)
	runtime.SetMutexProfileFraction(10)

	ds := &debugServer{
		version:   version,
		startTime: time.Now(),
		listener:  listener,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ds.render(mainTmpl, ds.getMain))
	mux.HandleFunc("/info", ds.render(infoTmpl, ds.getInfo))
	mux.HandleFunc("/memory", ds.render(memoryTmpl, ds.getMemory))
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	ds.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: httpserver.DefaultReadHeaderTimeout,
		IdleTimeout:       httpserver.DefaultIdleTimeout,
	}
	go func() { _ = ds.server.Serve(listener) }()

	return ds, nil
}

// Addr returns the address the debug server is listening on.
func (ds *debugServer) Addr() net.Addr {
	return ds.listener.Addr()
}

// Close shuts down the debug server.
func (ds *debugServer) Close() error {
	return ds.server.Close()
}

type serverInfo struct {
	Version      string
	StartTime    time.Time
	Uptime       string
	PID          int
	GoVersion    string
	GOOS         string
	GOARCH       string
	NumCPU       int
	GOMAXPROCS   int
	NumGoroutine int
	BuildInfo    string
}

func (ds *debugServer) getServerInfo() serverInfo {
	uptime := time.Since(ds.startTime).Truncate(time.Second)
	var buildInfoStr string
	if bi, ok := debug.ReadBuildInfo(); ok {
		buildInfoStr = bi.String()
	}
	return serverInfo{
		Version:      ds.version,
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		PID:          os.Getpid(),
		StartTime:    ds.startTime,
		Uptime:       uptime.String(),
		NumCPU:       runtime.NumCPU(),
		GOMAXPROCS:   runtime.GOMAXPROCS(0),
		NumGoroutine: runtime.NumGoroutine(),
		BuildInfo:    buildInfoStr,
	}
}

func (ds *debugServer) getMain(_ *http.Request) any {
	return ds.getServerInfo()
}

func (ds *debugServer) getInfo(_ *http.Request) any {
	return ds.getServerInfo()
}

func (ds *debugServer) getMemory(_ *http.Request) any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

type dataFunc func(*http.Request) any

func (ds *debugServer) render(tmpl *template.Template, fun dataFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.Execute(w, fun(r)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// commas formats a non-negative integer string with comma separators.
func commas(s string) string {
	for i := len(s); i > 3; {
		i -= 3
		s = s[:i] + "," + s[i:]
	}
	return s
}

func fuint64(v uint64) string {
	return commas(strconv.FormatUint(v, 10))
}

func fuint32(v uint32) string {
	return commas(strconv.FormatUint(uint64(v), 10))
}

var baseTmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"fuint64": fuint64,
	"fuint32": fuint32,
}).Parse(`
<html>
<head>
<title>{{template "title" .}}</title>
<style>
body {
	font-family: sans-serif;
	font-size: 1rem;
	line-height: 1.6;
	margin: 0;
	padding: 0;
}
nav {
	background: #2d2d2d;
	padding: 0.5rem 1rem;
}
nav a {
	color: #fff;
	text-decoration: none;
	margin-right: 1.5rem;
	font-size: 0.9rem;
}
nav a:hover {
	text-decoration: underline;
}
.content {
	padding: 1rem 2rem;
}
table {
	border-collapse: collapse;
	margin: 0.5rem 0;
}
td, th {
	padding: 0.25rem 0.75rem;
	text-align: left;
	border-bottom: 1px solid #eee;
}
td.value {
	text-align: right;
	font-family: monospace;
}
th {
	border-bottom: 2px solid #ddd;
	font-weight: 600;
}
pre {
	background: #f5f5f5;
	padding: 1rem;
	overflow-x: auto;
	font-size: 0.85rem;
}
h1 { margin-top: 0; }
.label { color: #555; }
</style>
{{block "head" .}}{{end}}
</head>
<body>
<nav>
<a href="/">Main</a>
<a href="/info">Info</a>
<a href="/memory">Memory</a>
<a href="/debug/pprof">Profiling</a>
</nav>
<div class="content">
<h1>{{template "title" .}}</h1>
{{block "body" .}}
Unknown page
{{end}}
</div>
</body>
</html>
`))

var mainTmpl = template.Must(template.Must(baseTmpl.Clone()).Parse(`
{{define "title"}}Buf LSP Debug{{end}}
{{define "body"}}
<h2>Server</h2>
<table>
<tr><td class="label">Version</td><td>{{.Version}}</td></tr>
<tr><td class="label">Go version</td><td>{{.GoVersion}}</td></tr>
<tr><td class="label">Platform</td><td>{{.GOOS}}/{{.GOARCH}}</td></tr>
<tr><td class="label">PID</td><td>{{.PID}}</td></tr>
<tr><td class="label">Started</td><td>{{.StartTime.Format "2006-01-02 15:04:05"}}</td></tr>
<tr><td class="label">Uptime</td><td>{{.Uptime}}</td></tr>
</table>

<h2>Debug Pages</h2>
<ul>
<li><a href="/info">Server info and build details</a></li>
<li><a href="/memory">Memory usage</a></li>
<li><a href="/debug/pprof">Profiling (pprof)</a></li>
</ul>

<h2>Profiles</h2>
<ul>
<li><a href="/debug/pprof/goroutine?debug=1">Goroutines</a></li>
<li><a href="/debug/pprof/heap?debug=1">Heap</a></li>
<li><a href="/debug/pprof/allocs?debug=1">Allocs</a></li>
<li><a href="/debug/pprof/block?debug=1">Block</a></li>
<li><a href="/debug/pprof/mutex?debug=1">Mutex</a></li>
<li><a href="/debug/pprof/threadcreate?debug=1">Thread create</a></li>
</ul>
{{end}}
`))

var infoTmpl = template.Must(template.Must(baseTmpl.Clone()).Parse(`
{{define "title"}}Buf LSP Info{{end}}
{{define "body"}}
<h2>Server</h2>
<table>
<tr><td class="label">Version</td><td>{{.Version}}</td></tr>
<tr><td class="label">Go version</td><td>{{.GoVersion}}</td></tr>
<tr><td class="label">Platform</td><td>{{.GOOS}}/{{.GOARCH}}</td></tr>
<tr><td class="label">PID</td><td>{{.PID}}</td></tr>
<tr><td class="label">Started</td><td>{{.StartTime.Format "2006-01-02 15:04:05"}}</td></tr>
<tr><td class="label">Uptime</td><td>{{.Uptime}}</td></tr>
<tr><td class="label">NumCPU</td><td>{{.NumCPU}}</td></tr>
<tr><td class="label">GOMAXPROCS</td><td>{{.GOMAXPROCS}}</td></tr>
<tr><td class="label">Goroutines</td><td>{{.NumGoroutine}}</td></tr>
</table>

{{if .BuildInfo}}
<h2>Build Info</h2>
<pre>{{.BuildInfo}}</pre>
{{end}}
{{end}}
`))

var memoryTmpl = template.Must(template.Must(baseTmpl.Clone()).Parse(`
{{define "title"}}Buf LSP Memory{{end}}
{{define "body"}}
<h2>Stats</h2>
<table>
<tr><td class="label">Allocated bytes</td><td class="value">{{fuint64 .HeapAlloc}}</td></tr>
<tr><td class="label">Total allocated bytes</td><td class="value">{{fuint64 .TotalAlloc}}</td></tr>
<tr><td class="label">System bytes</td><td class="value">{{fuint64 .Sys}}</td></tr>
<tr><td class="label">Heap system bytes</td><td class="value">{{fuint64 .HeapSys}}</td></tr>
<tr><td class="label">Malloc calls</td><td class="value">{{fuint64 .Mallocs}}</td></tr>
<tr><td class="label">Frees</td><td class="value">{{fuint64 .Frees}}</td></tr>
<tr><td class="label">Idle heap bytes</td><td class="value">{{fuint64 .HeapIdle}}</td></tr>
<tr><td class="label">In use bytes</td><td class="value">{{fuint64 .HeapInuse}}</td></tr>
<tr><td class="label">Released to system bytes</td><td class="value">{{fuint64 .HeapReleased}}</td></tr>
<tr><td class="label">Heap object count</td><td class="value">{{fuint64 .HeapObjects}}</td></tr>
<tr><td class="label">Stack in use bytes</td><td class="value">{{fuint64 .StackInuse}}</td></tr>
<tr><td class="label">Stack from system bytes</td><td class="value">{{fuint64 .StackSys}}</td></tr>
<tr><td class="label">Bucket hash bytes</td><td class="value">{{fuint64 .BuckHashSys}}</td></tr>
<tr><td class="label">GC metadata bytes</td><td class="value">{{fuint64 .GCSys}}</td></tr>
<tr><td class="label">Off heap bytes</td><td class="value">{{fuint64 .OtherSys}}</td></tr>
</table>
<h2>By Size</h2>
<table>
<tr><th>Size</th><th>Mallocs</th><th>Frees</th></tr>
{{range .BySize}}<tr><td class="value">{{fuint32 .Size}}</td><td class="value">{{fuint64 .Mallocs}}</td><td class="value">{{fuint64 .Frees}}</td></tr>{{end}}
</table>
{{end}}
`))
