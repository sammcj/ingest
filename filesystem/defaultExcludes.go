package filesystem

import (
	"bufio"
	"fmt"
	"strings"
)

// defaultGlobContent contains the content of default.glob
const defaultGlobContent = `
# Directories
**/screenshots/**
**/dist/**
**/node_modules/**
**/.git/**
**/.github/**
**/.vscode/**
**/build/**
**/coverage/**
**/.venv/**
**/venv/**
**/pyenv/**
**/tmp/**
**/out/**
**/temp/**
**/conda/**
**/mamba/**
**/.devcontainer/**
**/.next/**
**/backups/**

# File patterns
**/*.bak
**/*.png
**/*.jpg
**/*.jpeg
**/*.gif
**/*.svg
**/*.mp4
**/*.webm
**/*.webp
**/*.ico
**/*.avi
**/*.mp3
**/*.wav
**/*.flac
**/*.zip
**/*.tar
**/*.gz
**/*.bz2
**/*.7z
**/*.iso
**/*.bin
**/*.exe
**/*.app
**/*.dmg
**/*.deb
**/*.rpm
**/*.apk
**/*.fig
**/*.xd
**/*.blend
**/*.fbx
**/*.obj
**/*.tmp
**/*.swp
**/*.pem
**/*.crt
**/*.key
**/*.cert
**/*.pub
**/*.lock
**/*.DS_Store
**/*.sqlite
**/*.log
**/*.sqlite3
**/*.dll
**/*.woff
**/*.woff2
**/*.ttf
**/*.doc
**/*.docx
**/*.ppt
**/*.pptx
**/*.odf
**/*.eot
**/*.otf
**/*.ico
**/*.icns
**/*.jar
**/*.war
**/*.msi
**/*.csv
**/*.xls
**/*.xlsx
**/*.pdf
**/*.dat
**/*.baseline
**/*.ps1
**/*.bmp
**/*.diff
**/*.heic
**/*.hiec

# Specific files
**/.editorconfig
**/.eslintignore
**/.eslintrc*
**/tsconfig.json
**/.gitignore
**/.npmrc
**/LICENSE*
**/esbuild.config.mjs
**/manifest.json
**/package-lock.json
**/version-bump.mjs
**/versions.json
**/yarn.lock
**/CONTRIBUTING*
**/CHANGELOG*
**/SECURITY*
**/TODO.md
**/.nvmrc
**/.env*
**/.prettierrc*
**/CODEOWNERS
**/commitlint.config.js
**/renovate.json
**/pre-commit-config.yaml
**/.vimrc
**/poetry.lock
**/changelog.md
**/contributing.md
**/.pretterignore
**/.stylelintrc*
**/go.sum
**/*.pyc
**/.DS_Store
**/.gitattributes
**/.gitmodules
**/.gitpod.yml
**/.gitlab-ci.yml
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
