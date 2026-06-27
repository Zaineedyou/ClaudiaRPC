document.addEventListener('DOMContentLoaded', () => {
    const statusMsg = document.getElementById('status-message')
    const inputs = document.querySelectorAll('#rpc-form input, #rpc-form select')
    const profileSelect = document.getElementById('profile-select')

    const pHeader    = document.getElementById('preview-header')
    const pAppName   = document.getElementById('preview-app-name')
    const pDetails   = document.getElementById('preview-details')
    const pState     = document.getElementById('preview-state')
    const pLargeImg  = document.getElementById('preview-large-img')
    const pSmallImg  = document.getElementById('preview-small-img')
    const pTimestamp = document.getElementById('preview-timestamp')
    const pButtons   = document.getElementById('preview-buttons')

    const statusDot     = document.querySelector('.status-dot')
    const statusText    = document.querySelector('.status-text')
    const statStatusText = document.getElementById('stat-status-text')
    const statStatusDot  = document.querySelector('#stat-status .status-dot-sm')
    const statActivity   = document.getElementById('stat-activity')
    const statUptime     = document.getElementById('stat-uptime')

    let timerInterval  = null
    let activeToken    = null
    let uptimeInterval = null
    let rpcStartTime   = null

    // ── Token restore ──────────────────────────────────────────
    const savedToken = sessionStorage.getItem('discord_token')
    if (savedToken) document.getElementById('token').value = savedToken

    // ── Uptime ─────────────────────────────────────────────────
    const formatUptime = (ms) => {
        const s = Math.floor(ms / 1000)
        const h = Math.floor(s / 3600)
        const m = Math.floor((s % 3600) / 60)
        const sec = s % 60
        return `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(sec).padStart(2,'0')}`
    }

    const startUptime = () => {
        if (rpcStartTime) return
        rpcStartTime = Date.now()
        clearInterval(uptimeInterval)
        uptimeInterval = setInterval(() => {
            if (statUptime) statUptime.textContent = formatUptime(Date.now() - rpcStartTime)
        }, 1000)
    }

    const stopUptime = () => {
        clearInterval(uptimeInterval)
        rpcStartTime = null
        if (statUptime) statUptime.textContent = '--:--:--'
        if (statActivity) statActivity.textContent = '\u2014'
    }

    // ── Char counters ──────────────────────────────────────────
    const setupCounter = (inputId, counterId) => {
        const input   = document.getElementById(inputId)
        const counter = document.getElementById(counterId)
        if (!input || !counter) return
        const update = () => {
            const len = input.value.length
            const max = parseInt(input.getAttribute('maxlength')) || 128
            counter.textContent = `${len}/${max}`
            counter.style.color = len > max * 0.9 ? '#ff4d6d' : ''
        }
        input.addEventListener('input', update)
        update()
    }
    setupCounter('details', 'counter-details')
    setupCounter('state',   'counter-state')

    // ── Timestamp convert ──────────────────────────────────────
    const convertTimestamp = (val) => {
        if (!val) return ''
        if (/^\d+$/.test(val)) return val
        const p = val.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/)
        if (p) {
            const d = new Date(+p[1], +p[2]-1, +p[3], +p[4], +p[5], +(p[6]||0))
            return String(Math.floor(d.getTime() / 1000))
        }
        return ''
    }

    // ── Form helpers ───────────────────────────────────────────
    const getFormData = () => {
        const data = {}
        inputs.forEach(input => {
            data[input.id] = input.id === 'type' ? parseInt(input.value) : input.value
        })
        data.timestamp_start = convertTimestamp(data.timestamp_start)
        data.timestamp_end   = convertTimestamp(data.timestamp_end)
        return data
    }

    const fillForm = (data) => {
        inputs.forEach(input => {
            if (data[input.id] !== undefined) {
                input.value = data[input.id] || (input.tagName === 'SELECT' ? '0' : '')
            }
        })
        setupCounter('details', 'counter-details')
        setupCounter('state',   'counter-state')
        updatePreview()
    }

    // ── Status message ─────────────────────────────────────────
    const showStatus = (msg, isError = false) => {
        statusMsg.textContent = msg
        statusMsg.style.display = 'block'
        statusMsg.style.backgroundColor = isError ? 'rgba(237,66,69,0.2)' : 'rgba(59,165,92,0.2)'
        statusMsg.style.color = isError ? '#ff7b72' : '#7ee787'
        setTimeout(() => { statusMsg.style.display = 'none' }, 8000)
    }

    // ── Connection status polling ──────────────────────────────
    const updateConnectionStatus = async () => {
        const token = document.getElementById('token').value || activeToken
        if (!token) return
        try {
            const res  = await fetch(`/api/rpc/status?token=${encodeURIComponent(token)}`)
            const data = await res.json()
            const cls  = data.status.toLowerCase().replace('...', '')
            statusText.textContent = data.status
            statusDot.className    = 'status-dot ' + cls
            if (statStatusText) statStatusText.textContent = data.status
            if (statStatusDot)  statStatusDot.className    = 'status-dot-sm ' + cls
        } catch {
            statusText.textContent = 'Disconnected'
            statusDot.className    = 'status-dot disconnected'
            if (statStatusText) statStatusText.textContent = 'Disconnected'
            if (statStatusDot)  statStatusDot.className    = 'status-dot-sm disconnected'
        }
    }
    setInterval(updateConnectionStatus, 3000)

    // ── Preview timer ──────────────────────────────────────────
    const stopTimer = () => { if (timerInterval) { clearInterval(timerInterval); timerInterval = null } }

    const startTimer = (ts) => {
        stopTimer()
        const startTime = /^\d+$/.test(ts) ? parseInt(ts) * 1000 : new Date(ts).getTime()
        const tick = () => {
            const diff = Math.floor((Date.now() - startTime) / 1000)
            if (diff < 0) { pTimestamp.textContent = 'Starting soon...'; return }
            const h = Math.floor(diff / 3600)
            const m = Math.floor((diff % 3600) / 60)
            const s = diff % 60
            pTimestamp.textContent = (h > 0
                ? `${h}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`
                : `${m}:${String(s).padStart(2,'0')}`) + ' elapsed'
        }
        tick()
        timerInterval = setInterval(tick, 1000)
    }

    // ── Preview ────────────────────────────────────────────────
    const previewImageUrl = (url) => {
        if (!url) return ''
        if (url.startsWith('http://') || url.startsWith('https://')) return url
        return ''
    }

    const updatePreview = () => {
        const data = getFormData()
        const typeLabels = { 0:'PLAYING A GAME', 1:'STREAMING', 2:'LISTENING TO', 3:'WATCHING', 5:'COMPETING IN' }
        pHeader.textContent  = typeLabels[data.type] || 'PLAYING A GAME'
        pAppName.textContent = data.app_name || 'ClaudiaRPC'
        pDetails.textContent = data.details  || 'Rich Presence Details'
        pState.textContent   = data.state    || 'State Info'

        const largeUrl = previewImageUrl(data.large_image)
        pLargeImg.src = largeUrl || 'https://discord.com/assets/2c21aeda16de354ba5334551a883b481.png'

        const smallUrl = previewImageUrl(data.small_image)
        if (smallUrl) { pSmallImg.src = smallUrl; pSmallImg.style.display = 'block' }
        else            pSmallImg.style.display = 'none'

        if (data.timestamp_start) { pTimestamp.style.display = 'block'; startTimer(data.timestamp_start) }
        else                       { pTimestamp.style.display = 'none';  stopTimer() }

        pButtons.innerHTML = ''
        ;[['button1_label','button1_url'],['button2_label','button2_url']].forEach(([lk,uk]) => {
            if (!data[lk]) return
            const a = document.createElement('a')
            a.className   = 'preview-btn'
            a.textContent = data[lk]
            if (data[uk]) { a.href = data[uk]; a.target = '_blank' }
            pButtons.appendChild(a)
        })
    }

    // ── Profiles ───────────────────────────────────────────────
    const loadProfiles = async () => {
        try {
            const res      = await fetch('/api/profiles')
            const profiles = await res.json()
            const current  = profileSelect.value
            profileSelect.innerHTML = '<option value="">-- Select Profile --</option>'
            Object.keys(profiles).forEach(name => {
                const opt = document.createElement('option')
                opt.value = opt.textContent = name
                profileSelect.appendChild(opt)
            })
            if (current) profileSelect.value = current
            return profiles
        } catch { return {} }
    }

    // Quick switch
    profileSelect.addEventListener('change', async () => {
        const name = profileSelect.value
        if (!name) return
        const profiles = await loadProfiles()
        if (profiles[name]) {
            fillForm(profiles[name])
            fetch('/api/profiles/last', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            }).catch(() => {})
        }
    })

    inputs.forEach(input => input.addEventListener('input', updatePreview))

    // ── Validation warning ────────────────────────────────────
    const validateWarnings = (data) => {
        const w = []
        if (!data.client_id || !data.client_id.trim()) w.push('Application ID kosong')
        if (data.button1_label && !data.button1_url)   w.push('Button 1 tidak ada URL')
        if (data.button2_label && !data.button2_url)   w.push('Button 2 tidak ada URL')
        if (data.button1_url && !data.button1_url.startsWith('http')) w.push('Button 1 URL tidak valid')
        if (data.button2_url && !data.button2_url.startsWith('http')) w.push('Button 2 URL tidak valid')
        return w
    }

    // ── Start RPC ─────────────────────────────────────────────
    let warnConfirmed = false
    document.getElementById('start-btn').addEventListener('click', async () => {
        const data  = getFormData()
        sessionStorage.setItem('discord_token', data.token)
        activeToken = data.token

        const warnings = validateWarnings(data)
        if (warnings.length && !warnConfirmed) {
            showStatus('\u26a0 ' + warnings.join(', ') + ' — Pencet Start lagi untuk tetap lanjut.', true)
            warnConfirmed = true
            return
        }
        warnConfirmed = false

        try {
            const res    = await fetch('/api/rpc/start', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(data) })
            const result = await res.json()
            if (result.error) throw new Error(result.error)
            showStatus('RPC Started!')
            startUptime()
            if (statActivity) statActivity.textContent = data.app_name || 'ClaudiaRPC'
            updateConnectionStatus()
        } catch (err) { showStatus(err.message, true) }
    })

    // ── Update RPC ─────────────────────────────────────────────
    document.getElementById('update-btn').addEventListener('click', async () => {
        const data = getFormData()
        if (!data.token) { showStatus('Token kosong.', true); return }
        activeToken = data.token
        try {
            const res    = await fetch('/api/rpc/start', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(data) })
            const result = await res.json()
            if (result.error) throw new Error(result.error)
            showStatus('RPC Updated!')
        } catch (err) { showStatus(err.message, true) }
    })

    // ── Stop RPC ───────────────────────────────────────────────
    document.getElementById('stop-btn').addEventListener('click', async () => {
        const token = document.getElementById('token').value || activeToken
        if (!token) { showStatus('Token kosong.', true); return }
        try {
            const res    = await fetch('/api/rpc/stop', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ token }) })
            const result = await res.json()
            if (result.error) throw new Error(result.error)
            showStatus('RPC Stopped.')
            stopTimer()
            pTimestamp.style.display = 'none'
            activeToken = null
            stopUptime()
            updateConnectionStatus()
        } catch (err) { showStatus(err.message, true) }
    })

    // ── Save profile ───────────────────────────────────────────
    document.getElementById('save-profile-btn').addEventListener('click', async () => {
        const name = prompt('Nama profile:')
        if (!name || !name.trim()) return
        const data = getFormData()
        try {
            const res    = await fetch('/api/profiles', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name: name.trim(), data }) })
            const result = await res.json()
            if (result.error) throw new Error(result.error)
            showStatus(`Profile "${name.trim()}" disimpan.`)
            await loadProfiles()
            profileSelect.value = name.trim()
        } catch (err) { showStatus(err.message, true) }
    })

    // ── Delete profile ─────────────────────────────────────────
    document.getElementById('delete-profile-btn').addEventListener('click', async () => {
        const name = profileSelect.value
        if (!name) { showStatus('Pilih profile dulu.', true); return }
        if (!confirm(`Hapus profile "${name}"?`)) return
        try {
            await fetch(`/api/profiles/${encodeURIComponent(name)}`, { method:'DELETE' })
            showStatus(`Profile "${name}" dihapus.`)
            await loadProfiles()
        } catch (err) { showStatus(err.message, true) }
    })

    // ── Export profiles ────────────────────────────────────────
    document.getElementById('export-profile-btn').addEventListener('click', async () => {
        try {
            const res      = await fetch('/api/profiles')
            const profiles = await res.json()
            const blob = new Blob([JSON.stringify(profiles, null, 2)], { type: 'application/json' })
            const a = document.createElement('a')
            a.href = URL.createObjectURL(blob)
            a.download = 'claudiarpc_profiles.json'
            a.click()
            showStatus('Profiles berhasil di-export!')
        } catch (err) { showStatus('Export gagal: ' + err.message, true) }
    })

    // ── Import profiles ────────────────────────────────────────
    document.getElementById('import-file').addEventListener('change', async (e) => {
        const file = e.target.files[0]
        if (!file) return
        try {
            const text     = await file.text()
            const profiles = JSON.parse(text)
            for (const [name, data] of Object.entries(profiles)) {
                await fetch('/api/profiles', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name, data }) })
            }
            showStatus(`${Object.keys(profiles).length} profile berhasil di-import!`)
            await loadProfiles()
        } catch (err) { showStatus('Import gagal: ' + err.message, true) }
        e.target.value = ''
    })

    // ── Upload image ───────────────────────────────────────────
    document.querySelectorAll('.btn-upload').forEach(btn => {
        btn.addEventListener('click', async () => {
            const input = document.getElementById(btn.getAttribute('data-target'))
            const url   = input.value.trim()
            if (!url) { showStatus('Isi URL gambar dulu.', true); return }
            if (!url.startsWith('http')) { showStatus('URL harus http:// atau https://', true); return }
            btn.textContent = '...'
            btn.disabled    = true
            try {
                const res    = await fetch('/api/image', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ url }) })
                const result = await res.json()
                if (result.error) throw new Error(result.error)
                input.value = result.url
                updatePreview()
                showStatus('Gambar berhasil di-upload!')
            } catch (err) { showStatus('Upload gagal: ' + err.message, true) }
            finally { btn.textContent = 'Upload'; btn.disabled = false }
        })
    })

    // ── Init ───────────────────────────────────────────────────
    const init = async () => {
        await loadProfiles()
        try {
            const res    = await fetch('/api/profiles/last')
            const { name } = await res.json()
            if (name && profileSelect.querySelector(`option[value="${CSS.escape(name)}"]`)) {
                profileSelect.value = name
                const profiles = await (await fetch('/api/profiles')).json()
                if (profiles[name]) fillForm(profiles[name])
            }
        } catch {}
        updatePreview()
    }

    init()
})
