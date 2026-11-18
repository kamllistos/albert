# Changelog

What changed in Albert, newest first.

## 0.1.10 (2026-11-17)

- Albert now tells you when a newer version is out after any command (checked at most once a day), with the right update command for how you installed it (Homebrew, install script or `go install`).
- On Linux, install Albert with the install script or `go install github.com/kamllistos/albert/cmd/albert@latest`. Homebrew is for Macs only: the Homebrew cask no longer offers Linux downloads.
- `albert version` shows the real version for builds made with `go install`.
- Messages and help say "this computer" instead of "this Mac", since Albert runs on Linux too.
- **More sources on Linux.** Albert now lists packages you installed with apt (only ones you chose, not their dependencies) and Snap, and can update, install and remove them. Updating asks for your password, so run `albert update` in Terminal for those.
- **Programs in `~/.local/bin`.** Albert lists standalone programs there on Mac and Linux and finds their version by running `--version` (or `version`) once. It remembers the answer until the file changes. Albert can't update these, since they don't come from a package manager.
- On Linux, the install script suggests adding Albert to `~/.bashrc` (not `~/.zshrc`) when it can't tell your shell.

## 0.1.9 (2026-11-16)

- **Linux.** Albert runs on 64-bit Linux (x86-64 and ARM). Install it with the install script. On Linux Albert opens in your browser without Docker.
- Albert needs macOS 12 (Monterey) or later. The README now says so, and the install script stops with a clear message on older Macs instead of installing a program that won't open.

## 0.1.8 (2026-11-10)

- The update hint now says `brew update && brew upgrade albert`, so it works even when your Homebrew hasn't refreshed its list of versions.
- **Search GitHub.** The "Install new" tab now also searches Go tools on GitHub and installs them with `go install`. Installed tools then show up under Go.
- **New look.** The logo, tabs, Refresh and Update all share one dark bar across the top, and the colours follow the kamllistos logo: ink, orange and bone, in light and dark mode.
- **Details dock on the right.** The details panel fills the right edge below the top bar, and the list makes room for it instead of being covered.
- **New logo.** Albert's mark is an A, used in the top bar, the browser tab icon and the session-ended screen.

## 2026-10-27

- Albert tells you when a newer release is out.
- Fixed the delete action, with safer removal of gems and Homebrew tools.

## 2026-10-24

- Uninstall says what it removed, and removes only Albert's own files.
- Homebrew's upgrade and install confirmation prompt no longer blocks updates.
- README has a demo GIF.
