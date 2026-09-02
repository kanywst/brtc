Output assets/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
# 24 rows for the first command's box, 27 for the second, plus the typed
# command and the prompt after it. At 600 the top border and the title
# line scrolled off the top of the frame.
Set Height 700
Set TypingSpeed 0.1
Set Theme "Dracula"
Set Margin 20
Set MarginFill "#282A36"
Set BorderRadius 10

# The 'brtc' binary is expected to already exist in the current working
# directory before VHS runs (built by the caller). The CI workflow does
# 'go build -o brtc .' beforehand; locally, run the same command first.
Hide
Type "export PATH=$PWD:$PATH"
Enter
Type "clear"
Enter
Show


# Step 1: Show a weak password
Type "brtc --algo bcrypt --hw rtx-4090 'shortpass'"
Sleep 500ms
Enter
Sleep 3.5s

Type "clear"
Enter

# Step 2: Show a strong password with a different hardware profile and budget
Type "brtc --algo bcrypt --cost 12 --hw aws-p5.48xlarge --budget 500000usd 'Tr0ub4dor&3'"
Sleep 500ms
Enter
Sleep 4s
