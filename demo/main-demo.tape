# main-demo.tape — the hero demo shown at the top of the README.
#
# Flow: launch rulem -> main menu -> "Save rules file" -> browse the file
# picker with its live markdown preview -> pick a file -> rename it -> save.
#
# Render (requires ttyd + ffmpeg on PATH):
#   ./demo/setup-demo-env.sh && vhs demo/main-demo.tape
# Run this from the repository root.

Output demo/gifs/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 750
Set Padding 24
Set Theme "Dracula"

# --- Hidden setup: point rulem at the demo sandbox and enter the project dir.
Hide
Type 'export XDG_CONFIG_HOME="$PWD/demo/sandbox/config"' Enter
Type 'PS1="$ "' Enter
Type 'cd demo/sandbox/project' Enter
Type 'clear' Enter
Show

# --- Launch the app (../../rulem-demo == demo/rulem-demo from the project dir).
Type '../../rulem-demo' Sleep 500ms Enter
Sleep 2.5s

# --- Main menu: "Save rules file" is already highlighted at the top.
Sleep 1.5s
Enter
Sleep 2s

# --- File picker: browse the local rule files, pausing on each live preview.
Down
Sleep 2.5s
Up
Sleep 2s

# --- Select the highlighted file to save.
Enter
Sleep 1.5s

# --- Filename step: clear the pre-filled name and type a new destination name.
Ctrl+U
Sleep 500ms
Type "coding-style.md"
Sleep 1.5s
Enter
Sleep 2.5s

# --- Success screen. Back to the main menu, then quit.
Type "m"
Sleep 1.5s
Type "q"
Sleep 1.5s
