class Minimap {
  constructor(canvas, size) {
    this.canvas = canvas;
    this.ctx = canvas.getContext("2d");
    this.dpr = window.devicePixelRatio || 1;
    this.displaySize = size || 200;
    this.canvas.width = this.displaySize * this.dpr;
    this.canvas.height = this.displaySize * this.dpr;
    this.canvas.style.width = this.displaySize + "px";
    this.canvas.style.height = this.displaySize + "px";
    this.ctx.scale(this.dpr, this.dpr);
    this.rooms = {};
    this.roomOrder = [];
    this.currentRoom = null;
    this.padding = 20;
    this.roomW = 16;
    this.roomH = 12;
    this._hoveredRoom = null;
    this._setupHover();
  }

  _setupHover() {
    const tooltip = document.getElementById("minimap-tooltip");
    if (!tooltip) return;
    this.canvas.addEventListener("mousemove", (e) => {
      const rect = this.canvas.getBoundingClientRect();
      const mx = (e.clientX - rect.left) * (this.displaySize / rect.width);
      const my = (e.clientY - rect.top) * (this.displaySize / rect.height);
      const room = this._roomAt(mx, my);
      if (room) {
        tooltip.textContent = room;
        tooltip.style.display = "block";
        tooltip.style.left = (e.clientX - this.canvas.closest(".minimap-wrapper").getBoundingClientRect().left + 8) + "px";
        tooltip.style.top = (e.clientY - this.canvas.closest(".minimap-wrapper").getBoundingClientRect().top - 24) + "px";
        this._hoveredRoom = room;
      } else {
        tooltip.style.display = "none";
        this._hoveredRoom = null;
      }
    });
    this.canvas.addEventListener("mouseleave", () => {
      tooltip.style.display = "none";
      this._hoveredRoom = null;
    });
  }

  _roomAt(mx, my) {
    for (const id of this.roomOrder) {
      const r = this.rooms[id];
      if (mx >= r.x - this.roomW / 2 && mx <= r.x + this.roomW / 2 &&
          my >= r.y - this.roomH / 2 && my <= r.y + this.roomH / 2) {
        return id;
      }
    }
    return null;
  }

  _dirOffset(dir) {
    const s = 36;
    switch (dir) {
      case "norden": return { dx: 0, dy: -s };
      case "sueden": case "süden": return { dx: 0, dy: s };
      case "osten": return { dx: s, dy: 0 };
      case "westen": return { dx: -s, dy: 0 };
      default: return { dx: 0, dy: 0 };
    }
  }

  _isOccupied(x, y) {
    for (const id of this.roomOrder) {
      const r = this.rooms[id];
      if (Math.abs(r.x - x) < this.roomW + 4 && Math.abs(r.y - y) < this.roomH + 4) {
        return true;
      }
    }
    return false;
  }

  // Deterministische Platzierung: fester Kandidaten-Lauf (kein Zufall), damit
  // das Layout über Neu-Renderings hinweg stabil bleibt und nicht „springt".
  _findFreePos(x, y) {
    const gx = this.roomW + 4;
    const gy = this.roomH + 4;
    const attempts = [
      [0, 0],
      [gx, 0], [-gx, 0],
      [0, gy], [0, -gy],
      [gx, gy], [-gx, gy], [gx, -gy], [-gx, -gy],
      [2 * gx, 0], [-2 * gx, 0],
      [0, 2 * gy], [0, -2 * gy],
      [2 * gx, 2 * gy], [-2 * gx, 2 * gy], [2 * gx, -2 * gy], [-2 * gx, -2 * gy]
    ];
    for (const [dx, dy] of attempts) {
      const cx = x + dx;
      const cy = y + dy;
      if (!this._isOccupied(cx, cy)) return { x: cx, y: cy };
    }
    return { x, y };
  }

  addRoom(id, exits, fromRoom, direction) {
    if (this.rooms[id]) {
      if (exits && exits.length) {
        this.rooms[id].exits = exits;
        this.render();
      }
      return;
    }
    let x, y;
    if (!fromRoom || !this.rooms[fromRoom]) {
      x = this.displaySize / 2;
      y = this.displaySize / 2;
    } else {
      const from = this.rooms[fromRoom];
      const off = this._dirOffset(direction);
      x = from.x + off.dx;
      y = from.y + off.dy;
      const clamped = this._findFreePos(x, y);
      x = clamped.x;
      y = clamped.y;
    }
    this.rooms[id] = { x, y, exits: exits || [] };
    this.roomOrder.push(id);
    this.render();
  }

  setCurrentRoom(id) {
    this.currentRoom = id;
    if (!this.rooms[id]) {
      this.addRoom(id, [], null, null);
    }
    this._centerOnRoom(id);
    this.render();
  }

  _centerOnRoom(id) {
    const r = this.rooms[id];
    if (!r) return;
    const cx = this.displaySize / 2;
    const cy = this.displaySize / 2;
    const dx = cx - r.x;
    const dy = cy - r.y;
    for (const rid of this.roomOrder) {
      this.rooms[rid].x += dx;
      this.rooms[rid].y += dy;
    }
  }

  clear() {
    this.rooms = {};
    this.roomOrder = [];
    this.currentRoom = null;
    this.render();
  }

  render() {
    const ctx = this.ctx;
    const s = this.displaySize;
    ctx.clearRect(0, 0, s, s);

    // Draw connections
    ctx.strokeStyle = "#2e3340";
    ctx.lineWidth = 1;
    for (const id of this.roomOrder) {
      const r = this.rooms[id];
      for (const dir of (r.exits || [])) {
        const off = this._dirOffset(dir);
        const nx = r.x + off.dx;
        const ny = r.y + off.dy;
        // Only draw if target room exists nearby
        for (const tid of this.roomOrder) {
          if (tid !== id) {
            const t = this.rooms[tid];
            if (Math.abs(t.x - nx) < 8 && Math.abs(t.y - ny) < 8) {
              ctx.beginPath();
              ctx.moveTo(r.x, r.y);
              ctx.lineTo(t.x, t.y);
              ctx.stroke();
            }
          }
        }
      }
    }

    // Draw rooms
    for (const id of this.roomOrder) {
      const r = this.rooms[id];
      const isCurrent = id === this.currentRoom;
      const isHovered = id === this._hoveredRoom;

      // Room rectangle
      ctx.fillStyle = isCurrent ? "#5b8def" : (isHovered ? "#22262f" : "#181b23");
      ctx.strokeStyle = isCurrent ? "#5b8def" : (isHovered ? "#5b8def" : "#2e3340");
      ctx.lineWidth = isCurrent ? 2 : 1;
      ctx.beginPath();
      ctx.roundRect(
        r.x - this.roomW / 2,
        r.y - this.roomH / 2,
        this.roomW,
        this.roomH,
        3
      );
      ctx.fill();
      ctx.stroke();

      // Room label (only for current on small maps)
      if (isCurrent) {
        ctx.fillStyle = "#0f1117";
        ctx.font = "bold 7px sans-serif";
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        ctx.fillText(id.length > 6 ? id.slice(0, 5) + "." : id, r.x, r.y);
      }
    }
  }

  resize(newSize) {
    this.displaySize = newSize;
    this.canvas.width = newSize * this.dpr;
    this.canvas.height = newSize * this.dpr;
    this.canvas.style.width = newSize + "px";
    this.canvas.style.height = newSize + "px";
    this.ctx.scale(this.dpr, this.dpr);
    // Re-center on current room
    if (this.currentRoom) this._centerOnRoom(this.currentRoom);
    this.render();
  }
}
