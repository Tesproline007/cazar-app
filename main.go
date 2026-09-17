package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	AdminUser  = "Cazarsense1234"
	AdminPass  = "Cazarsense1234"
	WebPort    = ":2053"
	DataFolder = "/app/data"
	ConfigFile = "/app/data/settings.json"
)

type Settings struct {
	Host string `json:"host"`
	Port string `json:"port"`
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
	var s Settings
	b, err := os.ReadFile(ConfigFile)
	if err == nil {
		_ = json.Unmarshal(b, &s)
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
	return err == nil && cookie.Value == "authenticated"
}

func main() {
	startXray()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodPost {
			s := Settings{
				Host: strings.TrimSpace(r.FormValue("host")),
				Port: strings.TrimSpace(r.FormValue("port")),
			}
			saveSettings(s)
		}

		current := getSettings()
		html := strings.ReplaceAll(pageHTML, "{{HOST}}", current.Host)
		html = strings.ReplaceAll(html, "{{PORT}}", current.Port)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.FormValue("username") == AdminUser && r.FormValue("password") == AdminPass {
				http.SetCookie(w, &http.Cookie{
					Name:     "cazar_session",
					Value:    "authenticated",
					Path:     "/",
					HttpOnly: true,
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
		http.SetCookie(w, &http.Cookie{Name: "cazar_session", Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	fmt.Printf("Server listening on %s\n", WebPort)
	_ = http.ListenAndServe(WebPort, nil)
}

const loginHTML = `<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ورود | Cazar Sense</title>
<style>
  body { background: #080d1a; color: #f1f5f9; font-family: system-ui, sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
  .card { background: #131d31; padding: 2.2rem; border-radius: 1rem; width: 100%; max-width: 360px; box-shadow: 0 10px 30px rgba(0,0,0,0.6); border: 1px solid #1e293b; }
  h2 { text-align: center; margin-bottom: 1.5rem; color: #38bdf8; }
  input { width: 100%; padding: 0.8rem; margin-bottom: 1.2rem; border-radius: 0.5rem; border: 1px solid #334155; background: #080d1a; color: white; box-sizing: border-box; }
  button { width: 100%; padding: 0.8rem; border-radius: 0.5rem; border: none; background: #38bdf8; color: #080d1a; font-weight: bold; cursor: pointer; font-size: 1rem; }
  button:hover { background: #0284c7; }
</style>
</head>
<body>
<div class="card">
  <h2>ورود به پنل</h2>
  <form method="POST">
    <input type="text" name="username" placeholder="نام کاربری" required autofocus>
    <input type="password" name="password" placeholder="کلمه عبور" required>
    <button type="submit">ورود</button>
  </form>
</div>
</body>
</html>`

const pageHTML = `<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cazar Sense Dashboard</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js"></script>
<style>
  body { background: #060913; color: #f8fafc; font-family: system-ui, -apple-system, sans-serif; margin: 0; padding: 1.2rem; display: flex; justify-content: center; }
  .box { width: 100%; max-width: 480px; background: #0f172a; padding: 1.8rem; border-radius: 1.2rem; box-shadow: 0 20px 40px rgba(0,0,0,0.7); border: 1px solid #1e293b; }
  .top-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.2rem; }
  .top-bar h1 { font-size: 1.25rem; color: #38bdf8; margin: 0; }
  .exit { color: #f87171; text-decoration: none; font-size: 0.85rem; border: 1px solid #ef4444; padding: 0.2rem 0.6rem; border-radius: 0.4rem; }
  
  .grid-loc { display: grid; grid-template-columns: 1fr; gap: 0.55rem; margin-bottom: 1.2rem; }
  .loc-card { background: #060913; border: 1px solid #1e293b; border-radius: 0.65rem; padding: 0.65rem 0.8rem; cursor: pointer; display: flex; justify-content: space-between; align-items: center; transition: 0.2s; }
  .loc-card:hover { border-color: #38bdf8; }
  .loc-card.active { border-color: #38bdf8; background: #0c192e; box-shadow: 0 0 0 1px #38bdf8; }
  .loc-title { font-size: 0.85rem; font-weight: bold; color: #f1f5f9; display: flex; align-items: center; gap: 0.4rem; }
  .loc-badge { font-size: 0.7rem; font-weight: bold; padding: 0.15rem 0.45rem; border-radius: 0.35rem; }
  .badge-normal { background: #064e3b; color: #6ee7b7; }
  .badge-ai { background: #4c1d95; color: #c4b5fd; }

  label { display: block; font-size: 0.8rem; margin-bottom: 0.35rem; color: #94a3af; }
  input { width: 100%; padding: 0.65rem; margin-bottom: 0.8rem; border-radius: 0.5rem; border: 1px solid #334155; background: #060913; color: white; box-sizing: border-box; font-family: monospace; font-size: 0.9rem; }
  .btn-save { width: 100%; padding: 0.65rem; border-radius: 0.5rem; border: none; background: #0284c7; color: white; font-weight: bold; cursor: pointer; margin-bottom: 1rem; }
  .btn-save:hover { background: #0369a1; }
  .qr-frame { background: white; padding: 0.8rem; border-radius: 0.65rem; width: fit-content; margin: 0 auto 1rem auto; display: flex; justify-content: center; }
  .btn-copy { width: 100%; padding: 0.8rem; border-radius: 0.5rem; border: none; background: #10b981; color: #060913; font-weight: bold; font-size: 0.95rem; cursor: pointer; }
  .btn-copy:hover { background: #059669; }
</style>
</head>
<body>
<div class="box">
  <div class="top-bar">
    <h1>Cazar Sense Panel</h1>
    <a href="/logout" class="exit">خروج</a>
  </div>

  <label>انتخاب موقعیت سرور و نوع کانفیگ:</label>
  <div class="grid-loc">
    <div class="loc-card active" id="loc0" onclick="selectLocation(0)">
      <span class="loc-title">🇳🇱 EU West (Amsterdam, Netherlands)</span>
      <span class="loc-badge badge-normal">🚀 دانلود و وب</span>
    </div>
    <div class="loc-card" id="loc1" onclick="selectLocation(1)">
      <span class="loc-title">🇸🇬 Southeast Asia (Singapore, Singapore)</span>
      <span class="loc-badge badge-normal">🚀 دانلود و وب</span>
    </div>
    <div class="loc-card" id="loc2" onclick="selectLocation(2)">
      <span class="loc-title">🇺🇸 US East (Virginia, USA)</span>
      <span class="loc-badge badge-ai">🤖 هوش مصنوعی</span>
    </div>
    <div class="loc-card" id="loc3" onclick="selectLocation(3)">
      <span class="loc-title">🇺🇸 US West (California, USA)</span>
      <span class="loc-badge badge-ai">🤖 هوش مصنوعی</span>
    </div>
  </div>

  <form method="POST">
    <label>دامنه TCP ریل‌وی (Host):</label>
    <input type="text" name="host" value="{{HOST}}" placeholder="مثال: junction.proxy.rlwy.net" required>
    <label>پورت اختصاص‌یافته ریل‌وی (Port):</label>
    <input type="number" name="port" value="{{PORT}}" placeholder="مثال: 38268" required>
    <button type="submit" class="btn-save">💾 ذخیره در مخزن دائمی</button>
  </form>

  <div class="qr-frame" id="qrcode"></div>
  <button class="btn-copy" onclick="copyConfig()">📋 کپی لینک کانفیگ</button>
</div>

<script>
  const locations = [
    { name: "🇳🇱 EU West (Amsterdam, Netherlands)", mode: "normal" },
    { name: "🇸🇬 Southeast Asia (Singapore, Singapore)", mode: "normal" },
    { name: "🇺🇸 US East (Virginia, USA)", mode: "ai" },
    { name: "🇺🇸 US West (California, USA)", mode: "ai" }
  ];

  let selectedIdx = 0;
  const idNormal = "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d";
  const idAI     = "f9e8d7c6-b5a4-3210-fedc-ba9876543210";
  const h = "{{HOST}}";
  const p = "{{PORT}}";
  let vlessURL = "";

  function selectLocation(idx) {
    selectedIdx = idx;
    for (let i = 0; i < locations.length; i++) {
      document.getElementById('loc' + i).classList.toggle('active', i === idx);
    }
    makeConfig();
  }

  function makeConfig() {
    if (!h || !p) {
      document.getElementById('qrcode').innerHTML = '<span style="color:#000;font-size:12px">هاست و پورت را ذخیره کنید</span>';
      return;
    }
    const loc = locations[selectedIdx];
    const uid = (loc.mode === 'normal') ? idNormal : idAI;
    vlessURL = 'vless://' + uid + '@' + h + ':' + p + '?path=%2F&security=none&encryption=none&type=ws#' + encodeURIComponent(loc.name);
    
    document.getElementById('qrcode').innerHTML = '';
    new QRCode(document.getElementById('qrcode'), { text: vlessURL, width: 180, height: 180 });
  }

  function copyConfig() {
    if(!vlessURL) return alert('ابتدا هاست و پورت را در کادرها ذخیره کنید.');
    navigator.clipboard.writeText(vlessURL);
    alert('لینک کانفیگ با نام لوکیشن کپی شد!');
  }

  makeConfig();
</script>
</body>
</html>`
