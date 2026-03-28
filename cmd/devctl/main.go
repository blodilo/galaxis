// galaxis-devctl — Prozessmanager für den Galaxis-Dev-Stack.
// Läuft auf :9191, muss aus dem Projekt-Root gestartet werden.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const listenAddr = ":9191"

// ── Log-Puffer ────────────────────────────────────────────────────────────────

type logBuf struct {
	mu   sync.RWMutex
	data []string
	subs []chan string
}

func (b *logBuf) add(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data = append(b.data, line)
	if len(b.data) > 300 {
		b.data = b.data[len(b.data)-300:]
	}
	for _, ch := range b.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (b *logBuf) snapshot() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, len(b.data))
	copy(out, b.data)
	return out
}

func (b *logBuf) subscribe() chan string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan string, 100)
	b.subs = append(b.subs, ch)
	return ch
}

func (b *logBuf) unsubscribe(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, s := range b.subs {
		if s == ch {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
			close(ch)
			return
		}
	}
}

// ── Komponente ────────────────────────────────────────────────────────────────

type compStatus string

const (
	stStopped  compStatus = "stopped"
	stStarting compStatus = "starting"
	stRunning  compStatus = "running"
	stError    compStatus = "error"
)

type component struct {
	id      string
	display string
	port    int

	mu        sync.Mutex
	st        compStatus
	cmd       *exec.Cmd
	startedAt time.Time
	errMsg    string
	buf       logBuf

	fnStart  func(*component) error // startet Prozess, setzt c.cmd
	fnStop   func(*component)       // stoppt Prozess; nil = SIGTERM
	fnHealth func() bool            // gibt true zurück wenn bereit
}

func (c *component) info() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	pid := 0
	if c.cmd != nil && c.cmd.Process != nil {
		pid = c.cmd.Process.Pid
	}
	uptime := ""
	if c.st == stRunning && !c.startedAt.IsZero() {
		uptime = time.Since(c.startedAt).Round(time.Second).String()
	}
	return map[string]any{
		"id": c.id, "display": c.display,
		"status": string(c.st), "port": c.port,
		"pid": pid, "uptime": uptime, "error": c.errMsg,
	}
}

func (c *component) start() {
	c.mu.Lock()
	if c.st == stRunning || c.st == stStarting {
		c.mu.Unlock()
		return
	}
	c.st = stStarting
	c.errMsg = ""
	c.mu.Unlock()

	c.buf.add("[devctl] Starting " + c.display + " …")

	go func() {
		if err := c.fnStart(c); err != nil {
			c.mu.Lock()
			c.st = stError
			c.errMsg = err.Error()
			c.mu.Unlock()
			c.buf.add("[devctl] Error: " + err.Error())
			return
		}
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if c.fnHealth() {
				c.mu.Lock()
				c.st = stRunning
				c.startedAt = time.Now()
				c.mu.Unlock()
				c.buf.add("[devctl] " + c.display + " bereit ✓")
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		c.mu.Lock()
		if c.st == stStarting {
			c.st = stError
			c.errMsg = "health check timeout"
		}
		c.mu.Unlock()
		c.buf.add("[devctl] Health-Check Timeout")
	}()
}

func (c *component) stop() {
	c.mu.Lock()
	if c.st == stStopped {
		c.mu.Unlock()
		return
	}
	cmd := c.cmd
	c.mu.Unlock()

	c.buf.add("[devctl] Stopping " + c.display + " …")
	if c.fnStop != nil {
		c.fnStop(c)
	} else if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			_ = cmd.Process.Kill()
		}
	}
	c.mu.Lock()
	c.st = stStopped
	c.cmd = nil
	c.mu.Unlock()
	c.buf.add("[devctl] " + c.display + " gestoppt")
}

func (c *component) pipe(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		c.buf.add(scanner.Text())
	}
}

func (c *component) watch(cmd *exec.Cmd) {
	_ = cmd.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == cmd {
		c.st = stStopped
		c.cmd = nil
		c.buf.add("[devctl] " + c.display + " beendet")
	}
}

// ── Hilfsfunktionen ───────────────────────────────────────────────────────────

func tcpAlive(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func httpAlive(url string) bool {
	cl := &http.Client{Timeout: time.Second}
	resp, err := cl.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}

func dbPassword() string {
	out, err := exec.Command("secret-tool", "lookup", "service", "galaxis-local", "account", "postgres").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "galaxis_dev"
	}
	return strings.TrimSpace(string(out))
}

func runShell(c *component, args ...string) error {
	return runShellEnv(c, nil, args...)
}

func runShellEnv(c *component, env []string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	if len(env) > 0 {
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			c.buf.add(line)
		}
	}
	return err
}

// ── Komponenten-Definitionen ──────────────────────────────────────────────────

func makePostgres() *component {
	c := &component{id: "postgres", display: "PostgreSQL", port: 5432}
	c.fnHealth = func() bool { return tcpAlive(5432) }
	c.fnStart = func(c *component) error {
		if tcpAlive(5432) {
			c.buf.add("[devctl] PostgreSQL läuft bereits (externer Container)")
			return nil
		}
		return runShell(c, "docker", "compose", "up", "-d", "postgres")
	}
	c.fnStop = func(c *component) {
		_ = runShell(c, "docker", "compose", "stop", "postgres")
	}
	if tcpAlive(5432) {
		c.st = stRunning
		c.startedAt = time.Now()
	} else {
		c.st = stStopped
	}
	return c
}

func makeNATS() *component {
	c := &component{id: "nats", display: "NATS", port: 4222}
	c.fnHealth = func() bool { return tcpAlive(4222) }
	c.fnStart = func(c *component) error {
		if tcpAlive(4222) {
			c.buf.add("[devctl] NATS läuft bereits")
			return nil
		}
		return runShell(c, "docker", "compose", "up", "-d", "nats")
	}
	c.fnStop = func(c *component) {
		_ = runShell(c, "docker", "compose", "stop", "nats")
	}
	if tcpAlive(4222) {
		c.st = stRunning
		c.startedAt = time.Now()
	} else {
		c.st = stStopped
	}
	return c
}

func makeGalaxisAPI() *component {
	c := &component{id: "galaxis-api", display: "Galaxis API", port: 8080}
	c.fnHealth = func() bool { return httpAlive("http://localhost:8080/health") }
	c.fnStart = func(c *component) error {
		c.buf.add("[devctl] Build …")
		if err := runShell(c, "go", "build", "-o", "bin/galaxis-api", "./cmd/server"); err != nil {
			return fmt.Errorf("build: %w", err)
		}
		c.buf.add("[devctl] Build ok")
		dbURL := "postgres://galaxis:" + dbPassword() + "@localhost:5432/galaxis?sslmode=disable"
		c.buf.add("[devctl] Migrate …")
		dbEnv := append(os.Environ(), "DATABASE_URL="+dbURL)
		if err := runShellEnv(c, dbEnv, "./bin/galaxis-api", "--migrate-only"); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		c.buf.add("[devctl] Migration ok")
		cmd := exec.Command("./bin/galaxis-api",
			"--config", "game-params_v1.8.yaml",
			"--nats", "nats://localhost:4222",
		)
		cmd.Env = append(os.Environ(), "DATABASE_URL="+dbURL)
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			return err
		}
		c.mu.Lock()
		c.cmd = cmd
		c.mu.Unlock()
		go c.pipe(stdout)
		go c.pipe(stderr)
		go c.watch(cmd)
		return nil
	}
	if httpAlive("http://localhost:8080/health") {
		c.st = stRunning
		c.startedAt = time.Now()
	} else {
		c.st = stStopped
	}
	return c
}

func makeFrontend() *component {
	c := &component{id: "galaxis-frontend", display: "Galaxis Frontend", port: 5175}
	c.fnHealth = func() bool { return tcpAlive(5175) }
	c.fnStart = func(c *component) error {
		cmd := exec.Command("npm", "run", "dev", "--prefix", "frontend")
		cmd.Env = os.Environ()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			return err
		}
		c.mu.Lock()
		c.cmd = cmd
		c.mu.Unlock()
		go c.pipe(stdout)
		go c.pipe(stderr)
		go c.watch(cmd)
		return nil
	}
	if tcpAlive(5175) {
		c.st = stRunning
		c.startedAt = time.Now()
	} else {
		c.st = stStopped
	}
	return c
}

// ── HTTP-Server ───────────────────────────────────────────────────────────────

type manager struct {
	comps  []*component
	byID   map[string]*component
}

func newManager() *manager {
	list := []*component{makePostgres(), makeNATS(), makeGalaxisAPI(), makeFrontend()}
	m := &manager{comps: list, byID: map[string]*component{}}
	for _, c := range list {
		m.byID[c.id] = c
	}
	return m
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (m *manager) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, uiHTML)
	})

	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) {
		out := make([]map[string]any, 0, len(m.comps))
		for _, c := range m.comps {
			out = append(out, c.info())
		}
		writeJSON(w, map[string]any{"components": out})
	})

	mux.HandleFunc("POST /api/start/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, ok := m.byID[r.PathValue("id")]
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		c.start()
		writeJSON(w, map[string]string{"ok": "1"})
	})

	mux.HandleFunc("POST /api/stop/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, ok := m.byID[r.PathValue("id")]
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		go c.stop()
		writeJSON(w, map[string]string{"ok": "1"})
	})

	mux.HandleFunc("POST /api/restart/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, ok := m.byID[r.PathValue("id")]
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		go func() { c.stop(); c.start() }()
		writeJSON(w, map[string]string{"ok": "1"})
	})

	// SSE: Log-Stream
	mux.HandleFunc("GET /api/logs/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, ok := m.byID[r.PathValue("id")]
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, canFlush := w.(http.Flusher)

		sendLine := func(line string) {
			b, _ := json.Marshal(line)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if canFlush {
				flusher.Flush()
			}
		}

		for _, line := range c.buf.snapshot() {
			sendLine(line)
		}

		ch := c.buf.subscribe()
		defer c.buf.unsubscribe(ch)

		for {
			select {
			case line, ok := <-ch:
				if !ok {
					return
				}
				sendLine(line)
			case <-r.Context().Done():
				return
			}
		}
	})
}

func main() {
	if _, err := os.Stat("go.mod"); err != nil {
		log.Fatal("galaxis-devctl muss aus dem Projekt-Root gestartet werden")
	}
	_ = os.MkdirAll("bin", 0755)

	m := newManager()
	mux := http.NewServeMux()
	m.register(mux)

	log.Printf("galaxis-devctl → http://localhost%s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
