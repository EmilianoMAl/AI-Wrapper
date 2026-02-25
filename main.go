package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// MiniShell representa el shell asistido por IA
type MiniShell struct {
	running bool
}

// NewMiniShell crea una nueva instancia del shell
func NewMiniShell() *MiniShell {
	return &MiniShell{running: true}
}

// setupSignalHandlers configura los manejadores de señales Unix
func (ms *MiniShell) setupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGINT:
				fmt.Println("^C (usa 'exit' para salir)")
				// No salir, solo volver al prompt
			case syscall.SIGTERM:
				fmt.Println("\nRecibido SIGTERM, cerrando limpiamente...")
				ms.running = false
				os.Exit(0)
			}
		}
	}()
}

// displayPrompt muestra el prompt del shell
func (ms *MiniShell) displayPrompt() string {
	return "neri> "
}

// shouldExit verifica si el usuario quiere salir
func (ms *MiniShell) shouldExit(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "exit" || input == "quit"
}

// checkAPIKey verifica si existe la API key y muestra advertencia si no
func (ms *MiniShell) checkAPIKey() {
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}

	// Solo verificar API key para providers que la necesitan
	if provider == "openai" || provider == "gemini" {
		apiKey := os.Getenv("AI_API_KEY")
		if apiKey == "" {
			fmt.Println("⚠️  ADVERTENCIA: No se encontró AI_API_KEY en las variables de entorno")
			fmt.Printf("   Para usar %s, configura: export AI_API_KEY=tu_clave\n", provider)
			fmt.Println("   El programa continuará pero las llamadas a la API fallarán.")
			fmt.Println()
		}
	}
}

// run ejecuta el loop principal REPL
func (ms *MiniShell) run() {
	fmt.Println("Mini-shell asistido por IA")
	fmt.Println("Escribe 'exit' o 'quit' para salir")
	fmt.Println()

	// Verificar configuración de API
	ms.checkAPIKey()

	ms.setupSignalHandlers()

	reader := bufio.NewReader(os.Stdin)

	for ms.running {
		// Mostrar prompt y leer input
		prompt := ms.displayPrompt()
		fmt.Print(prompt)

		userInput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error leyendo input: %v\n", err)
			continue
		}

		userInput = strings.TrimSpace(userInput)

		// Manejar input vacío
		if userInput == "" {
			continue
		}

		// Manejar comandos de salida
		if ms.shouldExit(userInput) {
			fmt.Println("Saliendo...")
			break
		}

		// Procesar comando a través de IA
		rawResponse, finalCommand, err := TranslateToCommand(userInput)
		if err != nil {
			fmt.Printf("Error procesando comando: %v\n", err)
			fmt.Println()
			continue
		}

		// Mostrar resultados (sin ejecutar)
		if rawResponse != "" {
			fmt.Printf("IA raw: %s\n", rawResponse)
		}
		fmt.Printf("CMD: %s\n", finalCommand)
		fmt.Println()
	}

	fmt.Println("Hasta luego!")
}

func main() {
	shell := NewMiniShell()
	shell.run()
}
