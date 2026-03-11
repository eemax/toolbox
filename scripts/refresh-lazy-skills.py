#!/usr/bin/env python3
"""Refresh lazy-skills index based on directories in lazy-loading/skills/"""

import re
from pathlib import Path

SKILL_INDEX = Path("skills/lazy-skills/SKILL.md")
LAZY_DIR = Path("lazy-loading/skills")


def extract_frontmatter_field(skill_md_path: Path, field: str) -> str | None:
    """Extract a named field from YAML frontmatter in a SKILL.md file."""
    if not skill_md_path.exists():
        return None

    content = skill_md_path.read_text()
    frontmatter_match = re.match(r"^---\n(.*?)\n---", content, re.DOTALL)
    if frontmatter_match:
        match = re.search(
            rf"^{field}:\s*(.+)", frontmatter_match.group(1), re.MULTILINE
        )
        if match:
            return match.group(1).strip()

    return None


def read_existing_names(index_path: Path) -> set[str]:
    if not index_path.exists():
        return set()
    return {
        m.group(1)
        for line in index_path.read_text().split("\n")
        if (m := re.match(r"^- name:\s*(\S+)", line))
    }


def main():
    if not LAZY_DIR.exists():
        print(f"Error: {LAZY_DIR} not found")
        return 1

    existing = read_existing_names(SKILL_INDEX)

    new_skills = [d for d in sorted(LAZY_DIR.iterdir()) if d.is_dir() and d.name not in existing]

    if not new_skills:
        print("No new skills to add. Index is up to date.")
        return 0

    with open(SKILL_INDEX, "a") as f:
        for skill_dir in new_skills:
            skill_md = skill_dir / "SKILL.md"
            name = extract_frontmatter_field(skill_md, "name") or skill_dir.name
            description = (
                extract_frontmatter_field(skill_md, "description") or "Lazy-loaded skill"
            )
            display = name.replace("-", " ").replace("_", " ").title()

            f.write(f"\n## {display}\n\n")
            f.write(f"- name: {name}\n")
            f.write(f"- description: {description}\n")
            f.write("- triggers:\n")
            f.write(f"- path: `lazy-loading/skills/{skill_dir.name}/SKILL.md`\n")
            print(f"Added: {display}")

    print(f"\nUpdated {SKILL_INDEX} with {len(new_skills)} new skill(s).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
