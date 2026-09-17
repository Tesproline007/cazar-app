package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	DefaultUser = "Cazarsense1234"
	DefaultPass = "Cazarsense1234"
	WebPort     = ":2053"
	DataFolder  = "/app/data"
	ConfigFile  = "/app/data/settings.json"
)

type Settings struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
}

var (
	sessionTokens = make(map[string]time.Time)
	sessionMutex  sync.RWMutex
)

func generateRandomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func startXray() {
	cmd := exec.Command("/usr/local/bin/xray", "run", "-config", "/app/config.json")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("Xray launch error: %v\n", err)
	}
}

func getSettings() Settings {
	s := Settings{
		Username: DefaultUser,
		Password: DefaultPass,
	}
	b, err := os.ReadFile(ConfigFile)
	if err == nil {
		_ = json.Unmarshal(b, &s)
		if s.Username == "" {
			s.Username = DefaultUser
		}
		if s.Password == "" {
			s.Password = DefaultPass
		}
	}
	return s
}

func saveSettings(s Settings) {
	_ = os.MkdirAll(DataFolder, 0755)
	b, _ := json.Marshal(s)
	_ = os.WriteFile(ConfigFile, b, 0644)
}

func checkAuth(r *http.Request) bool {
	cookie, err := r.Cookie("cazar_session")
	if err != nil || cookie.Value == "" {
		return false
	}
	sessionMutex.RLock()
	expiry, exists := sessionTokens[cookie.Value]
	sessionMutex.RUnlock()
	return exists && time.Now().Before(expiry)
}

func main() {
	startXray()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		current := getSettings()

		if r.Method == http.MethodPost {
			action := r.FormValue("action")
			if action == "save_connection" {
				current.Host = strings.TrimSpace(r.FormValue("host"))
				current.Port = strings.TrimSpace(r.FormValue("port"))
				saveSettings(current)
			} else if action == "save_security" {
				newU := strings.TrimSpace(r.FormValue("new_username"))
				newP := strings.TrimSpace(r.FormValue("new_password"))
				if newU != "" && newP != "" {
					current.Username = newU
					current.Password = newP
					saveSettings(current)
				}
			}
		}

		html := strings.ReplaceAll(pageHTML, "{{HOST}}", current.Host)
		html = strings.ReplaceAll(html, "{{PORT}}", current.Port)
		html = strings.ReplaceAll(html, "{{USERNAME}}", current.Username)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			creds := getSettings()
			user := r.FormValue("username")
			pass := r.FormValue("password")

			if user == creds.Username && pass == creds.Password {
				token := generateRandomToken()
				sessionMutex.Lock()
				sessionTokens[token] = time.Now().Add(30 * 24 * time.Hour)
				sessionMutex.Unlock()

				http.SetCookie(w, &http.Cookie{
					Name:     "cazar_session",
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					MaxAge:   86400 * 30,
				})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, loginHTML)
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("cazar_session")
		if err == nil && cookie.Value != "" {
			sessionMutex.Lock()
			delete(sessionTokens, cookie.Value)
			sessionMutex.Unlock()
		}
		http.SetCookie(w, &http.Cookie{Name: "cazar_session", Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	fmt.Printf("Server listening on %s\n", WebPort)
	_ = http.ListenAndServe(WebPort, nil)
}

const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Login - Cazar Sense</title>
<style>
  body { background: #070a12; color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
  .card { background: #0f172a; padding: 2.2rem; border-radius: 1rem; width: 100%; max-width: 360px; box-shadow: 0 20px 40px rgba(0,0,0,0.6); border: 1px solid #1e293b; }
  h2 { text-align: center; margin-bottom: 1.5rem; color: #38bdf8; font-size: 1.3rem; letter-spacing: -0.025em; }
  label { display: block; font-size: 0.8rem; margin-bottom: 0.35rem; color: #94a3b8; }
  input { width: 100%; padding: 0.75rem; margin-bottom: 1.2rem; border-radius: 0.5rem; border: 1px solid #334155; background: #070a12; color: white; box-sizing: border-box; font-size: 0.9rem; }
  input:focus { border-color: #38bdf8; outline: none; }
  button { width: 100%; padding: 0.8rem; border-radius: 0.5rem; border: none; background: #38bdf8; color: #070a12; font-weight: 600; cursor: pointer; font-size: 0.95rem; }
  button:hover { background: #0284c7; }
</style>
</head>
<body>
<div class="card">
  <h2>Cazar Sense Panel</h2>
  <form method="POST">
    <label>Username</label>
    <input type="text" name="username" required autofocus>
    <label>Password</label>
    <input type="password" name="password" required>
    <button type="submit">Sign In</button>
  </form>
</div>
</body>
</html>`

const pageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cazar Sense Dashboard</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js"></script>
<style>
  body { background: #060913; color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 1.5rem 1rem; display: flex; justify-content: center; }
  .box { width: 100%; max-width: 480px; background: #0f172a; padding: 1.8rem; border-radius: 1.2rem; box-shadow: 0 25px 50px rgba(0,0,0,0.7); border: 1px solid #1e293b; }
  .top-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.2rem; }
  .top-bar h1 { font-size: 1.25rem; color: #38bdf8; margin: 0; font-weight: 700; }
  .exit { color: #f87171; text-decoration: none; font-size: 0.8rem; border: 1px solid #ef4444; padding: 0.25rem 0.65rem; border-radius: 0.4rem; }
  
  .mode-selector { display: flex; background: #060913; border-radius: 0.75rem; padding: 0.35rem; margin-bottom: 1.2rem; border: 1px solid #1e293b; }
  .m-btn { flex: 1; text-align: center; padding: 0.7rem 0.5rem; font-size: 0.85rem; font-weight: 600; border-radius: 0.5rem; cursor: pointer; color: #94a3b8; transition: 0.2s; }
  .m-btn.active { background: #38bdf8; color: #060913; }
  
  .grid-loc { display: grid; grid-template-columns: 1fr; gap: 0.55rem; margin-bottom: 1.2rem; }
  .loc-card { background: #060913; border: 1px solid #1e293b; border-radius: 0.65rem; padding: 0.75rem 0.9rem; cursor: pointer; display: flex; justify-content: space-between; align-items: center; transition: 0.2s; }
  .loc-card:hover { border-color: #38bdf8; }
  .loc-card.active { border-color: #38bdf8; background: #0c192e; box-shadow: 0 0 0 1px #38bdf8; }
  .loc-title { font-size: 0.85rem; font-weight: 600; color: #f1f5f9; }

  label { display: block; font-size: 0.8rem; margin-bottom: 0.35rem; color: #94a3b8; }
  input { width: 100%; padding: 0.65rem; margin-bottom: 0.8rem; border-radius: 0.5rem; border: 1px solid #334155; background: #060913; color: white; box-sizing: border-box; font-family: monospace; font-size: 0.9rem; }
  input:focus { border-color: #38bdf8; outline: none; }
  
  .btn-action { width: 100%; padding: 0.65rem; border-radius: 0.5rem; border: none; background: #0284c7; color: white; font-weight: 600; cursor: pointer; font-size: 0.85rem; margin-bottom: 1.2rem; }
  .btn-action:hover { background: #0369a1; }
  
  .qr-frame { background: white; padding: 0.8rem; border-radius: 0.65rem; width: fit-content; margin: 0 auto 1rem auto; display: flex; justify-content: center; }
  .btn-copy { width: 100%; padding: 0.8rem; border-radius: 0.5rem; border: none; background: #10b981; color: #060913; font-weight: 700; font-size: 0.95rem; cursor: pointer; margin-bottom: 1.5rem; }
  .btn-copy:hover { background: #059669; }

  .divider { border-top: 1px solid #1e293b; margin: 1.5rem 0 1.2rem 0; }
  .section-title { font-size: 0.9rem; font-weight: 600; color: #cbd5e1; margin-bottom: 0.8rem; }
</style>
</head>
<body>
<div class="box">
  <div class="top-bar">
    <h1>Cazar Sense</h1>
    <a href="/logout" class="exit">Sign Out</a>
  </div>

  <div class="mode-selector">
    <div class="m-btn active" id="btnStd" onclick="setMode('standard')">Standard Mode</div>
    <div class="m-btn" id="btnAI" onclick="setMode('ai')">AI Mode</div>
  </div>

  <label>Select Location</label>
  <div class="grid-loc">
    <div class="loc-card active" id="loc0" onclick="selectLocation(0)">
      <span class="loc-title">🇳🇱 EU West (Amsterdam, Netherlands)</span>
    </div>
    <div class="loc-card" id="loc1" onclick="selectLocation(1)">
      <span class="loc-title">🇸🇬 Southeast Asia (Singapore, Singapore)</span>
    </div>
    <div class="loc-card" id="loc2" onclick="selectLocation(2)">
      <span class="loc-title">🇺🇸 US East (Virginia, USA)</span>
    </div>
    <div class="loc-card" id="loc3" onclick="selectLocation(3)">
      <span class="loc-title">🇺🇸 US West (California, USA)</span>
    </div>
  </div>

  <form method="POST">
    <input type="hidden" name="action" value="save_connection">
    <label>Railway TCP Host</label>
    <input type="text" name="host" value="{{HOST}}" placeholder="e.g. junction.proxy.rlwy.net" required>
    <label>Railway TCP Port</label>
    <input type="number" name="port" value="{{PORT}}" placeholder="e.g. 38268" required>
    <button type="submit" class="btn-action">Save Network Settings</button>
  </form>

  <div class="qr-frame" id="qrcode"></div>
  <button class="btn-copy" onclick="copyConfig()">Copy Config URL</button>

  <div class="divider"></div>
  <div class="section-title">Account Security</div>
  <form method="POST">
    <input type="hidden" name="action" value="save_security">
    <label>Current Username: {{USERNAME}}</label>
    <input type="text" name="new_username" placeholder="New Username" required>
    <input type="password" name="new_password" placeholder="New Password" required>
    <button type="submit" class="btn-action">Update Credentials</button>
  </form>
</div>

<script>
  const locations = [
    "🇳🇱 EU West (Amsterdam, Netherlands)",
    "🇸🇬 Southeast Asia (Singapore, Singapore)",
    "🇺🇸 US East (Virginia, USA)",
    "🇺🇸 US West (California, USA)"
  ];

  let currentMode = 'standard';
  let selectedIdx = 0;
  const idNormal = "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d";
  const idAI     = "f9e8d7c6-b5a4-3210-fedc-ba9876543210";
  const h = "{{HOST}}";
  const p = "{{PORT}}";
  let vlessURL = "";

  function setMode(mode) {
    currentMode = mode;
    document.getElementById('btnStd').classList.toggle('active', mode === 'standard');
    document.getElementById('btnAI').classList.toggle('active', mode === 'ai');
    render();
  }

  function selectLocation(idx) {
    selectedIdx = idx;
    for (let i = 0; i < locations.length; i++) {
      document.getElementById('loc' + i).classList.toggle('active', i === idx);
    }
    render();
  }

  function render() {
    if (!h || !p) {
      document.getElementById('qrcode').innerHTML = '<span style="color:#000;font-size:12px">Configure host and port first</span>';
      return;
    }
    
    let baseName = locations[selectedIdx];
    let finalName = (currentMode === 'ai') ? (baseName + " - AI") : baseName;
    let uid = (currentMode === 'ai') ? idAI : idNormal;

    vlessURL = 'vless://' + uid + '@' + h + ':' + p + '?path=%2F&security=none&encryption=none&type=ws#' + encodeURIComponent(finalName);
    
    document.getElementById('qrcode').innerHTML = '';
    new QRCode(document.getElementById('qrcode'), { text: vlessURL, width: 180, height: 180 });
  }

  function copyConfig() {
    if(!vlessURL) return alert('Configure host and port first.');
    navigator.clipboard.writeText(vlessURL);
    alert('Config copied to clipboard.');
  }

  render();
</script>
</body>
</html>`
