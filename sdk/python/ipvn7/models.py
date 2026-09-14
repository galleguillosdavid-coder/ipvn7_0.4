"""
ipvn7.models - Data models and enumerations for the ipvn7 Network OS Python SDK
"""

from dataclasses import dataclass, field
from enum import Enum, IntEnum
from typing import List, Optional, Dict, Any

class IntentScope(str, Enum):
    GENERAL = "general"
    TELEMETRY = "telemetría"
    CONTROL = "control"
    AI_AGENT = "agente_ia"
    SETTLEMENT = "settlement"

class BindingScope(IntEnum):
    GENERAL = 0
    TELEMETRY = 1
    CONTROL = 2
    AI_AGENT = 3
    SETTLEMENT = 4

@dataclass
class BindingRecord:
    key_id_hex: str
    scope: int
    valid_from: str
    valid_until: str
    sig_root_hex: str
    delegate_pub: Optional[str] = None

@dataclass
class UINPassport:
    root_id_hex: str
    entity_did: str
    mode: str
    active_bindings: List[Dict[str, Any]] = field(default_factory=list)
    revoked_bindings: List[str] = field(default_factory=list)
    created_at: Optional[str] = None

@dataclass
class MemoryArbiterStats:
    total_limit_bytes: int
    total_allocated_bytes: int
    usage_percentage: float
    total_rejections: int
    classes: Dict[str, Any] = field(default_factory=dict)

@dataclass
class NodeProfile:
    class_id: int
    class_name: str
    allowed_intents: List[str] = field(default_factory=list)
    reputation: float = 100.0

@dataclass
class TaskProofResult:
    status: str
    task_type: str
    local_did: str
    remote_did: str
    round_trip_ms: float
    task_proof_hash: str
    ztna_decision: str = "ACCEPT"
    peer_tier: str = "TIER_1_PRIORITY"
