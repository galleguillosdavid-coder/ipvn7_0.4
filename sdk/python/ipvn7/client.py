"""
ipvn7.client - High-level SovereignNode client interface for AI Agents
"""

import json
import time
import urllib.request
import urllib.error
from typing import List, Dict, Any, Optional

from .models import (
    IntentScope,
    BindingScope,
    BindingRecord,
    UINPassport,
    MemoryArbiterStats,
    NodeProfile,
    TaskProofResult,
)

class SovereignNode:
    """
    SovereignNode client connects an AI Agent to an ipvn7 Network OS node,
    enabling cryptographic identity, scoped delegations, and post-quantum communications.
    """

    def __init__(self, base_url: str = "http://localhost:7070", scope: IntentScope = IntentScope.AI_AGENT):
        self.base_url = base_url.rstrip("/")
        self.default_scope = scope

    def _get(self, endpoint: str) -> Dict[str, Any]:
        url = f"{self.base_url}{endpoint}"
        req = urllib.request.Request(url, headers={"User-Agent": "ipvn7-python-sdk/0.3.0"})
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode("utf-8"))

    def _post(self, endpoint: str, data: Dict[str, Any]) -> Dict[str, Any]:
        url = f"{self.base_url}{endpoint}"
        body = json.dumps(data).encode("utf-8")
        req = urllib.request.Request(
            url,
            data=body,
            headers={
                "Content-Type": "application/json",
                "User-Agent": "ipvn7-python-sdk/0.3.0"
            }
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode("utf-8"))

    def is_healthy(self) -> bool:
        """Check if local ipvn7 node is responding."""
        try:
            res = self._get("/api/status")
            return "did" in res or "version" in res
        except Exception:
            return False

    def get_status(self) -> Dict[str, Any]:
        """Fetch general node status and telemetry."""
        return self._get("/api/status")

    def get_passport(self) -> UINPassport:
        """Retrieve the UIN Digital Passport of the local sovereign node."""
        data = self._get("/api/uin/passport")
        return UINPassport(
            root_id_hex=data.get("root_id_hex", ""),
            entity_did=data.get("entity_did", ""),
            mode=data.get("mode", "hybrid"),
            active_bindings=data.get("active_bindings", []),
            revoked_bindings=data.get("revoked_bindings", []),
            created_at=data.get("created_at")
        )

    def issue_binding(self, scope: BindingScope = BindingScope.AI_AGENT, duration_days: int = 30) -> BindingRecord:
        """
        Issue an authenticated UIN BindingRecord delegating cryptographic authorization
        to an AI Agent sub-identity without altering the Root ID.
        """
        req_data = {
            "scope": int(scope),
            "duration_days": duration_days
        }
        res = self._post("/api/uin/binding/create", req_data)
        return BindingRecord(
            key_id_hex=res.get("key_id_hex", ""),
            scope=res.get("scope", int(scope)),
            valid_from=res.get("valid_from", ""),
            valid_until=res.get("valid_until", ""),
            sig_root_hex=res.get("sig_root_hex", "")
        )

    def get_memory_arbiter_stats(self) -> MemoryArbiterStats:
        """Fetch the Global Memory Arbiter quota telemetry (Anti-OOM DoS engine)."""
        data = self._get("/api/memory/arbiter")
        return MemoryArbiterStats(
            total_limit_bytes=data.get("total_limit_bytes", 0),
            total_allocated_bytes=data.get("total_allocated_bytes", 0),
            usage_percentage=float(data.get("usage_percentage", 0.0)),
            total_rejections=data.get("total_rejections", 0),
            classes=data.get("classes", {})
        )

    def get_hierarchy_profiles(self) -> List[NodeProfile]:
        """Retrieve node hierarchy profiles and allowed intent scopes."""
        data = self._get("/api/hierarchy/snapshot")
        profiles = []
        if isinstance(data, list):
            for item in data:
                profiles.append(NodeProfile(
                    class_id=item.get("class", 0),
                    class_name=item.get("class_name", ""),
                    allowed_intents=item.get("allowed_intents", []),
                    reputation=float(item.get("reputation", 100.0))
                ))
        return profiles

    def send_task(self, target_url: str, payload: str, task_type: str = "PQC_AI_TASK") -> TaskProofResult:
        """Dispatch a deterministic sovereign payload to a remote node with Zero-Copy and Tit-for-Tat tracking."""
        req_data = {
            "target_url": target_url,
            "payload": payload,
            "task_type": task_type
        }
        res = self._post("/api/task/compute", req_data)
        return TaskProofResult(
            status=res.get("status", "FAILED"),
            task_type=res.get("task_type", task_type),
            local_did=res.get("local_did", ""),
            remote_did=res.get("remote_did", ""),
            round_trip_ms=float(res.get("round_trip_ms", 0.0)),
            task_proof_hash=res.get("task_proof_hash", ""),
            ztna_decision=res.get("ztna_decision", "ACCEPT"),
            peer_tier=res.get("peer_tier", "TIER_1_PRIORITY")
        )

    def verify_antireplay(self, origin_did: str, session_id: int, sequence: int, timestamp: Optional[int] = None) -> Dict[str, Any]:
        """Test anti-replay window filter defense against spoofing or stale packets."""
        if timestamp is None:
            timestamp = int(time.time())
        req_data = {
            "origin_did": origin_did,
            "session_id": session_id,
            "sequence": sequence,
            "timestamp": timestamp
        }
        return self._post("/api/antireplay/verify", req_data)
