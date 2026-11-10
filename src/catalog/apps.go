package catalog

import "slices"

var Apps = slices.Concat(coreApps, AgentSkills)

const (
	HypervisorApp      = "tart"
	HypervisorLauncher = HypervisorApp + "-launcher"
)

var coreApps = []App{
	{Name: "homebrew", Description: "macOS package manager", Category: CategoryPackageManagers, Installs: OnMacOS(Script("https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh")), Provides: []string{"brew"}, Path: []string{"/opt/homebrew/bin", "/usr/local/bin"}, Version: VersionRead{Args: []string{"brew", "--version"}, Pattern: `^Homebrew (\S+)`}, Disk: diskHomebrew},
	{Name: "mise", Description: "Polyglot runtime manager: node, go, python, bun, rust, java", Category: CategoryPackageManagers, Installs: Everywhere(Brew("mise"), Script("https://mise.run")), Provides: []string{"mise"}, Disk: diskMise},
	{Name: "mas", Description: "Mac App Store command line", Category: CategoryPackageManagers, Installs: OnMacOS(Brew("mas")), Provides: []string{"mas"}},
	{Name: "uv", Description: "Fast Python package and project manager", Category: CategoryPackageManagers, Installs: Everywhere(Brew("uv"), Mise("uv", "latest")), Provides: []string{"uv"}, Disk: diskUv},
	{Name: "cocoapods", Description: "Dependency manager for iOS and macOS projects", Category: CategoryPackageManagers, Installs: OnMacOS(Brew("cocoapods")), Provides: []string{"pod"}, Disk: diskCocoapods},
	{Name: "pnpm", Description: "Fast, disk space efficient Node package manager", Category: CategoryPackageManagers, Installs: Everywhere(Brew("pnpm"), Mise("pnpm", "latest")), Provides: []string{"pnpm"}, Disk: diskPnpm},
	{Name: "skills", Description: "Agent skills package manager", Category: CategoryPackageManagers, Installs: Anywhere(Bun("skills")), Provides: []string{"skills"}, Requires: []string{"bun"}, Disk: diskSkills},

	{Name: "bun", Description: "Fast JavaScript runtime and package manager", Category: CategoryRuntimes, Installs: Anywhere(Mise("bun", "latest")), Provides: []string{"bun"}, Disk: diskBun},
	{Name: "go", Description: "Compiled systems programming language by Google", Category: CategoryRuntimes, Installs: Anywhere(Mise("go", "latest")), Provides: []string{"go"}, Disk: diskGo},
	{Name: "node", Description: "JavaScript runtime built on Chrome's V8 engine", Category: CategoryRuntimes, Installs: Anywhere(Mise("node", "lts")), Provides: []string{"node", "npm", "npx"}, Disk: diskNode},
	{Name: "java", Description: "Java Development Kit (Temurin)", Category: CategoryRuntimes, Installs: Anywhere(Mise("java", "temurin-21")), Provides: []string{"java"}, Version: VersionRead{Args: []string{"java", "-version"}, Pattern: `version "([^"]+)"`}},
	{Name: "python", Description: "Python programming language runtime", Category: CategoryRuntimes, Installs: Anywhere(Mise("python", "latest")), Provides: []string{"python3"}, Version: VersionRead{Args: []string{"python3", "--version"}, Pattern: `^Python (\S+)`}, Disk: diskPython},
	{Name: "rust", Description: "Systems programming language for safety and speed", Category: CategoryRuntimes, Installs: Anywhere(Mise("rust", "stable")), Provides: []string{"rustc", "cargo"}, Disk: diskRust},

	{Name: "ghostty", Description: "GPU-accelerated terminal emulator", Category: CategoryShellTerminal, Installs: OnMacOS(Cask("ghostty")), Bundle: "Ghostty", Then: []string{"ghostty-dotfiles"}, Disk: diskGhostty},
	{Name: "ohmyzsh", Description: "Oh My Zsh shell framework", Category: CategoryShellTerminal, Installs: Anywhere(ScriptUpgrade("https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh", []string{"RUNZSH=no", "CHSH=no", "KEEP_ZSHRC=yes"}, `ZSH="$HOME/.oh-my-zsh" exec sh "$HOME/.oh-my-zsh/tools/upgrade.sh"`)), Version: VersionRead{Path: "~/.oh-my-zsh", Args: []string{"git", "-C", "~/.oh-my-zsh", "rev-parse", "--short", "HEAD"}, Pattern: `^(\S+)`}, Disk: diskOhmyzsh},
	{Name: "tmux", Description: "Terminal multiplexer for persistent sessions", Category: CategoryShellTerminal, Installs: Everywhere(Brew("tmux"), Mise("tmux", "latest")), Provides: []string{"tmux"}, Then: []string{"tmux-dotfiles"}, Disk: diskTmux},
	{Name: "starship", Description: "Cross-shell prompt (path, git branch/status, versions)", Category: CategoryShellTerminal, Installs: Everywhere(Brew("starship"), Mise("starship", "latest")), Provides: []string{"starship"}, Then: []string{"starship-dotfiles"}, Disk: diskStarship},
	{Name: "zoxide", Description: "Smarter cd command", Replaces: "cd (jumping around)", Category: CategoryShellTerminal, Installs: Everywhere(Brew("zoxide"), Mise("zoxide", "latest")), Provides: []string{"zoxide"}, Disk: diskZoxide},

	{Name: "yazi", Description: "Terminal file manager", Replaces: "Finder in the terminal", Category: CategoryCLIApps, Installs: Everywhere(Brew("yazi"), Mise("yazi", "latest")), Provides: []string{"yazi"}, Disk: diskYazi},
	{Name: "bat", Description: "Cat clone with syntax highlighting", Replaces: "cat (for humans)", Category: CategoryCLIApps, Installs: Everywhere(Brew("bat"), Mise("bat", "latest")), Provides: []string{"bat"}, Disk: diskBat},
	{Name: "dust", Description: "Intuitive disk usage viewer", Replaces: "du -sh", Category: CategoryCLIApps, Installs: Everywhere(Brew("dust"), Mise("dust", "latest")), Provides: []string{"dust"}},
	{Name: "eza", Description: "Modern ls replacement", Replaces: "ls", Category: CategoryCLIApps, Installs: Everywhere(Brew("eza"), Mise("eza", "latest")), Provides: []string{"eza"}},
	{Name: "fd", Description: "Fast find alternative", Replaces: "find", Category: CategoryCLIApps, Installs: Everywhere(Brew("fd"), Mise("fd", "latest")), Provides: []string{"fd"}},
	{Name: "ripgrep", Description: "Fast grep, the rg agents and editors shell out to", Replaces: "grep (what agents and editors call)", Category: CategoryCLIApps, Installs: Everywhere(Brew("ripgrep"), Mise("ripgrep", "latest")), Provides: []string{"rg"}},
	{Name: "tgrep", Description: "Trigram-indexed grep for large codebases", Replaces: "grep -r", Category: CategoryCLIApps, Installs: OnMacOS(Brew("tgrep")), Provides: []string{"tgrep"}},
	{Name: "sd", Description: "Intuitive sed alternative", Replaces: "sed (simple replaces)", Category: CategoryCLIApps, Installs: Everywhere(Brew("sd"), Mise("sd", "latest")), Provides: []string{"sd"}},
	{Name: "difftastic", Description: "Structural diff", Replaces: "diff", Category: CategoryCLIApps, Installs: Everywhere(Brew("difftastic"), Mise("difftastic", "latest")), Provides: []string{"difft"}},
	{Name: "jq", Description: "Command-line JSON processor", Category: CategoryCLIApps, Installs: Everywhere(Brew("jq"), Mise("jq", "latest")), Provides: []string{"jq"}},

	{Name: "git", Description: "Distributed version control system, from xcode-select", Category: CategoryGit, Installs: OnMacOS(Shell("xcode-select --install")), Provides: []string{"git"}, Version: VersionRead{Args: []string{"git", "--version"}, Pattern: `git version (\S+)`}, Disk: diskGit},
	{Name: "gpg", Description: "GNU Privacy Guard for encryption and signing", Category: CategoryGit, Installs: OnMacOS(Brew("gnupg")), Provides: []string{"gpg"}, Then: []string{"git-signing"}, Disk: diskGpg},
	{Name: "lazygit", Description: "Terminal UI for git", Replaces: "raw git porcelain (interactive)", Category: CategoryGit, Installs: Everywhere(Brew("lazygit"), Mise("lazygit", "latest")), Provides: []string{"lazygit"}, Disk: diskLazygit},
	{Name: "gh", Description: "GitHub CLI for repository management", Replaces: "raw GitHub API / web UI", Category: CategoryGit, Installs: Everywhere(Brew("gh"), Mise("gh", "latest")), Provides: []string{"gh"}, Then: []string{"github-login"}, Disk: diskGh},

	{Name: "zed", Description: "Zed code editor", Category: CategoryEditors, Installs: OnMacOS(Cask("zed")), Bundle: "Zed", Then: []string{"zed-dotfiles"}, Disk: diskZed},
	{Name: "cursor", Description: "AI-powered code editor", Category: CategoryEditors, Installs: OnMacOS(Cask("cursor")), Bundle: "Cursor", Disk: diskCursor},
	{Name: "android-studio", Description: "Android development IDE", Category: CategoryEditors, Installs: OnMacOS(Cask("android-studio")), Bundle: "Android Studio", Disk: diskAndroidStudio},
	{Name: "xcode", Description: "Apple development IDE", Category: CategoryEditors, Installs: OnMacOS(Mas("Xcode", 497799835)), Provides: []string{"xcodebuild"}, Bundle: "Xcode", Version: VersionRead{Args: []string{"xcodebuild", "-version"}, Pattern: `^Xcode (\S+)`}, Disk: diskXcode},
	{Name: "xcodegen", Description: "Generates Xcode projects from a spec", Category: CategoryEditors, Installs: OnMacOS(Brew("xcodegen")), Provides: []string{"xcodegen"}},

	{Name: HypervisorApp, Description: "Virtual machines on Apple silicon: the engine of station's VMs", Category: CategoryContainersVMs, Installs: MacOnly(Shell(`d="$HOME/.local/share"; case "$XDG_DATA_HOME" in /*) d="$XDG_DATA_HOME" ;; esac; mkdir -p "$d/station/tart" "$d/station/vms" "$HOME/.local/bin" && curl -fsSL https://github.com/cirruslabs/tart/releases/latest/download/tart.tar.gz | tar -xz -C "$d/station/tart"`)), Provides: []string{"tart"}, Then: []string{"tart-launcher"}, Version: VersionRead{Args: []string{"tart", "--version"}, Pattern: `^(\S+)`}, Disk: diskTart},
	{Name: "docker", Description: "Container engine, inside the Linux VM", Category: CategoryContainersVMs, Installs: LinuxOnly(Script("https://get.docker.com")), Provides: []string{"docker"}, Version: VersionRead{Args: []string{"docker", "--version"}, Pattern: `Docker version ([^,\s]+)`}, Disk: diskDocker},
	{Name: "orbstack", Description: "Docker and Linux machines on the Mac, for exploration on a Mac", Category: CategoryContainersVMs, Installs: MacOnly(Cask("orbstack")), Provides: []string{"orb"}, Bundle: "OrbStack", Disk: diskOrbstack},
	{Name: "lens", Description: "Kubernetes IDE for managing clusters", Category: CategoryContainersVMs, Installs: OnMacOS(Cask("lens")), Bundle: "Lens"},
	{Name: "kubectl", Description: "Kubernetes command line", Category: CategoryContainersVMs, Installs: Everywhere(Brew("kubectl"), Mise("kubectl", "latest")), Provides: []string{"kubectl"}, Disk: diskKubectl},
	{Name: "helm", Description: "Kubernetes package manager", Category: CategoryContainersVMs, Installs: Everywhere(Brew("helm"), Mise("helm", "latest")), Provides: []string{"helm"}, Disk: diskHelm},
	{Name: "helmfile", Description: "Declarative deployments of Helm charts", Category: CategoryContainersVMs, Installs: Everywhere(Brew("helmfile"), Mise("helmfile", "latest")), Provides: []string{"helmfile"}},

	{Name: "ansible", Description: "Agentless IT automation and configuration management", Category: CategoryDeploy, Installs: Everywhere(Brew("ansible"), Mise("pipx:ansible", "latest")), Provides: []string{"ansible"}, Disk: diskAnsible},
	{Name: "eas", Description: "Expo's EAS CLI for app builds", Category: CategoryDeploy, Installs: Anywhere(Bun("eas-cli")), Provides: []string{"eas"}, Requires: []string{"bun"}, Disk: diskEas},
	{Name: "pulumi", Description: "Infrastructure as code using real programming languages", Category: CategoryDeploy, Installs: Everywhere(TrustedBrew("pulumi/tap/pulumi"), Mise("pulumi", "latest")), Provides: []string{"pulumi"}, Disk: diskPulumi},
	{Name: "terraform", Description: "Infrastructure as code provisioning", Category: CategoryDeploy, Installs: Everywhere(TrustedBrew("hashicorp/tap/terraform"), Mise("terraform", "latest")), Provides: []string{"terraform"}, Disk: diskTerraform},

	{Name: "claude", Description: "Claude Code CLI for agentic coding", Category: CategoryAIAgents, Installs: Anywhere(Script("https://claude.ai/install.sh")), Provides: []string{"claude"}, Version: VersionRead{Args: []string{"claude", "--version"}, Pattern: `^(\S+)`}, Disk: diskClaude},
	{Name: "codex", Description: "OpenAI's CLI coding agent", Category: CategoryAIAgents, Installs: Everywhere(Bun("@openai/codex"), Mise("codex", "latest")), Provides: []string{"codex"}, Requires: []string{"bun"}, Disk: diskCodex},
	{Name: "gemini", Description: "Google's CLI coding agent", Category: CategoryAIAgents, Installs: Everywhere(Brew("gemini-cli"), Mise("gemini", "latest")), Provides: []string{"gemini"}, Disk: diskGemini},
	{Name: "opencode", Description: "Open-source AI coding agent CLI", Category: CategoryAIAgents, Installs: Everywhere(Brew("opencode"), Mise("opencode", "latest")), Provides: []string{"opencode"}, Disk: diskOpencode},
	{Name: "hermes", Description: "Self-improving AI agent with persistent memory", Category: CategoryAIAgents, Installs: VMs(Script("https://hermes-agent.nousresearch.com/install.sh", "--skip-setup"), Script("https://hermes-agent.nousresearch.com/install.sh", "--skip-setup")), Provides: []string{"hermes"}, Version: VersionRead{Args: []string{"hermes", "--version"}, Pattern: `(\d+\.\d+\S*)`}, Disk: diskHermes},
	{Name: "conductor", Description: "Runs parallel AI coding agents in git worktrees", Category: CategoryAIAgents, Installs: MacOnly(Cask("conductor")), Bundle: "Conductor"},
	{Name: "herdr", Description: "Agent multiplexer that lives in your terminal", Category: CategoryAIAgents, Installs: Everywhere(Brew("herdr"), Mise("herdr", "latest")), Provides: []string{"herdr"}},
	{Name: "herdr-app", Description: "Desktop app that keeps coding agents running in the background", Category: CategoryAIAgents, Installs: MacOnly(Cask("herdr-app")), Bundle: "Herdr", Disk: diskHerdrApp},

	{Name: "ollama", Description: "Run large language models locally, on the Mac's GPU", Category: CategoryAITooling, Installs: MacOnly(Cask("ollama-app")), Provides: []string{"ollama"}, Bundle: "Ollama", Live: Liveness{Process: "ollama"}, Disk: diskOllama},
	{Name: "qmd", Description: "Local semantic search over markdown", Replaces: "grep over markdown notes", Category: CategoryAITooling, Installs: Anywhere(BunAs("github:tobi/qmd", "qmd")), Provides: []string{"qmd"}, Requires: []string{"bun"}, Disk: diskQmd},
	{Name: "rtk", Description: "Token-optimized CLI proxy for agents", Replaces: "raw dev commands (token-heavy output)", Category: CategoryAITooling, Installs: Everywhere(Brew("rtk"), Mise("rtk", "latest")), Provides: []string{"rtk"}, Disk: diskRtk},
	{Name: "browser-use", Description: "AI agent library for browser automation", Category: CategoryAITooling, Installs: VMs(Uv("browser-use"), Mise("pipx:browser-use", "latest")), Provides: []string{"browser-use"}, Requires: []string{"uv"}},
	{Name: "markitdown", Description: "Convert files to Markdown for LLMs", Category: CategoryAITooling, Installs: VMs(Uv("markitdown[all]"), Mise("pipx:markitdown[all]", "latest")), Provides: []string{"markitdown"}, Requires: []string{"uv"}},
	{Name: "playwright-mcp", Description: "Browser automation for AI agents via MCP", Category: CategoryAITooling, Installs: VMs(Npm("@playwright/mcp"), Mise("npm:@playwright/mcp", "latest")), Requires: []string{"node"}},
	{Name: "playwright-browsers", Description: "Chromium runtime for website specs (@jterrazz/test)", Category: CategoryAITooling, Installs: VMs(Shell("npx playwright install chromium"), Shell("npx playwright install chromium")), Requires: []string{"node"}, Version: VersionRead{Path: "~/Library/Caches/ms-playwright/chromium*"}, Disk: diskPlaywrightBrowsers},
	{Name: "inferrs", Description: "TurboQuant LLM inference server", Category: CategoryAITooling, Installs: OnMacOS(TrustedBrew("ericcurtin/inferrs/inferrs")), Provides: []string{"inferrs"}},
	{Name: "agent-browser", Description: "Browser automation CLI for AI agents", Category: CategoryAITooling, Installs: VMs(Bun("agent-browser"), Bun("agent-browser")), Provides: []string{"agent-browser"}, Requires: []string{"bun"}, Disk: diskAgentBrowser},
	{Name: "plannotator", Description: "Review AI agent plans and code before committing", Category: CategoryAITooling, Installs: Anywhere(Script("https://plannotator.ai/install.sh")), Provides: []string{"plannotator"}, Version: VersionRead{Args: []string{"plannotator", "--version"}, Pattern: `(\d+\.\d+\S*)`}},

	{Name: "chatgpt", Description: "OpenAI ChatGPT desktop app", Category: CategoryAIApps, Installs: MacOnly(Cask("chatgpt")), Bundle: "ChatGPT"},
	{Name: "claude-desktop", Description: "Anthropic Claude desktop app", Category: CategoryAIApps, Installs: MacOnly(Cask("claude")), Bundle: "Claude", Disk: diskClaudeDesktop},
	{Name: "typewhisper", Description: "Whisper-based dictation app", Category: CategoryAIApps, Installs: MacOnly(Cask("typewhisper")), Bundle: "TypeWhisper"},

	{Name: "brave", Description: "Privacy-focused web browser", Category: CategoryBrowsers, Installs: MacOnly(Cask("brave-browser")), Bundle: "Brave Browser"},
	{Name: "dia", Description: "The Browser Company's AI-first browser", Category: CategoryBrowsers, Installs: MacOnly(AppFrom("download it from its website")), Bundle: "Dia"},

	{Name: "discord", Description: "Voice and text chat", Category: CategoryCommunication, Installs: MacOnly(Cask("discord")), Bundle: "Discord", Disk: diskDiscord},
	{Name: "slack", Description: "Team communication", Category: CategoryCommunication, Installs: MacOnly(Cask("slack")), Bundle: "Slack"},
	{Name: "whatsapp", Description: "Messaging app", Category: CategoryCommunication, Installs: MacOnly(Cask("whatsapp")), Bundle: "WhatsApp"},
	{Name: "messenger", Description: "Facebook Messenger", Category: CategoryCommunication, Installs: MacOnly(AppFrom("no current source")), Bundle: "Messenger"},

	{Name: "linear", Description: "Project management", Category: CategoryProductivity, Installs: MacOnly(Cask("linear-linear")), Bundle: "Linear"},
	{Name: "notion", Description: "Workspace for notes and docs", Category: CategoryProductivity, Installs: MacOnly(Cask("notion")), Bundle: "Notion", Disk: diskNotion},
	{Name: "obsidian", Description: "Knowledge base and note-taking", Category: CategoryProductivity, Installs: MacOnly(Cask("obsidian")), Bundle: "Obsidian", Disk: diskObsidian},
	{Name: "keynote", Description: "Apple presentations", Category: CategoryProductivity, Installs: MacOnly(Mas("Keynote", 361285480)), Bundle: "Keynote"},
	{Name: "numbers", Description: "Apple spreadsheets", Category: CategoryProductivity, Installs: MacOnly(Mas("Numbers", 361304891)), Bundle: "Numbers"},
	{Name: "pages", Description: "Apple word processor", Category: CategoryProductivity, Installs: MacOnly(Mas("Pages", 361309726)), Bundle: "Pages"},
	{Name: "raindrop", Description: "Bookmark manager", Category: CategoryProductivity, Installs: MacOnly(Mas("Save to Raindrop.io", 1549370672)), Bundle: "Save to Raindrop.io"},

	{Name: "broadcasts", Description: "Internet radio and stream player", Category: CategoryMedia, Installs: MacOnly(Mas("Broadcasts", 1469995354)), Bundle: "Broadcasts"},
	{Name: "compressor", Description: "Apple video compression", Category: CategoryMedia, Installs: MacOnly(Mas("Compressor", 424390742)), Bundle: "Compressor"},
	{Name: "final-cut-pro", Description: "Professional video editor", Category: CategoryMedia, Installs: MacOnly(Mas("Final Cut Pro", 424389933)), Bundle: "Final Cut Pro"},
	{Name: "lightroom", Description: "Adobe photo editor", Category: CategoryMedia, Installs: MacOnly(Mas("Adobe Lightroom", 1451544217)), Bundle: "Adobe Lightroom"},
	{Name: "logic-pro", Description: "Professional music making", Category: CategoryMedia, Installs: MacOnly(Mas("Logic Pro", 634148309)), Bundle: "Logic Pro"},

	{Name: "jump-desktop-connect", Description: "Remote GUI recovery for headless Macs", Category: CategoryRemoteAccess, Installs: MacOnly(Cask("jump-desktop-connect")), Bundle: "Jump Desktop Connect"},
	{Name: "jump-desktop", Description: "Jump Desktop viewer/client app", Category: CategoryRemoteAccess, Installs: MacOnly(Cask("jump-desktop")), Bundle: "Jump Desktop"},
	{Name: "tailscale", Description: "The Tailscale app: each Mac is a node, and station reads it", Category: CategoryRemoteAccess, Installs: MacOnly(Cask("tailscale-app")), Supersedes: "tailscale", AppStore: true, Provides: []string{"tailscale"}, Bundle: "Tailscale", Then: []string{TailscaleCLISetting}, Disk: diskTailscale},

	{Name: "bitwarden", Description: "Password manager", Category: CategorySecurityApps, Installs: MacOnly(Cask("bitwarden")), Bundle: "Bitwarden"},
	{Name: "adguard", Description: "Ad blocker for Safari", Category: CategorySecurityApps, Installs: MacOnly(Mas("AdGuard for Safari", 1440147259)), Bundle: "AdGuard for Safari"},
	{Name: "passepartout", Description: "VPN client", Category: CategorySecurityApps, Installs: MacOnly(Mas("Passepartout", 1433648537)), Bundle: "Passepartout"},

	{Name: "mole", Description: "Tidy, uninstall, analyze, and optimize macOS from the terminal", Category: CategorySystemUtilities, Installs: MacOnly(Brew("mole")), Provides: []string{"mo"}, Disk: diskMole},
	{Name: "betterdisplay", Description: "Flexible HiDPI scaling and display management", Category: CategorySystemUtilities, Installs: MacOnly(Cask("betterdisplay")), Bundle: "BetterDisplay"},
	{Name: "pipifier", Description: "Picture-in-Picture for Safari", Category: CategorySystemUtilities, Installs: MacOnly(Mas("PiPifier", 1160374471)), Bundle: "PiPifier"},
	{Name: "snippety", Description: "Code snippet manager", Category: CategorySystemUtilities, Installs: MacOnly(Mas("Snippety", 1530751461)), Bundle: "Snippety"},
	{Name: "speedtest", Description: "Internet speed test", Category: CategorySystemUtilities, Installs: MacOnly(Mas("Speedtest", 1153157709)), Bundle: "Speedtest"},
}
