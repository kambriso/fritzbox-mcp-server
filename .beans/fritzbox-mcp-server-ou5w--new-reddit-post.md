---
# fritzbox-mcp-server-ou5w
title: New Reddit post
status: draft
type: task
priority: normal
created_at: 2026-02-17T12:57:35Z
updated_at: 2026-02-17T12:57:55Z
---

Ziel
- Einen neuen Reddit-Post textlich so aufsetzen, dass die 3 Hauptkritikpunkte vorab adressiert sind:
  1) Source verfügbar (und ob FOSS oder nicht: klar sagen)
  2) “läuft lokal” präzise erklären (MCP lokal; LLM lokal oder Cloud)
  3) Mehrwert vs “geht auch ohne KI” (Convenience / Natural-Language Interface)
- Ton: sachlich, kurz, kein Hype.
- Call to action: Feedback, Threat-Model-Review, Issues/PRs.

Post-Entwurf (Deutsch, anpassbar)
Titel:
FRITZ!Box per natürlicher Sprache steuern (MCP-Server, lokal) – Source verfügbar, Feedback gesucht

Text:
Ich habe einen MCP-Server gebaut, der einem KI-Assistenten (IDE/Desktop) Tool-Zugriff auf eine FRITZ!Box gibt (TR-064). Der MCP-Server läuft lokal bei euch, keine offenen Ports nötig.

Wichtig zur “lokal vs Cloud”-Frage:
- Der MCP-Server läuft lokal.
- Das LLM könnt ihr lokal oder in der Cloud betreiben.
- Wenn ihr ein Cloud-LLM nutzt, gehen Prompts/Antworten als Text an den Anbieter. Der FRITZ!Box-Zugriff selbst bleibt lokal (Toolcalls -> MCP -> TR-064).

Lizenz:
Der Source ist im Repo einsehbar (audit/build/run). Es ist keine OSI-Open-Source-Lizenz; Redistribution nur mit Erlaubnis (Details in LICENSE).

Beispiele:
- “Warum war das Internet gestern Nacht weg?”
- “Welche Geräte sind neu/unbekannt?”
- “Blockiere die PlayStation in 20 Minuten”

Repo:
https://github.com/kambriso/fritzbox-mcp-server

Ich suche Feedback zu:
- Sicherheits-/Threat-Model (Spoofing, Credentials, Scope)
- UX: welche Tools/Kommandos fehlen?
- Doku: ist “local vs cloud” jetzt verständlich?

Subreddit-Fit
- r/informatik (eher “wie gebaut / Architektur / Security”)
- r/selfhosted (englisch, falls du willst)
- r/de_EDV nur, wenn deren Regeln Werbung/Source-available zulassen.

Done-Kriterium
- Text finalisiert (1 Titel + 1 Post-Body + 1 kurzer Kommentar für erste Rückfragen).
