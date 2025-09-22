import os
import yaml
from typing import Optional
from dotenv import load_dotenv
from pathlib import Path

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

class Config:
    """Configuration settings for the LLM reverse proxy server."""
    
    # Provider Configuration
    LLM_PROVIDER: str = os.getenv("LLM_PROVIDER", "openai")
    LLM_MODEL: str = os.getenv("LLM_MODEL", "")
    
    # OpenAI Configuration - try environment first, then config.yaml
    OPENAI_API_KEY: str = (
        os.getenv("OPENAI_API_KEY") or 
        yaml_config.get("providers", {}).get("openai", {}).get("api_key", "")
    )
    OPENAI_BASE_URL: str = (
        os.getenv("OPENAI_BASE_URL") or 
        yaml_config.get("providers", {}).get("openai", {}).get("endpoint", "https://api.openai.com/v1")
    )
    OPENAI_DEFAULT_MODEL: str = (
        os.getenv("OPENAI_DEFAULT_MODEL") or 
        yaml_config.get("providers", {}).get("openai", {}).get("default_model", "gpt-4o-mini").replace("gpt-4.1-mini", "gpt-4o-mini")
    )
    
    # Ollama Configuration
    OLLAMA_BASE_URL: str = (
        os.getenv("OLLAMA_BASE_URL") or 
        yaml_config.get("providers", {}).get("ollama", {}).get("endpoint", "http://localhost:11434")
    )
    OLLAMA_DEFAULT_MODEL: str = (
        os.getenv("OLLAMA_DEFAULT_MODEL") or 
        yaml_config.get("providers", {}).get("ollama", {}).get("default_model", "mistral")
    )
    
    # DeepSeek Configuration
    DEEPSEEK_API_KEY: str = (
        os.getenv("DEEPSEEK_API_KEY") or 
        yaml_config.get("providers", {}).get("deepseek", {}).get("api_key", "")
    )
    DEEPSEEK_BASE_URL: str = (
        os.getenv("DEEPSEEK_BASE_URL") or 
        yaml_config.get("providers", {}).get("deepseek", {}).get("endpoint", "https://api.deepseek.com/v1")
    )
    DEEPSEEK_DEFAULT_MODEL: str = (
        os.getenv("DEEPSEEK_DEFAULT_MODEL") or 
        yaml_config.get("providers", {}).get("deepseek", {}).get("default_model", "deepseek-chat")
    )
    
    # ADK Configuration
    GOOGLE_GENAI_USE_VERTEXAI: bool = os.getenv("GOOGLE_GENAI_USE_VERTEXAI", "FALSE").upper() == "TRUE"
    GOOGLE_API_KEY: str = os.getenv("GOOGLE_API_KEY", "")
    
    # Server Configuration
    HOST: str = os.getenv("HOST", "0.0.0.0")
    PORT: int = int(os.getenv("PORT", "8000"))
    DEBUG: bool = os.getenv("DEBUG", "false").lower() == "true"
    
    # Preprocessing Configuration
    ENABLE_CONTEXT_INJECTION: bool = os.getenv("ENABLE_CONTEXT_INJECTION", "true").lower() == "true"
    SYSTEM_PROMPT_PREFIX: str = os.getenv("SYSTEM_PROMPT_PREFIX", "You are a helpful AI assistant.")
    MAX_CONTEXT_LENGTH: int = int(os.getenv("MAX_CONTEXT_LENGTH", "4000"))
    
    # Postprocessing Configuration
    ENABLE_RESPONSE_ANALYTICS: bool = os.getenv("ENABLE_RESPONSE_ANALYTICS", "true").lower() == "true"
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")
    
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
    
    @classmethod
    def validate(cls) -> bool:
        """Validate that required configuration is present."""
        if cls.current_provider == "openai" and not cls.OPENAI_API_KEY:
            raise ValueError("OPENAI_API_KEY is required when using OpenAI provider")
        elif cls.current_provider == "deepseek" and not cls.DEEPSEEK_API_KEY:
            raise ValueError("DEEPSEEK_API_KEY is required when using DeepSeek provider")
        return True

# Global config instance
config = Config() 