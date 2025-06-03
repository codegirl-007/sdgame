from typing import NamedTuple, Dict, Tuple
from enum import Enum


class LoadBalancerSpec(NamedTuple):
    capacity: float        # e.g. float('inf')
    baseLatency: int       # ms
    cost: int


class WebServerSmall(NamedTuple):
    capacity: int
    baseLatency: int
    penaltyPerRPS: float
    cost: int


class WebServerMedium(NamedTuple):
    capacity: int
    baseLatency: int
    penaltyPerRPS: float
    cost: int


class CacheStandard(NamedTuple):
    capacity: int
    baseLatency: int
    penaltyPer10RPS: float
    hitRates: Dict[str, float]
    cost: int


class CacheLarge(NamedTuple):
    capacity: int
    baseLatency: int
    penaltyPer10RPS: float
    hitRates: Dict[str, float]
    cost: int


class DbReadReplica(NamedTuple):
    readCapacity: int       # RPS
    baseReadLatency: int    # ms
    penaltyPer10RPS: float
    cost: int


class ComponentSpec(NamedTuple):
    loadBalancer: LoadBalancerSpec
    webServerSmall: WebServerSmall
    webServerMedium: WebServerMedium
    cacheStandard: CacheStandard
    cacheLarge: CacheLarge
    dbReadReplica: DbReadReplica


class Design(NamedTuple):
    numWebServerSmall: int
    numWebServerMedium: int
    cacheType: str     # Either "cacheStandard" or "cacheLarge"
    cacheTTL: str
    numDbReplicas: int
    promotionDelaySeconds: int


class Level(NamedTuple):
    id: int
    description: str
    targetRPS: int
    maxP95Latency: int
    maxMonthlyCost: int
    requiredAvailability: int
    failureEvents: list
    componentSpec: ComponentSpec
    simulatedDurationSeconds: int


class CacheType(Enum):
    STANDARD = "cacheStandard"
    LARGE = "cacheLarge"


class LevelSimulator:
    def __init__(self, level: Level, design: Design):
        self.level = level
        self.design = design
        self.specs = self.level.componentSpec

    def compute_cost(self) -> int:
        s = self.specs
        d = self.design

        cost_lb = s.loadBalancer.cost
        cost_ws_small = d.numWebServerSmall * s.webServerSmall.cost
        cost_ws_medium = d.numWebServerMedium * s.webServerMedium.cost

        if d.cacheType == CacheType.STANDARD.value:
            cost_cache = s.cacheStandard.cost
        else:
            cost_cache = s.cacheLarge.cost

        # “1” here stands for the master; add d.numDbReplicas for replicas
        cost_db = s.dbReadReplica.cost * (1 + d.numDbReplicas)

        return cost_lb + cost_ws_small + cost_ws_medium + cost_cache + cost_db

    def compute_rps(self) -> Tuple[float, float]:
        """
        Returns (hits_rps, misses_rps) for a read workload of size level.targetRPS.
        """
        s = self.specs
        d = self.design

        total_rps = self.level.targetRPS

        if d.cacheType == CacheType.STANDARD.value:
            hit_rate = s.cacheStandard.hitRates[d.cacheTTL]
        else:
            hit_rate = s.cacheLarge.hitRates[d.cacheTTL]

        hits_rps = total_rps * hit_rate
        misses_rps = total_rps * (1 - hit_rate)
        return hits_rps, misses_rps

    def compute_latencies(self) -> Dict[str, float]:
        """
        Computes:
          - L95_ws (worst P95 among small/medium, given misses_rps)
          - L95_cache (baseLatency)
          - L95_db_read (based on misses_rps and replicas)
          - L95_total_read = miss_path (since misses are slower)
        """
        s = self.specs
        d = self.design

        # 1) First compute hits/misses
        _, misses_rps = self.compute_rps()

        # 2) Web server P95
        cap_small = s.webServerSmall.capacity
        cap_medium = s.webServerMedium.capacity

        weighted_count = d.numWebServerSmall + (2 * d.numWebServerMedium)

        if weighted_count == 0:
            L95_ws = float("inf")
        else:
            load_per_weighted = misses_rps / weighted_count

            L95_ws_small = 0.0
            if d.numWebServerSmall > 0:
                if load_per_weighted <= cap_small:
                    L95_ws_small = s.webServerSmall.baseLatency
                else:
                    L95_ws_small = (
                        s.webServerSmall.baseLatency 
                            + s.webServerSmall.penaltyPerRPS
                                * (load_per_weighted - cap_small)
                    )

            L95_ws_medium = 0.0
            # <<== FIXED: change “> 00” to “> 0”
            if d.numWebServerMedium > 0:
                if load_per_weighted <= cap_medium:
                    L95_ws_medium = s.webServerMedium.baseLatency
                else:
                    L95_ws_medium = (
                        s.webServerMedium.baseLatency
                            + s.webServerMedium.penaltyPerRPS
                                * (load_per_weighted - cap_medium)
                    )

            L95_ws = max(L95_ws_small, L95_ws_medium)

        # 3) Cache P95
        if d.cacheType == CacheType.STANDARD.value:
            L95_cache = s.cacheStandard.baseLatency
        else:
            L95_cache = s.cacheLarge.baseLatency

        # 4) DB read P95
        read_cap = s.dbReadReplica.readCapacity
        base_read_lat = s.dbReadReplica.baseReadLatency
        pen_per10 = s.dbReadReplica.penaltyPer10RPS

        num_reps = d.numDbReplicas
        if num_reps == 0:
            if misses_rps <= read_cap:
                L95_db_read = base_read_lat
            else:
                excess = misses_rps - read_cap
                L95_db_read = base_read_lat + pen_per10 * (excess / 10.0)
        else:
            load_per_rep = misses_rps / num_reps
            if load_per_rep <= read_cap:
                L95_db_read = base_read_lat
            else:
                excess = load_per_rep - read_cap
                L95_db_read = base_read_lat + pen_per10 * (excess / 10.0)

        # 5) End-to-end P95 read = miss_path
        L_lb = s.loadBalancer.baseLatency
        miss_path = L_lb + L95_ws + L95_db_read
        L95_total_read = miss_path

        return {
            "L95_ws": L95_ws,
            "L95_cache": L95_cache,
            "L95_db_read": L95_db_read,
            "L95_total_read": L95_total_read,
        }
    def compute_availability(self) -> float:
        """
        If failureEvents=[], just return 100.0.
        Otherwise:
          - For each failure (e.g. DB master crash at t_crash),
            if numDbReplicas==0 → downtime = sim_duration - t_crash
            else if design has auto_failover:
              downtime = failover_delay
            else:
              downtime = sim_duration - t_crash
          - availability = (sim_duration - total_downtime) / sim_duration * 100
        """
        sim_duration = self.level.simulatedDurationSeconds  # you’d need this field
        total_downtime = 0
        for event in self.level.failureEvents:
            t_crash = event["time"]
            if event["type"] == "DB_MASTER_CRASH":
                if self.design.numDbReplicas == 0:
                    total_downtime += (sim_duration - t_crash)
                else:
                    # assume a fixed promotion delay (e.g. 5s)
                    delay = self.design.promotionDelaySeconds
                    total_downtime += delay
            # (handle other event types if needed)
        return (sim_duration - total_downtime) / sim_duration * 100

    def validate(self) -> dict:
        """
        1) Cost check
        2) Throughput checks (cache, DB, WS)
        3) Latency check
        4) Availability check (if there are failureEvents)
        Return { "pass": True, "metrics": {...} } or { "pass": False, "reason": "..." }.
        """
        total_cost = self.compute_cost()
        if total_cost > self.level.maxMonthlyCost:
            return { "pass": False, "reason": f"Budget ${total_cost} > ${self.level.maxMonthlyCost}" }

        hits_rps, misses_rps = self.compute_rps()

        # Cache capacity
        cache_cap = (
            self.specs.cacheStandard.capacity
            if self.design.cacheType == CacheType.STANDARD.value
            else self.specs.cacheLarge.capacity
        )
        if hits_rps > cache_cap:
            return { "pass": False, "reason": f"Cache overloaded ({hits_rps:.1f} RPS > {cache_cap})" }

        # DB capacity
        db_cap = self.specs.dbReadReplica.readCapacity
        if self.design.numDbReplicas == 0:
            if misses_rps > db_cap:
                return { "pass": False, "reason": f"DB overloaded ({misses_rps:.1f} RPS > {db_cap})" }
        else:
            per_rep = misses_rps / self.design.numDbReplicas
            if per_rep > db_cap:
                return {
                    "pass": False,
                    "reason": f"DB replicas overloaded ({per_rep:.1f} RPS/replica > {db_cap})"
                }

        # WS capacity
        total_ws_cap = (
            self.design.numWebServerSmall * self.specs.webServerSmall.capacity
                + self.design.numWebServerMedium * self.specs.webServerMedium.capacity
        )
        if misses_rps > total_ws_cap:
            return {
                "pass": False,
                "reason": f"Web servers overloaded ({misses_rps:.1f} RPS > {total_ws_cap})"
            }

        # Latency
        lat = self.compute_latencies()
        if lat["L95_total_read"] > self.level.maxP95Latency:
            return {
                "pass": False,
                "reason": f"P95 too high ({lat['L95_total_read']:.1f} ms > {self.level.maxP95Latency} ms)"
            }

        # Availability (only if failureEvents is nonempty)
        availability = 100.0
        if self.level.failureEvents:
            availability = self.compute_availability()
            if availability < self.level.requiredAvailability:
                return {
                    "pass": False,
                    "reason": f"Availability too low ({availability:.1f}% < "
                        f"{self.level.requiredAvailability}%)"
                }

        # If we reach here, all checks passed
        return {
            "pass": True,
            "metrics": {
                "cost": total_cost,
                "p95": lat["L95_total_read"],
                "achievedRPS": self.level.targetRPS,
                "availability": (
                    100.0 if not self.level.failureEvents else availability
                )
            }
        }
