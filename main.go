package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
)

type Config struct {
	ContainerName string
	Port          string
	Title         string
	AuthUser      string
	AuthPass      string
}

var (
	cfg       Config
	dockerCli *client.Client
)

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Vt323&display=swap" rel="stylesheet">
    <style>
        body {
            font-family: 'VT323', monospace;
            background-color: #241810;
            /* SUA IMAGEM DO PINTEREST AQUI */
            background-image: url('https://i.pinimg.com/474x/0e/1f/c2/0e1fc2e0638e878d3ba8db495152164c.jpg');
            background-size: 150px; 
            background-repeat: repeat;
            color: #e0e0e0;
            overflow: hidden;
        }

        /* Efeito de TV de Tubo (Scanlines) */
        .scanlines {
            background: linear-gradient(to bottom, rgba(255,255,255,0), rgba(255,255,255,0) 50%, rgba(0,0,0,0.2) 50%, rgba(0,0,0,0.2));
            background-size: 100% 4px;
            position: fixed;
            top: 0; left: 0; width: 100%; height: 100%;
            pointer-events: none;
            z-index: 10;
        }

        /* Sombra nas bordas (Vignette) */
        .vignette {
            position: fixed;
            top: 0; left: 0; width: 100%; height: 100%;
            background: radial-gradient(circle, rgba(0,0,0,0) 60%, rgba(0,0,0,0.9) 100%);
            pointer-events: none;
            z-index: 5;
        }
        
        /* GUI Container - Caixa cinza estilo menu antigo */
        .mc-gui {
            background-color: rgba(40, 40, 40, 0.8);
            border: 3px solid #555;
            box-shadow: 0 0 20px rgba(0,0,0,0.8);
            backdrop-filter: blur(2px);
        }

        /* Decoração de parafusos nos cantos */
        .screw {
            position: absolute; width: 8px; height: 8px; background-color: #777; border: 1px solid #333;
        }

        /* Botões estilo v2.0 (Pixelados, cinzas, texto amarelo no hover) */
        .mc-btn {
            background-color: #787878;
            border-top: 3px solid #aaa;
            border-left: 3px solid #aaa;
            border-bottom: 3px solid #222;
            border-right: 3px solid #222;
            color: #ddd;
            text-shadow: 2px 2px #222;
            cursor: pointer;
            image-rendering: pixelated;
            white-space: nowrap;
            transition: none;
        }
        .mc-btn:hover {
            background-color: #8b8b8b;
            border-top: 3px solid #bfbfbf;
            border-left: 3px solid #bfbfbf;
            color: #ffffa0; /* Texto amarelo igual no jogo */
        }
        .mc-btn:active {
            background-color: #555;
            border-top: 3px solid #222;
            border-left: 3px solid #222;
            border-bottom: 3px solid #aaa;
            border-right: 3px solid #aaa;
            transform: translateY(2px);
        }

        .mc-title { text-shadow: 3px 3px 0 #222; }
    </style>
</head>
<body class="h-screen flex flex-col items-center justify-center p-4">
    
    <div class="scanlines"></div>
    <div class="vignette"></div>

    <div class="mc-gui p-8 w-full max-w-xl flex flex-col items-center gap-6 relative z-20">
        <div class="screw top-2 left-2"></div>
        <div class="screw top-2 right-2"></div>
        <div class="screw bottom-2 left-2"></div>
        <div class="screw bottom-2 right-2"></div>

        <h1 class="mc-title text-4xl md:text-6xl tracking-wide mb-2 text-center">
            {{ .Title }}
        </h1>

        <div class="text-xl text-gray-400 mb-4 font-mono">
            Target: <span class="text-white tracking-wider">{{ .ContainerName }}</span>
        </div>

        <div hx-get="/status" hx-trigger="load, every 2s" hx-swap="innerHTML" class="w-full flex justify-center min-h-[160px]">
            <div class="text-2xl text-gray-500 animate-pulse mt-10 tracking-widest">
                SCANNING SYSTEM...
            </div>
        </div>
		</div>

</body>
</html>
`

const statusTemplate = `
{{ if .Running }}
    <div class="w-full flex flex-col items-center animate-fade-in w-full">
        <div class="flex items-center gap-3 mb-8 bg-black/20 px-6 py-2 rounded border border-green-900/50">
            <div class="flex gap-1 items-end h-8">
                <div class="w-2 h-3 bg-green-500"></div>
                <div class="w-2 h-5 bg-green-500"></div>
                <div class="w-2 h-7 bg-green-500"></div>
                <div class="w-2 h-9 bg-green-500 animate-pulse"></div>
            </div>
            <span class="text-4xl text-green-400 drop-shadow-md tracking-widest">ONLINE</span>
        </div>
        
        <div class="w-full flex flex-col gap-4">
            <button hx-post="/stop" hx-swap="none" 
                class="mc-btn w-full py-3 text-3xl active:text-red-300">
                STOP SERVER
            </button>
            
            <button hx-post="/restart" hx-swap="none" 
                class="mc-btn w-full py-3 text-3xl text-gray-400">
                RESTART SYSTEM
            </button>
        </div>
    </div>
{{ else }}
    <div class="w-full flex flex-col items-center animate-fade-in w-full">
        <div class="flex items-center gap-3 mb-8 bg-black/20 px-6 py-2 rounded border border-red-900/50">
            <div class="flex gap-1 items-end h-8 opacity-50">
                <div class="w-2 h-3 bg-red-600 animate-pulse"></div>
                <div class="w-2 h-5 bg-gray-800"></div>
                <div class="w-2 h-7 bg-gray-800"></div>
                <div class="w-2 h-9 bg-gray-800"></div>
            </div>
            <span class="text-4xl text-red-500 drop-shadow-md tracking-widest">OFFLINE</span>
        </div>
        
        <button hx-post="/start" hx-swap="none" 
            class="mc-btn w-full py-6 text-4xl text-white animate-pulse">
            INITIALIZE SERVER
        </button>
    </div>
{{ end }}
`

func loadConfig() {
	_ = godotenv.Load()
	cfg = Config{
		ContainerName: getEnv("CONTAINER_NAME", "mc-server"),
		Port:          getEnv("PORT", "8080"),
		Title:         getEnv("APP_TITLE", "MINECRAFT SERVER"),
		AuthUser:      getEnv("AUTH_USER", ""),
		AuthPass:      getEnv("AUTH_PASS", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.AuthUser == "" || cfg.AuthPass == "" {
			next(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != cfg.AuthUser || pass != cfg.AuthPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func main() {
	loadConfig()

	var err error
	dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Critical Docker connection error: %v", err)
	}

	http.HandleFunc("/", basicAuth(handleIndex))
	http.HandleFunc("/status", basicAuth(handleStatus))
	http.HandleFunc("/start", basicAuth(handleStart))
	http.HandleFunc("/stop", basicAuth(handleStop))
	http.HandleFunc("/restart", basicAuth(handleRestart))

	fmt.Printf("System online on port :%s\n", cfg.Port)
	fmt.Printf("Target container: %s\n", cfg.ContainerName)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Template error", 500)
		return
	}
	tmpl.Execute(w, cfg)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	running := isContainerRunning()
	tmpl, _ := template.New("status").Parse(statusTemplate)
	data := struct{ Running bool }{Running: running}
	tmpl.Execute(w, data)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	log.Println("Command: START")
	if err := dockerCli.ContainerStart(context.Background(), cfg.ContainerName, container.StartOptions{}); err != nil {
		log.Printf("Start error: %v", err)
		http.Error(w, err.Error(), 500)
	}
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	log.Println("Command: STOP")
	if err := dockerCli.ContainerStop(context.Background(), cfg.ContainerName, container.StopOptions{}); err != nil {
		log.Printf("Stop error: %v", err)
		http.Error(w, err.Error(), 500)
	}
}

func handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	log.Println("Command: RESTART")
	if err := dockerCli.ContainerRestart(context.Background(), cfg.ContainerName, container.StopOptions{}); err != nil {
		log.Printf("Restart error: %v", err)
		http.Error(w, err.Error(), 500)
	}
}

func isContainerRunning() bool {
	inspect, err := dockerCli.ContainerInspect(context.Background(), cfg.ContainerName)
	if err != nil {
		return false
	}
	return inspect.State.Running
}
