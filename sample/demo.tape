# hop demo tape.
#
# Regenerate the README GIF from the repo root with:
#     vhs sample/demo.tape
#
# Requires: vhs (brew install vhs) and hop on your PATH.
# The hidden setup sources sample/demo.sh, which indexes only a throwaway copy
# of sample/projects, so recording never touches your real hop index.

Output docs/demo.gif

Require hop

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 65ms

# --- Hidden setup: isolated demo environment ---
Hide
Type "source sample/demo.sh" Enter
Sleep 2500ms
# Recording-only prompt: show the working directory (cyan) so cd jumps are visible.
Type `PS1="\[\e[36m\]\w\[\e[0m\] \$ "` Enter
# Start from a neutral home dir, no real repo path on screen.
Type "cd ~" Enter
Type "clear" Enter
Sleep 500ms
Show

# 1. One keyword: jump straight to the best match.
Type "p blog" Sleep 700ms Enter
Sleep 1500ms

# 2. Several keywords scope by ordered sub-path: "work" then "api".
Type "p work api" Sleep 900ms Enter
Sleep 1700ms

# 3. No argument opens the interactive fuzzy Hub.
Type "p" Sleep 600ms Enter
Sleep 1500ms

# 4. Fuzzy-filter the list.
Type "web" Sleep 1500ms

# 5. Tab opens the per-project action menu.
Tab Sleep 1600ms

# 6. Pin it (marked with a star), which returns to the list.
Type "p" Sleep 1500ms

# 7. Enter drops into the project.
Enter Sleep 1400ms

# 8. Jump back to the previous project.
Type "p -" Sleep 600ms Enter
Sleep 1500ms

# 9. Track any folder, even a non-git one, to add it to the search list.
Type "hop track ~/hop-demo/downloads" Sleep 700ms Enter
Sleep 1500ms

# 10. It is now jumpable like any other project.
Type "p down" Sleep 700ms Enter
Sleep 1600ms

# 11. Everything is configurable from an interactive editor (no saving here).
Type "hop config" Sleep 500ms Enter
Sleep 1600ms
Down Sleep 700ms
Right Sleep 1100ms
Down Sleep 600ms
Down Sleep 800ms
Escape
Sleep 900ms
