package filesystem

import (
	"bufio"
	"fmt"
	"strings"
)

// defaultGlobContent contains the content of default.glob
const defaultGlobContent = `
# Directories
**/.cargo/**
**/.devcontainer/**
**/.git/**
**/.github/**
**/.next/**
**/.venv/**
**/.vim/**
**/.vscode-insiders/**
**/.vscode-oss/**
**/.vscode-print-resource-cache/**
**/.vscode-react-native/**
**/.vscode/**
**/.wasmedge/**
**/.widsurf/**
**/.widsurf/**
**/.wine/**
**/.wine32/**
**/.yarn/**
**/.zcompcache/**
**/.zfunc/**
**/.zgen/**
**/.zsh_sessions/**
**/.zsh.d/**
**/backups/**
**/build/**
**/conda/**
**/coverage/**
**/dist/**
**/mamba/**
**/node_modules/**
**/out/**
**/pyenv/**
**/__pycache__/**
**/screenshots/**
**/target/**
**/temp/**
**/tmp/**
**/venv/**
**/virtualenv/**
**/wineprefix/**

# File patterns
**/__tests__/*
**/_data/*
**/.aider*
**/.aider/*
**/.bash_history
**/.boto
**/.claude.json
**/.cline/*
**/.condarc
**/.cursor/*
**/.dream_history
**/.fzf.bash
**/.fzf.zsh
**/.git-credentials
**/.llamafile_history
**/.lscolors
**/.netrc
**/.psql_history
**/.python_history
**/.terraform/*
**/.webpack/*
**/.Xauthority
**/*.7z
**/*.apk
**/*.app
**/*.avi
**/*.bak
**/*.baseline
**/*.bin
**/*.blend
**/*.bmp
**/*.bz2
**/*.cert
**/*.crt
**/*.csv
**/*.dat
**/*.deb
**/*.db
**/*.diff
**/*.dll
**/*.dmg
**/*.doc
**/*.docx
**/*.DS_Store
**/*.eot
**/*.excalidrawlib
**/*.exe
**/*.fbx
**/*.fig
**/*.flac
**/*.gif
**/*.gguf
**/*.ggml
**/*.exl2
**/*.exl3
**/*.gz
**/*.heic
**/*.hiec
**/*.icns
**/*.ico
**/*.iso
**/*.jar
**/*.jpeg
**/*.jpg
**/*.key
**/*.lock
**/*.log*
**/*.mp3
**/*.mp4
**/*.msi
**/*.mvnw*
**/*.obj
**/*.odf
**/*.otf
**/*.pdf
**/*.partial
**/*.pem
**/*.png
**/*.ppt
**/*.pptx
**/*.ps1
**/*.pub
**/*.pyc
**/*.pyo
**/*.pysave
**/*.rpm
**/*.sqlite
**/*.sqlite3
**/*.svg
**/*.swp*
**/*.tar*
**/*.terraform.tfstate.lock.info
**/*.tfgraph
**/*.tmp
**/*.ttf*
**/*.war
**/*.wav
**/*.webm
**/*.webp
**/*.woff
**/*.woff2
**/*.xd
**/*.xls
**/*.xlsx
**/*.zip
**/terraform.tfstate.*
**/test/*
**/tests/*
**/vendor/*


# Specific files
**/.aiderrules
**/.aider.*
**/.clinerules
**/.cursorrules
**/.DS_Store
**/.editorconfig
**/.env*
**/.eslintignore
**/.eslintrc*
**/.gitattributes
**/.gitconfig
**/.gitconfig-no_push
**/.gitignore
**/.gitignoreglobal
**/.gitlab-ci.yml
**/.gitmodules
**/.gitpod.yml
**/.npmrc
**/.nvmrc
**/.pre-commit-config.yaml
**/.pre-commit-config.yml
**/.prettierignore
**/.prettierrc*
**/.saml2aws
**/.saml2aws-auto.yml
**/.stylelintrc*
**/.terraform.lock.hcl
**/.terraform.lock.hcl.lock
**/.vimrc
**/.whitesource
**/.zcompdump*
**/.claude/*.json
**/.mcp.json
**/bat-config
**/changelog.md
**/CHANGELOG*
**/CLA.md
**/CODE_OF_CONDUCT.md
**/CODEOWNERS
**/CONTRIBUTORS.md
**/commitlint.config.js
**/contributing.md
**/CONTRIBUTING*
**/dircolors
**/esbuild.config.mjs
**/go.mod
**/go.sum
**/LICENSE*
**/LICENCE*
**/manifest.json
**/package-lock.json
**/plan
**/plan.out
**/pnpm-lock.yaml
**/poetry.lock
**/pre-commit-config.yaml
**/renovate.json
**/SECURITY*
**/SUPPORT.md
**/terraform.rc
**/terraform.tfplan
**/terraform.tfplan.json
**/terraform.tfstate
**/terraform.tfstate.backup
**/TODO.md
**/TROUBLESHOOTING.md
**/tsconfig.json
**/version-bump.mjs
**/versions.json
**/yarn.lock
`

// GetDefaultExcludes returns a list of default exclude patterns
func GetDefaultExcludes() ([]string, error) {
	var defaultExcludes []string
	scanner := bufio.NewScanner(strings.NewReader(defaultGlobContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			defaultExcludes = append(defaultExcludes, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning default glob content: %w", err)
	}

	return defaultExcludes, nil
}
