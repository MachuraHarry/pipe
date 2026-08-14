# How to Write a Blog Post

No CMS, no database. Write a Markdown file, add an entry to `index.json`, push.

## Quick Start

1. Write your post as a Markdown file in this directory.
2. Add an entry to `index.json`.
3. Push to `master`.

That's it. The deploy runs `website/bloggen.py` in CI, which generates a static,
crawler-friendly page for every post (`blog/<id>.html`), plus `sitemap.xml` and
`feed.xml`.

## Step 1: Write the Post

Create a file like `website/blog/my-topic.md`. Use standard Markdown:

```markdown
# My Post Title

Some introductory paragraph.

## Section One

Content goes here. You can use **bold**, *italic*, `code`, tables, code blocks, etc.

## Section Two

More content…
```

**No frontmatter needed.** The title from step 2 goes into `index.json`, not the markdown file. The file's `<h1>` is shown as the post's heading when rendered.

## Step 2: Add to `index.json`

Add an object to the array in `website/blog/index.json`:

```json
{
  "id": "my-topic",
  "file": "my-topic.md",
  "date": "2026-08-15",
  "tag": "tutorial",
  "title": {
    "en": "My Post Title",
    "de": "Mein Beitragstitel"
  },
  "excerpt": {
    "en": "A short description shown in the post listing. Keep it to 2-3 sentences.",
    "de": "Eine kurze Beschreibung, die in der Beitragsliste angezeigt wird. 2-3 Sätze."
  }
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | URL-safe identifier. Used as the static page filename `blog/<id>.html`. No spaces. |
| `file` | string | The `.md` filename in this directory. |
| `date` | string | ISO date (`YYYY-MM-DD`). Shown in the listing and detail view. |
| `tag` | string? | Optional. Shown as a badge. Use `tutorial`, `release`, `security`, `insight`, etc. Omit the field entirely if you don't want a tag. |
| `title.en` | string | Title shown when the site is in English. |
| `title.de` | string | Title shown when the site is in German. |
| `excerpt.en` | string | Short summary (3-4 lines max). Shown in the post listing. |
| `excerpt.de` | string | Same, in German. |

**Only one language** (`en` or `de`)? Fill the other with the same text. The language switch still works — both show the same content.

**Internal links between posts:** use relative links to the generated pages, e.g. `[Part 2](sandbox-audit-2.html)` — not `?post=…`.

## Step 3: Push

```bash
git add website/blog/my-topic.md website/blog/index.json
git commit -m "blog: My new post"
git push
```

The GitHub Pages deploy runs automatically. It generates the static post page (`blog/<id>.html`), `sitemap.xml` and `feed.xml` via `website/bloggen.py`. Your post is live within ~60 seconds.

### Preview locally

```bash
python3 website/bloggen.py   # writes website/blog/<id>.html, sitemap.xml, feed.xml
# open website/blog/<id>.html in a browser
```

## Markdown Features

The built-in renderer supports:

| Feature | Syntax |
|---------|--------|
| Headings | `#`, `##`, `###`, `####` |
| Bold / Italic | `**bold**`, `*italic*`, `***both***` |
| Inline code | `` `code` `` |
| Code blocks | ` ``` ` fences |
| Links | `[text](url)` |
| Images | `![alt](url)` |
| Lists | `- item` or `1. item` |
| Tables | Pipe tables with `---` separator |
| Blockquotes | `> quote` |
| Horizontal rules | `---` |
| Strikethrough | `~~text~~` |
