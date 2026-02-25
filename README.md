# Mini-Shell Asistido por IA

Un shell interactivo que utiliza IA para traducir lenguaje natural a comandos de Unix/Linux.

## Requisitos

- Go 1.21 o superior
- Acceso a API de IA (OpenAI, Gemini, o Ollama local)

## Configuración

Variables de entorno opcionales:

```bash
# Proveedor de IA (openai, gemini, ollama)
export AI_PROVIDER=ollama

# URL base de la API
export AI_BASE_URL=http://localhost:11434/api/generate

# API key (solo OpenAI/Gemini)
export AI_API_KEY=tu-api-key

# Modelo a usar
export AI_MODEL=llama2
```

## Instalación y Ejecución

1. Clonar o descargar los archivos
2. Configurar variables de entorno (opcional)
3. Ejecutar:

```bash
go run .
```

## Uso

```
neri> listar archivos en el directorio actual
IA raw: ls -la
CMD: ls -la

neri> buscar texto "hola" en todos los archivos
IA raw: grep -r "hola" .
CMD: grep -r "hola" .

neri> exit
Saliendo...
```

## Comandos Soportados

- `exit` o `quit`: Salir del programa
- `Ctrl+C`: Interrumpir sin salir
- Cualquier texto en lenguaje natural será traducido a comandos Unix/Linux

## Proveedores Soportados

### OpenAI
```bash
export AI_PROVIDER=openai
export AI_API_KEY=sk-...
export AI_MODEL=gpt-3.5-turbo
```

### Gemini
```bash
export AI_PROVIDER=gemini
export AI_API_KEY=...
export AI_MODEL=gemini-pro
```

### Ollama (local)
```bash
export AI_PROVIDER=ollama
export AI_BASE_URL=http://localhost:11434/api/generate
export AI_MODEL=llama2
```

## Pruebas de Sanitización

Para probar la sanitización de comandos:

```bash
go run parser_engine.go
```

Esto ejecutará 5 casos de prueba para verificar la extracción correcta de comandos de las respuestas de IA.
