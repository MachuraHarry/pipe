class WorldcraftAPI {
  constructor() {
    this.baseUrl = "";
    this.sessionId = localStorage.getItem("wc_session_id") || null;
    this.secret = localStorage.getItem("wc_secret") || null;
    this.adminToken = localStorage.getItem("wc_admin_token") || null;
  }

  get hasSession() {
    return !!(this.sessionId && this.secret);
  }

  saveSession(id, secret) {
    this.sessionId = id;
    this.secret = secret;
    localStorage.setItem("wc_session_id", id);
    localStorage.setItem("wc_secret", secret);
  }

  clearSession() {
    this.sessionId = null;
    this.secret = null;
    localStorage.removeItem("wc_session_id");
    localStorage.removeItem("wc_secret");
  }

  setAdminToken(token) {
    this.adminToken = token;
    if (token) {
      localStorage.setItem("wc_admin_token", token);
    } else {
      localStorage.removeItem("wc_admin_token");
    }
  }

  _authHeaders() {
    const h = { "Content-Type": "application/json" };
    if (this.secret) h["Authorization"] = "Bearer " + this.secret;
    return h;
  }

  _adminHeaders() {
    const h = { "Content-Type": "application/json" };
    if (this.adminToken) h["Authorization"] = "Bearer " + this.adminToken;
    return h;
  }

    async _fetch(method, path, body, headers) {
    const opts = { method, headers: headers || {} };
    if (body !== undefined && body !== null) {
      opts.body = typeof body === "string" ? body : JSON.stringify(body);
    }
    // Timeout: ein hängender Request blockiert nicht ewig die UI.
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 30000);
    opts.signal = controller.signal;
    let res, text, data;
    try {
      res = await fetch(this.baseUrl + path, opts);
    } catch (netErr) {
      throw Object.assign(new Error(t("connection_lost")), { code: "network_error", data: { error: { code: "network_error", message: t("connection_lost") } } });
    } finally {
      clearTimeout(timer);
    }
    text = await res.text();
    try {
      data = JSON.parse(text);
    } catch {
      data = { error: { code: "parse_error", message: text } };
    }
    if (!res.ok || (data && data.error && res.status >= 400)) {
      const code = (data.error && (data.error.code || data.error.error_code)) || res.status;
      const err = new Error((data.error && data.error.message) || t("error_generic"));
      err.code = code;
      err.status = res.status;
      err.data = data;
      throw err;
    }
    return data;
  }

  async listWorlds() {
    return this._fetch("GET", "/api/v1/worlds", null, this._adminHeaders());
  }

  async createSession(config) {
    return this._fetch("POST", "/api/v1/sessions", config, this._adminHeaders());
  }

  async resumeSession() {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("POST", "/api/v1/sessions/" + this.sessionId + "/resume", null, this._authHeaders());
  }

  async getState() {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("GET", "/api/v1/sessions/" + this.sessionId + "/state", null, this._authHeaders());
  }

  async getBootStatus() {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("GET", "/api/v1/sessions/" + this.sessionId + "/boot", null, this._authHeaders());
  }

  async getCost() {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("GET", "/api/v1/sessions/" + this.sessionId + "/cost", null, this._authHeaders());
  }

  async sendCommand(input) {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("POST", "/api/v1/sessions/" + this.sessionId + "/command", { input }, this._authHeaders());
  }

  async expandWorld(wunsch) {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("POST", "/api/v1/sessions/" + this.sessionId + "/expand", { wunsch }, this._authHeaders());
  }

  async newAdventure() {
    if (!this.hasSession) throw new Error("No session");
    return this._fetch("POST", "/api/v1/sessions/" + this.sessionId + "/new-adventure", null, this._authHeaders());
  }

  async pollJob(jobId, onProgress, signal) {
    if (!this.hasSession) throw new Error("No session");
    const url = "/api/v1/sessions/" + this.sessionId + "/jobs/" + jobId;
    const start = Date.now();
    const timeout = this._pollTimeout || 300000;
    const interval = this._pollInterval || 500;

    return new Promise((resolve, reject) => {
      const cleanup = signal && typeof signal.removeEventListener === "function"
        ? () => signal.removeEventListener("abort", onAbort)
        : () => {};
      const onAbort = () => {
        cleanup();
        const err = new Error(t("job_cancelled"));
        err.name = "AbortError";
        reject(err);
      };
      const poll = async () => {
        try {
          if (Date.now() - start > timeout) {
            cleanup();
            reject(new Error("Timeout"));
            return;
          }
          if (signal && signal.aborted) {
            cleanup();
            onAbort();
            return;
          }
          const data = await this._fetch("GET", url, null, this._authHeaders());
          if (data.status === "done") {
            cleanup();
            if (data.error) {
              reject(new Error(data.error));
            } else {
              resolve(data.result);
            }
          } else if (data.status === "error") {
            cleanup();
            reject(new Error(data.error || "Job failed"));
          } else {
            if (onProgress) onProgress(data);
            setTimeout(() => {
              if (signal && signal.aborted) {
                cleanup();
                onAbort();
              } else {
                poll();
              }
            }, interval);
          }
        } catch (e) {
          cleanup();
          reject(e);
        }
      };
      if (signal && typeof signal.addEventListener === "function") {
        signal.addEventListener("abort", onAbort);
      }
      poll();
    });
  }

  async deleteSession() {
    if (!this.hasSession) throw new Error("No session");
    try {
      await this._fetch("DELETE", "/api/v1/sessions/" + this.sessionId, null, this._authHeaders());
    } finally {
      this.clearSession();
    }
  }
}
