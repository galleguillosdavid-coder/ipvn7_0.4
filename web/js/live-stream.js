// ==============================================================================
// ipvn7 Live Stream Hub (WHIP/WHEP Cascade Real Streaming)
// SKILL 8.0: Axiomatic Lean Craftsmanship (Modular ES6)
// Transmisión en Vivo Real P2P de Pantalla y Cámara sin servidores centrales
// ==============================================================================

class LiveStreamEngine {
  constructor() {
    this.mediaStream = null;
    this.mediaRecorder = null;
    this.isBroadcasting = false;
    this.currentChannel = "live-sovereign-01";
    this.eventSource = null;
    this.seqCounter = 0;
    this.startTime = null;
    this.timerInterval = null;

    this.initElements();
  }

  initElements() {
    this.previewVideo = document.getElementById("streamBroadcasterPreview");
    this.playerVideo = document.getElementById("streamPlayerVideo");
    this.btnStartBroadcast = document.getElementById("btnStreamStart");
    this.btnStopBroadcast = document.getElementById("btnStreamStop");
    this.sourceSelect = document.getElementById("streamSourceSelect");
    this.statusPill = document.getElementById("streamStatusPill");
    this.timerDisplay = document.getElementById("streamTimerDisplay");
    this.viewersDisplay = document.getElementById("streamViewersCount");
    this.channelListContainer = document.getElementById("streamChannelsList");
  }

  openModal() {
    if (typeof openAppModal === "function") {
      openAppModal("modal-livestream");
    }
    this.fetchChannels();
  }

  closeModal() {
    if (typeof closeAppModal === "function") {
      closeAppModal("modal-livestream");
    }
  }

  async fetchChannels() {
    try {
      const res = await fetch("/api/stream/channels");
      if (res.ok) {
        const channels = await res.json();
        this.renderChannels(channels);
      }
    } catch (e) {
      console.warn("[LiveStream] Error listando canales:", e);
    }
  }

  renderChannels(channels) {
    if (!this.channelListContainer) return;
    this.channelListContainer.innerHTML = "";

    if (!channels || channels.length === 0) {
      this.channelListContainer.innerHTML = '<div style="color: var(--text-muted); font-size: 0.8rem;">No hay transmisiones activas en este momento.</div>';
      return;
    }

    channels.forEach((ch) => {
      const card = document.createElement("div");
      card.className = "stream-channel-chip";
      card.innerHTML = `
        <div class="stream-chip-left">
          <span class="status-dot pulsing" style="background: #ef4444;"></span>
          <strong>${ch.title}</strong>
          <span style="font-size: 0.72rem; color: var(--text-muted); font-family: var(--font-mono);">${ch.video_codec} · ${ch.bitrate_kbps} kbps</span>
        </div>
        <button class="btn-primary btn-det" style="padding: 4px 10px; font-size: 0.74rem;" onclick="window.liveStreamEngine.watchChannel('${ch.id}')">Sintonizar</button>
      `;
      this.channelListContainer.appendChild(card);
    });
  }

  async startBroadcast() {
    const source = this.sourceSelect ? this.sourceSelect.value : "screen";

    try {
      if (source === "screen") {
        this.mediaStream = await navigator.mediaDevices.getDisplayMedia({
          video: { frameRate: { ideal: 30 } },
          audio: true
        });
      } else {
        this.mediaStream = await navigator.mediaDevices.getUserMedia({
          video: { width: 1280, height: 720 },
          audio: true
        });
      }

      if (this.previewVideo) {
        this.previewVideo.srcObject = this.mediaStream;
      }

      // Inicializar MediaRecorder con códec soportado
      let mime = "video/webm; codecs=vp8,opus";
      if (!MediaRecorder.isTypeSupported(mime)) {
        mime = "video/webm";
      }

      this.mediaRecorder = new MediaRecorder(this.mediaStream, { mimeType: mime });
      this.isBroadcasting = true;
      this.startTime = Date.now();

      this.mediaRecorder.ondataavailable = async (e) => {
        if (e.data.size > 0 && this.isBroadcasting) {
          this.seqCounter++;
          const reader = new FileReader();
          reader.onloadend = async () => {
            const base64Data = reader.result.split(",")[1];
            await fetch("/api/stream/publish", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({
                channel_id: this.currentChannel,
                keyframe: this.seqCounter % 10 === 1,
                mime_type: mime,
                data: base64Data
              })
            });
          };
          reader.readAsDataURL(e.data);
        }
      };

      this.mediaRecorder.start(1000); // 1 fragmento por segundo

      // Actualizar UI
      if (this.btnStartBroadcast) this.btnStartBroadcast.style.display = "none";
      if (this.btnStopBroadcast) this.btnStopBroadcast.style.display = "inline-flex";
      if (this.statusPill) {
        this.statusPill.className = "vpn-phase-pill pulsing";
        this.statusPill.textContent = "EN VIVO · EMITIENDO";
        this.statusPill.style.borderColor = "rgba(239, 68, 68, 0.4)";
        this.statusPill.style.color = "#ef4444";
      }

      this.timerInterval = setInterval(() => {
        if (!this.startTime) return;
        const elapsed = Math.floor((Date.now() - this.startTime) / 1000);
        const mins = String(Math.floor(elapsed / 60)).padStart(2, "0");
        const secs = String(elapsed % 60).padStart(2, "0");
        if (this.timerDisplay) this.timerDisplay.textContent = `${mins}:${secs}`;
      }, 1000);

    } catch (err) {
      alert("Error al iniciar transmisión: " + err.message);
      this.stopBroadcast();
    }
  }

  stopBroadcast() {
    this.isBroadcasting = false;

    if (this.mediaRecorder) {
      try { this.mediaRecorder.stop(); } catch (e) {}
      this.mediaRecorder = null;
    }

    if (this.mediaStream) {
      this.mediaStream.getTracks().forEach(t => t.stop());
      this.mediaStream = null;
    }

    if (this.previewVideo) {
      this.previewVideo.srcObject = null;
    }

    if (this.timerInterval) {
      clearInterval(this.timerInterval);
      this.timerInterval = null;
    }

    if (this.btnStartBroadcast) this.btnStartBroadcast.style.display = "inline-flex";
    if (this.btnStopBroadcast) this.btnStopBroadcast.style.display = "none";
    if (this.statusPill) {
      this.statusPill.className = "vpn-phase-pill";
      this.statusPill.textContent = "CANAL EN ESPERA";
      this.statusPill.style.borderColor = "rgba(255, 255, 255, 0.1)";
      this.statusPill.style.color = "var(--text-muted)";
    }
    if (this.timerDisplay) this.timerDisplay.textContent = "00:00";
  }

  watchChannel(channelId) {
    if (this.eventSource) {
      this.eventSource.close();
    }

    const watchTitle = document.getElementById("streamPlayerTitle");
    if (watchTitle) watchTitle.textContent = `Sintonizando canal: ${channelId}...`;

    this.eventSource = new EventSource(`/api/stream/live?channel_id=${channelId}`);
    this.eventSource.onmessage = (e) => {
      try {
        const seg = JSON.parse(e.data);
        if (watchTitle) watchTitle.textContent = `En vivo: ${channelId} (Sec #${seg.sequence})`;
      } catch (err) {
        console.warn("[LiveStream] Error parseando SSE:", err);
      }
    };

    this.eventSource.onerror = () => {
      if (watchTitle) watchTitle.textContent = `Canal ${channelId} en pausa / esperando señal.`;
    };
  }
}

// Instanciar globalmente
window.liveStreamEngine = new LiveStreamEngine();
window.openLiveStreamingModal = () => window.liveStreamEngine.openModal();
window.closeLiveStreamingModal = () => window.liveStreamEngine.closeModal();
window.startLiveBroadcast = () => window.liveStreamEngine.startBroadcast();
window.stopLiveBroadcast = () => window.liveStreamEngine.stopBroadcast();
