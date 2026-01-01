# Grove Demo - Git worktree management made simple
# Run with: vhs .github/demo.tape

Output .github/demo.mp4

Set Theme "Catppuccin Mocha"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 14
Set Width 1200
Set Height 700
Set Padding 20
Set TypingSpeed 50ms

Env PS1 "[\W] $ "
Env HOME "/tmp/grove-demo"

# Set up clean environment and shell integration
Hide
Type "rm -rf /tmp/grove-demo && mkdir -p ~/projects && cd ~/projects"
Enter
Type `eval "$(grove switch shell-init)"`
Enter
Type "clear"
Enter
Sleep 500ms
Show

# Clone the repo
Type "grove clone https://github.com/sQVe/grove demo"
Enter
Sleep 5s

Type "cd demo"
Enter
Sleep 500ms

Type "grove list"
Enter
Sleep 2s

# Start feature work - notice prompt changes to [feat-new-feature]
Type "grove add feat/new-feature --switch"
Enter
Sleep 2s

# Create a file in this worktree
Type "echo 'new feature code' > feature.txt"
Enter
Sleep 500ms

Type "git status --short"
Enter
Sleep 2s

# Urgent hotfix! Switch to main - prompt changes to [main]
Type "# Urgent hotfix needed!"
Enter
Sleep 500ms

Type "grove switch main"
Enter
Sleep 1s

# Prove we're in a different directory - feature.txt doesn't exist here
Type "ls feature.txt"
Enter
Sleep 2s

# Create hotfix worktree
Type "grove add hotfix/urgent --switch"
Enter
Sleep 2s

# Show all worktrees exist as separate directories
Type "ls .."
Enter
Sleep 3s

# Back to feature - our file is still there, no stashing!
Type "grove switch feat-new-feature"
Enter
Sleep 1s

Type "cat feature.txt"
Enter
Sleep 2s

Type "grove list"
Enter
Sleep 3s
