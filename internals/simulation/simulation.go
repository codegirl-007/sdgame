package simulation

import (
	"math"
)

type LoadBalancerSpec struct {
	Capacity    float64
	BaseLatency int
	Cost        int
}

type WebServerSmall struct {
	Capacity      int
	BaseLatency   int
	PenaltyPerRPS float64
	Cost          int
}

type WebServerMedium struct {
	Capacity      int
	BaseLatency   int
	PenaltyPerRPS float64
	Cost          int
}

type CacheStandard struct {
	Capacity        int
	BaseLatency     int
	PenaltyPer10RPS float64
	HitRates        map[string]float64
	Cost            int
}

type CacheLarge struct {
	Capacity        int
	BaseLatency     int
	PenaltyPer10RPS float64
	HitRates        map[string]float64
	Cost            int
}

type DbReadReplica struct {
	ReadCapacity    int // RPS
	BaseReadLatency int // ms
	PenaltyPer10RPS float64
	Cost            int
}

type ComponentSpec struct {
	LoadBalancer    LoadBalancerSpec
	WebServerSmall  WebServerSmall
	WebServerMedium WebServerMedium
	CacheStandard   CacheStandard
	CacheLarge      CacheLarge
	DbReadReplica   DbReadReplica
}

type Design struct {
	NumWebServerSmall     int
	NumWebServerMedium    int
	CacheType             CacheType // Enum (see below)
	CacheTTL              string
	NumDbReplicas         int
	PromotionDelaySeconds int
}

type FailureEvent struct {
	Type string
	Time int
}
type Level struct {
	ID                       int
	Description              string
	TargetRPS                int
	MaxP95Latency            int
	MaxMonthlyCost           int
	RequiredAvailability     int
	FailureEvents            []FailureEvent
	ComponentSpec            ComponentSpec
	SimulatedDurationSeconds int
}

type Metrics struct {
	Cost         int     `json:"cost"`
	P95          float64 `json:"p95"`
	AchievedRPS  int     `json:"achievedRPS"`
	Availability float64 `json:"availability"`
}

type EvaluationResult struct {
	Pass    bool    `json:"pass"`
	Metrics Metrics `json:"metrics"`
}

type Latencies struct {
	L95WS        float64
	L95Cache     float64
	L95DBRead    float64
	L95TotalRead float64
}

type ValidationMetrics struct {
	Cost         int
	P95          float64
	AchievedRPS  int
	Availability float64
}
type ValidationResults struct {
	Pass    bool
	Reason  *string
	Metrics *ValidationMetrics
}

type CacheType string

const (
	CacheStandardType CacheType = "cacheStandard"
	CacheLargeType    CacheType = "cacheLarge"
)

type LevelSimulator struct {
	Level  Level
	Design Design
	Specs  ComponentSpec
}

func (ls *LevelSimulator) ComputeCost() int {
	s := ls.Specs
	d := ls.Design

	costLb := s.LoadBalancer.Cost
	costWSSmall := d.NumWebServerSmall * s.WebServerSmall.Cost
	costWSMedium := d.NumWebServerMedium * s.WebServerSmall.Cost
	var costCache int

	if d.CacheType == CacheStandardType {
		costCache = s.CacheStandard.Cost
	} else {
		costCache = s.CacheLarge.Cost
	}

	costDB := s.DbReadReplica.Cost * (1 + d.NumDbReplicas)

	return costLb + costWSSmall + costWSMedium + costCache + costDB
}

func (ls *LevelSimulator) ComputeRPS() (float64, float64) {
	s := ls.Specs
	d := ls.Design
	l := ls.Level

	totalRPS := l.TargetRPS

	var hitRate float64
	if d.CacheType == CacheStandardType {
		hitRate = s.CacheStandard.HitRates[d.CacheTTL]
	} else if d.CacheType == CacheLargeType {
		hitRate = s.CacheLarge.HitRates[d.CacheTTL]
	} else {
		hitRate = 0.0
	}

	hitRPS := float64(totalRPS) * hitRate
	missesRPS := float64(totalRPS) * (1 - hitRate)
	return hitRPS, missesRPS
}

func (ls *LevelSimulator) ComputeLatencies() Latencies {
	s := ls.Specs
	d := ls.Design

	_, missesRPS := ls.ComputeRPS()

	capSmall := s.WebServerSmall.Capacity
	capMedium := s.WebServerMedium.Capacity

	weightedCount := d.NumWebServerSmall + (2 * d.NumWebServerSmall)

	var L95WS float64
	if weightedCount == 0 {
		L95WS = math.Inf(1)
	} else {
		loadPerWeighted := missesRPS / float64(weightedCount)

		L95WSSmall := 0.0
		if d.NumWebServerSmall > 0 {
			if loadPerWeighted <= float64(capSmall) {
				L95WSSmall = float64(s.WebServerSmall.BaseLatency)
			} else {
				L95WSSmall = (float64(s.WebServerSmall.BaseLatency) + s.WebServerSmall.PenaltyPerRPS*(loadPerWeighted-float64(capSmall)))
			}
		}

		L95WSMedium := 0.0
		if d.NumWebServerMedium > 0 {
			if loadPerWeighted <= float64(capMedium) {
				L95WSMedium = float64(s.WebServerSmall.BaseLatency)
			} else {
				L95WSMedium = (float64(s.WebServerSmall.BaseLatency) + s.WebServerMedium.PenaltyPerRPS*(loadPerWeighted-float64(capMedium)))
			}
		}

		L95WS = math.Max(L95WSSmall, L95WSMedium)
	}

	var L95Cache float64
	if d.CacheType == CacheStandardType {
		L95Cache = float64(s.CacheStandard.BaseLatency)
	} else if d.CacheType == CacheLargeType {
		L95Cache = float64(s.CacheLarge.BaseLatency)
	} else {
		L95Cache = 0.0
	}

	readCap := s.DbReadReplica.ReadCapacity
	baseReadLat := s.DbReadReplica.BaseReadLatency
	penPer10 := s.DbReadReplica.PenaltyPer10RPS

	numReps := d.NumDbReplicas
	var L95DbRead float64

	if numReps == 0 {
		if missesRPS <= float64(readCap) {
			L95DbRead = float64(baseReadLat)
		} else {
			excess := missesRPS - float64(readCap)
			L95DbRead = float64(baseReadLat) + penPer10*(excess/10.0)
		}
	} else {
		loadPerRep := missesRPS / float64(numReps)
		if loadPerRep <= float64(readCap) {
			L95DbRead = float64(baseReadLat)
		} else {
			loadPerRep := missesRPS / float64(numReps)
			if loadPerRep <= float64(readCap) {
				L95DbRead = float64(baseReadLat)
			} else {
				excess := loadPerRep - loadPerRep - float64(readCap)
				L95DbRead = float64(baseReadLat) + penPer10*(excess/10.0)
			}
		}
	}

	LLb := s.LoadBalancer.BaseLatency
	missPath := LLb + int(L95WS) + int(L95DbRead)
	L95TotalRead := missPath

	return Latencies{
		L95WS:        L95WS,
		L95Cache:     L95Cache,
		L95DBRead:    L95DbRead,
		L95TotalRead: float64(L95TotalRead),
	}
}

func (ls *LevelSimulator) ComputeAvailability() float64 {
	simDuration := ls.Level.SimulatedDurationSeconds
	totalDownTime := 0
	for _, event := range ls.Level.FailureEvents {
		tCrash := event.Time
		if event.Type == "DB_MASTER_CRASH" {
			if ls.Design.NumDbReplicas == 0 {
				totalDownTime += (simDuration - tCrash)
			} else {
				delay := ls.Design.PromotionDelaySeconds
				totalDownTime += delay
			}
		}
	}

	return (float64(simDuration) - float64(totalDownTime)) / float64(simDuration) * 100
}

func (ls *LevelSimulator) Validate() ValidationResults {
	totalCost := ls.ComputeCost()
	if totalCost > ls.Level.MaxMonthlyCost {
		return ValidationResults{
			Pass: false,
			Metrics: Metrics{
				Cost: totalCost,
				P95:  la,
			},
		}
	}
}
