# Implementation Notes

## Stitch plugin-bundle source support

- Scope is individual skill installation, not Codex plugin marketplace
  installation. Skillhub still models sources as catalogs of installable skill
  directories.
- Stitch plugin-bundle skills are discovered from
  `plugins/<plugin>/skills/**/SKILL.md` and materialized into the existing
  generated catalog cache.
- Generated install IDs stay compatible with Skillhub's current ID rules by
  using the plugin name as a prefix, for example
  `stitch-design_generate-design`. Namespaced frontmatter names such as
  `stitch::generate-design` stay unchanged inside `SKILL.md` and are added to
  search triggers.
- The existing 5-column `sources.tsv` format is unchanged.
- Source names are normalized to lower-case on input, including TUI add-source
  forms and qualified install arguments such as `Uncodixfy/uncodixfy`.
- Single-skill repositories with a root-level `SKILL.md` are materialized as a
  generated catalog row, using the frontmatter `name` when it is a valid ID and
  falling back to the normalized source name.
- Root-level single-skill repositories copy only the skill payload into the
  generated cache and skip `.git`, so installing a source does not vend the
  cloned repository metadata as part of the skill.
- Inline frontmatter parsing only treats known skill metadata keys as boundaries
  between fields. This keeps descriptions containing text such as
  `http://example.test` or `note: keep text` intact.
