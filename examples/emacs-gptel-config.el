;; Emacs gptel configuration for ADK LLM Golang Proxy
;; This configures gptel to use the local Golang proxy as an OpenAI-compatible backend

;; Installation:
;; 1. Install gptel: M-x package-install RET gptel RET
;; 2. Add this configuration to your init.el or evaluate it
;; 3. Start the Golang proxy: make go-run (or ./bin/proxy --config config-golang.yaml)
;; 4. Use gptel: M-x gptel

(use-package gptel
  :config
  ;; Define the ADK LLM Golang Proxy backend
  (setq gptel-backend
        (gptel-make-openai "ADK Proxy (Golang)"
          :host "localhost:8001"
          :endpoint "/v1/chat/completions"
          :stream t
          :key "dummy"  ; Not required for local proxy
          :models '("gpt-4o-mini"
                    "gpt-4o"
                    "claude-3-5-sonnet-20241022"
                    "deepseek-chat")))

  ;; Set default model
  (setq gptel-model "gpt-4o-mini")

  ;; Optional: Set custom header to select workflow
  ;; Available workflows: "default", "basic", "advanced"
  (setq gptel-api-extra-headers
        '(("X-Workflow" . "basic")))  ; Change to "default" or "advanced" as needed

  ;; Optional: Customize gptel behavior
  (setq gptel-default-mode 'org-mode)  ; Use org-mode for gptel buffers

  ;; Key bindings (optional)
  :bind (("C-c g" . gptel)
         ("C-c G" . gptel-menu)))

;; Alternative: Minimal configuration without use-package
;; Uncomment if you don't use use-package:
;;
;; (require 'gptel)
;; (setq gptel-backend
;;       (gptel-make-openai "ADK Proxy (Golang)"
;;         :host "localhost:8001"
;;         :endpoint "/v1/chat/completions"
;;         :stream t
;;         :key "dummy"
;;         :models '("gpt-4o-mini" "gpt-4o" "claude-3-5-sonnet-20241022" "deepseek-chat")))
;; (setq gptel-model "gpt-4o-mini")
;; (setq gptel-api-extra-headers '(("X-Workflow" . "basic")))

;; Usage:
;; 1. Start the proxy: make go-run
;; 2. In Emacs: M-x gptel
;; 3. Type your prompt and press C-c RET to send
;; 4. The response will stream back in real-time

;; Workflow descriptions:
;; - "default": Simple pass-through (returns "Hello World")
;; - "basic": Intent detection via regex/keywords (no LLM for reasoning)
;; - "advanced": Multi-agent orchestration (ADK Python + OpenAI native)

;; Troubleshooting:
;; - Check proxy is running: curl http://localhost:8001/health
;; - Check available workflows: curl http://localhost:8001/workflows
;; - View proxy logs in the terminal where you ran make go-run
;; - Test with curl:
;;   curl -X POST http://localhost:8001/v1/chat/completions \
;;     -H "Content-Type: application/json" \
;;     -H "X-Workflow: basic" \
;;     -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}],"stream":true}'
