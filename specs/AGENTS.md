# AI coding-agent instructions

Build an independent, WhatsApp-inspired desktop messenger **frontend only** with Rust + GPUI Kit. Read `README.md` and referenced specs first. Do not implement unofficial WhatsApp protocols, fake claims of real login, or backend network calls. Use deterministic fixtures and mock service interfaces.

Prioritize: correctness of navigation and interactions; reuse GPUI components; accessibility and native macOS window behavior; adaptive layout; performance. Agent has freedom to design high-fidelity appearance, constrained by tokenized Light/Dark design system. Treat low-fi SVGs as layout contracts, not pixel-perfect artworks. Do not generate 120 separate page components; implement reusable feature views with 120 discoverable states and tests. No forced full-screen API; maximize application window beneath OS menu bar. The menu bar drawn in wireframes is illustrative only.

Source of truth: `design-system/design-tokens.json`, `specifications/screen-map.json`, then flow files. When conflict occurs, explain it and choose user workflows and native accessibility. Version-pin gpui-kit, compile and test on macOS before claiming success. Verify working behavior and screenshots in both themes. Avoid architectural complexity not justified by mock-only scope.
