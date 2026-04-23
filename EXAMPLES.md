# bfzf — 50 Usage Examples

A practical reference demonstrating the breadth of `bfzf` in shell pipelines, scripts, and Go programs.

---

## File & Directory Navigation

**1. Pick a file in the current directory**
```bash
ls | bfzf
```

**2. Pick any file recursively**
```bash
find . -type f | bfzf
```

**3. Pick a file and open in $EDITOR**
```bash
$EDITOR "$(find . -type f | bfzf)"
```

**4. Pick a directory and cd into it**
```bash
cd "$(find . -type d | bfzf)"
```

**5. Pick a file with bat preview**
```bash
find . -type f | bfzf --preview 'bat --color=always {}'
```

**6. Pick a file from eza long listing (last field = filename)**
```bash
eza -l | bfzf --preview 'bat --color=always {-1}'
```

**7. Pick and stat a file**
```bash
ls | bfzf --preview 'stat {}'
```

**8. Multi-select files and copy to a destination**
```bash
src=$(ls | bfzf -m)
echo "$src" | xargs -I{} cp {} /tmp/backup/
```

**9. Preview file content in bottom pane**
```bash
find . -name "*.md" | bfzf --preview 'cat {}' \
  --preview-position bottom --preview-size 50
```

**10. Pick a recently modified file (sorted by mtime)**
```bash
find . -type f -printf '%T@ %p\n' | sort -rn | awk '{print $2}' | bfzf
```

---

## Git

**11. Checkout a branch**
```bash
git checkout "$(git branch --all | sed 's/^\* //' | bfzf)"
```

**12. Git log with diff preview**
```bash
git log --oneline | bfzf --preview 'git show --stat {1}'
```

**13. Stage files interactively**
```bash
git diff --name-only | bfzf -m | xargs git add
```

**14. Open a changed file in editor**
```bash
$EDITOR "$(git diff --name-only | bfzf)"
```

**15. Pick a tag and show its message**
```bash
git tag | bfzf --preview 'git tag -v {} 2>&1 || git show {} --stat'
```

**16. Delete a local branch**
```bash
git branch -d "$(git branch | grep -v '^\*' | bfzf)"
```

**17. Cherry-pick a commit from log**
```bash
git cherry-pick "$(git log --oneline | bfzf | awk '{print $1}')"
```

**18. Pick a stash and apply it**
```bash
git stash apply "$(git stash list | bfzf | awk -F: '{print $1}')"
```

---

## Process & System

**19. Kill a process**
```bash
kill "$(ps aux | bfzf --header-lines 1 | awk '{print $2}')"
```

**20. SSH into a host from ~/.ssh/config**
```bash
ssh "$(grep '^Host ' ~/.ssh/config | awk '{print $2}' | bfzf)"
```

**21. Mount a disk image (macOS)**
```bash
hdiutil mount "$(find ~ -name '*.dmg' | bfzf)"
```

**22. Pick a running Docker container**
```bash
docker exec -it "$(docker ps --format '{{.Names}}' | bfzf)" bash
```

**23. Remove a Docker image**
```bash
docker rmi "$(docker images --format '{{.Repository}}:{{.Tag}}' | bfzf)"
```

**24. Pick a systemd unit and show logs**
```bash
systemctl list-units --no-pager | bfzf --preview 'journalctl -u {1} -n 40'
```

---

## Shell History & Navigation

**25. Replay a history command**
```bash
eval "$(history | awk '{$1=""; print substr($0,2)}' | bfzf)"
```

**26. Jump to a recent directory (with zoxide)**
```bash
cd "$(zoxide query -l | bfzf)"
```

**27. Pick a tmux session to attach**
```bash
tmux attach -t "$(tmux list-sessions -F '#{session_name}' | bfzf)"
```

---

## Package Management

**28. Install an apt package with description preview**
```bash
apt-cache pkgnames | bfzf --preview 'apt-cache show {}'
```

**29. Remove an installed apt package**
```bash
dpkg --get-selections | awk '{print $1}' | bfzf -m | xargs sudo apt remove
```

**30. Install a brew formula with info preview**
```bash
brew formulae | bfzf --preview 'brew info {}'
```

**31. Pick and install a npm global package**
```bash
npm search --json '' 2>/dev/null | jq -r '.[].name' | bfzf
```

---

## Text & Data Processing

**32. Pick a line from a CSV and print it**
```bash
cat data.csv | bfzf
```

**33. Filter and edit a config key**
```bash
cat ~/.config/myapp/config.toml | bfzf | xargs -I{} echo "Chosen: {}"
```

**34. Pick a word from a dictionary**
```bash
cat /usr/share/dict/words | bfzf
```

**35. Pick a JSON field value to inspect**
```bash
cat data.json | jq -r 'keys[]' | bfzf --preview 'cat data.json | jq ".{}"'
```

---

## Development Workflow

**36. Run a Make target interactively**
```bash
make "$(grep -E '^[a-zA-Z_-]+:' Makefile | awk -F: '{print $1}' | bfzf)"
```

**37. Open a TODO comment in the editor**
```bash
grep -rn "TODO" . --include='*.go' | bfzf | awk -F: '{print "+"$2" "$1}' | xargs $EDITOR
```

**38. Pick a Go test to run**
```bash
go test -list '.*' ./... 2>/dev/null | bfzf | xargs -I{} go test -run {} ./...
```

**39. Jump to a function definition**
```bash
ctags -R --output-format=json . 2>/dev/null | \
  jq -r '.name + "\t" + .path' | bfzf | awk '{print $2}' | xargs $EDITOR
```

**40. Pick a Kubernetes pod to describe**
```bash
kubectl get pods | bfzf --preview 'kubectl describe pod {1}'
```

**41. Delete a Kubernetes resource**
```bash
kubectl get all | bfzf -m | awk '{print $1}' | xargs kubectl delete
```

---

## bfzf CLI — Style & Appearance

**42. Full border preset**
```bash
ls | bfzf --style full --header "Files" --preview 'cat {}'
```

**43. Custom color scheme**
```bash
ls | bfzf --color "fg+:212,hl:220,border:99"
```

**44. Custom cursor and markers (multi-select)**
```bash
ls | bfzf -m --cursor '→ ' --marker checkmarks
```

**45. Inline height mode (below cursor, 15 lines)**
```bash
ls | bfzf --height 15
```

**46. Percentage height mode**
```bash
ls | bfzf --height 40%
```

**47. Popup mode in tmux (center)**
```bash
ls | bfzf --popup center
```

**48. Popup on the left, 40% wide**
```bash
ls | bfzf --popup left,40%
```

**49. No input, pure navigation list**
```bash
printf 'Option A\nOption B\nOption C' | bfzf --no-input
```

**50. Full pipeline: pick, preview, open**
```bash
find . -name "*.go" | \
  bfzf --style full \
       --header "Go source files" \
       --preview 'bat --color=always --line-range :80 {}' \
       --preview-position right \
       --preview-size 55 | \
  xargs $EDITOR
```

---

## Go Library Snippets

**Multi-select with checkmark markers and color override**
```go
m := bfzf.New(items,
    bfzf.WithLimit(0),
    bfzf.WithMarkerStyle(bfzf.MarkerCheckmarks),
    bfzf.WithColor("fg+:212,hl:220"),
    bfzf.WithPreset(bfzf.PresetFull),
    bfzf.WithListTitle("Choose items"),
)
```

**Embedded component at a fixed height**
```go
m := bfzf.New(items,
    bfzf.WithHeight(12),
    bfzf.WithPreview(func(item bfzf.Item) string {
        return "Detail: " + item.Label()
    }),
)
```

**Custom key binding overrides**
```go
m := bfzf.New(items,
    bfzf.WithKeyMapFunc(func(km *bfzf.KeyMap) {
        km.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
    }),
)
```
