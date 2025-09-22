# 🤖 ADK LLM Proxy

A smart, streaming-first LLM proxy that thinks before it responds. Built with Google's Agent Development Kit for intelligent request processing.

## Why This Exists

Ever wished your LLM could be smarter about how it processes requests? This proxy adds an intelligent agent layer that:

- **Preprocesses** your requests to understand context better
- **Reasons** about the best way to handle each query
- **Postprocesses** responses to add value and insights
- **Streams everything** for that snappy real-time feel

Think of it as giving your LLM a brain upgrade 🧠

## ⚡ Quick Start

```bash
# Clone and install
git clone https://github.com/yourusername/adk-llm-proxy.git
cd adk-llm-proxy
pip install -r requirements.txt

# Start with OpenAI (most common)
python main.py -provider openai -model gpt-4o-mini

# Or try with local Ollama
python main.py -provider ollama -model mistral
```

Set your API key first:
```bash
export OPENAI_API_KEY="your-key-here"
```

That's it! Server runs on `http://localhost:8001` 🚀

## 🎯 What Makes It Cool

### Smart Agent Pipeline
Every request flows through intelligent agents:
```
Your Request → 🔍 Preprocessing → 🧠 Reasoning → 🤖 LLM → ✨ Postprocessing → Response
```

### Multiple Providers
- **OpenAI** (GPT-4, GPT-3.5, etc.)
- **Ollama** (local models like Mistral, Llama)
- **DeepSeek** (cost-effective alternative)

### Streaming-First
No waiting around. Responses stream as they're generated.

## 💬 Usage

Drop-in replacement for OpenAI's API:

```bash
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "What is the meaning of life?"}],
    "stream": true
  }'
```

Works with any OpenAI-compatible client or library.

## 🏗️ How It's Built

Clean, maintainable architecture using Domain-Driven Design:

```
src/
├── application/     # Business logic & orchestration
├── domain/          # Core reasoning services
├── infrastructure/  # External integrations (ADK, APIs)
└── presentation/    # FastAPI web layer
```

## 🔧 Configuration

Want to customize the agent behavior? Check out `src/infrastructure/config/` for:

- Provider settings (API keys, endpoints)
- Agent configurations (preprocessing rules, reasoning logic)
- Server options (host, port, debug mode)

## 🚨 Troubleshooting

**"Google ADK not found"**
```bash
pip install google-adk
```

**"Can't connect to Ollama"**
Make sure Ollama is running: `ollama serve`

**Import errors**
Run from project root, not subdirectories.

## 🤝 Contributing

Found a bug? Want to add a feature? PRs welcome!

1. Fork it
2. Create your feature branch (`git checkout -b my-cool-feature`)
3. Commit your changes (`git commit -am 'Add cool feature'`)
4. Push to the branch (`git push origin my-cool-feature`)
5. Create a Pull Request

## 📝 License

MIT - do whatever you want with it!

---

*Built with ❤️ and way too much coffee*