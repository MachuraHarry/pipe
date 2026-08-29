const app = {
  _isAuthError(e) {
    return e.status === 401 || e.code === 401 || e.code === "unauthorized" ||
      (e.message && e.message.includes("401"));
  },
  screen: "splash",
  api: new WorldcraftAPI(),
  state: null,
  log: [],
  worlds: [],
  selectedWorld: null,
  selectedProfile: "normal",
  lang: localStorage.getItem("wc_lang") || "de",
  volume: parseFloat(localStorage.getItem("wc_volume") || "0.5"),
  cost: null,
  minimap: null,
  visitedRooms: new Set(),
  roomInfo: {},
  audio: null,
  _fullscreenMap: false,
  _isLoading: false,
  _apiOptions: null,
  _loadingStartTime: 0,
  _loadingTimer: null,
  _installPrompt: null,
  _jobAbort: null,

  profiles: [
    { id: "kaempfer", name: null, desc: null },
    { id: "haendlerin", name: null, desc: null },
    { id: "erkunderin", name: null, desc: null },
    { id: "normal", name: null, desc: null }
  ],

  async init() {
    window._wcLang = this.lang;
    window._wcSoundEnabled = localStorage.getItem("wc_sound") !== "false";

    // PWA-Install-Prompt vorab abfangen (zeigt später optionaler Button)
    window.addEventListener("beforeinstallprompt", (e) => {
      e.preventDefault();
      this._installPrompt = e;
    });

    this.audio = new GameAudio();
    window._wcAudio = this.audio;

    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.register("/sw.js").catch(() => {});
    }

    if (this.api.hasSession) {
      try {
        const data = await this.api.resumeSession();
        if (data && data.state) {
          this.state = data.state;
          this.log = [];
          if (data.state.room) this.visitedRooms.add(data.state.room);
          this.addLog("system", data.world || data.state.world || "");
          this.renderGame();
          return;
        }
      } catch {
        // Session ungültig — aber NICHT löschen, damit "Fortsetzen" Button erscheint
      }
    }

    this.renderSplash();
  },

  renderSplash() {
    this.screen = "splash";
    this._updateProfileNames();
    UI.renderSplash(
      () => this.renderNewGame(),
      () => this.onResume(),
      () => this.renderSettings(),
      () => this.installPrompt()
    );
  },

  installPrompt() {
    if (!this._installPrompt) {
      UI.toast(t("install_not_supported"));
      return;
    }
    const p = this._installPrompt;
    this._installPrompt = null;
    p.prompt();
    p.userChoice.finally(() => {});
  },

  _updateProfileNames() {
    this.profiles[0].name = t("profile_kaempfer");
    this.profiles[0].desc = t("profile_kaempfer_desc");
    this.profiles[1].name = t("profile_haendlerin");
    this.profiles[1].desc = t("profile_haendlerin_desc");
    this.profiles[2].name = t("profile_erkunderin");
    this.profiles[2].desc = t("profile_erkunderin_desc");
    this.profiles[3].name = t("profile_normal");
    this.profiles[3].desc = t("profile_normal_desc");
  },

  async renderNewGame() {
    this.screen = "newgame";
    this.worlds = [];
    this._needsToken = false;
    if (this.api.adminToken) {
      try {
        const data = await this.api.listWorlds();
        this.worlds = data.worlds || [];
      } catch (e) {
        if (this._isAuthError(e)) {
          this._needsToken = true;
        }
      }
    } else {
      this._needsToken = true;
    }
    this._updateProfileNames();
    UI.renderNewGame(
      this.worlds,
      this.profiles,
      this._needsToken,
      (w) => { this.selectedWorld = w; },
      (p) => { this.selectedProfile = p; },
      (token) => {
        if (token) this.api.setAdminToken(token);
      },
      () => this.onStartGame(),
      () => this.renderSplash()
    );
  },

  async onStartGame() {
    const briefing = document.getElementById("briefing-input");
    const briefingText = briefing ? briefing.value.trim() : "";
    const nameInput = document.getElementById("world-name-input");
    const worldName = nameInput ? nameInput.value.trim() : "";

    if (!this.selectedWorld && !briefingText && !worldName) {
      UI.showToast(t("error_generic"), "error");
      return;
    }

    this.screen = "booting";
    UI.renderBooting();

    try {
      let config;
      if (this.selectedWorld) {
        config = { world: this.selectedWorld };
      } else {
        config = {
          world: worldName || ("web_" + Date.now()),
          briefing: briefingText,
          profil: { stil: this.selectedProfile, schwierigkeit: "normal" }
        };
      }

      const data = await this.api.createSession(config);
      this.api.saveSession(data.session_id, data.secret);

      if (data.ready && data.state) {
        this.state = data.state;
        this.log = [];
        if (data.state.room) this.visitedRooms.add(data.state.room);
        this.addLog("system", data.world || data.state.world || "");
        this.audio.init();
        this.audio.loadAll();
        this.renderGame();
      } else {
        await this._pollBoot();
      }
    } catch (e) {
      if (this._isAuthError(e)) {
        UI.showToast(t("error_unauthorized"), "error");
        this._needsToken = true;
        this.renderNewGame();
      } else {
        UI.showToast(e.message || t("error_generic"), "error");
        this.renderSplash();
      }
    }
  },

  async _pollBoot() {
    const bootStart = Date.now();
    const timeout = 180000;
    let delay = 1000;

    const fail = (msg) => {
      UI.renderBootError(msg);
      this.api.clearSession();
      setTimeout(() => this.renderSplash(), 4000);
    };

    while (Date.now() - bootStart < timeout) {
      try {
        const data = await this.api.getBootStatus();
        if (data && data.ready) {
          try {
            const st = await this.api.getState();
            this.state = (st && st.state) || st;
            this.log = [];
            if (this.state && this.state.room) this.visitedRooms.add(this.state.room);
            this.addLog("system", (this.state && this.state.world) || "");
            this.audio.init();
            this.audio.loadAll();
            UI.finishBoot();
            this.renderGame();
            this._refreshCost();
            return;
          } catch (e) {
            const c = (e.data && e.data.error && e.data.error.error_code) || e.code;
            if (c === "boot_failed") {
              fail((e.data && e.data.error && e.data.error.message) || t("boot_error"));
              return;
            }
            // State noch nicht abrufbar — weiter pollen
          }
        } else if (data && data.error) {
          fail(data.error);
          return;
        } else if (data) {
          UI.updateBootStatus(data);
        }
      } catch (e) {
        const code = (e.data && e.data.error && e.data.error.error_code) || e.code;
        if (code === "boot_failed") {
          fail((e.data && e.data.error && e.data.error.message) || t("boot_error"));
          return;
        }
        // 503/Netzwerk-Toleranz: beim Booten weiter pollen
      }
      await new Promise(r => setTimeout(r, delay));
      delay = Math.min(delay * 1.6, 5000);
    }

    fail(t("boot_timeout"));
  },

  async onResume() {
    if (!this.api.hasSession) return;
    this.screen = "booting";
    UI.renderBooting();
    try {
      const data = await this.api.resumeSession();
      if (data && data.state) {
        this.state = data.state;
        this.log = [];
        if (data.state.room) this.visitedRooms.add(data.state.room);
        this.addLog("system", data.world || data.state.world || "");
        this.audio.init();
        this.audio.loadAll();
        this.renderGame();
      } else {
        UI.showToast(t("error_generic"), "error");
        this.renderSplash();
      }
    } catch (e) {
      // Nur löschen, wenn die Session wirklich tot ist (falsches/fehlendes
      // Secret oder Server kennt sie nicht). Transiente Fehler (Netzwerk,
      // Worker gerade nicht erreichbar) lassen den Spielstand intakt, so dass
      // „Fortsetzen" gleich erneut versucht werden kann.
      if (e.code === "unauthorized" || e.code === "unknown_session") {
        this.api.clearSession();
      } else if (e.code === "network_error") {
        UI.showToast(e.message || t("connection_lost"), "error");
        this.renderSplash();
        return;
      }
      UI.showToast(e.message || t("connection_lost"), "error");
      this.renderSplash();
    }
  },

  renderGame() {
    this.screen = "game";
    this.audio.init();

    if (this.state && this.state.room) {
      this.visitedRooms.add(this.state.room);
    }

    UI.renderGame(
      this.state,
      this.log,
      this._isLoading,
      {
        onCommand: (cmd) => this.onCommand(cmd),
        onToggleSound: () => this.onToggleSound(),
        onSettings: () => this.renderSettings(),
        onFullscreenMap: () => this.onFullscreenMap()
      }
    );

    setTimeout(() => {
      // Merke die Exits pro Raum, sobald wir ihn betreten — der Server liefert
      // pro State nur die Ausgänge des aktuellen Raums (keine Welt-Karte).
      if (this.state && this.state.room) {
        this.roomInfo[this.state.room] = { exits: this.state.exits || [] };
      }
      const canvas = document.getElementById("minimap-canvas");
      if (canvas) {
        this.minimap = new Minimap(canvas, canvas.width || 160);
        for (const rid of this.visitedRooms) {
          if (!this.minimap.rooms[rid]) {
            const info = this.roomInfo[rid];
            this.minimap.addRoom(rid, info ? (info.exits || []) : [], null, null);
          }
        }
        if (this.state && this.state.room) {
          this.minimap.setCurrentRoom(this.state.room);
        }
      }
    }, 50);
    this._refreshCost();
  },

  _startLoading(cmd) {
    this._isLoading = true;
    this._loadingStartTime = Date.now();
    // Update loading indicator in DOM (mit Cancel-Button für lange Jobs)
    UI.showLoading(cmd, () => this.onCancelJob());
    // Update timer every 100ms
    this._loadingTimer = setInterval(() => {
      UI.updateLoadingTime(Date.now() - this._loadingStartTime);
    }, 100);
  },

  _stopLoading() {
    this._isLoading = false;
    if (this._loadingTimer) {
      clearInterval(this._loadingTimer);
      this._loadingTimer = null;
    }
    UI.hideLoading();
  },

  async _refreshCost() {
    if (!this.api.hasSession) return;
    try {
      this.cost = await this.api.getCost();
    } catch {
      this.cost = null;
    }
    UI.updateCost(this.cost);
  },

  addLog(type, text, dialog) {
    const entry = { type, text, dialog: dialog || null, timestamp: Date.now() };
    this.log.push(entry);
    // Log-Begrenzung: unbegrenzt wachsende Arrays machen den Voll-Rerender bei
    // langen Sessions spürbar träge. 300 Einträge decken Stunden Spiellaufzeit ab.
    if (this.log.length > 300) {
      this.log.splice(0, this.log.length - 300);
    }
    if (this.screen === "game") {
      UI.appendLog(entry);
    }
  },

  _expandWunschOf(cmd) {
    const words = cmd.trim().split(/\s+/);
    words.shift();
    return words.join(" ").trim();
  },

  _isNewAdventureCmd(cmd) {
    return /^\s*neues\s+abenteuer\s*[.!?]*\s*$/i.test(cmd);
  },

  _isExpandCmd(cmd) {
    const first = cmd.trim().split(/\s+/)[0] || "";
    return first === "erweitere" || first === "erweitern";
  },

  async onCommand(input) {
    if (!input || !input.trim()) return;
    if (this._isLoading) return;
    const cmd = input.trim();

    this.addLog("user", cmd);
    this._apiOptions = null; // Clear previous options

    // Expand
    if (this._isExpandCmd(cmd)) {
      const wunsch = this._expandWunschOf(cmd);
      this._startLoading(cmd);
      this._jobAbort = new AbortController();
      try {
        const data = await this.api.expandWorld(wunsch);
        const result = await this.api.pollJob(data.job_id, null, this._jobAbort.signal);
        if (result && result.narration) {
          this.addLog("narrator", result.narration);
          if (result.state) this.state = result.state;
          this.renderGame();
        }
      } catch (e) {
        if (!(e && e.name === "AbortError")) this.addLog("error", e.message);
      } finally {
        this._jobAbort = null;
        this._stopLoading();
        this.renderGame();
      }
      return;
    }

    // New adventure
    if (this._isNewAdventureCmd(cmd)) {
      this._startLoading(cmd);
      this._jobAbort = new AbortController();
      try {
        const data = await this.api.newAdventure();
        const result = await this.api.pollJob(data.job_id, null, this._jobAbort.signal);
        if (result && result.narration) {
          this.addLog("narrator", result.narration);
          if (result.state) this.state = result.state;
          if (result.won || (this.state && this.state.won)) {
            this._stopLoading();
            this.audio.play("victory");
            this.renderGameOver();
            return;
          }
        }
      } catch (e) {
        if (!(e && e.name === "AbortError")) this.addLog("error", e.message);
      } finally {
        this._jobAbort = null;
        this._stopLoading();
        this.renderGame();
      }
      return;
    }

    // Normal command
    this._startLoading(cmd);
    this._jobAbort = new AbortController();
    try {
      const data = await this.api.sendCommand(cmd);
      const result = await this.api.pollJob(data.job_id, null, this._jobAbort.signal);

      if (!result) {
        this.addLog("error", t("error_generic"));
        this._stopLoading();
        this.renderGame();
        return;
      }

      const type = result.type || "narrator";
      const narration = result.narration || "";

      if (type === "fast") {
        this.addLog("fast", narration);
        this.audio.play("step");
      } else if (result.dialog) {
        if (narration) this.addLog("narrator", narration);
        this.addLog("dialog", narration, result.dialog);
        this.audio.play("dialog");
      } else {
        this.addLog("narrator", narration);
      }

      // Track prevRoom BEFORE updating state
      const prevRoom = this.state ? this.state.room : null;

      // Update state
      if (result.state) {
        this.state = result.state;
        if (this.state.room && this.state.room !== prevRoom) {
          this.visitedRooms.add(this.state.room);
          if (type !== "fast") this.audio.play("step");
        }
      }

      // Also use result.options if provided by API
      if (result.options) {
        this._apiOptions = result.options;
      }

      // Update minimap
      if (this.minimap && this.state && this.state.room) {
        this.minimap.addRoom(this.state.room, this.state.exits || [], prevRoom, null);
        this.minimap.setCurrentRoom(this.state.room);
      }

      // Check win
      if (result.won || (this.state && this.state.won)) {
        this._stopLoading();
        this.audio.play("victory");
        setTimeout(() => this.renderGameOver(), 800);
        return;
      }

    } catch (e) {
      if (e && e.name === "AbortError") {
        // bereits via onCancelJob geloggt
      } else if (e.message && e.message.includes("busy")) {
        this.addLog("system", t("error_busy"));
      } else {
        this.addLog("error", e.message || t("error_generic"));
      }
    } finally {
      this._jobAbort = null;
      this._stopLoading();
      this.renderGame();
    }
  },

  onCancelJob() {
    if (!this._jobAbort || this._jobAbort.signal.aborted) return;
    this._jobAbort.abort();
    const label = document.getElementById("loading-text");
    if (label) label.textContent = t("cancelling");
  },

  renderGameOver() {
    this.screen = "gameover";
    UI.renderGameOver(
      this.state,
      () => this.onNewAdventure(),
      () => this.renderGame(),
      () => this.renderSettings()
    );
  },

  async onNewAdventure() {
    this.screen = "booting";
    UI.renderBooting();
    try {
      const data = await this.api.newAdventure();
      const result = await this.api.pollJob(data.job_id);
      if (result && result.state) {
        this.state = result.state;
        this.addLog("system", result.narration || "");
        this.renderGame();
      } else {
        this.renderGame();
      }
    } catch (e) {
      this.addLog("error", e.message);
      this.renderGame();
    }
  },

  onToggleSound() {
    this.audio.init();
    const enabled = this.audio.toggle();
    window._wcSoundEnabled = enabled;
    const btn = document.getElementById("sound-toggle");
    if (btn) {
      btn.textContent = enabled ? "\u{1F50A}" : "\u{1F507}";
      btn.classList.toggle("active", enabled);
    }
  },

  onFullscreenMap() {
    if (this._fullscreenMap) return;
    this._fullscreenMap = true;

    const overlay = document.createElement("div");
    overlay.className = "minimap-overlay";

    const canvas = document.createElement("canvas");
    canvas.className = "minimap-canvas";
    const size = Math.min(window.innerWidth * 0.85, 500);
    canvas.width = size;
    canvas.height = size;

    const closeBtn = document.createElement("button");
    closeBtn.className = "minimap-overlay-close";
    closeBtn.textContent = "\u2715";
    closeBtn.addEventListener("click", () => {
      overlay.remove();
      this._fullscreenMap = false;
    });
    overlay.addEventListener("click", (e) => {
      if (e.target === overlay) {
        overlay.remove();
        this._fullscreenMap = false;
      }
    });

    overlay.appendChild(canvas);
    overlay.appendChild(closeBtn);
    document.body.appendChild(overlay);

    const bigMap = new Minimap(canvas, size);
    for (const rid of this.visitedRooms) {
      if (!bigMap.rooms[rid]) {
        const info = this.roomInfo[rid];
        bigMap.addRoom(rid, info ? (info.exits || []) : [], null, null);
      }
    }
    if (this.state && this.state.room) {
      bigMap.setCurrentRoom(this.state.room);
    }
  },

  renderSettings() {
    this.screen = "settings";
    UI.renderSettings(this, {
      onBack: () => {
        if (this.state) this.renderGame();
        else this.renderSplash();
      },
      onToast: (msg, type) => UI.showToast(msg, type),
      onDeleteSession: () => this.onDeleteSession(),
      onLanguageChange: () => {
        this._updateProfileNames();
        if (this.state) this.renderGame();
        else this.renderSplash();
      }
    });
  },

  async onDeleteSession() {
    try {
      await this.api.deleteSession();
    } catch {}
    this.state = null;
    this.log = [];
    this.visitedRooms.clear();
    this._apiOptions = null;
    if (this.minimap) {
      this.minimap.clear();
      this.minimap = null;
    }
    UI.showToast(t("delete_session"), "success");
    this.renderSplash();
  }
};

window._wcApp = app;

document.addEventListener("DOMContentLoaded", () => {
  app.init().catch(() => app.renderSplash());
});
