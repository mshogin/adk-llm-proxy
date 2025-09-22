#!/usr/bin/env python3
"""
Content Filter for LLM Reverse Proxy
Filters reasoning and analysis content from actual LLM conversation history.
"""

import re
import logging
from typing import Dict, List, Any, Optional

logger = logging.getLogger(__name__)

# Reasoning content markers
REASONING_START_MARKER = "🧠 **REASONING START**"
REASONING_END_MARKER = "🧠 **REASONING END**"
ANALYSIS_START_MARKER = "📊 **ANALYSIS START**"
ANALYSIS_END_MARKER = "📊 **ANALYSIS END**"

def is_reasoning_content(content: str) -> bool:
    """Check if content is reasoning-related and should be filtered from LLM conversation."""
    if not content:
        return False

    reasoning_indicators = [
        "🧠 **Reasoning**:",
        "🧠 **Analysis**:",
        "🧠 **Context**:",
        "🧠 **Enhancement**:",
        "Enhanced Request to LLM:",
        "**Response Analysis:**",
        "**Request Analysis:**",
        "📊 Quality Score:",
        "📝 Content Type:",
        "📏 Word Count:",
        "😊 Sentiment:",
        "🔍 Intent:",
        "🎯 Complexity:",
        "🏷️ Domains:",
        "mshogin",
        "REASONING START",
        "REASONING END"
    ]

    return any(indicator in content for indicator in reasoning_indicators)

def is_analysis_content(content: str) -> bool:
    """Check if content is response analysis and should be filtered."""
    if not content:
        return False

    analysis_indicators = [
        "**Response Analysis:**",
        "📊 Quality Score:",
        "📝 Content Type:",
        "📏 Word Count:",
        "😊 Sentiment:",
        "mshogin"
    ]

    return any(indicator in content for indicator in analysis_indicators)

def filter_message_content(content: str) -> str:
    """Filter reasoning and analysis content from message content."""
    if not content:
        return content

    # Remove reasoning blocks
    content = re.sub(r'🧠 \*\*.*?\*\*.*?(?=\n|$)', '', content, flags=re.MULTILINE | re.DOTALL)

    # Remove analysis blocks
    content = re.sub(r'📊 \*\*.*?\*\*.*?(?=\n|$)', '', content, flags=re.MULTILINE | re.DOTALL)

    # Remove code blocks containing "Enhanced Request to LLM"
    content = re.sub(r'```\s*Enhanced Request to LLM:.*?```', '', content, flags=re.MULTILINE | re.DOTALL)

    # Remove response analysis blocks
    content = re.sub(r'\*\*Response Analysis:\*\*.*?mshogin', '', content, flags=re.MULTILINE | re.DOTALL)

    # Remove standalone "mshogin"
    content = re.sub(r'\nmshogin\s*$', '', content)

    # Clean up extra whitespace and separators
    content = re.sub(r'\n\s*---\s*\n', '\n', content)
    content = re.sub(r'\n{3,}', '\n\n', content)
    content = content.strip()

    return content

def filter_messages_for_llm(messages: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Filter reasoning and analysis content from messages before sending to LLM."""
    filtered_messages = []

    for message in messages:
        if not message:
            continue

        content = message.get("content", "")
        if not content:
            filtered_messages.append(message)
            continue

        # Filter the content
        filtered_content = filter_message_content(content)

        # Only include messages with meaningful content after filtering
        if filtered_content.strip():
            filtered_message = message.copy()
            filtered_message["content"] = filtered_content
            filtered_messages.append(filtered_message)
        elif message.get("role") == "system":
            # Keep system messages even if content becomes empty
            filtered_message = message.copy()
            filtered_message["content"] = filtered_content or "You are a helpful AI assistant."
            filtered_messages.append(filtered_message)

    logger.debug(f"🔧 Filtered {len(messages)} → {len(filtered_messages)} messages for LLM")
    return filtered_messages

def add_reasoning_markers(content: str) -> str:
    """Add markers around reasoning content for easier filtering."""
    if is_reasoning_content(content):
        return f"{REASONING_START_MARKER}\n{content}\n{REASONING_END_MARKER}"
    return content

def add_analysis_markers(content: str) -> str:
    """Add markers around analysis content for easier filtering."""
    if is_analysis_content(content):
        return f"{ANALYSIS_START_MARKER}\n{content}\n{ANALYSIS_END_MARKER}"
    return content

def should_include_in_conversation_history(content: str) -> bool:
    """Determine if content should be included in ongoing conversation history."""
    # Don't include reasoning or analysis content in conversation history
    return not (is_reasoning_content(content) or is_analysis_content(content))

# Test the filtering
if __name__ == "__main__":
    test_content = """🧠 **Reasoning**: Analyzing your request...

This is actual user content that should be kept.

🧠 **Analysis**: Simple request detected (programming)

More real content here.

```
Enhanced Request to LLM:
**User**: Help me code
```

**Response Analysis:**
📊 Quality Score: 85.0/100
📝 Content Type: text
mshogin"""

    print("Original:")
    print(test_content)
    print("\nFiltered:")
    print(filter_message_content(test_content))