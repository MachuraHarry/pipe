class GameAudio {
  constructor() {
    this.ctx = null;
    this.enabled = localStorage.getItem("wc_sound") !== "false";
    this.volume = parseFloat(localStorage.getItem("wc_volume") || "0.5");
    this.buffers = {};
    this._ready = false;
  }

  init() {
    if (this.ctx) return;
    try {
      this.ctx = new (window.AudioContext || window.webkitAudioContext)();
      this._ready = true;
    } catch (e) {
      console.warn("Web Audio not supported");
    }
  }

  async load(name, url) {
    if (!this.ctx) return;
    try {
      const res = await fetch(url);
      if (!res.ok) return;
      const data = await res.arrayBuffer();
      this.buffers[name] = await this.ctx.decodeAudioData(data);
    } catch {
      // Silently fail for placeholder/missing audio
    }
  }

  async loadAll() {
    if (!this.ctx) return;
    const sounds = ["step", "hit", "pickup", "dialog", "victory"];
    for (const s of sounds) {
      await this.load(s, "/audio/" + s + ".wav");
    }
  }

  // Fallback: hörbare Töne auch dann, wenn die geladene Datei fehlt
  // (erste Sekunden, gescheiterter Fetch, Whitelist fehlt, …).
  _fallbackTone(name) {
    const map = {
      step: [150, 0.08, "sine"],
      hit: [320, 0.09, "square"],
      pickup: [660, 0.07, "sine"],
      dialog: [440, 0.1, "sine"],
      victory: [523.25, 0.4, "sine"],
    };
    const cfg = map[name] || map.step;
    if (!cfg) return;
    this.playTone(cfg[0], cfg[1], cfg[2]);
  }

  play(name) {
    if (!this.enabled || !this.ctx) return;
    if (!this.buffers[name]) {
      this._fallbackTone(name);
      return;
    }
    try {
      if (this.ctx.state === "suspended") {
        this.ctx.resume();
      }
      const source = this.ctx.createBufferSource();
      source.buffer = this.buffers[name];
      const gain = this.ctx.createGain();
      gain.gain.value = this.volume;
      source.connect(gain).connect(this.ctx.destination);
      source.start(0);
    } catch {
      // Ignore playback errors
    }
  }

  playTone(freq, duration, type) {
    if (!this.enabled || !this.ctx) return;
    try {
      if (this.ctx.state === "suspended") this.ctx.resume();
      const osc = this.ctx.createOscillator();
      const gain = this.ctx.createGain();
      osc.type = type || "sine";
      osc.frequency.value = freq;
      gain.gain.value = this.volume * 0.3;
      gain.gain.exponentialRampToValueAtTime(0.001, this.ctx.currentTime + (duration || 0.2));
      osc.connect(gain).connect(this.ctx.destination);
      osc.start();
      osc.stop(this.ctx.currentTime + (duration || 0.2));
    } catch {}
  }

  setVolume(v) {
    this.volume = v;
    localStorage.setItem("wc_volume", String(v));
  }

  toggle() {
    this.enabled = !this.enabled;
    localStorage.setItem("wc_sound", this.enabled ? "true" : "false");
    return this.enabled;
  }
}
