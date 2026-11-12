# Contributing to Lucy

Thanks for taking the time to help. Lucy is a small, focused project, so a short conversation before
a large change saves everyone time.

## Before you start

- **Bugs:** search the [existing issues](https://github.com/zeybekserdar/lucy/issues) first, then
  open a bug report with steps to reproduce.
- **Features:** open a feature request and describe the problem before writing code. Lucy
  deliberately leaves some things out: user and role management, public status pages, maintenance
  windows, notification channels other than ntfy, and TCP, ping, DNS or heartbeat monitors.
- **Security issues:** do not open a public issue. Follow [SECURITY.md](SECURITY.md).

## Development setup

You need Go 1.27 and Node.js 24.

```bash
git clone https://github.com/zeybekserdar/lucy.git
cd lucy
npm --prefix web install
```

Run the API and the interface in two terminals:

```bash
go run ./cmd/lucy
```

```bash
npm --prefix web run dev
```

Open http://localhost:5173. Vite proxies `/api` to the Go server on port 8080. To work on the
interface without a backend, add `?state=ok`, `incident`, `offline`, `empty` or `error` to the
address for sample data.

To try the production build, run `docker compose up -d --build`.

## Making a change

1. Fork the repository and create a branch from `main`.
2. Keep the change focused on one thing. Unrelated clean-ups belong in their own pull request.
3. Add or update tests for behavior you change. Go tests live next to the code (`*_test.go`), and
   interface logic is tested with Vitest (`*.test.ts`).
4. Make sure everything passes:

   ```bash
   go test ./...
   go vet ./...
   gofmt -l .
   npm --prefix web test
   npm --prefix web run lint
   npm --prefix web run build
   ```

5. Open a pull request and fill in the template.

## Guidelines

- **Interface text** goes through i18n. Add every new string to both `web/src/i18n/en.ts` and
  `web/src/i18n/tr.ts`. Notification texts live in `internal/notify/compose.go` and also come in both
  languages.
- **Design** follows the existing tokens in `web/src/styles/tokens.css`. Check new screens in light
  and dark themes and at 375 px wide; nothing may scroll horizontally.
- **Database changes** are new entries appended to the `migrations` list in
  `internal/store/store.go`. Never edit a migration that has already shipped.
- **Accessibility:** interactive elements need a visible focus state and an accessible name, and
  status must not rely on color alone.
- **Dependencies:** prefer the standard library. Explain any new dependency in the pull request.

## Commit messages

Write the subject in the imperative mood and keep it under about 70 characters. Use the body to
explain why the change is needed when it is not obvious.

## License

By contributing, you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE).
