#!/usr/bin/env python3
"""
test_sdk.py - Automated Unit & Integration Tests for ipvn7 Python SDK
"""

import sys
import os
import unittest

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from ipvn7 import (
    SovereignNode,
    IntentScope,
    BindingScope,
    UINPassport,
    BindingRecord,
    MemoryArbiterStats,
)

class TestIPVN7SDK(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.endpoint = os.environ.get("IPVN7_NODE_URL", "http://172.22.72.89:7070")
        cls.node = SovereignNode(cls.endpoint, scope=IntentScope.AI_AGENT)

    def test_01_health(self):
        healthy = self.node.is_healthy()
        self.assertTrue(healthy, f"El nodo en {self.endpoint} debe responder")

    def test_02_passport(self):
        passport = self.node.get_passport()
        self.assertIsInstance(passport, UINPassport)
        self.assertEqual(len(passport.root_id_hex), 64)
        self.assertTrue(passport.entity_did.startswith("did:ipvn7:uin:"))
        self.assertEqual(passport.mode, "hybrid")

    def test_03_issue_binding(self):
        binding = self.node.issue_binding(scope=BindingScope.AI_AGENT, duration_days=30)
        self.assertIsInstance(binding, BindingRecord)
        self.assertEqual(binding.scope, int(BindingScope.AI_AGENT))
        self.assertTrue(len(binding.key_id_hex) > 0)
        self.assertTrue(len(binding.sig_root_hex) > 0)

    def test_04_memory_arbiter(self):
        arbiter = self.node.get_memory_arbiter_stats()
        self.assertIsInstance(arbiter, MemoryArbiterStats)
        self.assertEqual(arbiter.total_limit_bytes, 64 * 1024 * 1024)
        self.assertIn("replay", arbiter.classes)
        self.assertIn("qos", arbiter.classes)
        self.assertIn("bindings", arbiter.classes)

    def test_05_hierarchy(self):
        profiles = self.node.get_hierarchy_profiles()
        self.assertIsInstance(profiles, list)
        self.assertGreaterEqual(len(profiles), 1)
        self.assertTrue(hasattr(profiles[0], "class_name"))

    def test_06_antireplay_verification(self):
        import time
        now = int(time.time())
        sess = int(time.time() * 1000) % 1000000 + 5000
        did = "did:ipvn7:uin:sdk_test_runner"

        # Trama 1: Legítima
        r1 = self.node.verify_antireplay(did, sess, 1, now)
        self.assertTrue(r1.get("accepted"))

        # Trama 2: Replay inmediato duplicado
        r2 = self.node.verify_antireplay(did, sess, 1, now)
        self.assertFalse(r2.get("accepted"))

if __name__ == "__main__":
    unittest.main()
