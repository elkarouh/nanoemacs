# TODO

## Embedded terminal buffer

`M-x yazi` suspends nem and hands it the terminal. A real terminal buffer would
let yazi, htop, vim and a password prompt run inside a nem buffer instead.

- Run the program on a pty rather than pipes (the `*shell*` buffer uses pipes,
  see `editor/shellbuf.go`).
- Add a VT100/xterm emulator that keeps a screen grid and feeds it to a buffer
  or a dedicated view, with colours and the alternate screen.
- Pass keys through to the program while the buffer is focused, with one
  escape key (for example `C-c C-c`) back to nem's own keymap.
- Resize the pty with the window.
- Likely needs a pty library and a terminal-emulation library; check their
  licences against THIRD-PARTY.md.
- Then `yazi` could open in a window beside the file instead of replacing the
  screen.
