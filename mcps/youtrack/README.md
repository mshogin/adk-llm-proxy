# YouTrack MCP Server

A comprehensive Model Context Protocol (MCP) server for YouTrack integration, providing epic and task tracking, analysis, and reporting capabilities.

## Features

### Epic Management Tools
- **find_epic**: Search for epics by name, ID, or tag
- **get_epic_details**: Get detailed information about a specific epic
- **list_epic_tasks**: List all tasks in an epic
- **get_epic_status_summary**: Get current status summary for an epic

### Task Analysis Tools
- **get_task_details**: Get detailed information about a specific task
- **get_task_comments**: Get recent comments for a task
- **get_task_history**: Get change history for a task
- **analyze_task_activity**: Analyze task activity over the last week

### Analytics Tools
- **analyze_epic_progress**: Analyze epic progress trends and risks
- **generate_epic_report**: Generate comprehensive epic reports

### Resources
- **youtrack://epic/{epic_id}**: Raw YouTrack epic data as JSON

### Prompts
- **epic_analysis_prompt**: Generate prompts for epic analysis

## Setup

### Prerequisites
- Python 3.8+
- YouTrack instance (Cloud or Server)
- YouTrack API token

### Installation

1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Set environment variables:
```bash
export YOUTRACK_URL="https://your-company.youtrack.cloud"
export YOUTRACK_TOKEN="your-api-token-here"
```

### Getting YouTrack API Token

1. Go to your YouTrack instance
2. Navigate to your profile settings
3. Go to "Authentication" tab
4. Create a new "Permanent Token"
5. Copy the token and set it as `YOUTRACK_TOKEN`

### Configuration

The server uses the following environment variables:

- `YOUTRACK_URL`: YouTrack instance URL (required)
- `YOUTRACK_TOKEN`: API token (required)
- `YOUTRACK_TIMEOUT`: Request timeout in seconds (default: 30)
- `YOUTRACK_VERIFY_SSL`: SSL verification (default: true)

## Usage

### Running the Server

```bash
python server.py
```

### Testing

Run the comprehensive test suite:

```bash
python test_youtrack_server.py
```

### Using with MCP Client

The server implements the MCP protocol and can be used with any MCP-compatible client. Example tools:

```python
# Find epics
result = await client.call_tool("find_epic", {"search_term": "authentication"})

# Get epic details
result = await client.call_tool("get_epic_details", {"epic_id": "PROJ-123"})

# Generate epic report
result = await client.call_tool("generate_epic_report", {"epic_id": "PROJ-123"})
```

## Epic Analysis Workflow

1. **Find Epic**: Use `find_epic` to locate epics by name or tag
2. **Get Details**: Use `get_epic_details` for basic information
3. **List Tasks**: Use `list_epic_tasks` to see all related tasks
4. **Analyze Progress**: Use `analyze_epic_progress` for trend analysis
5. **Generate Report**: Use `generate_epic_report` for comprehensive analysis

## Task Analysis Workflow

1. **Get Details**: Use `get_task_details` for task information
2. **Review Comments**: Use `get_task_comments` for recent discussions
3. **Check History**: Use `get_task_history` for change timeline
4. **Analyze Activity**: Use `analyze_task_activity` for recent changes

## Custom Fields

The server supports common YouTrack custom fields:

- **State**: Task/epic status
- **Assignee**: Person assigned to the task
- **Priority**: Task priority level
- **Story Points**: Effort estimation
- **Type**: Issue type (Epic, Task, Bug, etc.)

## Error Handling

The server includes comprehensive error handling:

- Connection validation on startup
- Graceful handling of missing issues
- Detailed error messages for debugging
- Automatic retry for transient failures

## Performance

- Efficient API calls with field selection
- Caching of frequently accessed data
- Configurable timeouts
- Batch processing for multiple items

## Security

- Token-based authentication
- SSL/TLS verification
- No sensitive data in logs
- Secure environment variable handling

## Troubleshooting

### Common Issues

1. **Connection Failed**
   - Verify YOUTRACK_URL is correct
   - Check YOUTRACK_TOKEN is valid
   - Ensure network connectivity

2. **Issue Not Found**
   - Verify issue ID format (e.g., PROJ-123)
   - Check user permissions for the project
   - Confirm issue exists in YouTrack

3. **Timeout Errors**
   - Increase YOUTRACK_TIMEOUT value
   - Check YouTrack server performance
   - Reduce query complexity

### Debug Mode

Enable debug logging:

```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run the test suite
5. Submit a pull request

## License

MIT License - see LICENSE file for details