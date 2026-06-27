# ClaudiaRPC 🌸

**ClaudiaRPC** adalah aplikasi Rich Presence (RPC) kustom untuk Discord yang dibangun menggunakan Go (Golang) sebagai backend dan antarmuka web modern bertema *Sakura Midnight*. Kontrol status Discord lo langsung dari browser — tanpa install apapun selain Go.

## ✨ Fitur

- 🎮 **Activity Type** — Playing, Streaming, Listening, Watching, Competing
- 🖼️ **Image Upload** — Paste URL gambar, klik Upload, otomatis ter-host di Discord CDN
- 📂 **Profil Persistent** — Simpan, load, export, dan import profil sebagai JSON
- ⚡ **Quick Switch** — Pilih profil dari dropdown, langsung ter-load tanpa klik Load
- 🔄 **Auto-Reconnect** — Koneksi ke Discord Gateway otomatis reconnect kalau putus
- ⏱️ **Timestamp** — Start dan End timestamp support
- 🔘 **Button** — Dua button interaktif dengan label dan URL custom
- 📊 **RPC Status Card** — Monitor status koneksi, uptime, dan activity secara real-time
- ⚠️ **Validasi** — Warning sebelum Start kalau ada field yang kurang, tapi tetap bisa jalan

## 🛠️ Teknologi

- **Backend**: [Go](https://golang.org/) + [Chi Router](https://github.com/go-chi/chi)
- **Frontend**: HTML5, CSS3, Vanilla JS
- **Protokol**: Discord Gateway WebSocket (v10) + REST API

## 📦 Instalasi

### Prasyarat
- [Go 1.20+](https://golang.org/dl/)
- Akun Discord dan User Token

### Langkah-langkah

1. **Clone repo**
   ```bash
   git clone https://github.com/Zaineedyou/ClaudiaRPC.git
   cd ClaudiaRPC
   ```

2. **Install dependensi**
   ```bash
   go mod tidy
   ```

3. **Jalankan server**
   ```bash
   go run cmd/server/main.go
   ```
   Atau build binary dulu:
   ```bash
   go build -o ClaudiaRPC cmd/server/main.go
   ./ClaudiaRPC
   ```

4. **Buka browser**
   ```
   http://localhost:8080
   ```

## 🚀 Cara Penggunaan

1. Dapatkan **Discord User Token** lo (lihat bagian peringatan di bawah)
2. Masukkan token ke field **User Token**
3. Isi **Application Name** dan **Application ID** dari [Discord Developer Portal](https://discord.com/developers/applications)
4. Set **Activity Type** sesuai yang lo mau
5. Untuk gambar — paste URL gambar di field **Large/Small Image URL**, klik **Upload**, URL otomatis dikonversi ke format Discord CDN
6. Opsional: isi Timestamp, Button, dan State
7. Klik **Start RPC**

### Manajemen Profil
- **Save** — simpan konfigurasi sekarang sebagai profil
- **Quick Switch** — pilih profil dari dropdown, otomatis ter-load
- **↑ Export** — download semua profil sebagai file JSON
- **↓ Import** — import profil dari file JSON

## ⚠️ Peringatan

> Aplikasi ini menggunakan **User Token Discord**. Penggunaan user token secara teknis melanggar *Terms of Service* Discord. Gunakan dengan bijak dan tanggung risiko sendiri.
>
> **Jangan pernah share token atau file `profiles.json` lo ke siapapun.**

## 📄 Lisensi

[MIT License](LICENSE)

---
Dibuat dengan 🩷 oleh [Zaineedyou](https://github.com/Zaineedyou)
