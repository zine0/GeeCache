package geecache

import (
	"fmt"
	"github/zine0/GeeCache/consistenthash"
	"github/zine0/GeeCache/geecachepb"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

const defaultBasePath = "/_geecache/"
const defaultReplicas = 50

type HTTPPool struct {
	self     string
	basePath string
	mu sync.Mutex
	peers *consistenthash.Map
	httpGetters map[string]*httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP implements http.Handler.
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)

	body,err := proto.Marshal(&geecachepb.Response{
		Value: view.ByteSlice(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas,nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter,len(peers))

	for _,peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.basePath,
		}
	}
}

func (p *HTTPPool) PickPeer(key string) (PeerGetter,bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s",peer)
		return p.httpGetters[peer],true
	}
	return  nil,false
}

var _PeerPicker = (*HTTPPool)(nil)

type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(in *geecachepb.Request,out *geecachepb.Response) (error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)

	res,err := http.Get(u)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v",res.Status)
	}
	bytes,err := io.ReadAll(res.Body)
	if err=proto.Unmarshal(bytes,out);err !=nil {
		return fmt.Errorf("reading response body: %v",err)
	}

	return nil
}

var _PeerGetter = (*httpGetter)(nil)
