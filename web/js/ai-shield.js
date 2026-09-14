// ==============================================================================
// ipvn7 AI Shield & Token Governance (Zero Token Drain Policy)
// SKILL 8.0: Axiomatic Lean Craftsmanship (Modular ES6)
// Control de IA 100% opcional · Heurísticas locales en Go activas por defecto
// ==============================================================================

class AIShieldManager {
  constructor() {
    this.currentMode = "off"; // "off" | "manual" | "active"
    this.modes = ["off", "manual", "active"];
    this.tokensConsumed = 0;

    this.initElements();
    this.fetchMode();
  }

  initElements() {
    this.headerBtn = document.getElementById("ai-shield-btn");
    this.headerText = document.getElementById("ai-pill-text");
    this.headerDot = document.getElementById("ai-pill-dot");
  }

  async fetchMode() {
    try {
      const res = await fetch("/api/copilot/mode");
      if (res.ok) {
        const data = await res.json();
        this.currentMode = data.copilot_mode || "off";
        this.updateUI();
      }
    } catch (e) {
      console.warn("[AIShield] Error leyendo modo de IA:", e);
      this.currentMode = "off";
      this.updateUI();
    }
  }

  async cycleMode() {
    const nextIdx = (this.modes.indexOf(this.currentMode) + 1) % this.modes.length;
    const nextMode = this.modes[nextIdx];

    try {
      const res = await fetch("/api/copilot/mode", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode: nextMode })
      });
      if (res.ok) {
        const data = await res.json();
        this.currentMode = data.copilot_mode;
        this.updateUI();
      }
    } catch (err) {
      alert("Error cambiando modo de IA: " + err.message);
    }
  }

  updateUI() {
    if (!this.headerText || !this.headerDot) return;

    if (this.currentMode === "off") {
      this.headerText.textContent = "IA: Apagada (0 Tokens)";
      this.headerText.style.color = "var(--text-muted)";
      this.headerDot.className = "status-dot";
      this.headerDot.style.background = "rgba(255, 255, 255, 0.2)";
      if (this.headerBtn) this.headerBtn.title = "IA 100% apagada: Cero consumo de tokens. Operación por heurísticas locales.";
    } else if (this.currentMode === "manual") {
      this.headerText.textContent = "IA: On-Demand";
      this.headerText.style.color = "#f59e0b";
      this.headerDot.className = "status-dot";
      this.headerDot.style.background = "#f59e0b";
      if (this.headerBtn) this.headerBtn.title = "IA en modo manual: Solo consulta a la API bajo tu clic explícito.";
    } else {
      this.headerText.textContent = "IA: Activa";
      this.headerText.style.color = "#10b981";
      this.headerDot.className = "status-dot pulsing";
      this.headerDot.style.background = "#10b981";
      if (this.headerBtn) this.headerBtn.title = "IA Copilot en monitoreo continuo.";
    }
  }
}

// Instanciar globalmente
window.aiShield = new AIShieldManager();
window.toggleAICopilotMode = () => window.aiShield.cycleMode();
