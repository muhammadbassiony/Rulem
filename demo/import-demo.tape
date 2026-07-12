# import-demo.tape — importing a rule from the central repo into a project.
#
# Flow: launch rulem -> "Import rules (Copy)" -> browse the central repo's rule
# files in the picker (live preview) -> pick one -> choose the AGENTS.md target
# -> choose copy mode -> confirm. The file lands in the project directory.
#
# Render (requires ttyd + ffmpeg on PATH):
#   ./demo/setup-demo-env.sh && vhs demo/import-demo.tape
# Run this from the repository root.

Output demo/gifs/import.gif

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

# --- Launch the app.
Type '../../rulem-demo' Sleep 500ms Enter
Sleep 2.5s

# --- Main menu: move down to "Import rules (Copy)" and open it.
Down
Sleep 1.5s
Enter
Sleep 2s

# --- File picker over the central repo: browse a couple of files' previews.
Down
Sleep 2.5s
Down
Sleep 2.5s

# --- Select the highlighted rule file to import.
Enter
Sleep 2s

# --- Editor target list: filter to AGENTS.md, then select it.
Type "/"
Type "AGENTS"
Sleep 1.5s
Enter
Sleep 1s
Enter
Sleep 2s

# --- Import mode list: keep the default (Copy) and confirm.
Enter
Sleep 1.5s

# --- Confirmation screen: accept.
Type "y"
Sleep 2.5s

# --- Success screen. Back to the main menu, then quit.
Type "m"
Sleep 1.5s
Type "q"
Sleep 1.5s
