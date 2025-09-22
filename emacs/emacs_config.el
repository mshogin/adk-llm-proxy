;; Emacs Configuration for ADK-based LLM Reverse Proxy
;; This configuration sets up gptel and other tools to work with your local ADK proxy

;; Ensure gptel is installed
;; M-x package-install RET gptel RET

(use-package gptel
  :ensure t
  :config
  ;; Configure the ADK-based reverse proxy as the default backend
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "ADK Local Proxy"
                        :host "localhost:8000"
                        :endpoint "/v1/chat/completions"
                        :stream t
                        :key "dummy"  ; Not used but required
                        :models '("gpt-4o-mini" "gpt-4o" "gpt-3.5-turbo")))
  
  ;; Set default directives (these will be enhanced by ADK preprocessing)
  (setq gptel-directives
        '((default . "You are a helpful AI assistant.")
          (programming . "You are an expert programmer. Provide clear, well-commented code and explain your reasoning.")
          (writing . "You are a writing assistant. Help improve clarity, style, and grammar.")
          (research . "You are a research assistant. Provide accurate, well-sourced information.")
          (debug . "You are a debugging expert. Help identify and fix code issues.")
          (explain . "You are a teacher. Explain concepts clearly with examples.")))
  
  ;; Key bindings
  :bind (("C-c g g" . gptel)
         ("C-c g s" . gptel-send)
         ("C-c g r" . gptel-rewrite-menu)
         ("C-c g m" . gptel-menu)
         ("C-c g k" . gptel-abort)
         ("C-c g i" . gptel-add-file)))

;; Optional: Configure multiple backends for different use cases
(defun adk-setup-multiple-backends ()
  "Setup multiple ADK backends for different scenarios."
  
  ;; Main ADK proxy (with preprocessing/postprocessing)
  (setq adk-main-backend
        (gptel-make-openai "ADK Main"
          :host "localhost:8000"
          :endpoint "/v1/chat/completions"
          :stream t
          :key "dummy"
          :models '("gpt-4o-mini" "gpt-4o")))
  
  ;; Direct OpenAI (for comparison)
  (when (getenv "OPENAI_API_KEY")
    (setq adk-direct-backend
          (gptel-make-openai "OpenAI Direct"
            :key (getenv "OPENAI_API_KEY")
            :stream t
            :models '("gpt-4o-mini" "gpt-4o" "gpt-3.5-turbo"))))
  
  ;; Switch backends easily
  (defun adk-use-proxy ()
    "Switch to ADK proxy backend."
    (interactive)
    (setq gptel-backend adk-main-backend)
    (message "Using ADK Proxy backend"))
  
  (defun adk-use-direct ()
    "Switch to direct OpenAI backend."
    (interactive)
    (if (boundp 'adk-direct-backend)
        (progn
          (setq gptel-backend adk-direct-backend)
          (message "Using Direct OpenAI backend"))
      (message "Direct OpenAI backend not configured")))
  
  ;; Bind backend switching
  (global-set-key (kbd "C-c g p") 'adk-use-proxy)
  (global-set-key (kbd "C-c g d") 'adk-use-direct))

;; Setup multiple backends
(adk-setup-multiple-backends)

;; Optional: Custom functions for specific ADK features
(defun adk-test-connection ()
  "Test connection to ADK proxy server."
  (interactive)
  (let ((url "http://localhost:8000/health"))
    (url-retrieve url
                  (lambda (status)
                    (if (plist-get status :error)
                        (message "❌ ADK proxy connection failed: %s" (plist-get status :error))
                      (goto-char (point-min))
                      (search-forward "\n\n")
                      (let ((response (json-read)))
                        (if (string= (alist-get 'status response) "healthy")
                            (message "✅ ADK proxy is healthy")
                          (message "⚠️ ADK proxy health check failed"))))))))

(defun adk-show-server-status ()
  "Show ADK server configuration and status."
  (interactive)
  (let ((url "http://localhost:8000/health"))
    (with-current-buffer (url-retrieve-synchronously url)
      (goto-char (point-min))
      (search-forward "\n\n")
      (let* ((response (json-read))
             (config (alist-get 'config response)))
        (message "ADK Status: OpenAI=%s, Context=%s, Analytics=%s"
                 (if (alist-get 'openai_configured config) "✅" "❌")
                 (if (alist-get 'context_injection config) "✅" "❌")
                 (if (alist-get 'analytics config) "✅" "❌"))))))

;; Optional: Enhanced org-mode integration
(use-package org
  :config
  ;; Setup gptel for org-mode
  (add-hook 'org-mode-hook
            (lambda ()
              ;; Enable gptel in org buffers
              (setq-local gptel-use-context t)
              
              ;; Custom org-mode directive
              (setq-local gptel-directives
                          (append gptel-directives
                                  '((org . "You are helping with an Org-mode document. Maintain proper formatting and structure."))))))
  
  ;; Bind gptel in org-mode
  :bind (:map org-mode-map
         ("C-c g o" . gptel)))

;; Optional: Code-specific enhancements
(defun adk-code-assistant ()
  "Invoke gptel with programming-specific directive."
  (interactive)
  (let ((gptel-default-mode 'programming))
    (call-interactively 'gptel)))

(defun adk-debug-assistant ()
  "Invoke gptel for debugging help."
  (interactive)
  (let ((gptel-default-mode 'debug))
    (call-interactively 'gptel)))

(defun adk-explain-code ()
  "Ask gptel to explain selected code."
  (interactive)
  (if (use-region-p)
      (let ((code (buffer-substring-no-properties (region-beginning) (region-end)))
            (gptel-default-mode 'explain))
        (with-current-buffer (get-buffer-create "*GPTel Explanation*")
          (erase-buffer)
          (insert (format "Please explain this code:\n\n```%s\n%s\n```"
                         (if (derived-mode-p 'prog-mode)
                             (substring (symbol-name major-mode) 0 -5)
                           "")
                         code))
          (gptel-mode)
          (gptel-send)
          (switch-to-buffer-other-window (current-buffer))))
    (message "Please select code to explain")))

;; Bind code assistance functions
(global-set-key (kbd "C-c g c") 'adk-code-assistant)
(global-set-key (kbd "C-c g b") 'adk-debug-assistant)
(global-set-key (kbd "C-c g e") 'adk-explain-code)

;; Optional: Integration with other packages
(use-package markdown-mode
  :ensure t
  :config
  ;; Setup gptel for markdown
  (add-hook 'markdown-mode-hook
            (lambda ()
              (setq-local gptel-default-mode 'writing))))

;; Optional: Customization for ADK-specific features
(defcustom adk-proxy-url "http://localhost:8000"
  "URL of the ADK proxy server."
  :type 'string
  :group 'gptel)

(defcustom adk-enable-analytics t
  "Whether to enable analytics logging in ADK proxy."
  :type 'boolean
  :group 'gptel)

;; Helper function to restart gptel session
(defun adk-restart-session ()
  "Clear gptel context and start fresh."
  (interactive)
  (when (gptel--in-response-p)
    (gptel-abort))
  (setq gptel-context nil)
  (message "GPTel session restarted"))

(global-set-key (kbd "C-c g x") 'adk-restart-session)

;; Status line indicator
(defun adk-mode-line-status ()
  "Return ADK proxy status for mode line."
  (condition-case nil
      (let ((url-request-timeout 1))
        (with-current-buffer (url-retrieve-synchronously 
                             (concat adk-proxy-url "/health"))
          "ADK"))
    (error "ADK-?")))

;; Add to mode line (optional)
;; (setq-default mode-line-format
;;               (append mode-line-format
;;                       '(" [" (:eval (adk-mode-line-status)) "]")))

;; Useful commands summary:
;; C-c g g   - Start gptel chat
;; C-c g s   - Send current buffer/region to gptel
;; C-c g r   - Rewrite menu
;; C-c g m   - Gptel menu
;; C-c g k   - Abort current request
;; C-c g i   - Add file to context
;; C-c g p   - Switch to ADK proxy backend
;; C-c g d   - Switch to direct OpenAI backend
;; C-c g c   - Code assistant mode
;; C-c g b   - Debug assistant mode
;; C-c g e   - Explain selected code
;; C-c g x   - Restart session

(message "ADK-based LLM Reverse Proxy configuration loaded!")
(message "Use M-x adk-test-connection to verify proxy connectivity") 