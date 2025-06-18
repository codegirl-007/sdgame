package design

import "encoding/json"

type Node struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position Position               `json:"position"`
	Props    map[string]interface{} `json:"props"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Connection struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Label     string `json:"label,omitempty"`
	Direction string `json:"direction,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	TLS       bool   `json:"tls,omitempty"`
	Capacity  int    `json:"capacity,omitempty"`
}

type Design struct {
	Nodes       []Node       `json:"nodes"`
	Connections []Connection `json:"connections"`
}

type Cache struct {
	Label          string `json:"label"`
	CacheTTL       int    `json:"cacheTTL"`
	MaxEntries     int    `json:"maxEntries"`
	EvictionPolicy string `json:"evictionPolicy"`
}

type CDN struct {
	Label           string `json:"label"`
	TTL             int    `json:"ttl"`
	GeoReplication  string `json:"geoReplication"`
	CachingStrategy string `json:"cachingStrategy"`
	Compression     string `json:"compression"`
	HTTP2           string `json:"http2"`
}

type Database struct {
	Label       string `json:"label"`
	Replication int    `json:"replication"`
}

type DataPipeline struct {
	Label          string `json:"label"`
	BatchSize      int    `json:"batchSize"`
	Transformation string `json:"transformation"`
}

type LoadBalancer struct {
	Label     string `json:"label"`
	Algorithm string `json:"algorithm"`
}

type MessageQueue struct {
	Label            string `json:"label"`
	QueueCapacity    int    `json:"queueCapacity"`
	RetentionSeconds int    `json:"retentionSeconds"`
}

type Microservice struct {
	Label           string `json:"label"`
	InstanceCount   int    `json:"instanceCount"`
	CPU             int    `json:"cpu"`
	RAMGb           int    `json:"ramGb"`
	RPSCapacity     int    `json:"rpsCapacity"`
	MonthlyUSD      int    `json:"monthlyUsd"`
	ScalingStrategy string `json:"scalingStrategy"`
	APIVersion      string `json:"apiVersion"`
}

type Monitoring struct {
	Label          string `json:"label"`
	Tool           string `json:"tool"`
	AlertMetric    string `json:"alertMetric"`    // e.g., "cpu", "latency"
	ThresholdValue int    `json:"thresholdValue"` // e.g., 80
	ThresholdUnit  string `json:"thresholdUnit"`  // e.g., "percent", "ms"
}

type ThirdPartyService struct {
	Label    string `json:"label"`
	Provider string `json:"provider"`
	Latency  int    `json:"latency"`
}

type WebServer struct {
	CPU            int `json:"cpu"`
	RamGb          int `json:"ramGb"`
	RPSCapacity    int `json:"rpsCapacity"`
	MonthlyCostUsd int `json:"monthlyCostUsd"`
}

type Request struct {
	ID        string
	Timestamp int
	LatencyMS int
	Origin    string
	Type      string
	Path      []string
}

type TickSnapshot struct {
	TickMs     int
	QueueSizes map[string]int
	NodeHealth map[string]string
}

func (n *Node) UnmarshalJSON(data []byte) error {
	type Alias Node // avoid infinite recursion
	aux := &struct {
		Props json.RawMessage `json:"props"`
		*Alias
	}{
		Alias: (*Alias)(n),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	switch n.Type {
	case "cache":
		var p Cache
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "cdn":
		var p CDN
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "database":
		var p Database
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "data pipeline":
		var p DataPipeline
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "loadBalancer":
		var p LoadBalancer
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "messageQueue":
		var p MessageQueue
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "microservice":
		var p Microservice
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "monitoring/alerting":
		var p Monitoring
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "third party service":
		var p ThirdPartyService
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	case "webserver":
		var p WebServer
		if err := json.Unmarshal(aux.Props, &p); err != nil {
			return err
		}
		n.Props = structToMap(p)

	default:
		var generic map[string]interface{}
		if err := json.Unmarshal(aux.Props, &generic); err != nil {
			return err
		}
		n.Props = generic
	}

	return nil
}

func structToMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}
