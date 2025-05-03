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
**/.vscode/**
**/backups/**
**/build/**
**/conda/**
**/coverage/**
**/dist/**
**/mamba/**
**/node_modules/**
**/out/**
**/pyenv/**
**/screenshots/**
**/target/**
**/temp/**
**/tmp/**
**/venv/**

# File patterns
**/__tests__/*
**/_data/*
**/.aider*
**/.aider/*
**/.cline/*
**/.cursor/*
**/.terraform/*
**/.webpack/*
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
**/*.diff
**/*.dll
**/*.dmg
**/*.doc
**/*.docx
**/*.DS_Store
**/*.eot
**/*.exe
**/*.fbx
**/*.fig
**/*.flac
**/*.gif
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
**/*.pem
**/*.png
**/*.ppt
**/*.pptx
**/*.ps1
**/*.pub
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
**/.clinerules
**/.cursorrules
**/.DS_Store
**/.editorconfig
**/.env*
**/.eslintignore
**/.eslintrc*
**/.gitattributes
**/.gitignore
**/.gitlab-ci.yml
**/.gitmodules
**/.gitpod.yml
**/.npmrc
**/.nvmrc
**/.pre-commit-config.yaml
**/.pre-commit-config.yml
**/.pretterignore
**/.prettierrc*
**/.stylelintrc*
**/.terraform.lock.hcl
**/.terraform.lock.hcl.lock
**/.vimrc
**/*.pyc
**/changelog.md
**/CHANGELOG*
**/CLA.md
**/CODE_OF_CONDUCT.md
**/CODEOWNERS
**/commitlint.config.js
**/contributing.md
**/CONTRIBUTING*
**/esbuild.config.mjs
**/go.mod
**/go.sum
**/LICENSE*
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
