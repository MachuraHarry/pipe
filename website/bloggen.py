#!/usr/bin/env python3
"""Generate static per-post pages, sitemap.xml and feed.xml for the Pipe blog.

Run from repo root:  python3 website/bloggen.py
Writes:
  website/blog/<id>.html   static, crawler-friendly post page (DE/EN toggle)
  website/sitemap.xml      all post URLs + main pages
  website/feed.xml         RSS 2.0 with full content:encoded
"""

import html
import json
import math
import re
import sys
from datetime import datetime
from email.utils import formatdate
from pathlib import Path

ROOT = Path(__file__).resolve().parent          # website/
BLOG_DIR = ROOT / "blog"
BASE = "https://pipe-lang.com"

INDEX = BLOG_DIR / "index.json"

CSS = """
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#07070c;--bg2:#0d0d16;--bg3:#141420;--card:#11111c;
  --fg:#e8e8f0;--fg2:#9090a8;--fg3:#606078;
  --accent:#a855f7;--accent2:#7c3aed;--green:#3ce096;--orange:#c084fc;--red:#fc5c7c;
  --border:#1e1e2e;--radius:10px;
  --mono:'JetBrains Mono','Fira Code',monospace;--sans:system-ui,-apple-system,sans-serif;
}
html{scroll-behavior:smooth}
body{background:var(--bg);color:var(--fg);font-family:var(--sans);line-height:1.6;padding-top:64px;min-height:100vh;display:flex;flex-direction:column}
.container{max-width:820px;margin:0 auto;padding:0 20px}

nav{position:fixed;top:0;z-index:100;width:100%;background:rgba(7,7,12,0.85);backdrop-filter:blur(12px);border-bottom:1px solid var(--border);padding:10px 0}
nav .container{display:flex;justify-content:space-between;align-items:center}
nav .logo{font-weight:800;font-size:1.2rem;color:var(--accent);text-decoration:none;letter-spacing:-0.5px;display:flex;align-items:center}
nav .logo img{height:28px;margin-right:6px}
nav .logo span{color:var(--fg)}
nav .nl{display:flex;align-items:center;gap:14px}
nav a.nl-a{color:var(--fg2);text-decoration:none;font-size:0.8rem;transition:color .2s;white-space:nowrap}
nav a.nl-a:hover{color:var(--fg)}
.hamburger{display:none;background:none;border:none;color:var(--fg);font-size:1.4rem;cursor:pointer;padding:4px 8px}
.lang-switch{display:flex;background:var(--bg3);border:1px solid var(--border);border-radius:20px;overflow:hidden}
.lang-switch button{background:none;border:none;color:var(--fg2);padding:4px 10px;font-size:0.72rem;cursor:pointer;font-weight:600;transition:all .2s}
.lang-switch button.active{background:var(--accent);color:#fff}

main{flex:1;padding:48px 0}
.pdetail-back{display:inline-flex;align-items:center;gap:4px;margin-bottom:28px;color:var(--accent);font-size:0.82rem;text-decoration:none;transition:color .2s}
.pdetail-back:hover{color:var(--fg);text-decoration:none}
.pdetail-meta{display:flex;align-items:center;gap:12px;margin-bottom:8px;font-size:0.72rem;color:var(--accent);font-family:var(--mono);flex-wrap:wrap}
.pdetail-meta .pdetail-tag{background:var(--bg3);padding:2px 8px;border-radius:10px;font-size:0.68rem;color:var(--fg2);font-family:var(--sans)}
.pdetail-actions{display:flex;gap:8px;margin:28px 0 0}
.pdetail-actions button{background:var(--bg3);border:1px solid var(--border);color:var(--fg2);padding:6px 14px;border-radius:14px;font-size:0.75rem;cursor:pointer;font-family:var(--sans);transition:all .2s}
.pdetail-actions button:hover{border-color:var(--accent);color:var(--fg);background:var(--bg2)}

.md h1{font-size:1.8rem;font-weight:800;margin:0 0 24px;letter-spacing:-0.5px}
.md h2{font-size:1.3rem;font-weight:700;margin:36px 0 12px;padding-bottom:6px;border-bottom:1px solid var(--border)}
.md h3{font-size:1.05rem;font-weight:700;margin:24px 0 8px;color:var(--accent2)}
.md p{margin-bottom:12px;font-size:0.9rem;line-height:1.75}
.md ul,.md ol{margin-bottom:12px;padding-left:22px}
.md li{font-size:0.9rem;margin-bottom:4px;line-height:1.6}
.md a{color:var(--accent);text-decoration:none}
.md a:hover{text-decoration:underline}
.md strong{font-weight:700;color:var(--orange)}
.md code{background:var(--bg3);padding:2px 6px;border-radius:4px;font-family:var(--mono);font-size:0.83em;color:var(--accent2)}
.md pre{background:var(--bg2);border:1px solid var(--border);border-radius:var(--radius);padding:16px 20px;overflow-x:auto;margin-bottom:14px}
.md pre code{background:none;padding:0;font-size:0.82rem;color:var(--fg)}
.token.comment,.token.prolog,.token.doctype,.token.cdata{color:#666;font-style:italic}
.token.punctuation{color:#9898a8}
.token.boolean,.token.number,.token.constant,.token.symbol{color:#c084fc}
.token.string,.token.char,.token.builtin{color:#3ce096}
.token.operator,.token.entity,.token.url{color:#fc8c3c}
.token.keyword{color:#a78bfa;font-weight:600}
.token.function,.token.class-name{color:#5ce0fc}
.token.variable{color:#e0e0e8}
.md table{border-collapse:collapse;width:100%;margin-bottom:16px;font-size:0.85rem}
.md th{background:var(--bg2);border:1px solid var(--border);padding:8px 12px;text-align:left;font-weight:600}
.md td{border:1px solid var(--border);padding:8px 12px}
.md blockquote{border-left:3px solid var(--accent);padding:8px 16px;margin-bottom:12px;background:var(--bg2);border-radius:0 var(--radius) var(--radius) 0;color:var(--fg2);font-size:0.88rem}
.md hr{border:none;border-top:1px solid var(--border);margin:32px 0}
.md img{max-width:100%;border-radius:var(--radius)}

footer{background:var(--bg2);border-top:1px solid var(--border);padding:28px 0;text-align:center;margin-top:auto}
footer .ft{color:var(--fg3);font-size:0.75rem}
footer .ft a{color:var(--accent);text-decoration:none}

.toast{position:fixed;bottom:24px;left:50%;transform:translateX(-50%);background:var(--green);color:var(--bg);padding:10px 20px;border-radius:20px;font-size:0.82rem;font-weight:600;z-index:200;opacity:0;transition:opacity .3s;pointer-events:none}
.toast.show{opacity:1}

@media(max-width:768px){
  .hamburger{display:block}
  nav .nl{display:none;position:absolute;top:100%;left:0;right:0;background:rgba(7,7,12,0.98);border-bottom:1px solid var(--border);flex-direction:column;padding:16px 20px;gap:12px;backdrop-filter:blur(12px)}
  nav .nl.open{display:flex}
  .md h1{font-size:1.5rem}
}
"""

# ---- markdown (mirror of blog.html renderMarkdown + inline) ----

def inline(t):
    t = re.sub(r'\*\*\*(.+?)\*\*\*', r'<strong><em>\1</em></strong>', t)
    t = re.sub(r'\*\*(.+?)\*\*', r'<strong>\1</strong>', t)
    t = re.sub(r'`([^`]+)`', r'<code>\1</code>', t)
    t = re.sub(r'~~(.+?)~~', r'<del>\1</del>', t)
    t = re.sub(r'\[([^\]]+)\]\(([^)]+)\)', r'<a href="\2">\1</a>', t)
    return t


BLOCK_RE = re.compile(r'^(#{1,6}\s|^> |^```|^[\-\*\+]\s|^\d+\.\s|^---|^\|)')
UL_RE = re.compile(r'^[\-\*\+]\s+')
OL_RE = re.compile(r'^\d+\.\s+')
LANG_RE = re.compile(r'\[lang:([a-z]{2})\]([\s\S]*?)\[/lang\]')
SEP_RE = re.compile(r'^\|[\s\-:|]+\|$')


def render_markdown(t, lang):
    t = t.replace('<', '&lt;').replace('>', '&gt;')
    t = LANG_RE.sub(lambda m: m.group(2) if m.group(1) == lang else '', t)
    lines = t.split('\n')
    o = []
    i = 0
    n = len(lines)
    while i < n:
        ln = lines[i]
        if ln.startswith('```'):
            fence_lang = ln[3:].strip()
            i += 1
            cl = []
            while i < n and not lines[i].startswith('```'):
                cl.append(lines[i])
                i += 1
            i += 1
            cls = ' class="language-%s"' % fence_lang if fence_lang else ''
            o.append('<pre><code%s>' % cls + '\n'.join(cl) + '</code></pre>')
            continue
        m = re.match(r'^(#{1,6})\s+(.+)', ln)
        if m:
            lvl = len(m.group(1))
            o.append('<h%d>%s</h%d>' % (lvl, inline(m.group(2)), lvl))
            i += 1
            continue
        if re.match(r'^---', ln):
            o.append('<hr>')
            i += 1
            continue
        if ln.startswith('> '):
            q = []
            while i < n and lines[i].startswith('> '):
                q.append(lines[i][2:])
                i += 1
            o.append('<blockquote>' + inline('\n'.join(q)) + '</blockquote>')
            continue
        if '|' in ln and i + 1 < n and SEP_RE.match(lines[i + 1]):
            hr = ln
            i += 2
            br = []
            while i < n and '|' in lines[i].strip():
                br.append(lines[i])
                i += 1
            hc = [c for c in hr.split('|') if c.strip()]
            hs = ''.join('<th>' + inline(c.strip()) + '</th>' for c in hc)
            tb = ''
            for row in br:
                bc = [c for c in row.split('|') if c.strip()]
                tb += '<tr>' + ''.join('<td>' + inline(c.strip()) + '</td>' for c in bc) + '</tr>'
            o.append('<table><thead><tr>' + hs + '</tr></thead><tbody>' + tb + '</tbody></table>')
            continue
        if UL_RE.match(ln):
            ul = []
            while i < n and UL_RE.match(lines[i]):
                ul.append(UL_RE.sub('', lines[i]))
                i += 1
            o.append('<ul>' + ''.join('<li>' + inline(x) + '</li>' for x in ul) + '</ul>')
            continue
        if OL_RE.match(ln):
            ol = []
            while i < n and OL_RE.match(lines[i]):
                ol.append(OL_RE.sub('', lines[i]))
                i += 1
            o.append('<ol>' + ''.join('<li>' + inline(x) + '</li>' for x in ol) + '</ol>')
            continue
        if ln.strip() == '':
            i += 1
            continue
        pl = []
        while i < n and lines[i].strip() != '' and not BLOCK_RE.match(lines[i]):
            pl.append(lines[i])
            i += 1
        if pl:
            o.append('<p>' + inline(' '.join(pl)) + '</p>')
    return '\n'.join(o)


def read_time(words):
    return max(1, math.ceil(words / 238))


def esc(s):
    return html.escape(s, quote=True)


def esc_json_ld(s):
    return s.replace('</', '<\\/')


# ---- page generation ----

NAV = """<nav><div class="container">
    <a href="../index.html" class="logo"><img src="../logo.svg" alt="Pipe">Pipe<span>.lang</span></a>
    <div class="lang-switch">
      <button id="btn-en" class="active" onclick="setLang('en')">EN</button>
      <button id="btn-de" onclick="setLang('de')">DE</button>
    </div>
    <button class="hamburger" onclick="document.getElementById('nm').classList.toggle('open')">☰</button>
    <div class="nl" id="nm">
      <a class="nl-a" href="../index.html">Home</a>
      <a class="nl-a" href="../docs.html">Docs</a>
      <a class="nl-a" href="../examples.html">Examples</a>
      <a class="nl-a" href="../install.html">Install</a>
      <a class="nl-a" href="../benchmarks.html">Benchmarks</a>
      <a class="nl-a" href="../playground.html">Playground</a>
      <a class="nl-a" href="../blog.html">Blog</a>
      <a class="nl-a" href="https://github.com/MachuraHarry/pipe">GitHub</a>
    </div>
  </div></nav>"""

PAGE_SCRIPT = """<script>
var POST_TITLE={POST_TITLE};
(function(){
  var s=localStorage.getItem('pipe-lang');
  if(!s)s=navigator.language.startsWith('de')?'de':'en';
  function apply(l){
    document.documentElement.lang=l;
    var a=document.querySelectorAll('[data-lang]');
    for(var i=0;i<a.length;i++)a[i].style.display=(a[i].getAttribute('data-lang')===l)?'':'none';
    document.getElementById('btn-en').classList.toggle('active',l==='en');
    document.getElementById('btn-de').classList.toggle('active',l==='de');
    if(POST_TITLE[l])document.title=POST_TITLE[l]+' — Pipe Blog';
  }
  window.setLang=function(l){localStorage.setItem('pipe-lang',l);apply(l)};
  apply(s);
})();
function copyLink(){
  navigator.clipboard.writeText(location.origin+location.pathname).then(function(){
    var t=document.getElementById('toast');t.classList.add('show');setTimeout(function(){t.classList.remove('show')},2000);
  });
}
</script>"""


def build_page(p, md_text):
    pid = p["id"]
    post_url = BASE + "/blog/" + pid + ".html"
    en_title = p["title"]["en"]
    de_title = p["title"]["de"]
    en_excerpt = p["excerpt"]["en"]
    de_excerpt = p["excerpt"]["de"]
    date = p["date"]
    tag = p.get("tag", "")

    en_html = render_markdown(md_text, "en")
    de_html = render_markdown(md_text, "de")

    rt_en = read_time(len(en_excerpt.split()))
    rt_de = read_time(len(de_excerpt.split()))

    schema = {
        "@context": "https://schema.org",
        "@type": "BlogPosting",
        "headline": en_title,
        "description": en_excerpt,
        "datePublished": date,
        "dateModified": date,
        "author": {"@type": "Organization", "name": "Pipe (SPR)"},
        "publisher": {
            "@type": "Organization",
            "name": "Pipe (SPR)",
            "logo": {"@type": "ImageObject", "url": BASE + "/logo.svg"},
        },
        "image": BASE + "/og-image.png",
        "mainEntityOfPage": post_url,
        "url": post_url,
        "inLanguage": "en",
    }
    if tag:
        schema["keywords"] = tag

    lang_sw = ("<span data-lang=\"en\">" + str(rt_en) + " min read</span>"
               "<span data-lang=\"de\">" + str(rt_de) + " Min. Lesezeit</span>")
    back_sw = ('<span data-lang="en">← All posts</span><span data-lang="de">← Alle Beiträge</span>')

    page = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<link rel="icon" type="image/svg+xml" href="../logo.svg">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="description" content="{desc}">
<meta property="og:title" content="{otitle}">
<meta property="og:description" content="{odesc}">
<meta property="og:type" content="article">
<meta property="og:image" content="{base}/og-image.png">
<meta property="og:url" content="{url}">
<meta property="og:site_name" content="Pipe (SPR)">
<meta name="twitter:card" content="summary_large_image">
<link rel="canonical" href="{url}">
<link rel="alternate" hreflang="en" href="{url}">
<link rel="alternate" hreflang="de" href="{url}">
<link rel="alternate" hreflang="x-default" href="{url}">
<link rel="alternate" type="application/rss+xml" title="Pipe Blog RSS" href="../feed.xml">
<script type="application/ld+json">{schema}</script>
<title>{ttitle} — Pipe Blog</title>
<style>{css}</style>
</head>
<body>
{nav}
<main><div class="container">
  <a class="pdetail-back" href="../blog.html">{back}</a>
  <div class="pdetail-meta"><span data-lang="en">{en_date}</span><span data-lang="de">{de_date}</span>{tag}<span>{rt}</span></div>
  <div class="md" data-lang="en">{en_html}</div>
  <div class="md" data-lang="de">{de_html}</div>
  <div class="pdetail-actions"><button onclick="copyLink()">🔗 <span data-lang="en">Copy link</span><span data-lang="de">Link kopieren</span></button></div>
</div></main>
<div class="toast" id="toast">Link copied ✓</div>
<footer><div class="ft">© 2026 Pipe (SPR) · MIT License · <a href="../feed.xml">RSS</a></div></footer>
<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/prism.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-json.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-bash.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-go.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-markup.min.js"></script>
<script src="../prism-pipe.js"></script>
{script}
</body>
</html>""".format(
        desc=esc(en_excerpt),
        otitle=esc(en_title),
        odesc=esc(en_excerpt),
        base=BASE,
        url=post_url,
        ttitle=esc(en_title),
        css=CSS,
        nav=NAV,
        back=back_sw,
        en_date=fmt_date(date),
        de_date=fmt_date_de(date),
        tag='<span class="pdetail-tag">' + esc(tag) + '</span>' if tag else '',
        rt=lang_sw,
        en_html=en_html,
        de_html=de_html,
        schema=esc_json_ld(json.dumps(schema, ensure_ascii=False)),
        script=PAGE_SCRIPT.replace('{POST_TITLE}', '{"en":' + json.dumps(en_title, ensure_ascii=False) + ',"de":' + json.dumps(de_title, ensure_ascii=False) + '}'),
    )
    return page


MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
MONTHS_DE = ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']


def fmt_date(d):
    y, m, day = d.split('-')
    return '%s %d, %s' % (MONTHS[int(m) - 1], int(day), y)


def fmt_date_de(d):
    y, m, day = d.split('-')
    return '%d. %s %s' % (int(day), MONTHS_DE[int(m) - 1], y)


# ---- sitemap ----

def build_sitemap(posts):
    main = [
        ("https://pipe-lang.com/", "1.0", "weekly"),
        ("https://pipe-lang.com/playground.html", "0.9", "weekly"),
        ("https://pipe-lang.com/install.html", "0.9", "monthly"),
        ("https://pipe-lang.com/docs.html", "0.8", "weekly"),
        ("https://pipe-lang.com/examples.html", "0.8", "weekly"),
        ("https://pipe-lang.com/benchmarks.html", "0.7", "monthly"),
        ("https://pipe-lang.com/blog.html", "0.7", "weekly"),
    ]
    urls = "".join(
        '  <url><loc>%s</loc><priority>%s</priority><changefreq>%s</changefreq></url>\n' % (loc, prio, freq)
        for loc, prio, freq in main
    )
    for p in posts:
        urls += ('  <url><loc>%s/blog/%s.html</loc><lastmod>%s</lastmod>'
                 '<priority>0.7</priority><changefreq>monthly</changefreq></url>\n'
                 % (BASE, p["id"], p["date"]))
    return '<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n' + urls + '</urlset>\n'


# ---- rss ----

def build_rss(posts):
    now = formatdate(usegmt=True)
    items = ""
    for p in posts:
        post_url = BASE + "/blog/" + p["id"] + ".html"
        md_text = (BLOG_DIR / p["file"]).read_text(encoding="utf-8")
        content = render_markdown(md_text, "en")
        try:
            pubdate = formatdate(datetime.fromisoformat(p["date"]).timestamp(), usegmt=True)
        except Exception:
            pubdate = now
        items += """  <item>
    <title>{title}</title>
    <link>{url}</link>
    <guid isPermaLink="true">{url}</guid>
    <pubDate>{pubdate}</pubDate>
    <description>{desc}</description>
    <content:encoded><![CDATA[{content}]]></content:encoded>
    {tag}
  </item>
""".format(
            title=esc(p["title"]["en"]),
            url=post_url,
            pubdate=pubdate,
            desc=esc(p["excerpt"]["en"]),
            content=content,
            tag='<category>%s</category>' % esc(p.get("tag", "")) if p.get("tag") else '',
        )
    return """<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:content="http://purl.org/rss/1.0/modules/content/">
<channel>
  <title>Pipe Blog</title>
  <link>https://pipe-lang.com/blog.html</link>
  <description>Updates, tutorials, and deep dives into the Semantic Pipeline Runtime.</description>
  <language>en</language>
  <lastBuildDate>{now}</lastBuildDate>
  <atom:link href="https://pipe-lang.com/feed.xml" rel="self" type="application/rss+xml"/>
{items}</channel>
</rss>
""".format(now=now, items=items)


def main():
    posts = json.loads(INDEX.read_text(encoding="utf-8"))
    if not posts:
        print("no posts found in " + str(INDEX))
        sys.exit(1)

    for p in posts:
        md_path = BLOG_DIR / p["file"]
        md_text = md_path.read_text(encoding="utf-8")
        page = build_page(p, md_text)
        out = BLOG_DIR / (p["id"] + ".html")
        out.write_text(page, encoding="utf-8")
        print("wrote %s (%d bytes)" % (out.relative_to(ROOT.parent), len(page)))

    (ROOT / "sitemap.xml").write_text(build_sitemap(posts), encoding="utf-8")
    print("wrote sitemap.xml (%d posts)" % len(posts))

    rss = build_rss(posts)
    (ROOT / "feed.xml").write_text(rss, encoding="utf-8")
    print("wrote feed.xml (%d items)" % len(posts))


if __name__ == "__main__":
    main()
