## Summary
What changed and why.

## Type
- [ ] fix (bug fix)
- [ ] feat (new feature)
- [ ] perf / refactor
- [ ] docs / chore

## Checklist (required — same as CI)
- [ ] `go vet ./...` clean
- [ ] `go test ./...` passing
- [ ] `npm --prefix frontend run lint:css` 0 errors
- [ ] `npm --prefix frontend run build` succeeds, **0 unused-CSS warnings**
- [ ] `wails build -tags "webkit2_41 netgo"` succeeds
- [ ] No heavy dependencies added without strong justification (see [the lean philosophy](../CONTRIBUTING.md#philosophy-stay-lean))
- [ ] No secrets / `*.db` / binaries committed
- [ ] Heavy DB work in event handlers offloaded to `a.bg()` (whatsmeow handlers are synchronous)

## Additional notes
Screenshots/GIFs if there are UI changes.
