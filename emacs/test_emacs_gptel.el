;;; test_emacs_gptel.el --- Test configuration for gptel with ADK proxy

;;; Commentary:
;; This file contains test configuration for gptel to work with the ADK proxy server
;; Run these functions step by step to debug the connection

;;; Code:

;; Basic gptel configuration for ADK proxy
(defun setup-adk-proxy ()
  "Set up gptel to use the ADK proxy server."
  (interactive)
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "Local ADK Proxy"
                        :host "localhost:8000"
                        :endpoint "/v1/chat/completions"
                        :stream t
                        :key "dummy"
                        :models '("gpt-4o-mini")))
  (message "✅ ADK proxy configured: %s" gptel-backend))

;; Alternative configuration forcing HTTP
(defun setup-adk-proxy-http ()
  "Set up gptel to use HTTP explicitly (not HTTPS)."
  (interactive)
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "Local ADK Proxy HTTP"
                        :host "http://localhost:8000"  ; Explicit HTTP URL
                        :endpoint "/v1/chat/completions"
                        :stream t
                        :key "dummy"
                        :models '("gpt-4o-mini")
                        :curl-args '("--insecure" "--http1.1")))  ; Force HTTP/1.1
  (message "✅ ADK proxy (HTTP) configured: %s" gptel-backend))

;; Configuration with curl args to force HTTP
(defun setup-adk-proxy-curl ()
  "Set up gptel with curl args to force HTTP."
  (interactive)
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "Local ADK Proxy CURL"
                        :host "localhost:8000"
                        :endpoint "/v1/chat/completions"
                        :stream t
                        :key "dummy"
                        :models '("gpt-4o-mini")
                        :curl-args '("--http1.1" "--no-alpn" "--insecure")))
  (message "✅ ADK proxy (CURL) configured: %s" gptel-backend))

;; Test health endpoint
(defun test-adk-health ()
  "Test the health endpoint of the ADK proxy."
  (interactive)
  (let ((url-request-method "GET")
        (url "http://localhost:8000/health"))
    (with-current-buffer (url-retrieve-synchronously url)
      (goto-char (point-min))
      (search-forward "\n\n")
      (let ((response (buffer-substring (point) (point-max))))
        (message "Health response: %s" response)
        response))))

;; Test models endpoint
(defun test-adk-models ()
  "Test the models endpoint of the ADK proxy."
  (interactive)
  (let ((url-request-method "GET")
        (url "http://localhost:8000/v1/models"))
    (with-current-buffer (url-retrieve-synchronously url)
      (goto-char (point-min))
      (search-forward "\n\n")
      (let ((response (buffer-substring (point) (point-max))))
        (message "Models response: %s" response)
        response))))

;; Test simple chat completion (non-streaming)
(defun test-adk-chat-simple ()
  "Test a simple non-streaming chat completion."
  (interactive)
  (let ((url-request-method "POST")
        (url-request-extra-headers 
         '(("Content-Type" . "application/json")
           ("Authorization" . "Bearer dummy")))
        (url-request-data 
         (json-encode '((model . "gpt-4o-mini")
                       (messages . [((role . "user") 
                                   (content . "Say hello briefly"))])
                       (stream . :false)
                       (max_tokens . 50))))
        (url "http://localhost:8000/v1/chat/completions"))
    (with-current-buffer (url-retrieve-synchronously url)
      (goto-char (point-min))
      (search-forward "\n\n")
      (let ((response (buffer-substring (point) (point-max))))
        (message "Chat response: %s" response)
        response))))

;; Test with debugging
(defun test-adk-with-debug ()
  "Test ADK proxy with detailed debugging."
  (interactive)
  (let ((url-debug t)
        (url-show-status t))
    (message "🔍 Testing health endpoint...")
    (test-adk-health)
    (sit-for 1)
    
    (message "🔍 Testing models endpoint...")
    (test-adk-models)
    (sit-for 1)
    
    (message "🔍 Testing simple chat...")
    (test-adk-chat-simple)
    (sit-for 1)
    
    (message "✅ All tests completed")))

;; Test streaming with manual HTTP
(defun test-adk-streaming ()
  "Test streaming chat completion manually."
  (interactive)
  (let ((proc (start-process "adk-test" "*adk-test*" "curl" 
                            "-X" "POST"
                            "-H" "Content-Type: application/json"
                            "-H" "Authorization: Bearer dummy"
                            "-H" "Accept: text/event-stream"
                            "-H" "User-Agent: emacs/29.1"
                            "-H" "Cache-Control: no-cache"
                            "-d" (json-encode '((model . "gpt-4o-mini")
                                              (messages . [((role . "user") 
                                                          (content . "Count to 5"))])
                                              (stream . t)
                                              (max_tokens . 100)))
                            "http://localhost:8000/v1/chat/completions")))
    (set-process-sentinel proc
                         (lambda (process event)
                           (message "Streaming test %s: %s" process event)))
    (message "Started streaming test process")))

;; Advanced gptel test with custom settings
(defun test-gptel-advanced ()
  "Test gptel with advanced settings that might trigger issues."
  (interactive)
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "ADK Proxy Debug"
                        :host "localhost:8000"
                        :endpoint "/v1/chat/completions"
                        :stream t
                        :key "dummy"
                        :models '("gpt-4o-mini")
                        :curl-args '("--http1.1" "--no-keepalive" "--verbose")))
  
  ;; Create a test buffer
  (with-current-buffer (get-buffer-create "*gptel-adk-test*")
    (erase-buffer)
    (insert "Test message for ADK proxy: Hello, can you respond with just 'OK'?")
    (gptel-mode)
    (goto-char (point-max))
    (message "🚀 Sending test message via gptel...")
    (gptel-send)))

;; Debug HTTP connections
(defun debug-adk-connection ()
  "Debug connection issues with detailed logging."
  (interactive)
  (let ((url-debug t)
        (url-show-status t)
        (url-automatic-caching nil)
        (url-request-method "POST")
        (url-request-extra-headers 
         '(("Content-Type" . "application/json")
           ("Authorization" . "Bearer dummy")
           ("User-Agent" . "emacs/debug-test")
           ("Accept" . "text/event-stream")
           ("Connection" . "close")))
        (url-request-data 
         (json-encode '((model . "gpt-4o-mini")
                       (messages . [((role . "user") 
                                   (content . "Test"))])
                       (stream . t))))
        (url "http://localhost:8000/v1/chat/completions"))
    
    (message "🔍 Making debug request...")
    (url-retrieve url
                  (lambda (status)
                    (message "Debug request completed with status: %s" status)
                    (when (buffer-live-p (current-buffer))
                      (message "Response buffer content:")
                      (message "%s" (buffer-string)))))))

;; Run all tests in sequence
(defun run-all-adk-tests ()
  "Run all ADK proxy tests in sequence."
  (interactive)
  (message "🚀 Starting comprehensive ADK proxy tests...")
  
  (setup-adk-proxy)
  (sit-for 1)
  
  (test-adk-with-debug)
  (sit-for 2)
  
  (test-adk-streaming)
  (sit-for 2)
  
  (debug-adk-connection)
  (sit-for 2)
  
  (test-gptel-advanced)
  
  (message "✅ All ADK tests initiated. Check *Messages* and *adk-test* buffers for results."))

;; Quick setup and test
(defun quick-adk-test ()
  "Quick setup and test of ADK proxy."
  (interactive)
  (setup-adk-proxy)
  (test-adk-health)
  (message "✅ Quick test completed. Try: M-x gptel"))

;; Test all HTTP configurations
(defun test-all-http-configs ()
  "Test all different HTTP configuration approaches."
  (interactive)
  (message "🔧 Testing HTTP configuration approaches...")
  
  ;; Test 1: Basic setup
  (message "🔍 Test 1: Basic setup")
  (setup-adk-proxy)
  (sit-for 1)
  
  ;; Test 2: Explicit HTTP URL
  (message "🔍 Test 2: Explicit HTTP URL")
  (setup-adk-proxy-http)
  (sit-for 1)
  
  ;; Test 3: curl args
  (message "🔍 Test 3: curl args approach")
  (setup-adk-proxy-curl)
  (sit-for 1)
  
  (message "✅ All configurations tested. Use M-x gptel to try the latest config"))

;; Debug server configuration (port 8002)
(defun setup-debug-server ()
  "Set up gptel to use the debug server on port 8002."
  (interactive)
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "Debug Server"
                        :host "http://localhost:8002"  ; Explicit HTTP
                        :endpoint "/v1/chat/completions"
                        :stream nil  ; Start with non-streaming for debugging
                        :key "dummy"
                        :models '("gpt-4o-mini")))
  (message "🐛 Debug server configured: %s" gptel-backend))

;; Test with debug server
(defun test-debug-server ()
  "Test with the raw debug server to see HTTP requests."
  (interactive)
  (setup-debug-server)
  (message "🔍 Configured for debug server on port 8002")
  (message "📝 Start debug server: python debug_raw_server.py")
  (message "💬 Send a test message to see raw HTTP data")
  
  ;; Create test buffer and send message
  (with-current-buffer (get-buffer-create "*gptel-debug*")
    (erase-buffer)
    (insert "Debug test: Can you see this request?")
    (gptel-mode)
    (goto-char (point-max))
    (message "🚀 Sending debug message...")
    (gptel-send)))

;; Test streaming with debug server
(defun test-debug-streaming ()
  "Test streaming with debug server."
  (interactive)
  (setq gptel-model "gpt-4o-mini"
        gptel-backend (gptel-make-openai "Debug Streaming"
                        :host "http://localhost:8002"  ; Explicit HTTP
                        :endpoint "/v1/chat/completions"
                        :stream t  ; Enable streaming
                        :key "dummy"
                        :models '("gpt-4o-mini")
                        :curl-args '("--http1.1" "--insecure")))  ; Force HTTP
  (message "🐛 Debug streaming configured")
  
  (with-current-buffer (get-buffer-create "*gptel-debug-stream*")
    (erase-buffer)
    (insert "Streaming test: Count to 5")
    (gptel-mode)
    (goto-char (point-max))
    (message "🚀 Sending debug streaming message...")
    (gptel-send)))

(provide 'test_emacs_gptel)
;;; test_emacs_gptel.el ends here 