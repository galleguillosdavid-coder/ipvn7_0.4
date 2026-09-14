// ==============================================================================
// ipvn7 Network Sentinel & Voluntary Probe (Censorship & Health Monitor)
// SKILL 8.0: Axiomatic Lean Craftsmanship (Modular ES6)
// Sonda voluntaria opt-in · Cero consumo de CPU al estar apagada · Zero-PII
// ==============================================================================

class NetworkProbeManager {
  constructor() {
    this.isActive = false;
    this.pollingInterval = null;

    this.initElements();
    this.fetchInitialState();
  }

  initElements() {
    this.headerBtn = document.getElementById("probe-header-btn");
    this.headerText = document.getElementById("probe-header-text");
    this.headerDot = document.getElementById("probe-pill-dot");
    this.modalSwitch = document.getElementById("probeModalSwitch");
    this.healthScoreDisplay = document.getElementById("probeHealthScore");
    this.latencyDisplay = document.getElementById("probeLatencyVal");
    this.jitterDisplay = document.getElementById("probeJitterVal");
    this.lossDisplay = document.getElementById("probeLossVal");
    this.peersAuditedDisplay = document.getElementById("probePeersCount");
    this.incidentsContainer = document.getElementById("probeIncidentsList");
  }

  openModal() {
    if (typeof openAppModal === "function") {
      openAppModal("modal-network-probe");
    }
    this.refreshReport();
  }

  closeModal() {
    if (typeof closeAppModal === "function") {
      closeAppModal("modal-network-probe");
    }
  }

  async fetchInitialState() {
    try {
      const res = await fetch("/api/probe/report");
      if (res.ok) {
        const rep = await res.json();
        this.isActive = rep.is_active;
        this.updateUI(rep);
      }
    } catch (e) {
      console.warn("[Probe] No se pudo leer reporte inicial:", e);
    }
  }

  async toggle() {
    try {
      const res = await fetch("/api/probe/toggle", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ active: !this.isActive })
      });
      if (res.ok) {
        const data = await res.json();
        this.isActive = data.is_active;
        await this.refreshReport();

        if (this.isActive && !this.pollingInterval) {
          this.pollingInterval = setInterval(() => this.refreshReport(), 10000);
        } else if (!this.isActive && this.pollingInterval) {
          clearInterval(this.pollingInterval);
          this.pollingInterval = null;
        }
      }
    } catch (err) {
      alert("Error alternando la sonda de red: " + err.message);
    }
  }

  async refreshReport() {
    try {
      const res = await fetch("/api/probe/report");
      if (res.ok) {
        const rep = await res.json();
        this.updateUI(rep);
      }
    } catch (e) {
      console.warn("[Probe] Error actualizando reporte:", e);
    }
  }

  updateUI(rep) {
    this.isActive = rep.is_active;

    // Header Pill
    if (this.headerText) {
      this.headerText.textContent = this.isActive ? "Sonda: Activa" : "Sonda: Off";
      this.headerText.style.color = this.isActive ? "#10b981" : "var(--text-muted)";
    }
    if (this.headerDot) {
      this.headerDot.className = this.isActive ? "status-dot pulsing" : "status-dot";
      this.headerDot.style.background = this.isActive ? "#10b981" : "rgba(255, 255, 255, 0.2)";
    }

    // Modal Switch
    if (this.modalSwitch) {
      this.modalSwitch.checked = this.isActive;
    }

    // Métricas
    if (this.healthScoreDisplay) {
      this.healthScoreDisplay.textContent = `${rep.health_score.toFixed(1)}%`;
      this.healthScoreDisplay.style.color = rep.health_score > 90 ? "#10b981" : "#f59e0b";
    }
    if (this.latencyDisplay) this.latencyDisplay.textContent = `${rep.latency_ewma_ms.toFixed(1)} ms`;
    if (this.jitterDisplay) this.jitterDisplay.textContent = `${rep.jitter_ms.toFixed(2)} ms`;
    if (this.lossDisplay) this.lossDisplay.textContent = `${rep.packet_loss_pct.toFixed(1)}%`;
    if (this.peersAuditedDisplay) this.peersAuditedDisplay.textContent = `${rep.peers_audited} pares Kleinberg`;

    // Incidentes de Censura e Interferencia
    const humanBadge = document.getElementById("humanProbeBadge");
    if (this.incidentsContainer) {
      this.incidentsContainer.innerHTML = "";
      if (!rep.censorship_incidents || rep.censorship_incidents.length === 0) {
        this.incidentsContainer.innerHTML = '<div style="color: #10b981; font-size: 0.8rem; padding: 10px;">✓ Cero interferencias detectadas. Tu conexión está completamente libre y sin censura.</div>';
        if (humanBadge) {
          humanBadge.textContent = "100% Limpia";
          humanBadge.style.color = "#10b981";
        }
        return;
      }

      if (humanBadge) {
        humanBadge.textContent = `${rep.censorship_incidents.length} Alerta(s)`;
        humanBadge.style.color = "#f59e0b";
      }

      rep.censorship_incidents.forEach((inc) => {
        const row = document.createElement("div");
        row.className = "probe-incident-chip";
        row.innerHTML = `
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px;">
            <span class="badge badge-amber" style="font-size: 0.68rem;">🚨 ${inc.anomaly_type}</span>
            <span style="font-size: 0.7rem; color: var(--text-muted);">${new Date(inc.timestamp).toLocaleTimeString()}</span>
          </div>
          <div style="font-size: 0.78rem; font-weight: 500; color: #fff; margin-bottom: 2px;">Objetivo: <code>${inc.target_host}</code></div>
          <div style="font-size: 0.74rem; color: var(--text-muted); margin-bottom: 4px;">${inc.observed_behavior}</div>
          <div style="font-size: 0.72rem; color: #10b981; font-family: var(--font-mono);">⚡ Mitigación: ${inc.mitigation_applied}</div>
        `;
        this.incidentsContainer.appendChild(row);
      });
    }
  }
}

// Instanciar globalmente
window.probeManager = new NetworkProbeManager();
window.toggleNetworkProbe = () => window.probeManager.toggle();
window.openNetworkProbeModal = () => window.probeManager.openModal();
window.closeNetworkProbeModal = () => window.probeManager.closeModal();
