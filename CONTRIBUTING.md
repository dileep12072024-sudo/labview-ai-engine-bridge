# Contributing to LabVIEW-AI-Engine-Bridge

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/labview-ai-engine-bridge.git
   cd labview-ai-engine-bridge
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Requirements
- Go 1.22 or later
- Windows 10+ (for LabVIEW integration)
- LabVIEW 2025+ (for testing VI generation)

### Building
```bash
go mod tidy
go test ./...
go build -o dist/bridge.exe .
```

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Test MCP connection
.\dist\bridge.exe -mock
```

## Code Style

- Follow standard Go conventions (gofmt, golint)
- Keep functions small and focused
- Add comments for exported functions
- Use meaningful variable names

## Submitting Changes

1. **Make your changes** and test thoroughly
2. **Commit with clear messages**:
   ```bash
   git commit -m "feat: add support for X" -m "Description of change"
   ```
3. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```
4. **Open a Pull Request** with:
   - Clear description of changes
   - Link to any related issues
   - Explanation of how to test

## Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

Example:
```
feat: add support for multiple control types

- Add new control types: tank, gauge, ring
- Update create_vi_control tool documentation
- Add tests for new control types

Closes #12
```

## Reporting Issues

When reporting bugs, please include:
- LabVIEW version
- Windows version
- Steps to reproduce
- Expected vs actual behavior
- Error messages/logs

## Questions?

Feel free to open a discussion or issue for questions about the codebase.

Thank you for contributing! 🚀
