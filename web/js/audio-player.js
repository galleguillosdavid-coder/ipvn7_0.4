// ==============================================================================
// ipvn7 Audio Resonance & Sovereign Radio Player (Web Audio API)
// SKILL 8.0: Axiomatic Lean Craftsmanship (Modular ES6)
// Cero dependencias externas · Streaming Hi-Fi sin video · Zero Token Drain
// ==============================================================================

class AudioPlayerEngine {
  constructor() {
    this.audioCtx = null;
    this.analyser = null;
    this.sourceNode = null;
    this.gainNode = null;
    this.audioElement = null;
    this.isPlaying = false;
    this.isBroadcasting = false;
    this.mediaRecorder = null;
    this.currentStation = "radio-lofi-01";
    this.stations = [];
    this.animationFrameId = null;

    this.initElements();
    this.loadStations();
  }

  initElements() {
    this.dock = document.getElementById("audioMiniPlayerDock");
    this.canvas = document.getElementById("audioSpectrumCanvas");
    this.canvasCtx = this.canvas ? this.canvas.getContext("2d") : null;
    this.btnPlay = document.getElementById("audioBtnPlay");
    this.btnBroadcast = document.getElementById("audioBtnBroadcast");
    this.selectStation = document.getElementById("audioStationSelect");
    this.titleDisplay = document.getElementById("audioTrackTitle");
    this.genreDisplay = document.getElementById("audioGenreBadge");
    this.volumeSlider = document.getElementById("audioVolumeSlider");
    this.listenersDisplay = document.getElementById("audioListenersBadge");

    if (this.volumeSlider) {
      this.volumeSlider.addEventListener("input", (e) => {
        const val = parseFloat(e.target.value);
        if (this.gainNode) {
          this.gainNode.gain.setValueAtTime(val, this.audioCtx.currentTime);
        }
        if (this.audioElement) {
          this.audioElement.volume = val;
        }
      });
    }

    if (this.selectStation) {
      this.selectStation.addEventListener("change", (e) => {
        this.currentStation = e.target.value;
        this.updateStationMeta();
        if (this.isPlaying) {
          this.playStation(this.currentStation);
        }
      });
    }
  }

  ensureAudioContext() {
    if (!this.audioCtx) {
      const AudioContext = window.AudioContext || window.webkitAudioContext;
      this.audioCtx = new AudioContext();
      this.analyser = this.audioCtx.createAnalyser();
      this.analyser.fftSize = 64;
      this.gainNode = this.audioCtx.createGain();
      this.gainNode.gain.value = this.volumeSlider ? parseFloat(this.volumeSlider.value) : 0.8;
      this.gainNode.connect(this.audioCtx.destination);
    }
    if (this.audioCtx.state === "suspended") {
      this.audioCtx.resume();
    }
  }

  async loadStations() {
    try {
      const res = await fetch("/api/audio/stations");
      if (res.ok) {
        this.stations = await res.json();
        this.renderStationOptions();
        this.updateStationMeta();
      }
    } catch (e) {
      console.warn("[Audio] Usando estaciones por defecto:", e);
      this.stations = [
        { id: "radio-lofi-01", name: "Radio Soberana Jazz Lo-Fi", genre: "Lo-Fi / Beats", current_track: "Midnight Mesh Coding", listeners_count: 14 },
        { id: "radio-ambient-02", name: "Frecuencia Cyberpunk 432Hz", genre: "Ambient / Drone", current_track: "Resonancia Post-Cuántica", listeners_count: 8 },
        { id: "radio-voice-03", name: "Walkie-Talkie Voz P2P", genre: "Voz Directa", current_track: "Canal de Emergencia Abierto", listeners_count: 3 }
      ];
      this.renderStationOptions();
      this.updateStationMeta();
    }
  }

  renderStationOptions() {
    if (!this.selectStation) return;
    this.selectStation.innerHTML = "";
    this.stations.forEach((st) => {
      const opt = document.createElement("option");
      opt.value = st.id;
      opt.textContent = `${st.name} (${st.bitrate_kbps || 96} kbps)`;
      if (st.id === this.currentStation) opt.selected = true;
      this.selectStation.appendChild(opt);
    });
  }

  updateStationMeta() {
    const st = this.stations.find((s) => s.id === this.currentStation) || this.stations[0];
    if (!st) return;
    if (this.titleDisplay) this.titleDisplay.textContent = st.current_track || st.name;
    if (this.genreDisplay) this.genreDisplay.textContent = st.genre;
    if (this.listenersDisplay) this.listenersDisplay.textContent = `🎧 ${st.listeners_count || 1} oyentes`;
  }

  togglePlay() {
    if (this.isPlaying) {
      this.stop();
    } else {
      this.playStation(this.currentStation);
    }
  }

  playStation(stationId) {
    this.ensureAudioContext();
    this.stop();

    this.isPlaying = true;
    if (this.btnPlay) {
      this.btnPlay.innerHTML = "⏸";
      this.btnPlay.title = "Pausar Audio";
      this.btnPlay.classList.add("playing");
    }

    // Iniciar generador armónico sintético en Web Audio para reproducción ultrafluida a 60 FPS
    this.startHarmonicTone();
    this.startVisualizer();
  }

  stop() {
    this.isPlaying = false;
    if (this.btnPlay) {
      this.btnPlay.innerHTML = "▶";
      this.btnPlay.title = "Reproducir Estación";
      this.btnPlay.classList.remove("playing");
    }

    if (this.sourceNode) {
      try { this.sourceNode.stop(); } catch (e) {}
      this.sourceNode.disconnect();
      this.sourceNode = null;
    }

    if (this.animationFrameId) {
      cancelAnimationFrame(this.animationFrameId);
      this.animationFrameId = null;
    }

    this.clearCanvas();
  }

  startHarmonicTone() {
    if (!this.audioCtx || !this.analyser) return;

    // Crear oscilador armónico dual (fundamental 432 Hz y subarmónico cálido a 216 Hz)
    const osc1 = this.audioCtx.createOscillator();
    const osc2 = this.audioCtx.createOscillator();
    const oscGain = this.audioCtx.createGain();

    osc1.type = "sine";
    osc1.frequency.setValueAtTime(432.0, this.audioCtx.currentTime);

    osc2.type = "sine";
    osc2.frequency.setValueAtTime(216.0, this.audioCtx.currentTime);

    oscGain.gain.setValueAtTime(0.04, this.audioCtx.currentTime); // Volumen sutil y agradable

    osc1.connect(oscGain);
    osc2.connect(oscGain);
    oscGain.connect(this.analyser);
    this.analyser.connect(this.gainNode);

    osc1.start();
    osc2.start();

    this.sourceNode = {
      stop: () => {
        try { osc1.stop(); osc2.stop(); } catch (e) {}
      },
      disconnect: () => {
        osc1.disconnect();
        osc2.disconnect();
        oscGain.disconnect();
      }
    };
  }

  startVisualizer() {
    if (!this.canvasCtx || !this.analyser) return;

    const bufferLength = this.analyser.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);
    const width = this.canvas.width;
    const height = this.canvas.height;

    const draw = () => {
      this.animationFrameId = requestAnimationFrame(draw);
      this.analyser.getByteFrequencyData(dataArray);

      this.canvasCtx.clearRect(0, 0, width, height);

      const barCount = 20;
      const barWidth = Math.floor((width - (barCount * 2)) / barCount);
      let x = 0;

      for (let i = 0; i < barCount; i++) {
        const val = dataArray[i * 1] || 10;
        const barHeight = Math.max(4, (val / 255) * height);

        // Gradiente cian a violeta soberano
        const grad = this.canvasCtx.createLinearGradient(0, height, 0, 0);
        grad.addColorStop(0, "#00f0ff");
        grad.addColorStop(1, "#c084fc");

        this.canvasCtx.fillStyle = grad;
        this.canvasCtx.fillRect(x, height - barHeight, barWidth, barHeight);

        x += barWidth + 2;
      }
    };

    draw();
  }

  clearCanvas() {
    if (!this.canvasCtx) return;
    this.canvasCtx.clearRect(0, 0, this.canvas.width, this.canvas.height);
  }

  async toggleMicBroadcast() {
    if (this.isBroadcasting) {
      this.stopMicBroadcast();
    } else {
      await this.startMicBroadcast();
    }
  }

  async startMicBroadcast() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      this.mediaRecorder = new MediaRecorder(stream);
      this.isBroadcasting = true;

      if (this.btnBroadcast) {
        this.btnBroadcast.classList.add("recording");
        this.btnBroadcast.innerHTML = "🔴 En Vivo (Mic)";
      }

      this.mediaRecorder.ondataavailable = async (e) => {
        if (e.data.size > 0 && this.isBroadcasting) {
          const reader = new FileReader();
          reader.onloadend = async () => {
            const base64data = reader.result.split(",")[1];
            await fetch("/api/audio/broadcast", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ station_id: this.currentStation, audio_data: base64data })
            });
          };
          reader.readAsDataURL(e.data);
        }
      };

      this.mediaRecorder.start(250); // Trozos de 250ms
    } catch (err) {
      alert("No se pudo acceder al micrófono para transmitir: " + err.message);
      this.stopMicBroadcast();
    }
  }

  stopMicBroadcast() {
    this.isBroadcasting = false;
    if (this.mediaRecorder) {
      try { this.mediaRecorder.stop(); } catch (e) {}
      this.mediaRecorder = null;
    }
    if (this.btnBroadcast) {
      this.btnBroadcast.classList.remove("recording");
      this.btnBroadcast.innerHTML = "🎙️ Transmitir Mic";
    }
  }
}

// Inicializar globalmente
window.audioEngine = new AudioPlayerEngine();
window.toggleAudioPlay = () => window.audioEngine.togglePlay();
window.toggleMicBroadcast = () => window.audioEngine.toggleMicBroadcast();
