package main

import (
	"fmt"
	"math"
)

type (
	MainCategory struct {
		Name          string
		SubCategories []SubCategory
	}

	SubCategory struct {
		Name        string
		Description string
		Score       int // 1 to 100
	}
)

func (mc MainCategory) Score() int {
	var score int
	for _, sc := range mc.SubCategories {
		score += sc.Score
	}
	return int(math.Round(float64(score) / float64(len(mc.SubCategories)) * 100.0))
}

var (
	mainCategoryIndex int
	subCategoryIndex  int
)

func clamp(min, x, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func GetCurrentCategory() string {
	return fmt.Sprintf(
		"%s (%s)",
		Categories[mainCategoryIndex].SubCategories[subCategoryIndex].Name,
		Categories[mainCategoryIndex].Name,
	)
}

func ApplyRating(score int) {
	score = clamp(0, score, 100)
	Categories[mainCategoryIndex].SubCategories[subCategoryIndex].Score = score
	subCategoryIndex++
	if subCategoryIndex >= len(Categories[mainCategoryIndex].SubCategories) {
		subCategoryIndex = 0
		mainCategoryIndex++
	}
	if mainCategoryIndex >= len(Categories) {
		mainCategoryIndex = 0
		subCategoryIndex = 0
		// no more categories, end the program
		teaProgram.Quit()
		return
	}

	// initialize chat for the new category
	aiMessageHistory = nil
	Begin()
}

var Categories = []MainCategory{
	{
		Name: "Computer Science",
		SubCategories: []SubCategory{
			{
				Name:        "Algorithms & Data Structures",
				Description: "Algorithms and data structures. Sorting, searching, ect. Trees, graphs, etc. Big-O notation, time complexity, space complexity, etc.",
			},
			{
				Name:        "Discrete Mathematics & Logic",
				Description: "Set theory, logic, probability, etc.",
			},
			{
				Name:        "Version control systems",
				Description: "3-way merge, conflictless merge, snapshot vs diff, etc.",
			},
			{
				Name:        "Compression",
				Description: "Compression, Huffman coding, lossy vs lossless, etc.",
			},
		},
	},
	{
		Name: "Low-Level and Systems Programming",
		SubCategories: []SubCategory{
			{
				Name:        "Operating Systems Principles",
				Description: "Processes, threads, memory management, etc. File systems, networking, etc.",
			},
			{
				Name:        "Computer Architecture",
				Description: "CPUs, memory, caches, etc.",
			},
			{
				Name:        "Low-Level Programming",
				Description: "Assembly, C, C++, Rust, Zig, etc.",
			},
		},
	},
	{
		Name: "Web",
		SubCategories: []SubCategory{
			{
				Name:        "Frontend",
				Description: "HTML, CSS, (Sass, Tailwind), JavaScript, TypeScript, etc.",
			},
			{
				Name:        "Web UI Libraries and Frameworks",
				Description: "React/Vue/Angular/Svelte/Solid, Next.js, HTMX, Alpine.js, ect.",
			},
			{
				Name:        "DOM",
				Description: "Document Object Model, events, TSX, virtual DOM, canvas, webgl/webgpu, etc.",
			},
			{
				Name:        "Backend",
				Description: "Go, Node/Deno/Bun, .NET",
			},
			{
				Name:        "High-level Networking",
				Description: "HTTP, TCP/UDP, DNS, WebSockets, etc.",
			},
		},
	},
	{
		Name: "Infrastructure",
		SubCategories: []SubCategory{
			{
				Name:        "Cloud",
				Description: "AWS, GCP, Azure, DigitalOcean, Cloudflare, Vercel, etc.",
			},
			{
				Name:        "Serverless",
				Description: "AWS Lambda, Vercel Edge Functions, Next.js, etc.",
			},
			{
				Name:        "Virtualization",
				Description: "Virtual machines, containers, Vagrant, Docker, etc.",
			},
			{
				Name:        "Low-Level Networking",
				Description: "DNS, DHCP, IPv4, IPv6, VPN, etc.",
			},
		},
	},
	{
		Name: "Data formats",
		SubCategories: []SubCategory{
			{
				Name:        "JSON",
				Description: "JavaScript Object Notation, Schema, Progressive JSON, JSON5, JSONC, etc.",
			},
			{
				Name:        "YAML",
				Description: "Anchors, aliases, etc.",
			},
			{
				Name:        "CSV",
				Description: "Separator, quoting, escaping, etc.",
			},
			{
				Name:        "Binary",
				Description: "Protocol buffers, ect.",
			},
		},
	},
	{
		Name: "Tools",
		SubCategories: []SubCategory{
			{
				Name:        "Terminal",
				Description: "Bash, Zsh, Fish, PowerShell, etc.",
			},
			{
				Name:        "Terminal tools",
				Description: "tmux, TUI, CLI, etc.",
			},
			{
				Name:        "Text Editors and IDEs",
				Description: "Vim, Emacs, VSCode, JetBrains IntelliJ, Language Server Protocol, Debug Adapter Protocol, etc.",
			},
			{
				Name:        "Version Control",
				Description: "Git, Mercurial, Subversion, Jujutsu, Perforce, etc.",
			},
			{
				Name:        "Build Tools",
				Description: "Make, CMake, Gradle, Bazel, Nix, etc.",
			},
			{
				Name:        "Linters and Formatters",
				Description: "Gofmt, Prettier, ESLint, clippy, rustfmt, Biome, TypeScript, etc.",
			},
			{
				Name:        "Package Managers",
				Description: "NPM/Yarn/PNPM, Go modules, Maven central, homebrew/apt/nix/pacman, Docker registry, etc.",
			},
			{
				Name:        "Compilers and interpreters",
				Description: "Musl vs glibc, CGO, LLVM, Lexer, Parser, abstract syntax tree, Java virtual machine, Just in time compiler, etc.",
			},
		},
	},
	{
		Name: "Mobile apps",
		SubCategories: []SubCategory{
			{
				Name:        "Android",
				Description: "Kotlin, Java, Jetpack Compose, etc.",
			},
			{
				Name:        "iOS",
				Description: "Swift, Objective-C, etc.",
			},
			{
				Name:        "React Native",
				Description: "JavaScript, TypeScript, etc.",
			},
			{
				Name:        "Installable PWAs",
				Description: "\"Problem child safari iOS\", special permissions for PWAs, etc.",
			},
		},
	},
	{
		Name: "Desktop apps",
		SubCategories: []SubCategory{
			{
				Name:        "Windows",
				Description: "WPF, WinForms, win32 API, UWP, etc.",
			},
			{
				Name:        "macOS",
				Description: "Cocoa, SwiftUI, etc.",
			},
			{
				Name:        "Cross-platform",
				Description: "Electron, GTK, Qt, etc.",
			},
		},
	},
	{
		Name: "Games",
		SubCategories: []SubCategory{
			{
				Name:        "Ready to use game engines",
				Description: "Unreal/Unity/Godot",
			},
			{
				Name:        "3D",
				Description: "3D models, triangles, vertex shaders, fragment shaders, etc.",
			},
			{
				Name:        "2D",
				Description: "Sprites, tilemaps, bitmap fonts, etc.",
			},
		},
	},
	{
		Name: "Security",
		SubCategories: []SubCategory{
			{
				Name:        "Encryption",
				Description: "Symmetric, asymmetric, etc.",
			},
			{
				Name:        "Hashing, signatures",
				Description: "Cryptographic random number generators, cryptographic hash functions, digital signatures, SSH, PGP, etc.",
			},
			{
				Name:        "Authentication",
				Description: "Secure passwords, multi-factor, biometrics, passwordless authentication, passkeys, password managers, secret managers (like HashiCorp Vault), OAuth2, ect.",
			},
			{
				Name:        "Network security",
				Description: "Firewalls, VPNs, etc.",
			},
			{
				Name:        "Web security",
				Description: "XSS, CSRF, SQL injection, etc.",
			},
			{
				Name:        "Malware",
				Description: "Anti-virus, anti-malware, etc.",
			},
			{
				Name:        "Reverse engineering",
				Description: "Disassembly, decompilation, Tools (IDA Pro, Ghidra, etc.)",
			},
		},
	},
	{
		Name: "Databases",
		SubCategories: []SubCategory{
			{
				Name:        "Relational",
				Description: "SQL, SQLite, PostgreSQL, MySQL, Turso, PlanetScale, etc.",
			},
			{
				Name:        "Object",
				Description: "MongoDB, Firestore, DynamoDB, etc.",
			},
			{
				Name:        "Graph",
				Description: "Neo4j, Dgraph, etc.",
			},
			{
				Name:        "Key-value",
				Description: "Redis, Memcached, etc.",
			},
			{
				Name:        "Time-series",
				Description: "InfluxDB, Prometheus, etc.",
			},
			{
				Name:        "Algorithms",
				Description: "B-trees, etc.",
			},
		},
	},
	{
		Name: "Data science",
		SubCategories: []SubCategory{
			{
				Name:        "Data visualization",
				Description: "D3.js, Plotly, etc.",
			},
			{
				Name:        "Data analysis",
				Description: "Pandas, NumPy, etc.",
			},
			{
				Name:        "Data pipelines",
				Description: "Apache Spark, etc.",
			},
			{
				Name:        "Data engineering",
				Description: "ETL, ELT, ELR, etc.",
			},
			{
				Name:        "Machine learning",
				Description: "Random forests, neural networks, etc.",
			},
		},
	},
}
