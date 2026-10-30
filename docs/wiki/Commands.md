<!-- Generated from nem's code by TestWikiReferenceIsCurrent in editor/wiki_test.go.
     Do not edit: change the code, then run NEM_UPDATE_WIKI=1 go test ./editor -run TestWikiReferenceIsCurrent. -->

# Commands

Every command nem has, by name. `M-x` runs any of them by name, with fuzzy completion: `M-x fwc` finds `forward-char`. [`nem.bind`](Lua-API#nembind) binds a key to any of them, and [`nem.run`](Lua-API#nemrun) runs one from a command of your own. Commands you define with [`nem.command`](Lua-API#nemcommand) join this list, as equals.

A key in a mode - `dired: f` - works only in that mode's buffers; see [Keybindings](Keybindings).

| Command | Keys | Does |
|---|---|---|
| `add-next-occurrence` | `C-d` | Select the word at point, then each further press adds a cursor at the next occurrence of the selection. Edits then apply at every cursor. |
| `async-shell-command` | `M-&` | Run a shell command, showing its output as it comes, in *Async Shell Command*. |
| `back-to-indentation` | `M-m` | Move to the first non-blank character on the line. |
| `backward-char` | wdired: `C-b` | Move point one grapheme backward. |
| `backward-kill-word` | `M-<backspace>` | Kill backward to the start of the previous word. |
| `backward-paragraph` | `M-{` | Move to the start of the paragraph, ARG paragraphs back. |
| `backward-sexp` | `C-M-b` | Move back over the previous balanced expression, ARG times. |
| `backward-word` | `M-b` | Move point to the start of the previous word. |
| `beginning-of-buffer` | `M-<` `C-<home>` | Move point to the start of the buffer. |
| `bookmark-file` | `C-x r f` | Add this buffer's file to the favourites, shown in the side panel's Favorites tab; again to remove it. |
| `bracketed-paste` |  | Insert text pasted through the terminal, verbatim. *(Not offered by `M-x`.)* |
| `capitalize-word` | `M-c` | Capitalize the following word. |
| `comment-dwim` | `M-;` | Comment or uncomment the lines of the region, or the current line. |
| `compilation-display-error` | compilation: `o` `C-o` | Show the place the line at point names in the other window, staying here. |
| `compilation-next-error` | compilation: `n` | Move to the next line naming a place, ARG on, showing it in the other window. |
| `compilation-previous-error` | compilation: `p` | Move to the previous line naming a place, ARG back, showing it in the other window. |
| `compile` |  | Run a command - a build, the tests - showing its output and leading to the places it names. |
| `compile-goto-error` | compilation: `RET` | Go to the place the line at point names, in the other window. |
| `count-words` | `M-=` | Count the lines, words and characters in the region, or the buffer. |
| `dabbrev-expand` | `M-/` | Complete the word before point from words already in the text; repeat for the next. |
| `delete-backward-char` | `<backspace>` | Delete the character before point. |
| `delete-char` | `<delete>` | Delete the character after point. |
| `delete-horizontal-space` | `M-\` | Delete all spaces and tabs around point. |
| `delete-indentation` | `M-^` | Join this line to the previous one, with one space between. |
| `delete-other-windows` | `C-x 1` | Make the current window fill the frame. |
| `delete-trailing-whitespace` |  | Delete the spaces and tabs at the ends of the buffer's lines, or the region's. |
| `delete-window` | `C-x 0` | Remove the current window. |
| `describe-bindings` | `C-h b` `<f1> b` | Show every key binding in a help buffer. |
| `describe-key` | `C-h k` `<f1> k` | Read a key sequence and report the command it runs. |
| `dired` | `C-x d` | List a directory, to browse it and act on its files. |
| `dired-bookmark-delete` | dired: `C-x r k` | Delete a bookmark, read by its name with completion. |
| `dired-bookmark-jump` | dired: `C-x r b` | Go to the directory of a bookmark, read by its name with completion. |
| `dired-bookmark-list` | dired: `C-x r l` | List the bookmarks, each name with its directory, in a read-only buffer. |
| `dired-bookmark-set` | dired: `C-x r m` | Bookmark the directory of this dired buffer under a name, which C-x r b goes to. |
| `dired-copy-filename` | dired: `w` | Copy the names of the marked files, or of the file at point. |
| `dired-create-directory` | dired: `+` | Create a directory, and any missing parents. |
| `dired-do-copy` | dired: `C` | Copy the file at point, or the marked files into a directory. |
| `dired-do-delete` | dired: `D` | Delete the marked files, or the file at point, after asking. |
| `dired-do-flagged-delete` | dired: `x` | Delete the files flagged with d, after asking. |
| `dired-do-open` | dired: `E` | Open the marked files, or the file at point, with the system app. |
| `dired-do-rename` | dired: `M` | Rename the file at point, or move the marked files into a directory. |
| `dired-fd` | dired: `F` | Run fd, with the arguments typed, in the directory of this dired buffer, and list the files it finds. |
| `dired-find-file` | dired: `e` `f` `RET` `M-<down>` | Visit the file at point, or list the directory at point. |
| `dired-find-file-other-window` | dired: `o` | Visit the file at point in another window. |
| `dired-first-file` | dired: `M-<` | Move to the first file in the listing, below the parent entry. |
| `dired-flag-file-deletion` | dired: `d` | Flag the file at point for deletion by x, and move down. |
| `dired-history-back` | dired: `M-p` | List the directory listed before this one, as a browser goes back. |
| `dired-history-forward` | dired: `M-n` | List the directory gone back from, as a browser goes forward. |
| `dired-jump` | `C-x C-j` | List the directory of the current file, with point on the file. |
| `dired-last-file` | dired: `M->` | Move to the last file in the listing. |
| `dired-mark` | dired: `m` | Mark the file at point for the next operation, and move down. |
| `dired-next-line` | dired: `n` `C-n` `SPC` `<down>` | Move to the next file, ARG files down. |
| `dired-previous-line` | dired: `p` `C-p` `<up>` | Move to the previous file, ARG files up. |
| `dired-revert` | dired: `g` | Read the directory again. |
| `dired-rg` | dired: `R` | Run rg, with the arguments typed, in the directory of this dired buffer, and list the matching lines. |
| `dired-sort-toggle` | dired: `s` | Sort by name, then by time, then by size. |
| `dired-toggle-details` | dired: `(` | Show or hide permissions, sizes and dates. |
| `dired-toggle-hidden` | dired: `.` | Show or hide dotfiles. |
| `dired-toggle-marks` | dired: `t` | Mark every unmarked file and unmark every marked one. |
| `dired-unmark` | dired: `u` | Remove the mark on the file at point, and move down. |
| `dired-unmark-all` | dired: `U` | Remove every mark and deletion flag. |
| `dired-unmark-backward` | dired: `<backspace>` | Move up a file and remove its mark. |
| `dired-up-directory` | dired: `^` `M-<up>` | List the parent directory, with point on this one. |
| `downcase-region` | `C-x C-l` | Convert the region to lower case. |
| `downcase-word` | `M-l` | Convert the following word to lower case. |
| `edit-language` |  | Edit how a language is coloured, in its .syntax file beside init.lua; saving it recolours every buffer. |
| `end-of-buffer` | `M->` `C-<end>` | Move point to the end of the buffer. |
| `exchange-point-and-mark` | `C-x C-x` | Put point where the mark is, and the mark where point was. |
| `execute-extended-command` | `M-x` | Read a command name in the minibuffer and run it. |
| `fd-find-file` |  | Visit the file at point in the fd results, or list the directory there in dired. |
| `fill-paragraph` | `M-q` | Re-wrap the paragraph at point to the fill column. |
| `find-file` | `C-x C-f` | Visit a file in the current window, creating it if absent. |
| `forward-char` | `C-f` | Move point one grapheme forward. |
| `forward-paragraph` | `M-}` | Move to the end of the paragraph, ARG paragraphs forward. |
| `forward-sexp` | `C-M-f` | Move over the next balanced expression, ARG times. |
| `forward-word` | `M-f` | Move point to the end of the next word. |
| `fzf` | `M-s f` | Open a file of the buffer's project, found by a few letters of its path; C-u asks where. |
| `fzf-lines` | `M-s l` | Go to a line of the buffer, found by a few letters of it. |
| `goto-line` | `M-g M-g` | Move point to the start of a numbered line. |
| `grep-display-match` | grep: `o` | Show the match at point in the other window, staying here. |
| `grep-goto-match` | grep: `RET` | Go to the match at point, in the other window. |
| `grep-next-file` | grep: `}` | Move to the next file's matches, ARG files on. |
| `grep-next-line` | grep: `C-n` `<down>` | Move to the next match, ARG matches on. |
| `grep-next-match` | grep: `n` | Move to the next match, ARG matches on, showing it in the other window. |
| `grep-previous-file` | grep: `{` | Move to the previous file's matches, ARG files back. |
| `grep-previous-line` | grep: `C-p` `<up>` | Move to the previous match, ARG matches back. |
| `grep-previous-match` | grep: `p` | Move to the previous match, ARG matches back, showing it in the other window. |
| `grep-revert` | grep: `g` | Search again, for the same thing in the same project or buffer. |
| `image-view` |  | Show the active buffer's file, a PNG, JPEG or GIF, as an image over the window, in terminals that draw kitty graphics (kitty, Ghostty, WezTerm). Any key closes it. |
| `imenu` | `M-g i` | Go to a definition in this buffer - a function, a type, a heading - by name. |
| `indent-for-tab-command` | `TAB` | Indent at point, or the region's lines, with the file's tabs or spaces. |
| `indent-rigidly` | `C-x TAB` | Shift the region's lines right by ARG columns, or one level; left with a negative ARG. |
| `indent-rigidly-left-to-tab-stop` | `<backtab>` | Shift the region's lines, or the current line, one level left. |
| `isearch-backward` | `C-r` | Search incrementally backward as you type. |
| `isearch-forward` | `C-s` | Search incrementally forward as you type. |
| `just-one-space` | `M-SPC` | Leave exactly one space around point. |
| `keyboard-quit` | `C-g` | Abandon the current operation. |
| `kill-all-buffers` |  | Kill every buffer after asking once, and again for each with unsaved changes. *scratch* is kept. |
| `kill-buffer` | `C-x k` | Kill the current buffer, confirming if it is modified. |
| `kill-compilation` | compilation: `C-c C-k` | Stop the running compilation. |
| `kill-line` | `C-k` | Kill to the end of the line, or the newline if already there. |
| `kill-region` | `C-w` | Kill the region, or the whole line if nothing is selected, saving it on the kill ring. |
| `kill-ring-save` | `M-w` | Save the region, or the whole line if nothing is selected, on the kill ring without removing it. |
| `kill-sexp` | `C-M-k` | Kill the balanced expression after point. |
| `kill-some-buffers` |  | Ask about each buffer in turn and kill the ones you answer y to; q stops asking. *scratch* is kept. |
| `kill-word` | `M-d` | Kill forward to the end of the next word. |
| `kmacro-end-and-call-macro` | `C-x e` | Play the last keyboard macro, ARG times; then e plays it again. |
| `kmacro-end-macro` | `C-x )` | Stop recording a keyboard macro. |
| `kmacro-end-or-call-macro` | `<f4>` | Stop recording a keyboard macro, or play the last one, ARG times (0: until it fails). |
| `kmacro-start-macro` | `C-x (` | Start recording a keyboard macro. |
| `kmacro-start-macro-or-insert-counter` | `<f3>` | Start recording a keyboard macro, or while recording, insert the macro counter. |
| `left-char` | `<left>` | Move one character to the left: backward, or forward in a line that reads right to left. |
| `list-buffers` | `C-x C-b` | Display a listing of every live buffer. |
| `mark-whole-buffer` | `C-x h` | Put point at the start of the buffer and the mark at the end. |
| `markdown-preview` | `C-x x m` | Show the buffer rendered as Markdown, in its place, as mdcat renders it. The preview is updated on each save of the buffer; e shows the buffer to edit again. |
| `html-preview` | `C-x x h` | Show the buffer rendered as HTML text, in its place, as lynx lays it out. The preview is updated on each save of the buffer; e shows the buffer to edit again. |
| `move-beginning-of-line` | `C-a` `<home>` | Move point to the start of the line. |
| `move-end-of-line` | `C-e` `<end>` | Move point to the end of the line. |
| `move-lines-down` | `M-<down>` | Move the region's lines, or the current line, one line down. |
| `move-lines-up` | `M-<up>` | Move the region's lines, or the current line, one line up. |
| `newline` | `RET` | Insert a newline, copying the current line's indentation. |
| `next-buffer` | `M-.` `C-x <right>` | Show the next buffer in the tab strip in the current window. |
| `next-error` | `` C-x ` `` `M-g n` `M-g M-n` | Go to the next match of the last search, or error of the last compilation, ARG on. |
| `next-line` | `C-n` `<down>` | Move point down one line, keeping the goal column. |
| `occur` | `M-s o` | List the lines of this buffer that match a regexp, each leading to its line. |
| `open-externally` |  | Open this buffer's file, or the directory a listing shows, with the system app. |
| `open-in-emacs` | `S-<f4>` | Open this buffer's file in a running Emacs, at point's line. A listing opens its directory. Needs emacsclient and a server. |
| `open-in-sublime` | `S-<f5>` | Open this buffer's file in Sublime Text, at point's line. A listing opens its directory. Needs the subl command. |
| `open-line` | `C-o` | Insert a newline after point, leaving point in place. |
| `other-window` | `C-x o` | Select another window, ARG windows forward. |
| `preview-close` |  | Close the Markdown preview and show the buffer it renders in its place. |
| `previous-buffer` | `M-,` `C-x <left>` | Show the previous buffer in the tab strip in the current window. |
| `previous-error` | `M-g p` `M-g M-p` | Go to the previous match of the last search, or error of the last compilation, ARG back. |
| `previous-line` | `C-p` `<up>` | Move point up one line, keeping the goal column. |
| `project-compile` | `C-x p c` | Run a compilation from the top of the current project. |
| `project-dired` | `C-x p D` | List the current project's top directory. |
| `project-find-dir` | `C-x p d` | List a directory of the current project, found by name. |
| `project-find-file` | `C-x p f` | Open a file in the current project, found by name. |
| `project-find-regexp` | `M-s p` `C-x p g` | Search the current project's files for a regexp, listing the matching lines. |
| `project-forget-project` |  | Take a project off the known projects. |
| `project-kill-buffers` | `C-x p k` | Kill every buffer of the current project, after asking. |
| `project-query-replace` | `C-x p r` | Replace text across the current project's files, asking at each match. |
| `project-recentf` | `C-x p e` | Open one of the current project's recently opened files. |
| `project-save-buffers` | `C-x p S` | Save every modified file of the current project. |
| `project-switch-project` | `C-x p p` | Open a file in another project nem knows. |
| `project-switch-to-buffer` | `C-x p b` | Switch to a buffer of the current project. |
| `project-toggle-test` | `C-x p t` | Go from a file to its test, or from a test to the file it tests. |
| `query-replace` | `M-%` | Replace occurrences of a string, asking about each one. |
| `query-replace-regexp` | `C-M-%` | Replace matches of a regexp, asking about each one. |
| `quit-window` | dired: `q`; grep: `q`; compilation: `q` | Show another buffer here, putting this one at the back of the list. |
| `recenter-top-bottom` | `C-l` | Scroll point to the centre, then the top, then the bottom. |
| `recentf-open` | `C-x C-r` | Open one of the files opened recently. |
| `recompile` | compilation: `g` | Run the last compilation again, or the one this buffer shows. |
| `recover-file` |  | Replace this buffer with its autosaved contents. |
| `redo` | `M-_` | Redo the change most recently undone. |
| `remove-last-cursor` | `C-M-u` | Drop the cursor added most recently, giving its word back to the others. |
| `replace-regexp` |  | Replace a regexp's matches everywhere after point, or in the region, without asking. |
| `replace-string` |  | Replace a string everywhere after point, or in the region, without asking. |
| `reverse-region` |  | Reverse the order of the region's lines. |
| `revert-buffer` | `C-x x g` | Read the buffer's file again, throwing away any edits after asking; a listing is read again, a compilation run again. |
| `rg` | `M-s g` `M-s r` | Search the files of the buffer's project as the pattern is typed, ripgrep's options and all; C-u asks where. |
| `rg-find-match` |  | Visit the file of the match on the line at point in the rg results, at the matched line. |
| `right-char` | `<right>` | Move one character to the right: forward, or backward in a line that reads right to left. |
| `save-buffer` | `C-x C-s` | Save the current buffer, prompting for a name if it has none. |
| `save-buffers-kill-terminal` | `C-x C-c` | Offer to save modified buffers, then exit. |
| `save-some-buffers` | `C-x s` | Offer to save each modified buffer. |
| `scroll-down-command` | `M-v` `<pgup>` | Move backward one screenful. |
| `scroll-up-command` | `C-v` `<pgdn>` | Move forward one screenful. |
| `select-to-beginning-of-line` | `S-<home>` | Select from point to the beginning of the line; a selection already active is extended. |
| `select-to-end-of-line` | `S-<end>` | Select from point to the end of the line; a selection already active is extended. |
| `self-insert-command` |  | Insert the character just typed. |
| `set-language` |  | Colour this buffer as another language, remembered for its file; its own language undoes it. |
| `set-mark-command` | `C-@` | Set the mark where point is. |
| `shell-beginning-of-line` |  | Move to the start of the input on the prompt's line, or of the line elsewhere. |
| `shell-command` | `M-!` | Run a shell command and show its output; with C-u, insert it at point. A command ending in & runs in the background. |
| `shell-command-on-region` | `M-\|` | Run a shell command with the region as its input and show its output; with C-u, replace the region with it. |
| `shell-delete-or-eof` |  | Delete the character after point, or at an empty prompt end the shell's input. |
| `shell-interrupt` |  | Interrupt what the shell is running, as C-c does in a terminal. |
| `shell-next-input` |  | Replace the input with the one sent after it. |
| `shell-previous-input` |  | Replace the input with the one sent before it. |
| `shell-send-input` |  | Send the line being typed to the shell. |
| `shell-toggle` | `M-t` | Go to the shell buffer, or back to the buffer it was entered from. |
| `sidebar-add` |  | Add a favourite: a file, a directory, a web link or a tool. |
| `sidebar-bookmark` |  | Add the file or directory at point in the side panel's tree to the favourites; again to remove it. |
| `sidebar-buffers` |  | Show the side panel's Buffers tab. |
| `sidebar-favorites` |  | Show the side panel's Favorites tab. |
| `sidebar-next-tab` |  | Show the side panel's next tab. |
| `sidebar-open` |  | Open the entry at point: fold or unfold a directory or group, show a file, buffer or definition in the other window, open a web link, run a tool. |
| `sidebar-outline` |  | Show the side panel's Outline tab. |
| `sidebar-parent` |  | Make the parent of the tree's root the root. |
| `sidebar-previous-tab` |  | Show the side panel's previous tab. |
| `sidebar-refresh` |  | Read the side panel's directories and favourites again. |
| `sidebar-remove` |  | Remove the favourite at point in the side panel, or kill the buffer there. |
| `sidebar-set-root` |  | Make the directory at point the root of the side panel's tree. |
| `sidebar-toggle` | `C-b` | Open the side panel, with the directory tree, favourites, open buffers and outline, and move into it; in the panel, close it. |
| `sidebar-toggle-hidden` |  | Show or hide dotfiles in the side panel's tree. |
| `sidebar-tree` |  | Show the side panel's Tree tab. |
| `skip-next-occurrence` | `C-M-s` | Drop the most recently added cursor and select the occurrence after it, skipping the word just passed over. |
| `sort-lines` |  | Sort the region's lines; with a prefix argument, in reverse. |
| `split-window-below` | `C-x 2` | Split the current window, stacking the new one beneath it. |
| `split-window-right` | `C-x 3` | Split the current window, placing the new one beside it. |
| `switch-to-buffer` | `C-x b` | Display another buffer, creating it if the name is new. |
| `symbol-overlay` | `C-.` | Highlight the symbol at point throughout the buffer. |
| `symbol-overlay-first` |  | Move to the first occurrence of the highlighted symbol at point. |
| `symbol-overlay-last` |  | Move to the last occurrence of the highlighted symbol at point. |
| `symbol-overlay-next` |  | Move to the next occurrence of the highlighted symbol at point. |
| `symbol-overlay-occur` |  | List the lines containing the highlighted symbol at point. |
| `symbol-overlay-prev` |  | Move to the previous occurrence of the highlighted symbol at point. |
| `symbol-overlay-rename` |  | Rename every occurrence of the highlighted symbol at point. |
| `toggle-line-numbers` | `C-x n` | Show or hide the line-number gutter. |
| `toggle-line-wrap` | `C-x x t` | Wrap lines wider than the window, or cut them off at its edge. |
| `transpose-chars` | `C-t` | Transpose the characters around point. |
| `transpose-words` |  | Transpose the words around point. |
| `undo` | `C-_` `C-x u` | Undo the most recent change. |
| `upcase-region` | `C-x C-u` | Convert the region to upper case. |
| `upcase-word` | `M-u` | Convert the following word to upper case. |
| `vc-next-action` | `C-x v v` | Save the file, stage it and commit it, with the message typed on the prompt. The commit holds this file alone, as C-x v v does in emacs. |
| `wdired-abort-changes` | wdired: `C-c C-k` | Throw the edited names away, and return to the listing. |
| `wdired-beginning-of-line` | wdired: `C-a` `<home>` | Move to the start of the file name, or of the line off an entry. |
| `wdired-change-to-wdired-mode` | dired: `C-x C-q` | Edit the file names in this listing as text, to rename the files. |
| `wdired-end-of-line` | wdired: `C-e` `<end>` | Move to the end of the file name, or of the line off an entry. |
| `wdired-exit` | wdired: `C-x C-q` | Return to the listing, asking whether to apply any edited names. |
| `wdired-finish-edit` | wdired: `C-c C-c` `C-x C-s` | Rename the files to the names as edited, and return to the listing. |
| `what-cursor-position` | `C-x =` | Describe the character at point and where point is. |
| `windmove-down` | `S-<down>` | Select the window below this one. |
| `windmove-left` | `S-<left>` | Select the window to the left of this one. |
| `windmove-right` | `S-<right>` | Select the window to the right of this one. |
| `windmove-up` | `S-<up>` | Select the window above this one. |
| `write-file` | `C-x C-w` | Save the current buffer under a new name. |
| `yank` | `C-y` | Insert the current kill-ring entry at point. |
| `yank-pop` | `M-y` | Replace the text just yanked with the next-older kill-ring entry. |
| `yazi` | `C-ESC`; dired: `Y` | Run the yazi file manager in this buffer's directory, then open what it chose; quitting yazi with nothing chosen quits nem. Needs yazi on the PATH. |
| `yazi-kill-buffer` | `ESC` | Kill this buffer, confirming if it is modified, then run yazi in its directory and open what it chose. Quitting yazi with nothing chosen quits nem. |
| `yazi-other-window` | `C-x y` | Run the yazi file manager, then open what it chose in the other window, splitting the frame if there is only one. Needs yazi on the PATH. |
| `zap-to-char` | `M-z` | Kill up to and including the next occurrence of a character, ARG times. |

234 commands in all.
