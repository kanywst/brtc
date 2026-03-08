Output assets/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600
Set TypingSpeed 0.1
Set Theme "Dracula"
Set Margin 20
Set MarginFill "#282A36"
Set BorderRadius 10

Hide
Type "go build -o brtc"
Enter
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
