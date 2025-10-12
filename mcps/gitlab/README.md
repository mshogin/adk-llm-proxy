# GitLab MCP Server

A comprehensive Model Context Protocol (MCP) server for GitLab integration, providing repository analysis, code correlation with task tracking, and advanced code metrics.

## Features

### Repository Analysis Tools
- **find_project**: Search for GitLab projects by name or description
- **get_recent_commits**: Get recent commits with filtering by branch and date
- **analyze_commit_messages**: Analyze commit messages for patterns and insights
- **get_merge_requests**: Get merge requests with state filtering
- **analyze_branch_activity**: Analyze branch activity and identify stale branches

### Code-to-Task Correlation
- **link_commits_to_tasks**: Extract YouTrack task IDs from commits and create correlation
- **analyze_epic_code_changes**: Analyze code changes related to specific epics
- **get_code_metrics**: Get detailed code metrics (lines changed, files modified)
- **track_developer_activity**: Track developer activity patterns and trends

### Advanced Code Analysis
- **analyze_code_complexity**: Analyze complexity trends and identify hotspots
- **detect_code_patterns**: Detect architectural changes and patterns
- **track_technical_debt**: Identify technical debt indicators
- **analyze_code_quality_trends**: Analyze code quality over time
- **assess_refactoring_impact**: Assess impact of refactoring activities

### Resources
- **gitlab://project/{project_id}**: Raw GitLab project data with recent commits
- **gitlab://commit/{commit_sha}**: Detailed commit information
- **gitlab://merge_request/{mr_id}**: Merge request data
- **gitlab://metrics/{project_id}**: Project metrics and statistics

### Prompts
- **code_review_prompt**: Generate prompts for code review analysis
- **commit_analysis_prompt**: Generate prompts for commit pattern analysis
- **technical_debt_prompt**: Generate prompts for technical debt assessment

## Setup

### Prerequisites
- Python 3.8+
- GitLab instance (GitLab.com or self-hosted)
- GitLab API token with appropriate permissions

### Installation

1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Set environment variables:
```bash
export GITLAB_URL="https://gitlab.com"  # or your GitLab instance URL
export GITLAB_TOKEN="your-api-token-here"
export GITLAB_PROJECT_ID="123"  # optional default project ID
```

### Getting GitLab API Token

#### GitLab.com or GitLab Self-Hosted:
1. Go to your GitLab instance
2. Navigate to User Settings → Access Tokens
3. Create a new Personal Access Token with the following scopes:
   - `read_api` - for reading project information
   - `read_repository` - for accessing repository data
   - `read_user` - for user information
4. Copy the token and set it as `GITLAB_TOKEN`

#### Required Permissions:
- Read access to repositories
- Read access to merge requests
- Read access to commits and branches

### Configuration

The server uses the following environment variables:

- `GITLAB_URL`: GitLab instance URL (default: https://gitlab.com)
- `GITLAB_TOKEN`: API token (required)
- `GITLAB_PROJECT_ID`: Default project ID (optional)
- `GITLAB_TIMEOUT`: Request timeout in seconds (default: 30)
- `GITLAB_VERIFY_SSL`: SSL verification (default: true)

## Usage

### Running the Server

```bash
python server.py
```

### Testing

Run the comprehensive test suite:

```bash
python test_gitlab_server.py
```

### Using with MCP Client

The server implements the MCP protocol and can be used with any MCP-compatible client. Example tools:

```python
# Find projects
result = await client.call_tool("find_project", {"search_term": "my-app"})

# Get recent commits
result = await client.call_tool("get_recent_commits", {
    "project_id": "123",
    "days": 7,
    "branch": "main"
})

# Analyze commit messages
result = await client.call_tool("analyze_commit_messages", {
    "project_id": "123",
    "days": 30
})

# Link commits to tasks
result = await client.call_tool("link_commits_to_tasks", {"project_id": "123"})

# Get code metrics
result = await client.call_tool("get_code_metrics", {"project_id": "123"})
```

## Repository Analysis Workflow

1. **Find Project**: Use `find_project` to locate projects by name
2. **Get Commits**: Use `get_recent_commits` to see recent changes
3. **Analyze Messages**: Use `analyze_commit_messages` for insights
4. **Check Branches**: Use `analyze_branch_activity` for branch health
5. **Review MRs**: Use `get_merge_requests` for code review status

## Code Correlation Workflow

1. **Link Commits**: Use `link_commits_to_tasks` to correlate with YouTrack
2. **Epic Analysis**: Use `analyze_epic_code_changes` for epic-level insights
3. **Code Metrics**: Use `get_code_metrics` for quantitative analysis
4. **Developer Tracking**: Use `track_developer_activity` for team insights
5. **Complexity Assessment**: Use `analyze_code_complexity` for quality metrics

## YouTrack Integration

The server automatically extracts YouTrack task IDs from commit messages using patterns:

- `PROJECT-123` - Standard task ID format
- `PROJECT #123` - Alternative format
- `#PROJECT-123` - Hash-prefixed format

### Best Practices for Task Linking

1. **Consistent Format**: Use consistent task ID format in commit messages
2. **Meaningful Messages**: Include task context in commit messages
3. **Epic References**: Reference epic IDs in feature-related commits
4. **Conventional Commits**: Use conventional commit format when possible

Example commit message:
```
feat: Add user authentication PROJ-101

Implements secure login functionality for PROJ-100 epic.
- Add password validation
- Implement session management
- Add 2FA support

Closes PROJ-101
```

## Code Quality Insights

### Complexity Indicators
- **Large Commits**: Commits with >500 lines changed
- **Multi-file Commits**: Commits affecting >10 files
- **Hotspots**: Files changed frequently (>5 times)
- **Bug Fix Ratio**: Proportion of bug fixes vs features

### Quality Metrics
- **Lines Added/Deleted**: Code growth trends
- **File Type Distribution**: Technology usage
- **Developer Activity**: Contribution patterns
- **Refactoring Activity**: Code maintenance health

## Performance

- Efficient API usage with pagination
- Configurable request timeouts
- Batch processing for large datasets
- Caching for frequently accessed data

## Error Handling

- Comprehensive error handling for API failures
- Graceful degradation for missing data
- Detailed error messages for debugging
- Automatic retry for transient failures

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Verify GITLAB_TOKEN is correct
   - Check token permissions and scopes
   - Ensure token hasn't expired

2. **Project Not Found**
   - Verify project ID or path format
   - Check user permissions for the project
   - Confirm project exists and is accessible

3. **API Rate Limiting**
   - Reduce request frequency
   - Use smaller date ranges for analysis
   - Consider GitLab Premium for higher limits

4. **SSL/TLS Errors**
   - Set GITLAB_VERIFY_SSL=false for testing
   - Check certificate validity for self-hosted instances

### Debug Mode

Enable debug logging:

```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

## Integration with Other MCP Servers

### With YouTrack MCP Server

The GitLab MCP server is designed to work seamlessly with the YouTrack MCP server:

1. **Cross-Reference**: Link commits to YouTrack tasks automatically
2. **Epic Analysis**: Correlate GitLab changes with YouTrack epics
3. **Progress Tracking**: Map code changes to task completion
4. **Team Insights**: Combine developer activity with task assignments

### Example Workflow

1. Use YouTrack server to find epic tasks
2. Use GitLab server to analyze related code changes
3. Cross-reference task IDs between both systems
4. Generate comprehensive project reports

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run the test suite
5. Submit a pull request

## License

MIT License - see LICENSE file for details