import os
import yaml
from typing import Optional, Dict, List, Any
from dotenv import load_dotenv
from pathlib import Path
from dataclasses import dataclass, field

# Import MCP types after the try/except block
try:
    from ..mcp.client import MCPServerConfig as MCPClientConfig, MCPTransportType
except ImportError:
    # Fallback if MCP not available
    MCPClientConfig = None
    MCPTransportType = None

# Load environment variables from .env file
load_dotenv()

def load_yaml_config():
    """Load configuration from config.yaml file."""
    # Look for config.yaml in the main project directory (two levels up from this file)
    config_path = Path(__file__).parent.parent.parent / "config.yaml"
    if config_path.exists():
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    
    # Fallback to current directory
    config_path = Path("config.yaml")
    if config_path.exists():
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    return {}

# Load YAML configuration
yaml_config = load_yaml_config()

# MCPServerConfig is imported from mcp.client

class Config:
    """Configuration settings for the LLM reverse proxy server."""

    def __init__(self):
        """Initialize configuration with MCP server loading."""
        # Provider Configuration
        self.LLM_PROVIDER: str = os.getenv("LLM_PROVIDER", "openai")
        self.LLM_MODEL: str = os.getenv("LLM_MODEL", "")

        # OpenAI Configuration - try environment first, then config.yaml
        self.OPENAI_API_KEY: str = (
            os.getenv("OPENAI_API_KEY") or
            yaml_config.get("providers", {}).get("openai", {}).get("api_key", "")
        )
        self.OPENAI_BASE_URL: str = (
            os.getenv("OPENAI_BASE_URL") or
            yaml_config.get("providers", {}).get("openai", {}).get("endpoint", "https://api.openai.com/v1")
        )
        self.OPENAI_DEFAULT_MODEL: str = (
            os.getenv("OPENAI_DEFAULT_MODEL") or
            yaml_config.get("providers", {}).get("openai", {}).get("default_model", "gpt-4o-mini").replace("gpt-4.1-mini", "gpt-4o-mini")
        )

        # Ollama Configuration
        self.OLLAMA_BASE_URL: str = (
            os.getenv("OLLAMA_BASE_URL") or
            yaml_config.get("providers", {}).get("ollama", {}).get("endpoint", "http://localhost:11434")
        )
        self.OLLAMA_DEFAULT_MODEL: str = (
            os.getenv("OLLAMA_DEFAULT_MODEL") or
            yaml_config.get("providers", {}).get("ollama", {}).get("default_model", "mistral")
        )

        # DeepSeek Configuration
        self.DEEPSEEK_API_KEY: str = (
            os.getenv("DEEPSEEK_API_KEY") or
            yaml_config.get("providers", {}).get("deepseek", {}).get("api_key", "")
        )
        self.DEEPSEEK_BASE_URL: str = (
            os.getenv("DEEPSEEK_BASE_URL") or
            yaml_config.get("providers", {}).get("deepseek", {}).get("endpoint", "https://api.deepseek.com/v1")
        )
        self.DEEPSEEK_DEFAULT_MODEL: str = (
            os.getenv("DEEPSEEK_DEFAULT_MODEL") or
            yaml_config.get("providers", {}).get("deepseek", {}).get("default_model", "deepseek-chat")
        )

        # ADK Configuration
        self.GOOGLE_GENAI_USE_VERTEXAI: bool = os.getenv("GOOGLE_GENAI_USE_VERTEXAI", "FALSE").upper() == "TRUE"
        self.GOOGLE_API_KEY: str = os.getenv("GOOGLE_API_KEY", "")

        # Server Configuration
        self.HOST: str = os.getenv("HOST", "0.0.0.0")
        self.PORT: int = int(os.getenv("PORT", "8000"))
        self.DEBUG: bool = os.getenv("DEBUG", "false").lower() == "true"

        # Preprocessing Configuration
        self.ENABLE_CONTEXT_INJECTION: bool = os.getenv("ENABLE_CONTEXT_INJECTION", "true").lower() == "true"
        self.SYSTEM_PROMPT_PREFIX: str = os.getenv("SYSTEM_PROMPT_PREFIX", "You are a helpful AI assistant.")
        self.MAX_CONTEXT_LENGTH: int = int(os.getenv("MAX_CONTEXT_LENGTH", "4000"))

        # Postprocessing Configuration
        self.ENABLE_RESPONSE_ANALYTICS: bool = os.getenv("ENABLE_RESPONSE_ANALYTICS", "true").lower() == "true"
        self.LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")

        # MCP Configuration
        self.ENABLE_MCP: bool = os.getenv("ENABLE_MCP", "true").lower() == "true"
        self.MCP_SERVERS: List[Any] = []
        self.MCP_HEALTH_CHECK_INTERVAL: float = float(os.getenv("MCP_HEALTH_CHECK_INTERVAL", "60.0"))
        self.MCP_CONNECTION_TIMEOUT: float = float(os.getenv("MCP_CONNECTION_TIMEOUT", "30.0"))
        self.MCP_MAX_RETRY_ATTEMPTS: int = int(os.getenv("MCP_MAX_RETRY_ATTEMPTS", "3"))

        # Processing Configuration
        self.REASONING_WORKFLOW: str = yaml_config.get("processing", {}).get("reasoning_workflow", "workflows/default")

        # Load MCP servers from configuration
        self._load_mcp_servers()
    
    @property
    def current_provider(self) -> str:
        """Get the currently selected provider."""
        return self.LLM_PROVIDER.lower()
    
    @property
    def current_model(self) -> str:
        """Get the currently selected model."""
        if self.LLM_MODEL:
            return self.LLM_MODEL
        
        if self.current_provider == "openai":
            return self.OPENAI_DEFAULT_MODEL
        elif self.current_provider == "ollama":
            return self.OLLAMA_DEFAULT_MODEL
        elif self.current_provider == "deepseek":
            return self.DEEPSEEK_DEFAULT_MODEL
        else:
            return self.OPENAI_DEFAULT_MODEL
    
    @property
    def current_base_url(self) -> str:
        """Get the base URL for the current provider."""
        if self.current_provider == "openai":
            return self.OPENAI_BASE_URL
        elif self.current_provider == "ollama":
            return self.OLLAMA_BASE_URL
        elif self.current_provider == "deepseek":
            return self.DEEPSEEK_BASE_URL
        else:
            return self.OPENAI_BASE_URL
    
    @property
    def current_api_key(self) -> str:
        """Get the API key for the current provider."""
        if self.current_provider == "openai":
            return self.OPENAI_API_KEY
        elif self.current_provider == "ollama":
            return ""  # Ollama doesn't require API key
        elif self.current_provider == "deepseek":
            return self.DEEPSEEK_API_KEY
        else:
            return self.OPENAI_API_KEY
    
    def _load_mcp_servers(self):
        """Load MCP server configurations from YAML config."""
        mcp_config = yaml_config.get("mcp", {})
        servers_config = mcp_config.get("servers", [])

        self.MCP_SERVERS = []

        if not MCPClientConfig or not MCPTransportType:
            print("Warning: MCP library not available, skipping server configuration")
            return

        for server_data in servers_config:
            try:
                # Convert string transport to enum
                transport_str = server_data.get("transport", "stdio").upper()
                transport = getattr(MCPTransportType, transport_str, MCPTransportType.STDIO)

                # Create MCPServerConfig from YAML data using the client constructor
                server_config = MCPClientConfig(
                    name=server_data.get("name"),
                    transport=transport,
                    command=server_data.get("command"),
                    args=server_data.get("args", []),
                    env=server_data.get("env", {}),
                    url=server_data.get("url"),
                    headers=server_data.get("headers", {})
                )

                # Add additional attributes as needed by other parts of the system
                server_config.enabled = server_data.get("enabled", True)
                server_config.timeout = server_data.get("timeout", self.MCP_CONNECTION_TIMEOUT)
                server_config.retry_attempts = server_data.get("retry_attempts", self.MCP_MAX_RETRY_ATTEMPTS)
                server_config.retry_delay = server_data.get("retry_delay", 1.0)
                server_config.health_check_interval = server_data.get("health_check_interval", self.MCP_HEALTH_CHECK_INTERVAL)

                # Validate configuration
                server_config.validate()
                self.MCP_SERVERS.append(server_config)

            except Exception as e:
                print(f"Warning: Invalid MCP server configuration: {e}")

    def get_enabled_mcp_servers(self) -> List[Any]:
        """Get list of enabled MCP servers."""
        return [server for server in self.MCP_SERVERS if server.enabled]

    def get_mcp_server_by_name(self, name: str) -> Optional[Any]:
        """Get MCP server configuration by name."""
        for server in self.MCP_SERVERS:
            if server.name == name:
                return server
        return None

    @classmethod
    def validate(cls) -> bool:
        """Validate that required configuration is present."""
        config_instance = cls()

        if config_instance.current_provider == "openai" and not config_instance.OPENAI_API_KEY:
            raise ValueError("OPENAI_API_KEY is required when using OpenAI provider")
        elif config_instance.current_provider == "deepseek" and not config_instance.DEEPSEEK_API_KEY:
            raise ValueError("DEEPSEEK_API_KEY is required when using DeepSeek provider")

        # Validate MCP configurations if enabled
        if config_instance.ENABLE_MCP:
            for server in config_instance.MCP_SERVERS:
                try:
                    server.validate()
                except Exception as e:
                    raise ValueError(f"Invalid MCP server configuration: {e}")

        return True

# Global config instance
config = Config() 