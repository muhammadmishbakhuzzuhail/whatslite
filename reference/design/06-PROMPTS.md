# 06 · Prompts

Copy-paste these into the coding agent. Replace `<STACK>` with your stack (React + Vite, Tauri, GPUI, etc.).

## A · Bootstrap (run once)
> You are building **WhatsLite**, a desktop chat client. The complete design package is in this folder.
> Read in this order before writing code: `00-START-HERE.md`, `machine/tokens.json`, `01-SCREENS.md`, `02-COMPONENTS.md`, `03-BEHAVIOUR-RULES.md`, `04-COMING-SOON.md`, `05-BUILD-ORDER.md`.
> Stack: `<STACK>`. Set up the token layer first and prove dark mode is a pure value swap.
> Do not invent colours, strings, spacing or states. If something is missing, stop and ask.
> Report back: file tree, token layer, and a screenshot of the empty shell.

## B · One screen at a time
> Implement **<SCREEN NAME>** from `01-SCREENS.md` (page <NN>).
> Use only components listed in `02-COMPONENTS.md`. If a component does not exist yet, create it there first and add it to the registry.
> States required: default, hover, selected, focus-visible, empty, loading, failed, and the coming-soon fallback from `04-COMING-SOON.md`.
> Acceptance: matches the design at 1440×900, no raw hex, keyboard reachable.
> Return the component list you created and a screenshot.

## C · Component
> Build component `<NAME>` from `02-COMPONENTS.md`.
> Sizes and paddings come from the registry; colours from `machine/tokens.json`; type from the type scale in `docs/`.
> Implement every state in `03-BEHAVIOUR-RULES.md` that mentions it.
> Provide: props, variants, keyboard behaviour, and 3 usage examples in the app.

## D · Behaviour pass
> Re-read `03-BEHAVIOUR-RULES.md`. Audit each rule against the current implementation and list every violation with file and line. Fix them one rule at a time, showing a before/after screenshot per rule. Start with §1 bubble grouping.

## E · Coming-soon sweep
> For every row marked SOON in `04-COMING-SOON.md`, verify the control exists, is reachable, and raises exactly one toast with the stated copy. Produce a table: surface → control → copy → pass/fail.

## F · States sweep
> For each of the 18 lists in `04-COMING-SOON.md` and `05-BUILD-ORDER.md` §9, confirm empty / loading / failed exist. Loading must be a skeleton that mirrors the row and must not appear before 200ms. Failed must offer retry and must not look like empty.

## G · Keyboard & a11y
> Implement `03-BEHAVIOUR-RULES.md` §10 and §11: focus ring only for keyboard, logical tab order, Esc closes one layer, focus trap in modal/drawer, ARIA labels on icon-only buttons, reduced motion, text scaling to 150%. Prove each with a short recording or screenshot.

## H · Design QA (before calling it done)
> Compare the implementation against `design/whatslite.pen` page by page. For each screen list: layout differences, spacing differences, missing states, missing copy. Fix everything that changes hierarchy; report the rest. Finish with the checklist in `05-BUILD-ORDER.md` → Definition of done.

---

## Anti-patterns (put these in the agent's system prompt)
1. Do not use `#25D366` or WhatsApp assets.
2. Do not invent copy, users, avatars or timestamps.
3. Do not create a second UI for dark mode.
4. Do not skip states because “the API is not ready” — that is what the toast is for.
5. Do not copy-paste a component: instance it.
6. Do not add animations that are not in the motion scale (120/160/200/240ms, ease-out).
7. Do not restructure the sidebar or merge screens.
