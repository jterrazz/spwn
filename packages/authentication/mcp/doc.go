// Package mcp implements OAuth login + persistence for hosted MCP
// servers (Notion, Linear, Atlassian, etc.). It owns one host
// directory — ~/.spwn/credentials/mcp — that mcp2cli treats as its
// cache. The same directory is bind-mounted into every world spawn,
// so a single `spwn auth login <provider>` on the host unlocks the
// MCP for every world without env-vars or per-container reauth.
//
// The OAuth dance runs in a one-shot helper container
// (spwn-mcp-auth:latest, a thin Python image with mcp2cli) so the
// host stays clean — no Python or pipx requirements.
package mcp
