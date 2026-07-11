# settings-demo.tape — a tour of the settings menu (read-only, no mutations).
#
# Flow: launch rulem -> "Update settings" -> view the repositories list ->
# open a repository's actions menu -> browse the available actions -> back out.
# No destructive action is taken.
#
# Render (requires ttyd + ffmpeg on PATH):
#   ./demo/setup-demo-env.sh && vhs demo/settings-demo.tape
# Run this from the repository root.

Output demo/gifs/settings.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 750
Set Padding 24
Set Theme "Dracula"

# --- Hidden setup: point rulem at the demo sandbox.
Hide
Type 'export XDG_CONFIG_HOME="$PWD/demo/sandbox/config"' Enter
Type 'PS1="$ "' Enter
Type 'cd demo/sandbox/project' Enter
Type 'clear' Enter
Show

# --- Launch the app.
Type '../../rulem-demo' Sleep 500ms Enter
Sleep 2.5s

# --- Main menu: move down to "Update settings" and open it.
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 1s
Enter
Sleep 2.5s

# --- Repositories list: open the "Demo Rules" repository's actions.
Sleep 1.5s
Enter
Sleep 2s

# --- Repository actions menu: browse the available actions (no selection).
Down
Sleep 1.5s
Down
Sleep 1.5s
Up
Sleep 1.5s

# --- Back out: actions -> repo list -> main menu, then quit.
Escape
Sleep 1.5s
Escape
Sleep 1.5s
Type "q"
Sleep 1.5s
