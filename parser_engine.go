package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Configuración de la API IA
type AIConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

// Respuesta de OpenAI
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Respuesta de Gemini
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// Respuesta de Ollama
type OllamaResponse struct {
	Response string `json:"response"`
}

// getAIConfig obtiene la configuración desde variables de entorno
func getAIConfig() AIConfig {
	config := AIConfig{
		Provider: getEnvOrDefault("AI_PROVIDER", "ollama"),
		BaseURL:  "",
		APIKey:   "",
		Model:    "",
	}

	switch config.Provider {
	case "openai":
		config.BaseURL = getEnvOrDefault("AI_BASE_URL", "https://api.openai.com/v1/chat/completions")
		config.APIKey = os.Getenv("AI_API_KEY")
		config.Model = getEnvOrDefault("AI_MODEL", "gpt-3.5-turbo")
	case "gemini":
		config.BaseURL = getEnvOrDefault("AI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/models")
		config.APIKey = os.Getenv("AI_API_KEY")
		config.Model = getEnvOrDefault("AI_MODEL", "gemini-pro")
	case "ollama":
		config.BaseURL = getEnvOrDefault("AI_BASE_URL", "http://localhost:11434/api/generate")
		config.Model = getEnvOrDefault("AI_MODEL", "llama2")
	}

	return config
}

// getEnvOrDefault obtiene variable de entorno o valor por defecto
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// callAIAPI realiza la llamada HTTP a la API de IA
func callAIAPI(prompt string) (string, error) {
	config := getAIConfig()

	var payload interface{}
	var endpoint string

	switch config.Provider {
	case "openai":
		payload = map[string]interface{}{
			"model": config.Model,
			"messages": []map[string]string{
				{"role": "system", "content": "Eres un asistente que convierte lenguaje natural a comandos de Unix/Linux. Responde SOLO con el comando, sin explicaciones."},
				{"role": "user", "content": prompt},
			},
			"max_tokens": 100,
		}
		endpoint = config.BaseURL
	case "gemini":
		endpoint = fmt.Sprintf("%s/%s:generateContent?key=%s", config.BaseURL, config.Model, config.APIKey)
		payload = map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": fmt.Sprintf("Eres un asistente que convierte lenguaje natural a comandos de Unix/Linux. Responde SOLO con el comando, sin explicaciones. Usuario: %s", prompt)},
					},
				},
			},
		}
	case "ollama":
		payload = map[string]interface{}{
			"model":  config.Model,
			"prompt": fmt.Sprintf("Eres un asistente que convierte lenguaje natural a comandos de Unix/Linux. Responde SOLO con el comando, sin explicaciones. Usuario: %s", prompt),
			"stream": false,
		}
		endpoint = config.BaseURL
	default:
		return "", fmt.Errorf("proveedor no soportado: %s", config.Provider)
	}

	// Serializar payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error serializando payload: %v", err)
	}

	// Crear request HTTP
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creando request: %v", err)
	}

	// Setear headers
	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		if config.Provider == "openai" {
			req.Header.Set("Authorization", "Bearer "+config.APIKey)
		}
	}

	// Ejecutar request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error en request HTTP: %v", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error leyendo respuesta: %v", err)
	}

	// Manejar errores HTTP
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("error HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parsear respuesta según provider
	var rawResponse string
	switch config.Provider {
	case "openai":
		var openAIResp OpenAIResponse
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			return "", fmt.Errorf("error parseando respuesta OpenAI: %v", err)
		}
		if len(openAIResp.Choices) > 0 {
			rawResponse = openAIResp.Choices[0].Message.Content
		}
	case "gemini":
		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return "", fmt.Errorf("error parseando respuesta Gemini: %v", err)
		}
		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			rawResponse = geminiResp.Candidates[0].Content.Parts[0].Text
		}
	case "ollama":
		var ollamaResp OllamaResponse
		if err := json.Unmarshal(body, &ollamaResp); err != nil {
			return "", fmt.Errorf("error parseando respuesta Ollama: %v", err)
		}
		rawResponse = ollamaResp.Response
	}

	return rawResponse, nil
}

// sanitizeCommand limpia y extrae el comando ejecutable de la respuesta IA
func sanitizeCommand(raw string) string {
	// Trim espacios
	raw = strings.TrimSpace(raw)

	// Caso 1: Bloque de código con triple backticks
	backtickRegex := regexp.MustCompile("```(?:bash|sh|zsh|shell)?\n?(.*?)\n?```")
	matches := backtickRegex.FindStringSubmatch(raw)
	if len(matches) > 1 {
		command := strings.TrimSpace(matches[1])
		return getFirstNonEmptyLine(command)
	}

	// Caso 2: Inline code con backticks
	inlineRegex := regexp.MustCompile("`([^`]+)`")
	matches = inlineRegex.FindStringSubmatch(raw)
	if len(matches) > 1 {
		command := strings.TrimSpace(matches[1])
		return command
	}

	// Caso 3: Primera línea que parece comando
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !looksLikeExplanation(line) {
			// Limpiar prompts tipo $, neri>, Emiliano>
			line = regexp.MustCompile(`^\$|^\s*neri>|^\s*Emiliano>`).ReplaceAllString(line, "")
			return strings.TrimSpace(line)
		}
	}

	// Si nada funciona, retornar la primera línea no vacía
	return getFirstNonEmptyLine(raw)
}

// getFirstNonEmptyLine obtiene la primera línea no vacía
func getFirstNonEmptyLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// looksLikeExplanation verifica si una línea parece explicación
func looksLikeExplanation(line string) bool {
	explanationPatterns := []string{
		`^para`,
		`^usa`,
		`^este`,
		`^el comando`,
		`^la respuesta`,
		`^you can`,
		`^this will`,
		`^use`,
		`^the command`,
	}

	for _, pattern := range explanationPatterns {
		if matched, _ := regexp.MatchString(pattern, strings.ToLower(line)); matched {
			return true
		}
	}
	return false
}

// TranslateToCommand función principal que orquesta la traducción
func TranslateToCommand(userText string) (string, string, error) {
	rawResponse, err := callAIAPI(userText)
	if err != nil {
		// Mensaje de error más amigable
		return "", "", fmt.Errorf("no se pudo conectar con la IA (verifica tu conexión o API key): %v", err)
	}

	sanitizedCommand := sanitizeCommand(rawResponse)
	if sanitizedCommand == "" {
		return rawResponse, "", fmt.Errorf("la IA no pudo generar un comando válido")
	}

	return rawResponse, sanitizedCommand, nil
}
