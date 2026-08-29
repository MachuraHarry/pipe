// gen_audio.js — erzeugt deterministische Ohrwurm-Töne als WAV-Dateien.
// Aufruf: node tools/gen_audio.js  (im worldcraft/web-Verzeichnis)
// Ersetzt die bisherigen stillen .mp3-Platzhalter durch echte, kleine SFX.
// Determinismus: keine Zufallswerte, feste Frequenzen/Zeiten → reproduzierbar.

const fs = require("node:fs");
const path = require("node:path");

const SR = 22050;
const OUT = path.join(__dirname, "..", "audio");

function sine(freq, t) { return Math.sin(2 * Math.PI * freq * t); }

// Hüllkurve: kurzes Attack, exponentieller Decay
function env(t, dur, decay = 6) {
  const attack = Math.min(0.004, dur * 0.1);
  const a = t < attack ? t / attack : 1;
  return a * Math.exp(-decay * (t / dur));
}

function render(dur, gen) {
  const n = Math.floor(SR * dur);
  const pcm = Buffer.alloc(n * 2);
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const s = Math.max(-1, Math.min(1, gen(t, t / dur)));
    pcm.writeInt16LE(Math.round(s * 32767), i * 2);
  }
  return pcm;
}

function writeWav(name, pcm) {
  const header = Buffer.alloc(44);
  header.write("RIFF", 0);
  header.writeUInt32LE(36 + pcm.length, 4);
  header.write("WAVE", 8);
  header.write("fmt ", 12);
  header.writeUInt32LE(16, 16);      // fmt chunk size
  header.writeUInt16LE(1, 20);       // PCM
  header.writeUInt16LE(1, 22);       // mono
  header.writeUInt32LE(SR, 24);      // sample rate
  header.writeUInt32LE(SR * 2, 28);  // byte rate
  header.writeUInt16LE(2, 32);       // block align
  header.writeUInt16LE(16, 34);      // bits
  header.write("data", 36);
  header.writeUInt32LE(pcm.length, 40);
  fs.writeFileSync(path.join(OUT, name), Buffer.concat([header, pcm]));
}

const sounds = {
  // Schritt: tiefer, weicher "Tap" mit schnellem Abklingen
  "step.wav": render(0.08, (t) => 0.5 * sine(150 + 60 * (1 - t / 0.08), t) * env(t, 0.08, 14)),
  // Treffer: harter Rechteck-Klang mit steilem Abklingen
  "hit.wav": render(0.09, (t) => 0.5 * Math.sign(sine(340 - 180 * (t / 0.09), t)) * env(t, 0.09, 10)),
  // Aufsammeln: zwei steigende Sinus-Blipps ("Coin")
  "pickup.wav": render(0.14, (t) => {
    const a = sine(660, t) * env(t, 0.06, 12);
    const b = t >= 0.06 ? sine(990, t - 0.06) * env(t - 0.06, 0.08, 10) : 0;
    return 0.35 * (a + b);
  }),
  // Dialog: weicher, runder "Plink" (Quinte absteigend)
  "dialog.wav": render(0.13, (t) => 0.4 * sine(440 + 110 * (1 - t / 0.13), t) * env(t, 0.13, 8)),
  // Sieg: C-E-G-Arpeggio, jeder Ton leicht verklingend
  "victory.wav": render(0.6, (t) => {
    const notes = [523.25, 659.25, 783.99];
    let s = 0;
    for (let k = 0; k < notes.length; k++) {
      const start = k * 0.2;
      if (t >= start) {
        const tt = t - start;
        s += 0.35 * sine(notes[k], tt) * env(tt, 0.22, 5);
      }
    }
    return s;
  }),
};

fs.mkdirSync(OUT, { recursive: true });
for (const [file, pcm] of Object.entries(sounds)) {
  writeWav(file, pcm);
  console.log("geschrieben: audio/" + file);
}