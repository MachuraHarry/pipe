const UI = {
  _audioInited: false,
  _loadingEl: null,

  showLoading(cmd, onCancel) {
    const existing = document.getElementById("loading-bar");
    if (existing) existing.remove();
    const bar = document.createElement("div");
    bar.id = "loading-bar";
    bar.className = "loading-bar";
    bar.innerHTML = `
      <div class="loading-spinner-mini"></div>
      <span class="loading-text" id="loading-text">${t("thinking")}</span>
      <span class="loading-cmd">${this.esc(cmd)}</span>
      <span class="loading-time" id="loading-time">0.0s</span>
      ${onCancel ? `<button class="loading-cancel" id="btn-cancel-loading" type="button" aria-label="${this.esc(t("cancel"))}">✕ ${this.esc(t("cancel"))}</button>` : ""}
    `;
    document.body.appendChild(bar);
    const cancelBtn = bar.querySelector ? bar.querySelector("#btn-cancel-loading") : null;
    if (cancelBtn && onCancel) cancelBtn.onclick = onCancel;
  },

  updateLoadingTime(ms) {
    const el = document.getElementById("loading-time");
    if (el) el.textContent = (ms / 1000).toFixed(1) + "s";
  },

  hideLoading() {
    const el = document.getElementById("loading-bar");
    if (el) el.remove();
  },

  updateCost(cost) {
    const el = document.getElementById("cost-inline");
    if (el) {
      if (cost) {
        el.textContent = "≈ $" + Number(cost.cost_usd || 0).toFixed(4);
        el.title = t("ai_cost_title") + " " + cost.calls + " " + t("ai_cost_calls") + " · " + cost.tokens + " " + t("ai_cost_tokens");
      } else {
        el.remove();
      }
    }
    const detail = document.getElementById("cost-detail");
    if (detail) {
      detail.textContent = cost
        ? "≈ $" + Number(cost.cost_usd || 0).toFixed(4) + " · " + cost.calls + " " + t("ai_cost_calls") + " · " + cost.tokens + " " + t("ai_cost_tokens")
        : "-";
    }
  },

  renderSplash(onNewGame, onResume, onSettings, onInstall) {
    const root = document.getElementById("root");
    const hasSession = window._wcApp && window._wcApp.api.hasSession;
    const hasInstall = window._wcApp && window._wcApp._installPrompt;
    root.innerHTML = `
      <div class="screen splash-screen">
        <div class="splash-content">
          <div class="splash-logo">
            <div class="pixel-border splash-icon">
              <img src="/img/icon-192.png" alt="W" width="80" height="80"
                   style="image-rendering:pixelated; width:100%; height:100%;">
            </div>
          </div>
          <h1 class="splash-title">${t("title")}</h1>
          <p class="splash-subtitle">${t("subtitle")}</p>
          <div class="splash-buttons">
            ${hasSession ? `<button class="btn btn-primary" id="btn-resume">${t("continue")}</button>` : ""}
            <button class="btn btn-secondary" id="btn-new">${t("new_game")}</button>
            <button class="btn btn-outline" id="btn-settings">${t("settings")}</button>
            ${hasInstall ? `<button class="btn btn-outline" id="btn-install">${t("install")}</button>` : ""}
          </div>
        </div>
      </div>
    `;
    if (hasSession) {
      const resumeBtn = document.getElementById("btn-resume");
      if (resumeBtn) resumeBtn.onclick = onResume;
    }
    const newBtn = document.getElementById("btn-new");
    if (newBtn) newBtn.onclick = onNewGame;
    const settingsBtn = document.getElementById("btn-settings");
    if (settingsBtn) settingsBtn.onclick = onSettings;
    const installBtn = document.getElementById("btn-install");
    if (installBtn && onInstall) installBtn.onclick = onInstall;
  },

  renderNewGame(worlds, profiles, needsToken, onSelectWorld, onSelectProfile, onTokenChange, onStart, onBack) {
    const root = document.getElementById("root");
    let worldHtml = "";
    if (worlds.length > 0) {
      worldHtml = `
        <div class="world-select">
          <h3>${t("worlds")}</h3>
          <div class="world-chips" id="world-chips">
            ${worlds.map(w => `<div class="world-chip" data-world="${this._escAttr(w)}">${this.esc(w)}</div>`).join("")}
          </div>
        </div>
      `;
    }
    const tokenHtml = needsToken ? `
      <div class="token-section">
        <label>${t("admin_token")}</label>
        <input type="password" class="text-input" id="admin-token-input"
               value="${window._wcApp && window._wcApp.api.adminToken || ""}"
               placeholder="${t("admin_token_placeholder")}" autocomplete="off">
        <div class="dim-text" style="margin-top:4px">${t("admin_token_hint")}</div>
      </div>
    ` : "";
    root.innerHTML = `
      <div class="screen newgame-screen">
        <div class="top-bar">
          <button class="top-btn" id="btn-back">\u2190</button>
          <span>${t("new_game")}</span>
          <span></span>
        </div>
        <div class="newgame-content">
          ${tokenHtml}
          ${worldHtml}
          <div class="world-create">
            <h3>${t("new_world")}</h3>
            <input type="text" id="world-name-input" class="text-input"
                   placeholder="${t("world_name_placeholder")}" maxlength="40"
                   style="margin-bottom:12px">
            <h3>${t("your_briefing")}</h3>
            <textarea id="briefing-input" placeholder="${t("briefing_placeholder")}" rows="3"></textarea>
          </div>
          <div class="profile-select">
            <h3>${t("profiles")}</h3>
            <div class="profile-chips" id="profile-chips">
              ${profiles.map(p => `<div class="profile-chip${p.id === "normal" ? " selected" : ""}" data-profile="${p.id}">${p.name}</div>`).join("")}
            </div>
          </div>
          <button class="btn btn-primary btn-start" id="btn-start">${t("start_game")}</button>
        </div>
      </div>
    `;

    // Bind events
    document.getElementById("btn-back").onclick = onBack;

    // Token input
    const tokenInput = document.getElementById("admin-token-input");
    if (tokenInput && onTokenChange) {
      tokenInput.onchange = (e) => onTokenChange(e.target.value.trim());
    }

    const chips = document.querySelectorAll(".world-chip");
    chips.forEach(c => {
      c.onclick = () => {
        chips.forEach(cc => cc.classList.remove("selected"));
        c.classList.add("selected");
        onSelectWorld(c.dataset.world);
      };
    });

    const pchips = document.querySelectorAll(".profile-chip");
    pchips.forEach(c => {
      c.onclick = () => {
        pchips.forEach(cc => cc.classList.remove("selected"));
        c.classList.add("selected");
        onSelectProfile(c.dataset.profile);
      };
    });

    document.getElementById("btn-start").onclick = onStart;
  },

  renderBooting() {
    const root = document.getElementById("root");
    root.innerHTML = `
      <div class="screen booting-screen">
        <div class="boots-box pixel-border">
          <div class="boots-inner">
            <div class="boots-spinner"></div>
            <h2>${t("booting")}</h2>
            <p class="boot-phase" id="boot-phase">${t("boot_phase_start")}</p>
            <div class="boot-bar"><div class="boot-bar-fill" id="boot-bar-fill" style="width:0%"></div></div>
            <p class="boot-progress-text" id="boot-progress-text">0%</p>
            <p class="boot-elapsed" id="boot-elapsed">${t("boot_elapsed", "0:00")}</p>
            <p class="boots-tip">${t("booting_tip")}</p>
          </div>
        </div>
      </div>
    `;
    this._bootStart = Date.now();
    this._bootTimer = setInterval(() => this._updateBootElapsed(), 1000);
  },

  _updateBootElapsed() {
    const el = document.getElementById("boot-elapsed");
    if (!el || !this._bootStart) return;
    const s = Math.floor((Date.now() - this._bootStart) / 1000);
    const mm = Math.floor(s / 60);
    const ss = s % 60;
    el.textContent = t("boot_elapsed", mm + ":" + (ss < 10 ? "0" : "") + ss);
  },

  _bootPhaseText(phase) {
    return t("boot_phase_" + phase) || t("boot_phase_start");
  },

  updateBootStatus(data) {
    if (!data) return;
    const phaseEl = document.getElementById("boot-phase");
    const fill = document.getElementById("boot-bar-fill");
    const pctEl = document.getElementById("boot-progress-text");
    if (phaseEl) phaseEl.textContent = this._bootPhaseText(data.phase);
    let p = (typeof data.progress === "number") ? data.progress : 0;
    if (data.detail && phaseEl) {
      phaseEl.textContent = (this._bootPhaseText(data.phase) || "") + " · " + data.detail;
    }
    if (fill && pctEl) {
      p = Math.max(0, Math.min(100, p));
      fill.style.width = p + "%";
      pctEl.textContent = Math.round(p) + "%";
    }
    if (data.ready) {
      this._stopBootTimer();
    }
  },

  finishBoot() {
    this._stopBootTimer();
  },

  _stopBootTimer() {
    if (this._bootTimer) {
      clearInterval(this._bootTimer);
      this._bootTimer = null;
    }
  },

  renderBootError(msg) {
    this._stopBootTimer();
    const root = document.getElementById("root");
    root.innerHTML = `
      <div class="screen booting-screen">
        <div class="boots-box pixel-border">
          <div class="boots-inner">
            <div class="boots-error-icon">!</div>
            <h2>${t("boot_failed_title")}</h2>
            <p class="boot-error-msg">${this.esc(msg || t("boot_error"))}</p>
            <p class="boots-tip">${t("boot_failed_note")}</p>
          </div>
        </div>
      </div>
    `;
  },

  renderGame(state, log, isLoading, handlers) {
    const root = document.getElementById("root");
    const room = state ? state.room_description : "";
    const exits = state ? (state.exits || []) : [];
    const items = state ? (state.items || []) : [];
    const npcs = state ? (state.npcs || []) : [];
    const monsters = state ? (state.monsters || []) : [];
    const hp = state ? (state.hp || 0) : 0;
    const maxHp = state ? (state.max_hp || 10) : 10;
    const gold = state ? (state.gold || 0) : 0;
    const mana = state ? (state.mana || 0) : 0;
    const maxMana = state ? (state.max_mana || 0) : 0;
    const xp = state ? (state.xp || 0) : 0;
    const inventory = state ? (state.inventory || []) : [];
    const quests = state ? (state.quests || []) : [];
    const spells = state ? (state.spells || []) : [];
    const buffs = state ? (state.buffs || []) : [];
    const roomName = state ? (state.room_name || state.room || "") : "";

    // Get API options or build from state
    const app = window._wcApp;
    let apiOptions = [];
    if (app && app._apiOptions) {
      apiOptions = app._apiOptions;
      // Don't clear yet - persist until next command
    }

    const soundEnabled = window._wcSoundEnabled !== false;
    const lang = window._wcLang || "de";

    const cost = (app && app.cost) || null;
    const costChip = cost ? `<span class="cost-inline" id="cost-inline" title="${t("ai_cost_title")} ${cost.calls} ${t("ai_cost_calls")} · ${cost.tokens} ${t("ai_cost_tokens")}">≈ $${Number(cost.cost_usd || 0).toFixed(4)}</span>` : "";

    root.innerHTML = `
      <div class="screen game-screen" aria-busy="${isLoading ? "true" : "false"}">
        <div class="status-bar" id="status-bar">
          <div class="status-left">
            <span class="hp-inline" title="${hp}/${maxHp} HP">
              <span class="hp-icon">♥</span>
              <span class="hp-bar-mini-wrap">
                <span class="hp-bar-mini"><span class="hp-bar-fill" style="width:${Math.round((hp/maxHp)*100)}%"></span></span>
              </span>
              <span class="hp-num">${hp}/${maxHp}</span>
            </span>
            ${maxMana > 0 ? `
            <span class="mana-inline" title="${mana}/${maxMana} Mana">
              <span class="mana-icon">◆</span>
              <span class="mana-bar-mini-wrap">
                <span class="mana-bar-mini"><span class="mana-bar-fill" style="width:${Math.round((mana/maxMana)*100)}%"></span></span>
              </span>
              <span class="mana-num">${mana}/${maxMana}</span>
            </span>` : ""}
            <span class="xp-inline" title="${xp} XP">
              <span class="xp-icon">★</span>
              <span class="xp-num">${xp}</span>
            </span>
            <span class="gold-inline" title="${gold} Gold">
              <span class="gold-icon">☉</span>
              <span class="gold-num">${gold}</span>
            </span>
            ${costChip}
            ${buffs.length ? `<span class="buff-count" title="${buffs.length} ${t("buffs")}">⬡${buffs.length}</span>` : ""}
          </div>
          <div class="status-right">
            <button class="top-btn icon-btn" id="sound-toggle" title="Sound" aria-label="Sound">${soundEnabled ? "\u{1F50A}" : "\u{1F507}"}</button>
            <button class="top-btn icon-btn" id="btn-minimap-toggle" title="Map" aria-label="Karte">\u{1F5FA}</button>
            <button class="top-btn icon-btn" id="btn-settings-game" title="${t("settings")}" aria-label="${t("settings")}">\u2699</button>
          </div>
        </div>

        <div class="room-name" id="room-name">${this.esc(roomName)}</div>

        <div class="game-log" id="game-log" role="log" aria-live="polite">
          ${log.map(e => this._logEntryHtml(e, isLoading)).join("")}
        </div>

        <div class="actions-area" id="actions-area" data-loading="${isLoading ? "1" : "0"}">
          ${this._actionsGridHtml(exits, items, npcs, monsters, apiOptions, spells, isLoading)}
        </div>

        <div class="tab-bar" id="tab-bar">
          <button class="tab-btn" data-tab="inventory">${t("inventory")} (${inventory.length})</button>
          ${spells.length ? `<button class="tab-btn" data-tab="spells">${t("spells")} (${spells.length})</button>` : ""}
          ${buffs.length ? `<button class="tab-btn" data-tab="buffs">${t("buffs")} (${buffs.length})</button>` : ""}
          <button class="tab-btn" data-tab="quests">${t("quests")} (${quests.length})</button>
          <button class="tab-btn" data-tab="map">${t("map")}</button>
        </div>

        <div class="sidebar" id="sidebar">
          <div class="sidebar-tab" data-tab="inventory">
            <h3>${t("inventory")}</h3>
            <div class="items-panel">${inventory.length ? inventory.map(i =>
              `<span class="item-chip">${this.esc(i)}</span>`
            ).join("") : `<span class="dim-text">${t("no_items")}</span>`}</div>
          </div>
          <div class="sidebar-tab" data-tab="spells">
            <h3>${t("spells")}</h3>
            <div class="spells-panel">${spells.length ? spells.map(sp =>
              `<div class="spell-item">
                <span class="spell-name">${this.esc(sp.name || sp.id)}</span>
                <span class="spell-type">${this.esc(t("spell_type_" + sp.typ))}</span>
                <span class="spell-stats">◆${this.esc(sp.manakosten)} ⚔${this.esc(sp.schaden)}</span>
              </div>`
            ).join("") : `<span class="dim-text">${t("no_spells")}</span>`}</div>
          </div>
          <div class="sidebar-tab" data-tab="buffs">
            <h3>${t("buffs")}</h3>
            <div class="buffs-panel">${buffs.length ? buffs.map(b =>
              `<div class="buff-item">
                <span class="buff-art">${this.esc(b.art)}</span>
                <span class="buff-val">+${this.esc(b.wert)}</span>
                <span class="buff-rounds">${this.esc(b.runden)} ${t("buff_rounds")}</span>
              </div>`
            ).join("") : `<span class="dim-text">${t("no_buffs")}</span>`}</div>
          </div>
          <div class="sidebar-tab" data-tab="quests">
            <h3>${t("quests")}</h3>
            <div class="quests-panel">${quests.length ? quests.map(q =>
              `<div class="quest-item"><span class="quest-status ${this.esc(q.state)}"></span>${this.esc(q.name || q.id)}</div>`
            ).join("") : `<span class="dim-text">${t("no_quests")}</span>`}</div>
          </div>
          <div class="sidebar-tab" data-tab="map">
            <h3>${t("map")}</h3>
            <button class="btn btn-outline btn-fullscreen-map" id="btn-fullscreen-map">${t("fullscreen")}</button>
            <div class="minimap-wrap">
              <canvas id="minimap-canvas" width="160" height="160"></canvas>
            </div>
          </div>
        </div>

        <div class="input-bar" id="input-bar">
          <input type="text" id="cmd-input" class="cmd-input"
                 placeholder="${t("placeholder")}"
                 autocomplete="off"
                 ${isLoading ? "disabled" : ""}>
          <button class="btn btn-primary btn-send" id="btn-send" ${isLoading ? "disabled" : ""}>
            ${isLoading ? '<span class="loading-spinner-inline"></span>' : "\u27A4"}
          </button>
        </div>
      </div>
    `;

    this._bindGameEvents(handlers);
    this._scrollLogToBottom();
    this._refreshDialogButtons();
    // Desktop: Fokus zurück auf die Befehlseingabe (mobile ohne Tastatur-Pop)
    const inp0 = document.getElementById("cmd-input");
    if (inp0 && window.matchMedia && window.matchMedia("(hover: hover)").matches) {
      inp0.focus();
    }
  },

  _logEntryHtml(entry, isLoading) {
    let cls = "log-" + entry.type;
    let text = entry.text || "";
    let dialogHtml = "";

    if (entry.dialog) {
      const d = entry.dialog;
      dialogHtml = `
        <div class="dialog-bubble">
          <div class="dialog-speaker">${this.esc(d.speaker || "")}</div>
          <div class="dialog-reply">${this.esc(d.reply || "")}</div>
          ${d.choices && d.choices.length ? `
            <div class="dialog-choices" data-dialog-group="${this._escAttr(d.speaker || "")}">
              ${d.choices.map((c, ci) => {
                const cmd = c;
                const did = "dlg-" + Date.now() + "-" + ci;
                return `<button class="btn btn-dialog" data-cmd="${this._escAttr(cmd)}" data-dialog="${did}" ${isLoading ? "disabled" : ""}>${this.esc(c)}</button>`;
              }).join("")}
            </div>
          ` : ""}
        </div>
      `;
    }

    return `<div class="${cls}">${text ? `<span class="log-text">${this.esc(text)}</span>` : ""}${dialogHtml}</div>`;
  },

  _actionsGridHtml(exits, items, npcs, monsters, apiOptions, spells, isLoading) {
    const sections = [];
    const maybeDisabled = isLoading ? ' disabled="disabled"' : "";

    // Movement
    if (exits.length) {
      sections.push(`
        <div class="action-category">
          <div class="action-category-label">${t("movement")}</div>
          <div class="action-chips">
            ${exits.map(e => `<button ${maybeDisabled} class="action-chip chip-move" data-cmd="geh nach ${this._escAttr(e)}">${this.esc(e)}</button>`).join("")}
          </div>
        </div>
      `);
    }

    // Items on ground
    if (items.length) {
      sections.push(`
        <div class="action-category">
          <div class="action-category-label">${t("items")}</div>
          <div class="action-chips">
            ${items.map(i => {
              const id = i.id || i;
              const name = i.name || i;
              return `<button ${maybeDisabled} class="action-chip chip-item" data-cmd="nimm ${this._escAttr(id)}">${this.esc(name)}</button>`;
            }).join("")}
          </div>
        </div>
      `);
    }

    // NPCs
    if (npcs.length) {
      sections.push(`
        <div class="action-category">
          <div class="action-category-label">${t("npcs")}</div>
          <div class="action-chips">
            ${npcs.map(n => {
              const label = n.name || n.id;
              const role = n.role ? ` <span class="npc-role">${this.esc(n.role)}</span>` : "";
              return `<button ${maybeDisabled} class="action-chip chip-npc" data-cmd="rede mit ${this._escAttr(n.id)}">${this.esc(label)}${role}</button>`;
            }).join("")}
          </div>
        </div>
      `);
    }

    // Monsters
    if (monsters.length) {
      sections.push(`
        <div class="action-category">
          <div class="action-category-label">${t("monsters")}</div>
          <div class="action-chips">
            ${monsters.map(m => {
              const hpPct = m.max_hp ? Math.round((m.hp / m.max_hp) * 100) : 100;
              return `<button ${maybeDisabled} class="action-chip chip-monster" data-cmd="greife ${this._escAttr(m.id)} an">
                ${this.esc(m.name || m.id)}
                <span class="monster-hp"><span class="monster-hp-fill" style="width:${hpPct}%"></span></span>
              </button>`;
            }).join("")}
          </div>
        </div>
      `);
    }

    // Spell casting — one-tap attack spells on present monsters
    if (spells && spells.length && monsters.length) {
      const attackSpells = spells.filter(sp => sp.typ === "angriff");
      if (attackSpells.length) {
        sections.push(`
          <div class="action-category">
            <div class="action-category-label">${t("zauber")}</div>
            <div class="action-chips">
              ${attackSpells.map(sp => monsters.map(m =>
                `<button ${maybeDisabled} class="action-chip chip-spell" data-cmd="zauber ${this._escAttr(sp.id)} auf ${this._escAttr(m.id)}">${this.esc(sp.name || sp.id)} ▸ ${this.esc(m.name || m.id)}</button>`
              ).join("")).join("")}
            </div>
          </div>
        `);
      }
    }

    // System actions (API options)
    if (apiOptions.length) {
      sections.push(`
        <div class="action-category">
          <div class="action-category-label">${t("actions")}</div>
          <div class="action-chips">
            ${apiOptions.map(o => `<button ${maybeDisabled} class="action-chip chip-system" data-cmd="${this._escAttr(o)}">${this.esc(o)}</button>`).join("")}
          </div>
        </div>
      `);
    }

    return sections.join("") || `<div class="dim-text" style="padding:8px">${t("no_actions")}</div>`;
  },

  _escAttr(s) {
    return UI.esc(s);
  },

  esc(s) {
    return String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  },

  appendLog(entry) {
    const log = document.getElementById("game-log");
    if (!log) return;
    log.insertAdjacentHTML("beforeend", this._logEntryHtml(entry));
    this._refreshDialogButtons();
    this._scrollLogToBottom();
  },

  // Deaktiviert Choice-Buttons aller früheren Dialoge — nur die neueste
  // Dialog-Bubble ist klickbar (alte Optionen feuern sonst veraltete Züge).
  _refreshDialogButtons() {
    const groups = document.querySelectorAll(".dialog-bubble .dialog-choices");
    groups.forEach((g, idx) => {
      const isLast = idx === groups.length - 1;
      g.classList.toggle("stale", !isLast);
      g.querySelectorAll("button").forEach(b => { b.disabled = !isLast; });
    });
  },

  // Eingabe-Historie fuer Pfeiltasten ↑/↓ im Command-Input.
  _pushHistory(v) {
    if (!this._cmdHistory) this._cmdHistory = [];
    const h = this._cmdHistory;
    if (h[h.length - 1] !== v) h.push(v);
    if (h.length > 50) h.shift();
    this._historyIdx = undefined;
    this._draftBeforeHistory = "";
  },

  _scrollLogToBottom() {
    const log = document.getElementById("game-log");
    if (log) requestAnimationFrame(() => log.scrollTop = log.scrollHeight);
  },

  _bindGameEvents(handlers) {
    // Send button
    const sendBtn = document.getElementById("btn-send");
    const input = document.getElementById("cmd-input");
    if (sendBtn && input) {
      const submit = () => {
        const v = input.value.trim();
        if (!v) return;
        const app = window._wcApp;
        // Waehrend eines laufenden Zugs: Text behalten statt verwerfen.
        if (app && app._isLoading) return;
        this._pushHistory(v);
        input.value = "";
        handlers.onCommand(v);
      };
      sendBtn.onclick = submit;
      input.onkeydown = (e) => {
        if (e.key === "Enter") {
          submit();
        } else if (e.key === "ArrowUp" || e.key === "ArrowDown") {
          // Historie durchblaettern; laufender Zug blockiert das nicht.
          e.preventDefault();
          const h = this._cmdHistory || [];
          if (!h.length) return;
          if (this._historyIdx === undefined || this._historyIdx === null) {
            this._historyIdx = h.length;
            this._draftBeforeHistory = input.value;
          }
          this._historyIdx += (e.key === "ArrowUp" ? -1 : 1);
          this._historyIdx = Math.max(0, Math.min(h.length, this._historyIdx));
          input.value = (this._historyIdx === h.length) ? (this._draftBeforeHistory || "") : h[this._historyIdx];
        }
      };
    }

    // Sound toggle
    const soundBtn = document.getElementById("sound-toggle");
    if (soundBtn) soundBtn.onclick = handlers.onToggleSound;

    // Minimap toggle (mobile)
    const minimapBtn = document.getElementById("btn-minimap-toggle");
    if (minimapBtn) minimapBtn.onclick = handlers.onFullscreenMap;

    // Settings
    const settingsBtn = document.getElementById("btn-settings-game");
    if (settingsBtn) settingsBtn.onclick = handlers.onSettings;

    // Fullscreen map
    const fsMapBtn = document.getElementById("btn-fullscreen-map");
    if (fsMapBtn) fsMapBtn.onclick = handlers.onFullscreenMap;

    // Tab bar
    const tabs = document.querySelectorAll("#tab-bar .tab-btn");
    const panels = document.querySelectorAll("#sidebar .sidebar-tab");
    const sidebar = document.getElementById("sidebar");
    let activeTab = null;
    tabs.forEach(tab => {
      tab.onclick = () => {
        const target = tab.dataset.tab;
        if (activeTab === target) {
          // Toggle off
          activeTab = null;
          tabs.forEach(t => t.classList.remove("active"));
          panels.forEach(p => p.classList.remove("active"));
          if (sidebar) sidebar.classList.remove("open");
        } else {
          activeTab = target;
          tabs.forEach(t => t.classList.toggle("active", t.dataset.tab === target));
          panels.forEach(p => p.classList.toggle("active", p.dataset.tab === target));
          if (sidebar) sidebar.classList.add("open");
        }
      };
    });

    // Dialog choices — single-use: disable all in group after click
    const log = document.getElementById("game-log");
    if (log) {
      log.addEventListener("click", (e) => {
        const btn = e.target.closest("[data-dialog]");
        if (!btn || btn.disabled) return;
        const dialogId = btn.getAttribute("data-dialog");
        // Disable all buttons in same dialog group
        log.querySelectorAll(`[data-dialog]`).forEach(b => {
          if (b.getAttribute("data-dialog") !== dialogId) {
            // Same log entry, different button — find shared parent
            if (b.closest(".dialog-choices") === btn.closest(".dialog-choices")) {
              b.disabled = true;
              b.classList.add("disabled");
            }
          }
        });
        btn.classList.add("selected");
        btn.disabled = true;
        const cmd = btn.getAttribute("data-cmd");
        if (cmd && handlers.onCommand) handlers.onCommand(cmd);
      });
    }

    // Action chips (non-dialog buttons)
    document.querySelectorAll("[data-cmd]:not([data-dialog])").forEach(btn => {
      btn.addEventListener("click", (e) => {
        e.preventDefault();
        const cmd = btn.getAttribute("data-cmd");
        if (cmd && handlers.onCommand) {
          handlers.onCommand(cmd);
        }
      });
    });
  },

  renderGameOver(state, onNewAdventure, onBack, onSettings) {
    const root = document.getElementById("root");
    root.innerHTML = `
      <div class="screen gameover-screen">
        <div class="boots-box pixel-border">
          <div class="boots-inner">
            <h2>${t("victory")}</h2>
            <p>${state ? t("gold_earned") + " " + (state.gold || 0) : ""}</p>
            <div class="gameover-buttons">
              <button class="btn btn-primary" id="btn-new-adv">${t("new_adventure")}</button>
              <button class="btn btn-secondary" id="btn-back-game">${t("back_to_game")}</button>
              <button class="btn btn-outline" id="btn-settings-go">${t("settings")}</button>
            </div>
          </div>
        </div>
      </div>
    `;
    document.getElementById("btn-new-adv").onclick = onNewAdventure;
    document.getElementById("btn-back-game").onclick = onBack;
    document.getElementById("btn-settings-go").onclick = onSettings;
  },

  renderSettings(app, opts) {
    const root = document.getElementById("root");
    const lang = app.lang;
    root.innerHTML = `
      <div class="screen settings-screen">
        <div class="top-bar">
          <button class="top-btn" id="btn-settings-back">\u2190</button>
          <span>${t("settings")}</span>
          <span></span>
        </div>
        <div class="settings-list">
          <div class="setting-group">
            <label>${t("language")}</label>
            <div class="setting-row">
              <button class="btn btn-sm ${lang === "de" ? "btn-primary" : "btn-outline"}" id="lang-de">Deutsch</button>
              <button class="btn btn-sm ${lang === "en" ? "btn-primary" : "btn-outline"}" id="lang-en">English</button>
            </div>
          </div>
          <div class="setting-group">
            <label>${t("sound")}</label>
            <div class="setting-row">
              <input type="range" min="0" max="1" step="0.05"
                     value="${app.volume}" id="volume-slider" class="range-slider">
              <span class="dim-text" id="volume-label">${Math.round(app.volume * 100)}%</span>
            </div>
          </div>
          <div class="setting-group">
            <label>${t("admin_token")}</label>
            <input type="password" class="text-input" id="admin-token-input"
                   value="${app.api.adminToken || ""}" placeholder="...Secret..."
                   autocomplete="off">
            <div class="dim-text" style="margin-top:4px">${t("admin_token_hint")}</div>
          </div>
          <div class="setting-group">
            <label>${t("ai_cost")}</label>
            <div class="setting-row">
              <span class="dim-text" id="cost-detail">-</span>
            </div>
          </div>
          <div class="setting-group">
            <button class="btn btn-danger btn-full" id="btn-delete-session">${t("delete_session")}</button>
          </div>
        </div>
      </div>
    `;
    document.getElementById("btn-settings-back").onclick = opts.onBack;
    document.getElementById("lang-de").onclick = () => {
      app.lang = "de";
      localStorage.setItem("wc_lang", "de");
      window._wcLang = "de";
      opts.onLanguageChange();
    };
    document.getElementById("lang-en").onclick = () => {
      app.lang = "en";
      localStorage.setItem("wc_lang", "en");
      window._wcLang = "en";
      opts.onLanguageChange();
    };
    document.getElementById("volume-slider").oninput = (e) => {
      const v = parseFloat(e.target.value);
      app.volume = v;
      window._wcAudio.setVolume(v);
      document.getElementById("volume-label").textContent = Math.round(v * 100) + "%";
    };
    document.getElementById("admin-token-input").onchange = (e) => {
      app.api.setAdminToken(e.target.value.trim());
      opts.onToast(t("admin_token_saved"), "success");
    };
    document.getElementById("btn-delete-session").onclick = () => {
      if (confirm(t("confirm_delete"))) opts.onDeleteSession();
    };
    this.updateCost(app.cost);
  },

  showToast(msg, type) {
    const existing = document.getElementById("wc-toast");
    if (existing) existing.remove();
    const toast = document.createElement("div");
    toast.id = "wc-toast";
    toast.className = "toast toast-" + (type || "info");
    toast.textContent = msg;
    document.body.appendChild(toast);
    setTimeout(() => toast.classList.add("show"), 10);
    setTimeout(() => {
      toast.classList.remove("show");
      setTimeout(() => toast.remove(), 300);
    }, 3000);
  }
};

window.UI = UI;
