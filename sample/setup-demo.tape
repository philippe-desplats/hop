# hop setup wizard demo tape.
#
# Regenerate the README GIF from the repo root with:
#     vhs sample/setup-demo.tape
#
# Requires: vhs (brew install vhs) and hop on your PATH.
# The hidden setup sources sample/setup-demo.sh, which builds a throwaway $HOME
# with sample project folders and stub tools, so recording never touches your
# real home, config or index.

Output docs/demo-setup.gif

Require hop

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 65ms

# --- Hidden setup: isolated fake home + stub tools ---
# A static prompt (no working-directory token) keeps any real path off-screen.
Hide
Type "source sample/setup-demo.sh" Enter
Sleep 3000ms
Type "PS1='$ '" Enter
Type "clear" Enter
Sleep 500ms
Show

# Launch the guided setup.
Type "hop setup" Sleep 700ms Enter
Sleep 1900ms

# 1. Folder discovery: ~/Projects, ~/Developments and a custom ~/Labs, all found
#    (the last by content) and preselected. Move through the list, then continue.
Down Sleep 800ms
Down Sleep 1100ms
Enter Sleep 1500ms

# 2. Pick an editor from the detected list.
Down Sleep 900ms
Enter Sleep 1400ms

# 3. Pick the AI assistant for the c / r keys.
Down Sleep 900ms
Enter Sleep 1400ms

# 4. Offer to wire the integration into ~/.zshrc (keep "Yes").
Sleep 1600ms
Enter Sleep 1400ms

# 5. Confirm: review the summary, then save, scan and wire.
Sleep 2000ms
Enter Sleep 2400ms
