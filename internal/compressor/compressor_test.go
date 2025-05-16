package compressor

import (
	"os"
	"strings"
	"testing"
)

func TestGenericCompressor_Compress_Go(t *testing.T) {
	compressor := NewGenericCompressor()
	goCode := `
package main

import "fmt"

// This is a comment
type MyStruct struct {
FieldA int
FieldB string
}

func (s *MyStruct) MyMethod(val int) string {
// Method body
if val > 0 {
return fmt.Sprintf("Positive: %d", val)
}
return "Zero or Negative"
}

func main() {
// Main function body
instance := MyStruct{FieldA: 1, FieldB: "test"}
fmt.Println(instance.MyMethod(5))
fmt.Println("Hello, world!")
}
`
	expectedCompressedParts := []string{
		"package main",
		"import \"fmt\"",
		"// This is a comment",
		"type MyStruct struct",
		"func (s *MyStruct) MyMethod(val int) string { ... }",
		"func main() { ... }",
	}

	compressed, err := compressor.Compress([]byte(goCode), "go")
	if err != nil {
		t.Fatalf("Compress failed for Go: %v", err)
	}

	t.Logf("Compressed Go code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed output does not contain expected part: %s", part)
		}
	}

	if strings.Contains(compressed, `return fmt.Sprintf("Positive: %d", val)`) {
		t.Errorf("Compressed output unexpectedly contains function body content")
	}
	if strings.Contains(compressed, `fmt.Println("Hello, world!")`) {
		t.Errorf("Compressed output unexpectedly contains main function body content")
	}
}

func TestGenericCompressor_Compress_Python(t *testing.T) {
	compressor := NewGenericCompressor()
	pythonCode := `
# This is a Python comment
import os
from sys import argv

class MyClass:
    """
    A simple class
    """
    def __init__(self, name):
        self.name = name

    def greet(self, message):
        """Greets the person."""
        # Method body
        print(f"{message}, {self.name}!")
        if len(self.name) > 3:
            print("Long name")
        return True

def my_function(x, y):
    # Function body
    result = x + y
    print(f"Result is {result}")
    return result

if __name__ == "__main__":
    c = MyClass("Test")
    c.greet("Hello")
    my_function(1, 2)
`
	expectedCompressedParts := []string{
		"# This is a Python comment",
		"import os",
		"from sys import argv",
		"class MyClass: { ... } # Body removed",
		"def __init__(self, name): { ... } # Body removed",
		"def greet(self, message): { ... } # Body removed",
		"def my_function(x, y): { ... } # Body removed",
	}

	compressed, err := compressor.Compress([]byte(pythonCode), "python")
	if err != nil {
		t.Fatalf("Compress failed for Python: %v", err)
	}

	t.Logf("Compressed Python code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed Python output does not contain expected part: '%s'", part)
		}
	}

	if strings.Contains(compressed, `self.name = name`) {
		t.Errorf("Compressed Python output unexpectedly contains __init__ body content")
	}
	if strings.Contains(compressed, `print(f"{message}, {self.name}!")`) {
		t.Errorf("Compressed Python output unexpectedly contains greet method body content")
	}
	if strings.Contains(compressed, `result = x + y`) {
		t.Errorf("Compressed Python output unexpectedly contains my_function body content")
	}
	if strings.Contains(compressed, `c = MyClass("Test")`) {
		t.Errorf("Compressed Python output unexpectedly contains __main__ block content")
	}
}

func TestGenericCompressor_Compress_JavaScript(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	jsCode, err := os.ReadFile("testdata/example.js")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expectedPlaceholder := " { ... } // Body removed" // Generic placeholder for JS

	// Update expected parts for JS with its specific placeholder
	expectedCompressedParts := []string{
		"// This is a JavaScript comment",
		"import { something } from 'module';",
		"export class MyJSClass" + expectedPlaceholder,
		// constructor is a method_definition, its signature includes 'constructor'
		"constructor(name)" + expectedPlaceholder,
		"greet(message)" + expectedPlaceholder,
		"export function myJSFunction(x, y)" + expectedPlaceholder,
		"const myArrowFunc = (a, b) =>" + expectedPlaceholder,
		// Arrow function with expression body should be compressed
		"const myExpressionArrow = (x) =>" + expectedPlaceholder,
		"function* myGenerator()" + expectedPlaceholder,
		"export const myVar = 42;",            // Should be kept by @export.other
		"constructor()" + expectedPlaceholder, // Constructor of anonymous default class
	}

	compressed, err := compressor.Compress(jsCode, "javascript")
	if err != nil {
		t.Fatalf("Compress failed for JavaScript: %v", err)
	}

	t.Logf("Compressed JavaScript code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed JS output does not contain expected part: '%s'", part)
		}
	}

	if strings.Contains(compressed, `this.name = name;`) {
		t.Errorf("Compressed JS output unexpectedly contains constructor body content")
	}
	if strings.Contains(compressed, `console.log(message + ", " + this.name + "!");`) {
		t.Errorf("Compressed JS output unexpectedly contains greet method body content")
	}
	if strings.Contains(compressed, `const result = x + y;`) {
		t.Errorf("Compressed JS output unexpectedly contains myJSFunction body content")
	}
	if strings.Contains(compressed, `return a * b;`) {
		t.Errorf("Compressed JS output unexpectedly contains myArrowFunc body content")
	}
	if strings.Contains(compressed, `yield 1;`) {
		t.Errorf("Compressed JS output unexpectedly contains generator function body content")
	}
	if strings.Contains(compressed, `this.x = 1;`) {
		t.Errorf("Compressed JS output unexpectedly contains anonymous default class body content")
	}
	if strings.Contains(compressed, `x * x`) {
		t.Errorf("Compressed JS output unexpectedly contains expression arrow function body content")
	}
}

func TestGenericCompressor_Compress_JavaScript_AnonDefaultFunction(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	jsCode, err := os.ReadFile("testdata/example_anon_func.js")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expectedPlaceholder := " { ... } // Body removed" // Generic placeholder for JS

	// Update expected parts for JS with its specific placeholder
	expectedCompressedParts := []string{
		"// This is a JavaScript comment",
		"import { something } from 'module';",
		// Arrow function with expression body should be compressed
		"const myExpressionArrow = (x) =>" + expectedPlaceholder,
	}

	compressed, err := compressor.Compress(jsCode, "javascript")
	if err != nil {
		t.Fatalf("Compress failed for JavaScript: %v", err)
	}

	t.Logf("Compressed JavaScript code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed JS output does not contain expected part: '%s'", part)
		}
	}

	if strings.Contains(compressed, `console.log("Anon default func body");`) {
		t.Errorf("Compressed JS output unexpectedly contains anonymous default function body content")
	}
	if strings.Contains(compressed, `x * x`) {
		t.Errorf("Compressed JS output unexpectedly contains expression arrow function body content")
	}
}

func TestGenericCompressor_Compress_Bash(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	bashCode, err := os.ReadFile("testdata/example.sh")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expectedPlaceholder := " { ... } # Body removed" // Generic placeholder for Bash

	// Update expected parts for Bash
	expectedCompressedParts := []string{
		"# This is a bash comment",
		"function greet()" + expectedPlaceholder,
		"process_file()" + expectedPlaceholder,
		"echo \"Starting script\"",
	}

	compressed, err := compressor.Compress(bashCode, "bash")
	if err != nil {
		t.Fatalf("Compress failed for Bash: %v", err)
	}

	t.Logf("Compressed Bash code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed Bash output does not contain expected part: '%s'", part)
		}
	}

	if strings.Contains(compressed, `local name=$1`) {
		t.Errorf("Compressed Bash output unexpectedly contains function body content")
	}
}

func TestGenericCompressor_Compress_C(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	cCode, err := os.ReadFile("testdata/example.c")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expectedPlaceholder := " { ... } // Body removed" // Generic placeholder for C

	// Update expected parts for C
	expectedCompressedParts := []string{
		"#include <stdio.h>",
		"#include <stdlib.h>",
		"#include <string.h>",
		"struct Person {",
		"enum Color {",
		"union Data {",
		"typedef unsigned long int UINT32;",
		"int main(int argc, char *argv[])" + expectedPlaceholder,
		"void greet(const char* name)" + expectedPlaceholder,
		"int calculate_sum(int a, int b)" + expectedPlaceholder,
	}

	compressed, err := compressor.Compress(cCode, "c")
	if err != nil {
		t.Fatalf("Compress failed for C: %v", err)
	}

	t.Logf("Compressed C code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed C output does not contain expected part: '%s'", part)
		}
	}

	if strings.Contains(compressed, `strcpy(person.name, "John");`) {
		t.Errorf("Compressed C output unexpectedly contains function body content")
	}
}

func TestGenericCompressor_Compress_CSS(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	cssCode, err := os.ReadFile("testdata/example.css")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Update expected parts for CSS
	expectedCompressedParts := []string{
		"/* This is a CSS comment */",
		"@import url('https://fonts.googleapis.com/css2?family=Roboto:wght@400;700&display=swap')",
		":root {",
		"body {",
		"header {",
		"nav {",
		".card {",
		".button {",
		"@media (max-width: 768px) {",
		"@keyframes fadeIn {",
	}

	compressed, err := compressor.Compress(cssCode, "css")
	if err != nil {
		t.Fatalf("Compress failed for CSS: %v", err)
	}

	t.Logf("Compressed CSS code:\n%s", compressed)

	for _, part := range expectedCompressedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed CSS output does not contain expected part: '%s'", part)
		}
	}
}

func TestGenericCompressor_Compress_HTML(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	htmlCode, err := os.ReadFile("testdata/example.html")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// For HTML, we're just testing that the compression doesn't fail
	// Since we're using JavaScript parser for HTML, the results may vary
	compressed, err := compressor.Compress(htmlCode, "html")
	if err != nil {
		t.Fatalf("Compress failed for HTML: %v", err)
	}

	t.Logf("Compressed HTML code:\n%s", compressed)

	// Check that script content is properly handled
	if strings.Contains(compressed, "document.addEventListener('DOMContentLoaded', function()") {
		t.Errorf("Compressed HTML output unexpectedly contains script content that should be compressed")
	}

	// Test passes as long as compression doesn't fail
	t.Log("HTML compression completed without errors")
}

func TestGenericCompressor_Compress_Rust(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	rustCode, err := os.ReadFile("testdata/example.rs")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// For Rust, we're using JavaScript parser, so we're just testing that the compression doesn't fail
	compressed, err := compressor.Compress(rustCode, "rust")
	if err != nil {
		t.Fatalf("Compress failed for Rust: %v", err)
	}

	t.Logf("Compressed Rust code:\n%s", compressed)

	// Check for expected parts
	expectedParts := []string{
		"// This is a Rust comment",
		"use std::collections::HashMap;",
		"use std::io::{self, Read, Write};",
		"pub struct Person {",
		"pub trait Printable {",
		"pub enum Status {",
	}

	for _, part := range expectedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed Rust output does not contain expected part: '%s'", part)
		}
	}

	// Test passes as long as compression doesn't fail
	t.Log("Rust compression completed without errors")
}

func TestGenericCompressor_Compress_Java(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	javaCode, err := os.ReadFile("testdata/example.java")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// For Java, we're using JavaScript parser, so we're just testing that the compression doesn't fail
	compressed, err := compressor.Compress(javaCode, "java")
	if err != nil {
		t.Fatalf("Compress failed for Java: %v", err)
	}

	t.Logf("Compressed Java code:\n%s", compressed)

	// Check for expected parts
	expectedParts := []string{
		"// This is a Java comment",
		"package com.example.demo;",
		"import java.util.ArrayList;",
		"public class Person {",
		"interface Printable {",
		"enum Status {",
	}

	for _, part := range expectedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed Java output does not contain expected part: '%s'", part)
		}
	}

	// Test passes as long as compression doesn't fail
	t.Log("Java compression completed without errors")
}

func TestGenericCompressor_Compress_Swift(t *testing.T) {
	compressor := NewGenericCompressor()

	// Read test file content
	swiftCode, err := os.ReadFile("testdata/example.swift")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// For Swift, we're using JavaScript parser, so we're just testing that the compression doesn't fail
	compressed, err := compressor.Compress(swiftCode, "swift")
	if err != nil {
		t.Fatalf("Compress failed for Swift: %v", err)
	}

	t.Logf("Compressed Swift code:\n%s", compressed)

	// Check for expected parts
	expectedParts := []string{
		"// This is a Swift comment",
		"import Foundation",
		"struct Address {",
		"class Person {",
		"protocol Printable {",
		"enum Status {",
	}

	for _, part := range expectedParts {
		if !strings.Contains(compressed, part) {
			t.Errorf("Compressed Swift output does not contain expected part: '%s'", part)
		}
	}

	// Test passes as long as compression doesn't fail
	t.Log("Swift compression completed without errors")
}

func TestIdentifyLanguage(t *testing.T) {
	tests := []struct {
		filePath      string
		expectedLang  string
		expectedError bool
	}{
		{"main.go", "go", false},
		{"script.py", "python", false},
		{"app.js", "javascript", false},
		{"style.css", "css", false},
		{"script.sh", "bash", false},
		{"header.h", "c", false},
		{"program.c", "c", false},
		{"index.html", "html", false},
		{"page.htm", "html", false},
		{"main.rs", "rust", false},
		{"App.java", "java", false},
		{"Main.swift", "swift", false},
		{"README.md", "", true},
	}

	for _, tt := range tests {
		lang, err := IdentifyLanguage(tt.filePath)
		if tt.expectedError {
			if err == nil {
				t.Errorf("IdentifyLanguage(%s): expected error, got nil", tt.filePath)
			}
		} else {
			if err != nil {
				t.Errorf("IdentifyLanguage(%s): unexpected error: %v", tt.filePath, err)
			}
			if lang != tt.expectedLang {
				t.Errorf("IdentifyLanguage(%s): expected lang %s, got %s", tt.filePath, tt.expectedLang, lang)
			}
		}
	}
}

func TestGetLanguage(t *testing.T) {
	_, err := GetLanguage("go")
	if err != nil {
		t.Errorf("GetLanguage(go) failed: %v", err)
	}
	_, err = GetLanguage("nonexistent")
	if err == nil {
		t.Errorf("GetLanguage(nonexistent) expected error, got nil")
	}
}

func TestParseSource_ValidGo(t *testing.T) {
	content := []byte("package main\nfunc main() {}")
	lang, _ := GetLanguage("go")
	tree, err := ParseSource(content, lang)
	if err != nil {
		t.Fatalf("ParseSource failed for valid Go: %v", err)
	}
	if tree == nil {
		t.Fatalf("ParseSource returned nil tree for valid Go")
	}
	defer tree.Close()
	if tree.RootNode() == nil {
		t.Errorf("ParseSource returned tree with nil root node")
	}
}

func TestParseSource_InvalidGo(t *testing.T) {
	content := []byte("package main\nfunc main() {")
	lang, _ := GetLanguage("go")
	tree, err := ParseSource(content, lang)
	if err != nil {
		t.Fatalf("ParseSource returned an unexpected error for malformed Go: %v", err)
	}
	if tree == nil {
		t.Fatalf("ParseSource returned nil tree for malformed Go")
	}
	defer tree.Close()
	if tree.RootNode().HasError() {
		t.Logf("Malformed Go code parsed with errors, as expected.")
	} else {
		t.Logf("Malformed Go code parsed without explicit error nodes (might be recovered by parser).")
	}
}

func TestGetQuery_Go(t *testing.T) {
	queryStr, err := GetQuery("go")
	if err != nil {
		t.Fatalf("GetQuery(go) failed: %v", err)
	}
	if queryStr == "" {
		t.Errorf("GetQuery(go) returned empty query string")
	}
}

func TestCompileQuery_ValidGoQuery(t *testing.T) {
	lang, _ := GetLanguage("go")
	queryStr, _ := GetQuery("go")
	query, err := CompileQuery(queryStr, lang)
	if err != nil {
		t.Fatalf("CompileQuery failed for valid Go query: %v", err)
	}
	if query == nil {
		t.Fatalf("CompileQuery returned nil query for valid Go query")
	}
	defer query.Close()
}

func TestExecuteQuery_Go(t *testing.T) {
	content := []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hi\") }")
	lang, _ := GetLanguage("go")
	tree, _ := ParseSource(content, lang)
	defer tree.Close()
	queryStr, _ := GetQuery("go")
	query, _ := CompileQuery(queryStr, lang)
	defer query.Close()

	matches, err := ExecuteQuery(tree, query, content)
	if err != nil {
		t.Fatalf("ExecuteQuery failed for Go: %v", err)
	}
	if len(matches) == 0 {
		t.Errorf("ExecuteQuery returned no matches for basic Go code")
	}
	foundPackage := false
	foundImport := false
	foundFunc := false
	for _, match := range matches {
		for _, capture := range match.Captures {
			captureName := query.CaptureNameForId(capture.Index)
			if captureName == "package" {
				foundPackage = true
			}
			if captureName == "import" {
				foundImport = true
			}
			if captureName == "definition.function" {
				foundFunc = true
			}
		}
	}
	if !foundPackage {
		t.Errorf("Expected @package capture, not found")
	}
	if !foundImport {
		t.Errorf("Expected @import capture, not found")
	}
	if !foundFunc {
		t.Errorf("Expected @definition.function capture, not found")
	}
}
