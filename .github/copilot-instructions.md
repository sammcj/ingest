# GitHub Copilot Instructions

## Contribution Guidelines

### Before Committing

1. **Run linters:** `make lint` (must pass without warnings or errors)
2. **Run tests:** `make test` (must pass all tests)
3. **Build successfully:** `make build` (must compile without warnings or errors)

### Code Standards

- Follow Go best practices and idiomatic patterns
- Use Australian English spelling throughout code and documentation
- No marketing terms like "comprehensive" or "production-grade"
- Focus on clear, concise, actionable technical guidance
- Keep responses token-efficient (avoid returning unnecessary data)
verbosity

## Code Quality Checks

### General Code Quality
- Verify proper module imports and dependencies
- Check for hardcoded credentials or sensitive data
- Ensure proper resource cleanup (defer statements)
- Validate input parameters thoroughly
- Use appropriate data types and structures
- Follow consistent error message formatting

## Configuration & Environment
- Environment variables should have sensible defaults
- Configuration should be documented in README
- Support both development and production modes
- Handle missing optional dependencies gracefully

## General Guidelines

- Do not use marketing terms such as 'comprehensive' or 'production-grade' in documentation or code comments.
- Focus on clear, concise actionable technical guidance.

## Review Checklist for Every PR

Before approving any pull request, verify:

- [ ] Code follows the latest Golang best practices
- [ ] No security issues or vulnerabilities introduced
- [ ] All linting and tests pass successfully
- [ ] Documentation updated if required
- [ ] Australian English spelling used throughout, No American English spelling used
- [ ] Context cancellation handled properly if applicable
- [ ] Resource cleanup with defer statements if applicable

If you are re-reviewing a PR you've reviewed in the past and your previous comments / suggestions have been addressed or are no longer valid please resolve those previous review comments to keep the review history clean and easy to follow.
