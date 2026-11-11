# Changelog

What changed in Albert, newest first.

## Unreleased (2026-11-10)

- **Tabs across the top.** Installed and Install new now sit in a row above the list instead of down the sidebar.
- **Details dock on the right.** The details panel now fills the right edge, top to bottom, and the list makes room for it instead of being covered.

## 0.1.8 (2026-11-10)

- The update hint now says `brew update && brew upgrade albert`, so it works even when your Homebrew hasn't refreshed its list of versions.
- **Search GitHub.** The "Install new" tab now also searches Go tools on GitHub and installs them with `go install`. Installed tools then show up under Go.
- **New sidebar.** The top bar is now a dark left sidebar, and the colours follow the kamllistos logo: ink, orange and bone, in light and dark mode.
- **New logo.** Albert's mark is an A, used in the sidebar, the browser tab icon and the session-ended screen.

## 2026-10-27

- Albert tells you when a newer release is out.
- Fixed the delete action, with safer removal of gems and Homebrew tools.

## 2026-10-24

- Uninstall says what it removed, and removes only Albert's own files.
- Homebrew's upgrade and install confirmation prompt no longer blocks updates.
- README has a demo GIF.
