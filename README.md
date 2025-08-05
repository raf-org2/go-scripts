# GitHub Organization Security Analysis Tool

A Go application for analyzing GitHub organization-level security features and advanced security settings across all repositories in your organizations.

## 📋 Overview

This repository contains a comprehensive tool for GitHub organization security analysis:

- **`organization-check.go`** - Analyze security settings across all repositories in your GitHub organizations

## 🚀 Features

- **Organization Analysis**: List all organizations you're a member of with repository counts
- **Security Settings Review**: Check GitHub Advanced Security (GHAS) features across repositories
- **Comprehensive Coverage**: Analyze Secret Scanning, Code Scanning (CodeQL), Dependabot, and more
- **Enterprise Support**: Works with GitHub Enterprise and organization-level access
- **Dockerized Environment**: Run everything in a consistent containerized environment
- **Detailed Reporting**: Visual indicators for enabled/disabled security features

## 🛠️ Prerequisites

- Docker and Docker Compose
- GitHub Personal Access Token with appropriate permissions:
  - **`repo`** scope: For repository access
  - **`admin:org`** scope: For organization access (if needed)
  - **`security_events`** scope: For security feature analysis
  - Enterprise admin permissions (for enterprise features)

## ⚙️ Setup

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd go-scripts
   ```

2. **Set your GitHub token**:
   ```bash
   export GITHUB_TOKEN_ORG=your_github_token_here
   ```

3. **Initialize and build**:
   ```bash
   make build
   ```

## 📖 Usage

### Run Organization Security Analysis

Analyze all your organization memberships and their security settings:

```bash
make organization-check
```

This comprehensive analysis will show:

#### 👤 **User Information**
- Your authenticated user details
- Associated email and company

#### 🏢 **Organization Memberships**
- List of all organizations you're a member of
- Repository count for each organization
- Your role and membership status in each org

#### 🛡️ **Security Analysis for Each Repository**
- **Secret Scanning**: Status and configuration
- **Secret Scanning Push Protection**: Whether push protection is enabled
- **Advanced Security**: GitHub Advanced Security status
- **Dependabot Security Updates**: Dependency vulnerability scanning
- **Code Scanning (CodeQL)**: Static analysis security scanning
- **Dependabot Alerts**: Active dependency vulnerability alerts
- **Repository Type**: Private vs public (affects available features)

### Development & Debugging

Open a shell in the container for development:

```bash
make shell
```

## 🔧 Manual Usage (without Docker)

If you prefer to run the tool directly:

1. **Install dependencies**:
   ```bash
   go mod tidy
   ```

2. **Set environment variable**:
   ```bash
   export GITHUB_TOKEN=your_github_token_here
   ```

3. **Run the tool**:
   ```bash
   go run organization-check.go
   ```

### Sample Output


![sample](./images/sample-output.png)

## 📜 License

This project is licensed under the terms specified in the LICENSE file.

## ⚠️ Security Considerations

- **Never commit tokens**: Keep your GitHub token secure and out of version control
- **Use environment variables**: Store tokens in `GITHUB_TOKEN_ORG` environment variable
- **Minimal permissions**: Ensure your token has only required scopes
- **Regular rotation**: Rotate access tokens regularly for security

## 🆘 Troubleshooting

### Common Issues

**"GitHub token is required"**
- Set `GITHUB_TOKEN_ORG` environment variable
- Or ensure the token is properly exported in your shell

**"Failed to list organizations"**
- Verify token has `admin:org` scope (if needed)
- Check if you're actually a member of any organizations

**"Access denied or error" for security features**
- Ensure token has `security_events` scope
- Verify you have appropriate permissions for the repository
- Some features require admin access to the repository

**"Not configured" for Code Scanning or Dependabot**
- Features may not be enabled for the repository
- Check if GitHub Advanced Security is enabled for private repos
- Verify organization/enterprise policies

### Getting Help

If you encounter issues:
1. Check that your token has the required scopes
2. Verify your organization membership and permissions
3. Review GitHub's documentation on Advanced Security features
4. Check organization/enterprise policies that may affect feature availability

## 🔗 Additional Resources

- [GitHub Advanced Security Documentation](https://docs.github.com/en/enterprise-cloud@latest/get-started/learning-about-github/about-github-advanced-security)
- [GitHub API Documentation](https://docs.github.com/en/rest)
- [Managing Security and Analysis Settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-security-and-analysis-settings-for-your-repository)
