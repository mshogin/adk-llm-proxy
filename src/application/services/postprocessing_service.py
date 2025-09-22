import json
import logging
import time
from typing import Dict, List, Any, Optional
from google.adk.agents import LlmAgent
from google.adk.tools import ToolContext
from src.infrastructure.config.config import config

logger = logging.getLogger(__name__)

def create_unified_analysis(request_metadata: Optional[Dict[str, Any]], response_content: str, response_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Create a unified analysis combining request analysis and response analysis.

    Args:
        request_metadata: Metadata from the original request (reasoning analysis)
        response_content: The complete response content from the LLM
        response_metadata: Optional metadata about the response

    Returns:
        Dictionary with unified analysis results
    """
    try:
        # Generate response analysis
        response_analysis = analyze_response_content(response_content, response_metadata)

        # Combine request and response analysis
        unified_analysis = {
            "request_analysis": request_metadata.get("intent_analysis", {}) if request_metadata else {},
            "response_analysis": response_analysis.get("analysis", {}),
            "created_at": time.time(),
            "status": "success"
        }

        # Create analysis display content
        request_intent = unified_analysis["request_analysis"]
        response_info = unified_analysis["response_analysis"]

        analysis_content = "\n\n**Request & Response Analysis:**\n"

        # Request analysis section
        if request_intent:
            analysis_content += f"🔍 Intent: {request_intent.get('complexity', 'unknown')} request"
            domains = request_intent.get('domains', [])
            if domains:
                analysis_content += f" ({', '.join(domains)})"
            analysis_content += f"\n🎯 Complexity: {request_intent.get('word_count', 0)} words\n"

        # Response analysis section
        if response_info:
            analysis_content += f"📊 Quality Score: {response_info.get('quality_score', 0)}/100\n"
            analysis_content += f"📝 Content Type: {response_info.get('content_type', 'text')}\n"
            analysis_content += f"📏 Word Count: {response_info.get('length_words', 0)} words\n"
            analysis_content += f"😊 Sentiment: {response_info.get('sentiment', 'neutral')}\n"

        analysis_content += "mshogin"

        return {
            "status": "success",
            "unified_analysis": unified_analysis,
            "analysis_content": analysis_content,
            "should_display": True
        }

    except Exception as e:
        logger.error(f"Error creating unified analysis: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "analysis_content": "",
            "should_display": False
        }

def analyze_response_content(content: str, metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Analyze the content of the response for quality, safety, and insights.

    Args:
        content: The complete response content from the LLM
        metadata: Optional metadata about the response

    Returns:
        Dictionary with analysis results
    """
    logger.debug("ANALYZE_RESPONSE_CONTENT")
    try:
        analysis = {
            "length_chars": len(content),
            "length_words": len(content.split()),
            "length_sentences": content.count('.') + content.count('!') + content.count('?'),
            "has_code_blocks": '```' in content,
            "has_links": 'http' in content.lower(),
            "language_detected": "english",  # Simple detection
            "sentiment": "neutral",
            "quality_score": 0.0,
            "safety_flags": [],
            "content_type": "text"
        }

        # Basic quality scoring
        quality_score = 50.0  # Base score

        # Length quality
        if 50 <= analysis["length_words"] <= 500:
            quality_score += 20
        elif analysis["length_words"] > 500:
            quality_score += 15
        elif analysis["length_words"] >= 10:
            quality_score += 10

        # Structure quality
        if analysis["length_sentences"] >= 2:
            quality_score += 15

        if analysis["has_code_blocks"]:
            quality_score += 10
            analysis["content_type"] = "technical"

        # Simple sentiment analysis (very basic)
        positive_words = ["good", "great", "excellent", "helpful", "wonderful", "perfect"]
        negative_words = ["bad", "terrible", "awful", "horrible", "wrong", "error"]

        content_lower = content.lower()
        positive_count = sum(1 for word in positive_words if word in content_lower)
        negative_count = sum(1 for word in negative_words if word in content_lower)

        if positive_count > negative_count:
            analysis["sentiment"] = "positive"
            quality_score += 5
        elif negative_count > positive_count:
            analysis["sentiment"] = "negative"

        # Safety checks (basic)
        safety_keywords = ["password", "secret", "private", "confidential", "hack", "illegal"]
        for keyword in safety_keywords:
            if keyword in content_lower:
                analysis["safety_flags"].append(f"Contains: {keyword}")

        analysis["quality_score"] = min(100.0, quality_score)

        return {
            "status": "success",
            "analysis": analysis,
            "timestamp": time.time()
        }

    except Exception as e:
        logger.error(f"Error analyzing response content: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "analysis": {}
        }

def enhance_response(content: str, enhancement_type: str = "standard") -> Dict[str, Any]:
    """
    Enhance the response content based on specified enhancement type.

    Args:
        content: The original response content
        enhancement_type: Type of enhancement to apply

    Returns:
        Dictionary with enhanced content
    """
    try:
        enhanced_content = content
        enhancements_applied = []

        if enhancement_type == "standard":
            # Basic enhancements

            # Ensure proper sentence endings
            if enhanced_content and not enhanced_content.rstrip().endswith(('.', '!', '?')):
                enhanced_content = enhanced_content.rstrip() + "."
                enhancements_applied.append("Added sentence ending")

            # Remove excessive whitespace
            lines = enhanced_content.split('\n')
            cleaned_lines = []
            for line in lines:
                cleaned_line = ' '.join(line.split())  # Remove extra spaces
                cleaned_lines.append(cleaned_line)
            enhanced_content = '\n'.join(cleaned_lines)

            # Remove excessive empty lines
            while '\n\n\n' in enhanced_content:
                enhanced_content = enhanced_content.replace('\n\n\n', '\n\n')
                enhancements_applied.append("Cleaned whitespace")

        elif enhancement_type == "verbose":
            # Add more detailed explanations
            if len(content.split()) < 20:
                enhanced_content += "\n\nWould you like me to elaborate on any particular aspect of this response?"
                enhancements_applied.append("Added elaboration prompt")

        elif enhancement_type == "concise":
            # Make response more concise
            sentences = content.split('. ')
            if len(sentences) > 3:
                enhanced_content = '. '.join(sentences[:3]) + "."
                enhancements_applied.append("Condensed response")

        return {
            "status": "success",
            "enhanced_content": enhanced_content,
            "original_length": len(content),
            "enhanced_length": len(enhanced_content),
            "enhancements_applied": enhancements_applied
        }

    except Exception as e:
        logger.error(f"Error enhancing response: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "enhanced_content": content  # Return original on error
        }

def log_interaction(request_metadata: Dict[str, Any], response_metadata: Dict[str, Any],
                   analysis: Dict[str, Any]) -> Dict[str, Any]:
    """
    Log the complete interaction for analytics and monitoring.

    Args:
        request_metadata: Metadata from the original request
        response_metadata: Metadata from the OpenAI response
        analysis: Analysis of the response content

    Returns:
        Dictionary with logging results
    """
    try:
        if not config.ENABLE_RESPONSE_ANALYTICS:
            return {
                "status": "skipped",
                "message": "Analytics disabled"
            }

        interaction_log = {
            "timestamp": time.time(),
            "request": {
                "model": request_metadata.get("model"),
                "message_count": request_metadata.get("message_count", 0),
                "estimated_tokens": request_metadata.get("total_tokens_estimate", 0),
                "conversation_type": request_metadata.get("conversation_type"),
                "has_system_message": request_metadata.get("has_system_message", False)
            },
            "response": {
                "response_id": response_metadata.get("response_id"),
                "model_used": response_metadata.get("model_used"),
                "finish_reason": response_metadata.get("finish_reason"),
                "total_tokens": response_metadata.get("total_tokens", 0),
                "prompt_tokens": response_metadata.get("prompt_tokens", 0),
                "completion_tokens": response_metadata.get("completion_tokens", 0)
            },
            "analysis": {
                "quality_score": analysis.get("quality_score", 0),
                "content_type": analysis.get("content_type", "text"),
                "sentiment": analysis.get("sentiment", "neutral"),
                "safety_flags": analysis.get("safety_flags", []),
                "length_words": analysis.get("length_words", 0)
            }
        }

        # Log to appropriate destination (file, database, etc.)
        logger.info(f"Interaction logged: {json.dumps(interaction_log, indent=2)}")

        return {
            "status": "success",
            "logged": True,
            "log_entry": interaction_log
        }

    except Exception as e:
        logger.error(f"Error logging interaction: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "logged": False
        }

def add_chat_content(content: str, content_type: str = "analysis", metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Add custom content to the chat that will be visible to the user.

    Args:
        content: The original response content
        content_type: Type of content to add (analysis, summary, note, etc.)
        metadata: Optional metadata for the content

    Returns:
        Dictionary with the content to be added to the chat
    """
    try:
        chat_content = ""

        if content_type == "analysis":
            # Add analysis information
            analysis_result = analyze_response_content(content, metadata)
            if analysis_result.get("status") == "success":
                analysis = analysis_result.get("analysis", {})
                chat_content = f"\n\n---\n\n**Response Analysis:**\n"
                chat_content += f"📊 Quality Score: {analysis.get('quality_score', 0):.1f}/100\n"
                chat_content += f"📝 Content Type: {analysis.get('content_type', 'text')}\n"
                chat_content += f"📏 Word Count: {analysis.get('length_words', 0)}\n"
                chat_content += f"😊 Sentiment: {analysis.get('sentiment', 'neutral')}\n"

                if analysis.get('safety_flags'):
                    chat_content += f"⚠️ Safety Flags: {', '.join(analysis['safety_flags'])}\n"

        elif content_type == "summary":
            # Add a summary of the response
            words = content.split()
            chat_content = f"\n\n---\n\n**Summary:**\n"
            chat_content += f"📄 Response length: {len(words)} words\n"
            chat_content += f"📊 Key points: {min(3, len(words) // 10)} main ideas\n"

            # Extract first sentence as summary
            sentences = content.split('. ')
            if sentences:
                chat_content += f"💡 Main point: {sentences[0]}.\n"

        elif content_type == "custom":
            # Add custom content from metadata
            if metadata and "custom_message" in metadata:
                chat_content = f"\n\n---\n\n{metadata['custom_message']}\n"

        elif content_type == "enhancement":
            # Add enhancement information
            enhancement_result = enhance_response(content, "standard")
            if enhancement_result.get("status") == "success":
                enhancements = enhancement_result.get("enhancements_applied", [])
                if enhancements:
                    chat_content = f"\n\n---\n\n**Enhancements Applied:**\n"
                    for enhancement in enhancements:
                        chat_content += f"✅ {enhancement}\n"

        chat_content += "mshogin"
        return {
            "status": "success",
            "chat_content": chat_content,
            "content_type": content_type,
            "should_add": len(chat_content.strip()) > 0
        }

    except Exception as e:
        logger.error(f"Error adding chat content: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "chat_content": "",
            "should_add": False
        }

def filter_response(content: str, filter_type: str = "basic") -> Dict[str, Any]:
    """
    Apply content filtering to the response.

    Args:
        content: The response content to filter
        filter_type: Type of filtering to apply

    Returns:
        Dictionary with filtering results
    """
    try:
        filtered_content = content
        filters_applied = []

        if filter_type == "basic":
            # Basic safety filtering

            # Remove potential sensitive information patterns
            import re

            # Email patterns (simple)
            email_pattern = r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
            if re.search(email_pattern, filtered_content):
                filtered_content = re.sub(email_pattern, '[EMAIL]', filtered_content)
                filters_applied.append("Masked email addresses")

            # Phone number patterns (simple US format)
            phone_pattern = r'\b\d{3}-\d{3}-\d{4}\b|\b\(\d{3}\)\s*\d{3}-\d{4}\b'
            if re.search(phone_pattern, filtered_content):
                filtered_content = re.sub(phone_pattern, '[PHONE]', filtered_content)
                filters_applied.append("Masked phone numbers")

            # Credit card patterns (simple)
            cc_pattern = r'\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b'
            if re.search(cc_pattern, filtered_content):
                filtered_content = re.sub(cc_pattern, '[CARD]', filtered_content)
                filters_applied.append("Masked credit card numbers")

        return {
            "status": "success",
            "filtered_content": filtered_content,
            "original_content": content,
            "filters_applied": filters_applied,
            "content_modified": len(filters_applied) > 0
        }

    except Exception as e:
        logger.error(f"Error filtering response: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "filtered_content": content  # Return original on error
        }

# Create the postprocessing agent using new ADK API
from google.adk.agents import Agent

postprocessing_agent = Agent(
    name="postprocessing_agent",
    model="gemini-2.0-flash",
    tools=[analyze_response_content, enhance_response, log_interaction, filter_response, add_chat_content],
    instruction="""You are a postprocessing agent for an LLM reverse proxy server.

    Your responsibilities:
    1. Analyze response content using analyze_response_content()
    2. Apply content filtering using filter_response()
    3. Enhance responses when appropriate using enhance_response()
    4. Log interactions for analytics using log_interaction()

    When processing a response:
    1. First analyze the content for quality and safety
    2. Apply any necessary content filtering
    3. Enhance the response if requested or beneficial
    4. Log the complete interaction for monitoring

    Preserve the original response integrity while adding value through analysis and filtering.
    Always return the processed content ready for delivery to the client.""",
    description="Handles response postprocessing, analysis, filtering, and enhancement"
)
