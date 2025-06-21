#!/usr/bin/env python3
import os
import re
from pathlib import Path
from datetime import datetime, timedelta
import argparse

VAULT_ROOT = Path(".")
GUIDES_DIR = VAULT_ROOT / "99-System/03-Guides"
LOGS_DIR = VAULT_ROOT / "99-System/logs"
WEEKLY_DIR = Path("08-Journal/weekly")


EXCLUDED_DIRS = {
    "08-Journal", "09-Archive", "99-System", ".git", ".obsidian", ".trash"
}

# ========== Helpers ==========

def kebab_case(s):
    return re.sub(r'[^a-z0-9]+', '-', s.lower()).strip('-')

def is_valid_dir(path):
    return path.is_dir() and path.name not in EXCLUDED_DIRS and not path.name.startswith(".")

# ========== Commands ==========

def rename_files_to_kebab_case():
    for path in VAULT_ROOT.rglob('*.md'):
        new_name = kebab_case(path.stem) + '.md'
        if new_name != path.name:
            path.rename(path.with_name(new_name))
            print(f"Renamed: {path} → {new_name}")

def generate_indexes():
    GUIDES_DIR.mkdir(parents=True, exist_ok=True)
    for folder in VAULT_ROOT.iterdir():
        if is_valid_dir(folder):
            lines = [f"# 📚 Index of {folder.name}", ""]
            for file in sorted(folder.glob("*.md")):
                title = file.stem.replace('-', ' ').title()
                lines.append(f"- [{title}]({file.name})")
            if lines:
                index_file = GUIDES_DIR / f"{folder.name.lower()}-index.md"
                index_file.write_text("\n".join(lines), encoding="utf-8")
                print(f"Index created: {index_file}")

def add_toc_to_markdown():
    HEADER_PATTERN = re.compile(r'^(#{2,6})\s+(.*)', re.MULTILINE)
    for md_file in VAULT_ROOT.rglob("*.md"):
        content = md_file.read_text(encoding="utf-8")
        if "## 📑 Table of Contents" in content:
            continue
        headers = HEADER_PATTERN.findall(content)
        if headers:
            toc_lines = ["## 📑 Table of Contents", ""]
            for hashes, title in headers:
                anchor = re.sub(r'[^\w\- ]', '', title).strip().lower().replace(' ', '-')
                toc_lines.append(f"{' ' * (len(hashes)-2)*2}- [{title}](#{anchor})")
            new_content = "\n".join(toc_lines) + "\n\n" + content
            md_file.write_text(new_content, encoding="utf-8")
            print(f"TOC added: {md_file}")

def create_weekly_review():
    today = datetime.now()
    week_start = (today - timedelta(days=today.weekday())).strftime('%Y-%m-%d')
    week = today.strftime('%Y-w%U')

    WEEKLY_DIR.mkdir(parents=True, exist_ok=True)
    path = WEEKLY_DIR / f"{week}.md"

    if path.exists():
        print(f"ℹ️ Weekly note already exists: {path}")
        return

    content = f"""# Weekly Review {week}

## ✅ Achievements
- Professional:
- Personal:
- Learning:

## 🚧 Blockers
- 

## 📈 Summary
- Completed professional tasks:
- Completed personal tasks:
- Learning hours:

## 🎯 Next Week Goals
- Professional:
- Personal:
- Learning:

#weekly-review #{week}
"""

    path.write_text(content, encoding="utf-8")
    print(f"✅ Created weekly note: {path}")

def check_broken_links():
    LOGS_DIR.mkdir(parents=True, exist_ok=True)
    all_md_files = list(VAULT_ROOT.rglob("*.md"))
    all_stems = {f.stem.lower(): f for f in all_md_files}

    broken_links = {}

    for file in all_md_files:
        content = file.read_text(encoding="utf-8")
        links = re.findall(r'\[\[([^\]]+)\]\]', content)
        for link in links:
            link_kebab = kebab_case(link)
            found = any(link_kebab == stem for stem in all_stems)
            if not found:
                broken_links.setdefault(str(file), []).append(link)

    date_suffix = datetime.now().strftime("-%Y-%m-%d")
    log_path = LOGS_DIR / f"broken-links{date_suffix}.md"
    if broken_links:
        with open(log_path, "w") as f:
            for file, links in broken_links.items():
                f.write(f"{file}:\n")
                for l in links:
                    f.write(f"  - [[{l}]]\n")
                f.write("\n")
        print(f"❌ Broken links found. Report written to: {log_path}")
    else:
        print("✅ No broken links found.")
    return log_path


# ========== CLI ==========

def main():
    parser = argparse.ArgumentParser(description="Vault maintenance tool")
    parser.add_argument("command", choices=["rename", "index", "toc", "weekly", "links"], help="Command to run")
    args = parser.parse_args()

    if args.command == "rename":
        rename_files_to_kebab_case()
    elif args.command == "index":
        generate_indexes()
    elif args.command == "toc":
        add_toc_to_markdown()
    elif args.command == "weekly":
        create_weekly_review()
    elif args.command == "links":
        check_broken_links()

if __name__ == "__main__":
    main()
