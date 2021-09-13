package entity

import (
	"time"
)

type BaseInfo struct {
	ID         string `json:"id"`
	CreateTime int64  `json:"create_time,omitempty"`
	UpdateTime int64  `json:"update_time,omitempty"`
}

func (info *BaseInfo) GetBaseInfo() *BaseInfo {
	return info
}

func (info *BaseInfo) Updating() {
	info.UpdateTime = time.Now().Unix()
}

func (info *BaseInfo) KeyCompat(key string) {
	if info.ID == "" && key != "" {
		info.ID = key
	}
}

type BaseInfoSetter interface {
	GetBaseInfo() *BaseInfo
}

type Status uint8

type Timeout struct {
	Connect int `json:"connect,omitempty"`
	Send    int `json:"send,omitempty"`
	Read    int `json:"read,omitempty"`
}

// --- structures for upstream start ---

type Node struct {
	Host     string      `json:"host,omitempty"`
	Port     int         `json:"port,omitempty"`
	Weight   int         `json:"weight"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type NodesSetter interface {
	SetNodes([]*Node)
}

type Healthy struct {
	Interval     int `json:"interval,omitempty"`
	HttpStatuses int `json:"http_statuses,omitempty"`
	Successes    int `json:"successes,omitempty"`
}

type UnHealthy struct {
	Interval     int   `json:"interval,omitempty"`
	HTTPStatuses []int `json:"http_statuses,omitempty"`
	TCPFailures  int   `json:"tcp_failures,omitempty"`
	Timeouts     int   `json:"timeouts,omitempty"`
	HTTPFailures int   `json:"http_failures,omitempty"`
}

type Active struct {
	Type                   string    `json:"type,omitempty"`
	Timeout                int       `json:"timeout,omitempty"`
	Concurrency            int       `json:"concurrency,omitempty"`
	Host                   string    `json:"host,omitempty"`
	Port                   int       `json:"port,omitempty"`
	HTTPPath               string    `json:"http_path,omitempty"`
	HTTPSVerifyCertificate string    `json:"https_verify_certificate,omitempty"`
	Healthy                Healthy   `json:"healthy,omitempty"`
	UnHealthy              UnHealthy `json:"unhealthy,omitempty"`
	ReqHeaders             []string  `json:"req_headers,omitempty"`
}

type Passive struct {
	Type      string    `json:"type,omitempty"`
	Healthy   Healthy   `json:"healthy,omitempty"`
	UnHealthy UnHealthy `json:"unhealthy,omitempty"`
}

type HealthChecker struct {
	Active  Active  `json:"active,omitempty"`
	Passive Passive `json:"passive,omitempty"`
}

type UpstreamTLS struct {
	ClientCert string `json:"client_cert,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
}

type KeepLivePool struct {
	Size        int `json:"size,omitempty"`
	IdleTimeout int `json:"idle_timeout,omitempty"`
	Requests    int `json:"requests,omitempty"`
}

type UpstreamArg struct {
	NamespaceID string `json:"namespace_id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
}

type UpstreamDef struct {
	Nodes         interface{}       `json:"nodes,omitempty"`
	Retries       int               `json:"retries,omitempty"`
	Timeout       *Timeout          `json:"timeout,omitempty"`
	Type          string            `json:"type,omitempty"`
	Checks        *HealthChecker    `json:"checks,omitempty"`
	HashOn        string            `json:"hash_on,omitempty"`
	Key           string            `json:"key,omitempty"`
	Scheme        string            `json:"scheme,omitempty"`
	DiscoveryType string            `json:"discovery_type,omitempty"`
	DiscoveryArgs *UpstreamArg      `json:"discovery_args,omitempty"`
	PassHost      string            `json:"pass_host,omitempty"`
	UpstreamHost  string            `json:"upstream_host,omitempty"`
	Name          string            `json:"name,omitempty"`
	Desc          string            `json:"desc,omitempty"`
	ServiceName   string            `json:"service_name,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	TLS           *UpstreamTLS      `json:"tls,omitempty"`
	Pool          *KeepLivePool     `json:"keepalive_pool,omitempty"`
}

func (u *UpstreamDef) SetNodes(nodes []*Node) {
	u.Nodes = NodesFormat(nodes)
}

type Upstream struct {
	BaseInfo
	UpstreamDef
}

func (u *Upstream) SetNodes(nodes []*Node) {
	(u.UpstreamDef).SetNodes(nodes)
}

// --- structures for upstream end ---

type Route struct {
	BaseInfo
	URI             string                 `json:"uri,omitempty"`
	Uris            []string               `json:"uris,omitempty"`
	Name            string                 `json:"name" validate:"max=50"`
	Desc            string                 `json:"desc,omitempty" validate:"max=256"`
	Priority        int                    `json:"priority,omitempty"`
	Methods         []string               `json:"methods,omitempty"`
	Host            string                 `json:"host,omitempty"`
	Hosts           []string               `json:"hosts,omitempty"`
	RemoteAddr      string                 `json:"remote_addr,omitempty"`
	RemoteAddrs     []string               `json:"remote_addrs,omitempty"`
	Timeout         *Timeout               `json:"timeout,omitempty"`
	Vars            []interface{}          `json:"vars,omitempty"`
	FilterFunc      string                 `json:"filter_func,omitempty"`
	Script          interface{}            `json:"script,omitempty"`
	ScriptID        interface{}            `json:"script_id,omitempty"` // For debug and optimization(cache), currently same as Route's ID
	Plugins         map[string]interface{} `json:"plugins,omitempty"`
	PluginConfigID  interface{}            `json:"plugin_config_id,omitempty"`
	Upstream        *UpstreamDef           `json:"upstream,omitempty"`
	ServiceID       interface{}            `json:"service_id,omitempty"`
	UpstreamID      interface{}            `json:"upstream_id,omitempty"`
	ServiceProtocol string                 `json:"service_protocol,omitempty"`
	Labels          map[string]string      `json:"labels,omitempty"`
	EnableWebsocket bool                   `json:"enable_websocket,omitempty"`
	Status          Status                 `json:"status"`
}

func (r *Route) SetNodes(nodes []*Node) {
	r.Upstream.SetNodes(nodes)
}

type Service struct {
	BaseInfo
	Name            string                 `json:"name,omitempty"`
	Desc            string                 `json:"desc,omitempty"`
	Upstream        *UpstreamDef           `json:"upstream,omitempty"`
	UpstreamID      interface{}            `json:"upstream_id,omitempty"`
	Plugins         map[string]interface{} `json:"plugins,omitempty"`
	Script          string                 `json:"script,omitempty"`
	Labels          map[string]string      `json:"labels,omitempty"`
	EnableWebsocket bool                   `json:"enable_websocket,omitempty"`
}

func (s *Service) SetNodes(nodes []*Node) {
	s.Upstream.SetNodes(nodes)
}
