# Contributing to Go-Carbon

Thank you for your interest in contributing to Go-Carbon! We welcome contributions from the community.

## 🚀 Quick Start

1. **Fork the repository**
2. **Clone your fork**
   ```bash
   git clone https://github.com/YOUR_USERNAME/go-carbon.git
   cd go-carbon
   ```
3. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```
4. **Make your changes**
5. **Run tests**
   ```bash
   go test ./...
   ```
6. **Commit your changes**
   ```bash
   git commit -m "Add some amazing feature"
   ```
7. **Push to your fork**
   ```bash
   git push origin feature/amazing-feature
   ```
8. **Open a Pull Request**

## 📋 Guidelines

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Run `gofmt` before committing: `go fmt ./...`
- Run `go vet` to catch common issues: `go vet ./...`
- Use meaningful variable and function names
- Add comments for exported functions and types

### Testing

- Write tests for new features
- Ensure all tests pass: `go test ./...`
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Add edge case tests for complex logic

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples:**
```
feat(codegen): add support for nested enum types

fix(pipeline): resolve race condition in shutdown

docs(readme): update installation instructions

test(codegen): add edge case tests for complex types
```

### Pull Requests

- Keep PRs focused on a single feature or fix
- Update documentation if needed
- Add tests for new functionality
- Ensure CI passes
- Link related issues in the PR description

### Code Generation

If contributing to the code generator:

1. **Understand the architecture**
   - Read [docs/codegen.md](docs/codegen.md)
   - Study existing generators in `internal/codegen/gen/`

2. **Test thoroughly**
   - Add unit tests for new functionality
   - Test with real Anchor IDL files
   - Verify generated code compiles

3. **Update documentation**
   - Update [docs/codegen.md](docs/codegen.md) for new features
   - Add examples if needed
   - Update CHANGELOG.md

### Plugin Development

If creating new plugins:

1. **Follow plugin interface**
   - Implement required methods
   - Handle initialization and shutdown
   - Add proper error handling

2. **Test with examples**
   - Create example usage in `examples/`
   - Document configuration options
   - Test integration with pipeline

## 🐛 Reporting Issues

### Bug Reports

Include:
- Go version (`go version`)
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

### Feature Requests

Include:
- Clear description of the feature
- Use case and motivation
- Possible implementation approach
- Any breaking changes

## 📖 Documentation

- Documentation lives in `docs/`
- Examples go in `examples/`
- Update README.md for significant changes
- Keep CHANGELOG.md up to date

## 💬 Getting Help

- **Documentation**: Check [docs/](docs/)
- **Examples**: See [examples/](examples/)
- **Issues**: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions)

## 🎯 Areas We Need Help

### High Priority

- [ ] Comprehensive test coverage (>80%)
- [ ] Yellowstone gRPC datasource implementation
- [ ] Helius websocket datasource
- [ ] More protocol decoders (Metaplex, Serum, etc.)

### Medium Priority

- [ ] Prometheus metrics backend
- [ ] WebSocket live updates
- [ ] GraphQL API
- [ ] Database integrations

### Good First Issues

Look for issues labeled `good first issue` or `help wanted` in the GitHub issue tracker.

## 🔐 Security

If you discover a security vulnerability, please email [security contact] instead of creating a public issue.

## 📄 License

By contributing to Go-Carbon, you agree that your contributions will be licensed under the MIT License.

## 🙏 Thank You

Every contribution, no matter how small, is valuable and appreciated. Thank you for helping make Go-Carbon better!

---

**Questions?** Feel free to open an issue or discussion!
