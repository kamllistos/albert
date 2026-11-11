# Changelog

What changed in Albert, newest first.

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
