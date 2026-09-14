package l3_test

import (
	"bytes"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
	"ipvn7/pkg/l2"
	"ipvn7/pkg/l3"
)

func TestMCPServerToolsAndExecution(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	router := l1.NewKleinbergRouter(id)
	telemetry := l2.NewTelemetryRingBuffer()

	peerID, _ := l0.GenerateIdentity()
	_ = router.AddOrUpdatePeer(peerID.DID(), &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7777}, 1.2)

	server := l3.NewMCPServer(id, router, telemetry)

	// Probar petición initialize
	initReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	resp := server.HandleRequest(&initReq)
	if resp.Error != nil {
		t.Fatalf("initialize falló con error: %v", resp.Error.Message)
	}

	// Probar tools/list
	listReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	respList := server.HandleRequest(&listReq)
	if respList.Error != nil {
		t.Fatalf("tools/list falló: %v", respList.Error.Message)
	}

	// Probar tools/call get_node_status
	callReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "get_node_status", "arguments": {}}`),
	}
	respCall := server.HandleRequest(&callReq)
	if respCall.Error != nil {
		t.Fatalf("tools/call get_node_status falló: %v", respCall.Error.Message)
	}

	// Probar stdio stream
	inBuf := bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/list\"}\n")
	var outBuf bytes.Buffer

	err := server.ServeStdio(inBuf, &outBuf)
	if err != nil {
		t.Fatalf("ServeStdio falló: %v", err)
	}

	if !strings.Contains(outBuf.String(), "get_node_status") {
		t.Errorf("Salida JSON-RPC no contiene get_node_status: %s", outBuf.String())
	}
}

func TestMCPExtendedTools(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	router := l1.NewKleinbergRouter(id)
	telemetry := l2.NewTelemetryRingBuffer()
	fw := l1.NewZTNAFirewall(true)
	dag := l1.NewDAGStore(id)
	wot := l1.NewWebOfTrust()
	mp := l1.NewMultipathScheduler()
	copilot := l3.NewAICopilotEngine(fw, mp, telemetry)

	server := l3.NewMCPServer(id, router, telemetry)
	server.AttachSubsystems(
		fw, dag, wot, mp, copilot,
		func(name string) (string, string, string, bool) {
			if name == "notebook.ipv7" {
				return "did:ipvn7:notebook_mock", "fd07::106", "10.7.0.106", true
			}
			return "", "", "", false
		},
		func(peerDID string) (string, []string, error) {
			return "123456", []string{"🚀", "🛡️", "🔑", "⚡"}, nil
		},
	)

	// 1. Probar ipvn7_firewall_rule
	setRuleReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "ipvn7_firewall_rule", "arguments": {"action": "set", "did": "did:ipvn7:test_peer", "allow_inbound": true}}`),
	}
	resp := server.HandleRequest(&setRuleReq)
	if resp.Error != nil {
		t.Fatalf("ipvn7_firewall_rule falló: %v", resp.Error.Message)
	}

	// 2. Probar ipvn7_dag_put
	dagReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      11,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "ipvn7_dag_put", "arguments": {"payload": "test block content from MCP"}}`),
	}
	respDAG := server.HandleRequest(&dagReq)
	if respDAG.Error != nil {
		t.Fatalf("ipvn7_dag_put falló: %v", respDAG.Error.Message)
	}

	// 3. Probar ipvn7_wot_vouch
	wotReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      12,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "ipvn7_wot_vouch", "arguments": {"subject_did": "did:ipvn7:test_peer", "trust_level": 0.9, "reason": "Trusted node"}}`),
	}
	respWoT := server.HandleRequest(&wotReq)
	if respWoT.Error != nil {
		t.Fatalf("ipvn7_wot_vouch falló: %v", respWoT.Error.Message)
	}

	// 4. Probar ipvn7_ddns_resolve
	ddnsReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      13,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "ipvn7_ddns_resolve", "arguments": {"name": "notebook.ipv7"}}`),
	}
	respDDNS := server.HandleRequest(&ddnsReq)
	if respDDNS.Error != nil {
		t.Fatalf("ipvn7_ddns_resolve falló: %v", respDDNS.Error.Message)
	}

	// 5. Probar ipvn7_sas_derive
	sasReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      14,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "ipvn7_sas_derive", "arguments": {"peer_did": "did:ipvn7:notebook_mock"}}`),
	}
	respSAS := server.HandleRequest(&sasReq)
	if respSAS.Error != nil {
		t.Fatalf("ipvn7_sas_derive falló: %v", respSAS.Error.Message)
	}

	// 6. Probar ipvn7_ai_diagnose
	aiReq := l3.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      15,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "ipvn7_ai_diagnose", "arguments": {"recent_latency_ms": 65.0, "drops_count": 8}}`),
	}
	respAI := server.HandleRequest(&aiReq)
	if respAI.Error != nil {
		t.Fatalf("ipvn7_ai_diagnose falló: %v", respAI.Error.Message)
	}
}

