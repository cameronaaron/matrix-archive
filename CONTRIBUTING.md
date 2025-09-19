# Contributing to Matrix Archive

Thank you for your interest in contributing to Matrix Archive! This document provides guidelines for contributing to the project.

## Code of Conduct

Please be respectful and constructive in all interactions. We aim to maintain a welcoming environment for all contributors.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/matrix-archive.git
   cd matrix-archive
   ```
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Build the project**:
   ```bash
   make build
   ```

## Development Guidelines

### Code Style

- Follow standard Go conventions and best practices
- Use `gofmt` and `golint` to ensure code quality
- Write clear, self-documenting code with appropriate comments
- Follow the existing naming conventions established in the codebase

### Testing

- Write tests for new functionality
- Ensure all tests pass before submitting PRs:
  ```bash
  make test
  ```
- Aim for good test coverage:
  ```bash
  make coverage
  ```

### Commit Messages

- Use clear, descriptive commit messages
- Follow the format: `type(scope): description`
- Examples:
  - `feat(export): add YAML export format support`
  - `fix(auth): resolve credential saving issue`
  - `docs(readme): update installation instructions`

## Pull Request Process

1. **Create a feature branch** from `master`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the development guidelines

3. **Test thoroughly**:
   ```bash
   make test
   make build
   ```

4. **Update documentation** if needed (README, code comments, etc.)

5. **Submit a pull request** with:
   - Clear description of the changes
   - Reference to any related issues
   - Screenshots/examples if applicable

## Reporting Issues

When reporting issues, please include:

- Go version and operating system
- Steps to reproduce the issue
- Expected vs actual behavior
- Any relevant log output or error messages
- Sample data or configuration (without sensitive information)

## Security Considerations

- **Never commit sensitive data** (credentials, tokens, personal messages)
- Review the `.gitignore` file to understand what should not be committed
- Be mindful of privacy when sharing logs or examples
- Report security vulnerabilities privately to the maintainers

## Areas for Contribution

- **Export formats**: Add support for new export formats
- **Bridge support**: Improve username mapping for different bridge types
- **Performance**: Optimize message processing and export generation
- **Documentation**: Improve guides, examples, and code documentation
- **Testing**: Expand test coverage and add integration tests
- **UI/Templates**: Enhance HTML export templates and styling

## Questions?

Feel free to open an issue for questions or discussions about contributing to the project.