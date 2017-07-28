package http_client_cluster

import (
	"bytes"
	"container/heap"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"crypto/tls"
	"context"
	"golang.org/x/net/http2"
)

// per host to a client
var ClientMap map[string]ConfigClient = make(map[string]ConfigClient, 1)
var ClientRWLock *sync.RWMutex = new(sync.RWMutex)

type HttpClusterConfig struct {

	// A HeaderTimeoutPerRequest of zero means no timeout.
	HeaderTimeoutPerRequest time.Duration
	// retry times when request err
	Retry    int
	Redirect bool
	// cert for request https
	Cert tls.Certificate
}

var cfg *HttpClusterConfig

func DefaultConfig() *HttpClusterConfig {
	return &HttpClusterConfig{
		HeaderTimeoutPerRequest: DefaultRequestTimeout,
		Retry:    DefaultRetry,
		Redirect: Redirect,
	}
}

type ConfigClient struct {
	client *HttpClusterClient
	config *HttpClusterConfig
}

func formatSchemeHost(scheme, host string) string {
	return fmt.Sprintf("%s://%s", scheme, host)
}

func GetClient(scheme, host string) (*HttpClusterClient, error) {
	ClientRWLock.RLock()
	config_client, ok := ClientMap[formatSchemeHost(scheme, host)]
	ClientRWLock.RUnlock()
	if ok {
		return config_client.client, nil
	}

	var clientCfg *HttpClusterConfig
	if nil == cfg {
		clientCfg = DefaultConfig()
	} else {
		clientCfg = cfg
	}

	c, err := New(scheme, host, clientCfg)
	return c, err
}

// the Response.Body has closed after reading into body.
func HttpClientClusterDo(request *http.Request) (*http.Response, error) {
	var resp *http.Response
	client, err := GetClient(request.URL.Scheme, request.URL.Host)
	if nil != err {
		log.Println("new http cluster client err: %v", err)
		return nil, err
	}

	if nil == client {
		err = fmt.Errorf("nil client")
		return nil, err
	}

	resp, err = client.Do(request)
	return resp, err
}

// eg SetDefaultConfig(1, 120*time.Second)
func SetDefaultConfig(defaultRetry int, defaultRequestTimeout time.Duration) error {

	DefaultRetry = defaultRetry
	DefaultRequestTimeout = defaultRequestTimeout
	return nil
}

func SetRedirect(ok bool) {
	Redirect = ok
}

func SetClientConfig(scheme, host string, cfg *HttpClusterConfig) error {

	c := &HttpClusterClient{
		cfg:    cfg,
		scheme: scheme,
		host:   host,
	}

	c.updateClientAddr()
	go func(c *HttpClusterClient) {
		timer := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-timer.C:
				c.updateClientAddr()
			}
		}
	}(c)

	ClientRWLock.Lock()
	defer ClientRWLock.Unlock()
	cofig_client := ConfigClient{
		client: c,
		config: cfg,
	}
	ClientMap[formatSchemeHost(scheme, host)] = cofig_client
	return nil
}

func New(scheme, host string, cfg *HttpClusterConfig) (*HttpClusterClient, error) {

	c := &HttpClusterClient{
		cfg:    cfg,
		scheme: scheme,
		host:   host,
	}

	c.updateClientAddr()
	go func(c *HttpClusterClient) {
		timer := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-timer.C:
				c.updateClientAddr()
			}
		}
	}(c)

	ClientRWLock.Lock()
	defer ClientRWLock.Unlock()
	cofig_client := ConfigClient{
		client: c,
		config: cfg,
	}
	ClientMap[formatSchemeHost(scheme, host)] = cofig_client
	return c, nil
}

func noCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 1 {
		fmt.Println("check redirect ", http.ErrUseLastResponse.Error(), len(via))
		return http.ErrUseLastResponse
	}
	return nil
}

func newHTTPClient(addr, ip string, port int, headerTimeout time.Duration, redirect bool,
	cert tls.Certificate) *http.Client {
	fmt.Println("newHTTPClient ", addr, ip)
	dial := func(network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout:   headerTimeout,
			KeepAlive: 75 * time.Second,
		}
		if addr == address {
			fmt.Println("dail ", ip, port)
			return d.Dial(network, fmt.Sprintf("%s:%d", ip, port))
		} else {
			fmt.Println("dail ", address)
			return d.Dial(network, address)
		}
	}
	// proxy := func(_ *http.Request) (*url.URL, error) {
	// 	return url.Parse("http://127.0.0.1:8888")
	// }
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}
	if len(cert.Certificate) > 0 {
		tlsConfig.BuildNameToCertificate()
	}

	var tr = &http.Transport{
		// Proxy: proxy,
		Dial:                  dial,
		TLSHandshakeTimeout:   headerTimeout,
		ResponseHeaderTimeout: headerTimeout,
		TLSClientConfig:       tlsConfig,
		MaxIdleConns:          100,
		IdleConnTimeout:       75 * time.Second,
		MaxIdleConnsPerHost:   30,
	}
	// log.Println(fmt.Sprintf("transport: %+v", tr))

	err := http2.ConfigureTransport(tr)
	if err != nil {
		log.Println("http2 ConfigureTransport err: ", err)
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   headerTimeout,
	}
	if !redirect {
		client.CheckRedirect = noCheckRedirect
	}

	return client
}

type weightClientStats struct {
	UseCount   int64 `json:"use_count"`
	ErrorCount int64 `json:"error_count"`
}

/*********************************************************************************
 带优先级的httpClient，weight越低优先级越高
 封装redirectFollowingHTTPClient

 The httpWeightClient with priority has encapsulate redirectFollowingHTTPClient,
 the lower the weight, the higher the priority.
**********************************************************************************/
type httpWeightClient struct {
	client   *http.Client
	endpoint string // "127.0.0.1"
	index    int
	weight   uint64
	errcnt   int
	stats    weightClientStats
}

/************************************************************************
 带集群功能的client，封装httpWeightClient

 The client with cluster functionality has encapsulate httpWeightClient
************************************************************************/
type HttpClusterClient struct {
	sync.RWMutex
	endpoints []string
	cfg       *HttpClusterConfig
	scheme    string
	host      string
	clients   []*httpWeightClient
}

func (c *HttpClusterClient) Len() int {
	return len(c.clients)
}

func (c *HttpClusterClient) Swap(i, j int) {
	c.clients[i], c.clients[j] = c.clients[j], c.clients[i]
	c.clients[i].index = i
	c.clients[j].index = j
}

func (c *HttpClusterClient) Less(i, j int) bool {
	return c.clients[i].weight < c.clients[j].weight
}

func (c *HttpClusterClient) Pop() (client interface{}) {
	c.clients, client = c.clients[:c.Len()-1], c.clients[c.Len()-1]
	return
}

func (c *HttpClusterClient) Push(client interface{}) {
	weightClient := client.(*httpWeightClient)
	weightClient.index = c.Len()
	c.clients = append(c.clients, weightClient)
}

func (c *HttpClusterClient) exist(addr string) bool {
	c.RLock()
	for _, cli := range c.clients {
		if cli.endpoint == addr {
			c.RUnlock()
			return true
		}
	}
	c.RUnlock()
	return false
}

func (c *HttpClusterClient) add(addr string, client *http.Client) {
	c.Lock()
	defer c.Unlock()

	for _, cli := range c.clients {
		if cli.endpoint == addr {
			return
		}
	}
	heap.Push(c, &httpWeightClient{client: client, endpoint: addr})

	if c.Len() == minHeapSize {
		heap.Init(c)
	}
}

// update clients with new addrs, remove the no use client
func (c *HttpClusterClient) clear(addrs []string) {
	c.Lock()
	var rm []*httpWeightClient
	for _, cli := range c.clients {
		var has_cli bool
		for _, addr := range addrs {
			if cli.endpoint == addr {
				has_cli = true
				break
			}
		}
		if !has_cli {
			rm = append(rm, cli)
		} else if cli.errcnt > 0 {
			/*
				if cli.weight >= errWeight*uint64(cli.errcnt) {
					cli.weight -= errWeight * uint64(cli.errcnt)
					cli.errcnt = 0
					if c.Len() >= minHeapSize {
						heap.Fix(c, cli.index)
					}
				}
			*/
		}
	}

	for _, cli := range rm {
		// p will up, down, or not move, so append it to rm list.
		heap.Remove(c, cli.index)
	}
	c.Unlock()
}

func (c *HttpClusterClient) get() *httpWeightClient {
	c.Lock()
	defer c.Unlock()

	size := c.Len()
	if size == 0 {
		return nil
	}

	if size < minHeapSize {
		var index int = 0
		for i := 1; i < size; i++ {
			if c.Less(i, index) {
				index = i
			}
		}

		return c.clients[index]
	}

	client := heap.Pop(c).(*httpWeightClient)
	heap.Push(c, client)
	return client
}

func (c *HttpClusterClient) use(client *httpWeightClient) {
	c.Lock()
	client.weight++
	if c.Len() >= minHeapSize {
		heap.Fix(c, client.index)
	}
	client.stats.UseCount++
	c.Unlock()
}

func (c *HttpClusterClient) done(client *httpWeightClient) {
	/*
		c.Lock()
		if client.weight > 0 {
			client.weight--
		}
		if c.Len() >= minHeapSize {
			heap.Fix(c, client.index)
		}
		c.Unlock()
	*/
}

func (c *HttpClusterClient) occurErr(client *httpWeightClient, err error) {
	c.Lock()
	if nil != err {
		client.weight += errWeight
		client.errcnt++
		if c.Len() >= minHeapSize {
			heap.Fix(c, client.index)
		}

		client.stats.ErrorCount++
	} else {
		/*
			if client.errcnt > 0 {
				if client.weight >= errWeight {
					client.weight -= errWeight
				}
				client.errcnt--
				if c.Len() >= minHeapSize {
					heap.Fix(c, client.index)
				}
			}
		*/
	}
	c.Unlock()
}

func (c *HttpClusterClient) updateClientAddr() {
	addr := strings.Split(c.host, ":")
	addrs, err := net.LookupHost(addr[0])
	if nil != err {
		log.Println("lookup host err: ", c.host, err)
		return
	}
	// only ipv4
	var ips []string
	for _, s := range addrs {
		ip := net.ParseIP(s)
		if ip != nil && len(ip.To4()) == net.IPv4len {
			ips = append(ips, s)
		}
	}

	c.endpoints = ips

	var port int
	if len(addr) > 1 {
		port, err = strconv.Atoi(addr[1])
		if nil != err {
			log.Println("parse port err: ", err)
			return
		}
	} else {
		switch c.scheme {
		case "http", "HTTP":
			port = 80
		case "https", "HTTPS":
			port = 443
		}
	}

	//统计打印
	Debugf("############clients stats :%s#############", c.host)
	c.RLock()
	for i := range c.clients {
		Debugf("## ip :%s  use count :%d  index : %d  err total :%d  err peroid :%d  weights : %d",
			c.clients[i].endpoint, c.clients[i].stats.UseCount, c.clients[i].index, c.clients[i].stats.ErrorCount,
			c.clients[i].errcnt, c.clients[i].weight)
	}
	c.RUnlock()

	c.clear(ips)

	for i := range ips {
		if !c.exist(ips[i]) {
			c.add(ips[i], newHTTPClient(fmt.Sprintf("%s:%d", addr[0], port), ips[i], port,
				c.cfg.HeaderTimeoutPerRequest, c.cfg.Redirect, c.cfg.Cert))
		}
	}

	if c.Len() == 0 {
		log.Println("cluster has no client to use")
	}

}

func (c *HttpClusterClient) Do(request *http.Request) (*http.Response, error) {

	resp, err := c.DoRequest(request)
	return resp, err
}

func (c *HttpClusterClient) DoRequest(request *http.Request) (*http.Response, error) {
	var err error
	var retry int
	cerr := &ClusterError{}
	var resp *http.Response

	// hold body content
	var bodyBytes []byte
	if request.Body != nil {
		bodyBytes, err = ioutil.ReadAll(request.Body)
		if err != nil {
			log.Printf("read request body error:%v", err)
			return nil, err
		}
	}
	for retry = 0; retry < c.cfg.Retry; retry++ {
		// restore body content
		if bodyBytes != nil {
			request.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		}
		client := c.get()
		if client == nil {
			c.updateClientAddr()
			err = fmt.Errorf("nil client")
			continue
		}

		c.use(client)

		resp, err = client.client.Do(request)

		c.done(client)
		c.occurErr(client, err)

		if nil != err {
			cerr.Errors = append(cerr.Errors, err)
			// mask previous errors with context error, which is controlled by user
			if err == context.Canceled || err == context.DeadlineExceeded {
				// return nil, nil, err
				log.Println("context err, retry")
			}

			// c.occurErr(client, err)
			log.Printf("cluster: put client back %v err: %v", client.endpoint, err)
			continue
		}

		return resp, err
	}
	if retry >= c.cfg.Retry && cerr.Errors != nil {
		log.Printf("cluster call failed after %v times", c.cfg.Retry)
	}

	if nil == resp && nil == err {
		err = fmt.Errorf("Unknown Err")
	}
	return resp, err
}

type ClusterError struct {
	Errors []error
}

func (ce *ClusterError) Error() string {
	return ErrClusterUnavailable.Error()
}

func (ce *ClusterError) Detail() string {
	s := ""
	for i, e := range ce.Errors {
		s += fmt.Sprintf("error #%d: %s\n", i, e)
	}
	return s
}

type Log interface {
	Error(format string, args ...interface{})
	Info(format string, args ...interface{})
	Notice(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

var gLogger Log
var logMode bool

func Debugf(format string, args ...interface{}) {
	if logMode {
		gLogger.Warning(format, args...)
	}
}

func RegLog(logger Log) {
	gLogger = logger
	logMode = true
}

const errWeight uint64 = 10
const minHeapSize = 1

var DefaultRequestTimeout = 10 * time.Second
var DefaultRetry = 1
var Redirect = true

var (
	ErrNoEndpoints           = errors.New("client: no endpoints available")
	ErrTooManyRedirects      = errors.New("client: too many redirects")
	ErrClusterUnavailable    = errors.New("client: cluster is unavailable or misconfigured")
	ErrNoLeaderEndpoint      = errors.New("client: no leader endpoint available")
	errTooManyRedirectChecks = errors.New("client: too many redirect checks")
)
