# Guide

Gothic builds full-stack apps in Go. You write one `.gth` file with ordinary Go
for your model and a small view syntax for your HTML. The compiler emits Go that
renders the page on the server and JavaScript that updates it in the browser.

This guide is meant to be read in order. Each chapter builds on the previous
one and ends with a working example.

1. [Quickstart](01-quickstart.md)
2. [State and commands](02-state.md)
3. [Components](03-components.md)
4. [Routing and documents](04-routing.md)
5. [Browser interop](05-browser.md)
6. [Long lists](06-long-lists.md)
7. [Architecture](07-architecture.md)
8. [The server](08-server.md)

If you prefer to learn by reading code, the `examples/` directory has working
apps for every feature in this guide. If you want the full grammar in one place,
keep [SYNTAX.md](../SYNTAX.md) open.
