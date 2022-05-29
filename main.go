package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/itsabgr/go-handy"
	"github.com/itsabgr/w3s-proxy/pkg/w3s"
	"github.com/valyala/fasthttp"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var flagAddr = flag.String("addr", "0.0.0.0:80", "http server listening address")
var flagHost = flag.String("host", "https://api.web3.storage", "web3storage host")
var flagToken = flag.String("token", "", "web3storage token")

func main() {
	defer handy.Catch(func(recovered error) {
		log.Println("FATAL", recovered.Error())
	})
	log.Println("INFO", "start", time.Now())
	server := fasthttp.Server{}
	server.DisableHeaderNamesNormalizing = true
	server.Name = "w3s-proxy/1.0"
	server.DisablePreParseMultipartForm = true
	server.NoDefaultContentType = true
	server.NoDefaultDate = true
	server.NoDefaultServerHeader = true
	server.TCPKeepalive = true
	server.TCPKeepalivePeriod = 2 * time.Second
	server.DisableKeepalive = false
	server.Concurrency = runtime.NumCPU() * 128
	server.MaxRequestBodySize = 5e+7 //50MB
	server.CloseOnShutdown = true
	server.LogAllErrors = false
	server.SleepWhenConcurrencyLimitsExceeded = time.Second * 2
	server.WriteTimeout = time.Second * 2
	server.ReadTimeout = 60 * time.Second
	server.SecureErrorLogMessage = true
	server.ErrorHandler = func(ctx *fasthttp.RequestCtx, err error) {
		ctx.SetStatusCode(400)
		ctx.SetBodyString(limitedString(err.Error(), 200))
	}
	server.Handler = func(ctx *fasthttp.RequestCtx) {
		defer handy.Catch(func(recovered error) {
			server.ErrorHandler(ctx, recovered)
		})
		if string(ctx.Method()) == fasthttp.MethodGet {
			path := string(ctx.Path())
			resp, err := http.Get(fmt.Sprintf("https://ipfs.io/ipfs/%s", path))
			if err != nil {
				ctx.SetStatusCode(400)
				ctx.SetBodyString(limitedString(err.Error(), 200))
				return
			}
			ctx.SetContentType(resp.Header.Get("Content-Type"))
			ctx.SetBodyStream(resp.Body, int(resp.ContentLength))
			return
		}
		exts, err := mime.ExtensionsByType(string(ctx.Request.Header.ContentType()))
		if err != nil || len(exts) == 0 {
			exts = []string{".b"}
		}
		token := strings.TrimSpace(string(ctx.Request.Header.Peek("Authorization")))
		host := strings.TrimSpace(string(ctx.Request.URI().QueryArgs().Peek("host")))
		if host == "" {
			host = *flagHost
		}
		if token == "" {
			token = *flagToken
		}
		if strings.HasPrefix(token, "Bearer ") {
			token = token[len("Bearer "):]
		}
		client, err := w3s.NewClient(w3s.WithEndpoint(host), w3s.WithToken(token))
		if err != nil {
			ctx.SetStatusCode(400)
			ctx.SetBodyString(limitedString(err.Error(), 200))
			return
		}
		fileName := "w3s-proxy" + exts[0]
		tmp := NewMemFile(fileName)
		defer handy.Close(tmp)
		err = ctx.Request.BodyWriteTo(tmp)
		if err != nil {
			ctx.SetStatusCode(400)
			ctx.SetBodyString(limitedString(err.Error(), 200))
			return
		}
		cid, err := client.Put(ctx, tmp)
		ctx.SetBodyString(filepath.Join(cid.String(), fileName))
	}
	defer server.Shutdown()
	handy.Throw(server.ListenAndServe(*flagAddr))
}
func limitedString(str string, limit int) string {
	if len(str) < limit {
		return str
	}
	return str[:limit]
}

type memFile struct {
	name string
	buff *bytes.Buffer
}

func NewMemFile(name string) *memFile {
	return &memFile{name: name, buff: new(bytes.Buffer)}
}

func (mf *memFile) Read(b []byte) (int, error) {
	return mf.buff.Read(b)
}

type memInfo struct {
	name    string
	size    int64
	modtime time.Time
}

func (mf *memFile) Write(p []byte) (int, error) {
	return mf.buff.Write(p)
}
func (m memInfo) Name() string {
	return m.name
}

func (m memInfo) Size() int64 {
	return m.size
}

func (m memInfo) Mode() fs.FileMode {
	return 0666
}

func (m memInfo) ModTime() time.Time {
	return m.modtime
}

func (m memInfo) IsDir() bool {
	return false
}

func (m memInfo) Sys() any {
	return nil
}

func (mf *memFile) Stat() (fs.FileInfo, error) {
	return memInfo{name: mf.name, size: int64(mf.buff.Len()), modtime: time.Now()}, nil
}

func (mf *memFile) Close() error {
	mf.buff.Reset()
	return nil
}
