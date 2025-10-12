#!/usr/bin/env python3
"""
Test the enhanced multi-agent reasoning system.
"""
import requests
import json
import time

def test_enhanced_reasoning():
    """Test the enhanced multi-agent reasoning system."""
    url = "http://localhost:8001/v1/chat/completions"

    headers = {
        "Content-Type": "application/json",
        "Authorization": "Bearer test-key"
    }

    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "I would like to see all tickets assigned to me"}
        ],
        "stream": True
    }

    print("🧠 Testing Enhanced Multi-Agent Reasoning System...")
    print("📝 Request: 'I would like to see all tickets assigned to me'")
    print("🎯 Expected: LLM-powered intent analysis → plan generation → recursive execution → context evaluation")
    print("=" * 80)

    try:
        response = requests.post(url, headers=headers, json=payload, stream=True)
        response.raise_for_status()

        print("📡 Enhanced Reasoning Response:")
        reasoning_phases = []

        for line in response.iter_lines():
            if line:
                line_str = line.decode('utf-8')
                if line_str.startswith('data: '):
                    data_str = line_str[6:]  # Remove 'data: ' prefix
                    if data_str.strip() == '[DONE]':
                        break
                    try:
                        data = json.loads(data_str)
                        if 'choices' in data and len(data['choices']) > 0:
                            delta = data['choices'][0].get('delta', {})
                            content = delta.get('content', '')
                            if content:
                                print(content, end='', flush=True)

                                # Track reasoning phases
                                if "Intent Analysis" in content:
                                    reasoning_phases.append("Intent Analysis")
                                elif "Plan Generation" in content:
                                    reasoning_phases.append("Plan Generation")
                                elif "Plan Execution" in content:
                                    reasoning_phases.append("Plan Execution")
                                elif "Context Evaluation" in content:
                                    reasoning_phases.append("Context Evaluation")
                                elif "LLM-Powered" in content:
                                    reasoning_phases.append("LLM-Powered")

                    except json.JSONDecodeError:
                        pass

        print("\n" + "=" * 80)
        print("✅ Enhanced Reasoning Test Complete")
        print(f"🔍 Detected phases: {set(reasoning_phases)}")

        # Check if we got the expected enhanced reasoning phases
        expected_phases = ["Intent Analysis", "Plan Generation", "Plan Execution", "Context Evaluation"]
        detected_phases = set(reasoning_phases)

        if all(phase in str(reasoning_phases) for phase in expected_phases):
            print("✅ SUCCESS: All expected reasoning phases detected!")
        else:
            print(f"⚠️  Some phases may be missing. Expected: {expected_phases}")

    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
    except Exception as e:
        print(f"❌ Test failed: {e}")

if __name__ == "__main__":
    test_enhanced_reasoning()