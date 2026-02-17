---
# fritzbox-mcp-server-455c
title: README und Lizenz schärfen + Local-vs-Cloud Architektur erklären (Graphviz digraph)
status: todo
type: task
priority: high
created_at: 2026-02-17T12:49:42Z
updated_at: 2026-02-17T12:49:42Z
---

Ziel
- README.adoc und README-de.adoc so anpassen, dass “MCP-Server lokal” und “LLM lokal oder Cloud” klar getrennt sind.
- 2 Graphviz digraph-Architekturen hinzufügen (kein Mermaid, kein SVG). Fokus: Datenfluss lokal vs Cloud.
- es existiert schon ein digraph dieses berücksichtigen nicht überschreiben (da stehen u.a. color drin die fehlen unten)

Konkrete Änderungen (Acceptance Criteria)
1) README.adoc
   - bleibe in der bestehenden Sprache - Deutsch ist die führende Sprache.
   - Ersetze/entschärfe Aussagen wie “No cloud services / everything runs locally” durch:
     - MCP-Server läuft lokal, keine offenen Ports nötig.
     - Wenn ein Cloud-LLM verwendet wird, gehen Prompts/Antworten an den Anbieter; FRITZ!Box-Zugriff bleibt lokal über MCP-Toolcalls.
     - Für “alles lokal” wird ein lokales LLM empfohlen.
   - Füge einen Abschnitt “Privacy / Data flow” hinzu:
     - “Local LLM mode” vs “Cloud LLM mode”
     - Was verlässt den Rechner (nur Text) und was nicht (Router-Creds, TR-064 Endpoint).
3) Architekturdiagramme als Graphviz digraph:
   - Lege Dateien an oder besser benutze die existierende dot (falls extern):
     - docs/arch-local-llm.dot
     - docs/arch-cloud-llm.dot
   - Binde sie in README.adoc als Codeblock oder include ein (AsciiDoc), z.B.:
     - [source,dot] ---- … ----
     - oder via asciidoctor-diagram include, wenn im Projekt genutzt.

verifiziere diese beiden Graphen:

Graphviz digraph 1: Local LLM (copy into docs/arch-local-llm.dot)
digraph LocalLLM {
  rankdir=LR;
  node [shape=box];

  subgraph cluster_local_machine {
    label="Local machine";
    style="rounded";
    "User" [shape=oval];
    "AI Assistant (IDE/Desktop)";
    "Local LLM";
    "MCP Client";
    "fritzbox-mcp-server";
  }

  subgraph cluster_home_net {
    label="Home network (LAN)";
    style="rounded";
    "FRITZ!Box (TR-064)";
    "LAN devices";
  }

  "User" -> "AI Assistant (IDE/Desktop)" [label="natural language"];
  "AI Assistant (IDE/Desktop)" -> "Local LLM" [label="prompt"];
  "Local LLM" -> "AI Assistant (IDE/Desktop)" [label="response/tool plan"];

  "AI Assistant (IDE/Desktop)" -> "MCP Client" [label="tool calls"];
  "MCP Client" -> "fritzbox-mcp-server" [label="MCP over stdio"];
  "fritzbox-mcp-server" -> "FRITZ!Box (TR-064)" [label="HTTPS + credentials"];
  "FRITZ!Box (TR-064)" -> "LAN devices" [label="mesh / devices"];
}

Graphviz digraph 2: Cloud LLM (copy into docs/arch-cloud-llm.dot)
digraph CloudLLM {
  rankdir=LR;
  node [shape=box];

  subgraph cluster_local_machine {
    label="Local machine";
    style="rounded";
    "User" [shape=oval];
    "AI Assistant (IDE/Desktop)";
    "MCP Client";
    "fritzbox-mcp-server";
  }

  subgraph cluster_internet {
    label="Internet";
    style="rounded";
    "Cloud LLM Provider";
  }

  subgraph cluster_home_net {
    label="Home network (LAN)";
    style="rounded";
    "FRITZ!Box (TR-064)";
    "LAN devices";
  }

  "User" -> "AI Assistant (IDE/Desktop)" [label="natural language"];
  "AI Assistant (IDE/Desktop)" -> "Cloud LLM Provider" [label="prompt (text leaves device)"];
  "Cloud LLM Provider" -> "AI Assistant (IDE/Desktop)" [label="response/tool plan"];

  "AI Assistant (IDE/Desktop)" -> "MCP Client" [label="tool calls"];
  "MCP Client" -> "fritzbox-mcp-server" [label="MCP over stdio"];
  "fritzbox-mcp-server" -> "FRITZ!Box (TR-064)" [label="HTTPS + credentials (local)"];
  "FRITZ!Box (TR-064)" -> "LAN devices" [label="mesh / devices"];
}

Hinweise für Formulierungen (kurz, ohne Marketing)
- “MCP server runs locally. LLM can be local or cloud.”
- “If you use a cloud LLM, your prompts may contain sensitive device names/IPs; use a local LLM if you want to keep all text local.”
- “Source-available license (not OSI open source, not FOSS). You may build/run/audit; redistribution requires permission.”

