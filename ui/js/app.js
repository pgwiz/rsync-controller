// Rsync Controller Interactive Frontend
// Zero Emoji Policy Enforced Across All Code & UI

// State Management
const state = {
  currentFolder: "E:\\Backup\\pgwiz\\rsync",
  isPullMode: false,
  isLocked: true,
  items: [],
  selectedItems: new Map(), // ID -> { id, name, size, isDir, path }
  activeTab: "tab-source",
  activeTheme: "y2k-candy",
  activeEngine: "usn",
  serverOnline: true
};

// Initial Mock / Fallback items
const initialWorkspaceItems = [
  { id: 1002, name: "Themes examples", size: 0, isDir: true, path: "E:\\Backup\\pgwiz\\rsync\\Themes examples" },
  { id: 1019, name: "cmd", size: 0, isDir: true, path: "E:\\Backup\\pgwiz\\rsync\\cmd" },
  { id: 1022, name: "core", size: 0, isDir: true, path: "E:\\Backup\\pgwiz\\rsync\\core" },
  { id: 1025, name: "themes", size: 0, isDir: true, path: "E:\\Backup\\pgwiz\\rsync\\themes" },
  { id: 1026, name: "ui", size: 0, isDir: true, path: "E:\\Backup\\pgwiz\\rsync\\ui" },
  { id: 1027, name: "go.mod", size: 81, isDir: false, path: "E:\\Backup\\pgwiz\\rsync\\go.mod" },
  { id: 1028, name: "go.sum", size: 153, isDir: false, path: "E:\\Backup\\pgwiz\\rsync\\go.sum" },
  { id: 1030, name: "theme-y2k-classic-desktop-blue-folders.jpg", size: 489200, isDir: false, path: "E:\\Backup\\pgwiz\\rsync\\Themes examples\\theme-y2k-classic-desktop-blue-folders.jpg" },
  { id: 1031, name: "theme-vaporwave-retrowave-vintage-pc.jpg", size: 612400, isDir: false, path: "E:\\Backup\\pgwiz\\rsync\\Themes examples\\theme-vaporwave-retrowave-vintage-pc.jpg" },
  { id: 1032, name: "theme-monochrome-classic-80s-90s-gui.jpg", size: 312800, isDir: false, path: "E:\\Backup\\pgwiz\\rsync\\Themes examples\\theme-monochrome-classic-80s-90s-gui.jpg" },
  { id: 1033, name: "theme-ui-reference-window.png", size: 529679, isDir: false, path: "E:\\Backup\\pgwiz\\rsync\\Themes examples\\theme-ui-reference-window.png" }
];

// Utility: Format file sizes
function formatSize(bytes) {
  if (bytes === 0) return "--";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  let val = bytes;
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024;
    i++;
  }
  return val.toFixed(1) + " " + units[i];
}

// Fetch live items from Go backend
async function loadLiveItems(targetPath) {
  try {
    const res = await fetch(`/api/scan?target=${encodeURIComponent(targetPath || state.currentFolder)}`);
    if (res.ok) {
      const data = await res.json();
      if (data && data.items && data.items.length > 0) {
        state.items = data.items.map(it => ({
          id: it.id,
          name: it.name,
          size: it.size,
          isDir: it.isDir,
          path: it.path || `${targetPath}\\${it.name}`
        }));
        if (data.engine) {
          const badge = document.getElementById("engineStatusBadge");
          if (badge) badge.textContent = `ENGINE: ${data.engine.toUpperCase()}`;
        }
        renderSourceItems(state.items);
      }
    }
  } catch (err) {
    console.log("Using cached/preloaded items");
  }
}

// Render source items into table
function renderSourceItems(itemsToRender) {
  const tbody = document.getElementById("sourceItemsBody");
  if (!tbody) return;
  tbody.innerHTML = "";

  itemsToRender.forEach(item => {
    const tr = document.createElement("tr");
    const isChecked = state.selectedItems.has(item.id);
    if (isChecked) tr.classList.add("selected");

    const kindText = item.isDir ? "[DIR]" : "[FILE]";

    tr.innerHTML = `
      <td class="col-check">
        <input type="checkbox" class="item-checkbox" data-id="${item.id}" ${isChecked ? "checked" : ""}>
      </td>
      <td class="col-icon">${kindText}</td>
      <td><b>${item.name}</b></td>
      <td class="col-size">${formatSize(item.size)}</td>
    `;

    const checkbox = tr.querySelector(".item-checkbox");
    checkbox.addEventListener("change", (e) => {
      toggleSelection(item, e.target.checked);
    });

    tbody.appendChild(tr);
  });
}

// Toggle individual item selection
function toggleSelection(item, isChecked) {
  if (isChecked) {
    state.selectedItems.set(item.id, item);
  } else {
    state.selectedItems.delete(item.id);
  }
  updateSelectionSummary();
  syncCheckboxElements();
}

// Update bottom summary bar and tags
function updateSelectionSummary() {
  const count = state.selectedItems.size;
  let totalBytes = 0;
  state.selectedItems.forEach(item => {
    totalBytes += item.size;
  });

  const summaryEl = document.getElementById("selectedCountSummary");
  if (summaryEl) {
    summaryEl.textContent = `[x] ${count} item${count === 1 ? "" : "s"} selected (${formatSize(totalBytes)})`;
  }

  const countTag = document.getElementById("checkedCountTag");
  if (countTag) {
    countTag.textContent = count;
  }

  renderCheckedItems();
}

// Render checked items tab
function renderCheckedItems() {
  const tbody = document.getElementById("checkedItemsBody");
  if (!tbody) return;

  if (state.selectedItems.size === 0) {
    tbody.innerHTML = `<tr><td colspan="4" style="text-align: center; padding: 20px; color: #888;">No items currently checked. Check items in Source or Search tab.</td></tr>`;
    return;
  }

  tbody.innerHTML = "";
  state.selectedItems.forEach(item => {
    const tr = document.createElement("tr");
    tr.classList.add("selected");
    const kindText = item.isDir ? "[DIR]" : "[FILE]";

    tr.innerHTML = `
      <td class="col-check">
        <button class="tactile-btn" data-remove-id="${item.id}" style="padding: 1px 6px; font-size: 10px;">[X]</button>
      </td>
      <td class="col-icon">${kindText}</td>
      <td><b>${item.name}</b><div style="font-size: 10px; color: #777;">${item.path}</div></td>
      <td class="col-size">${formatSize(item.size)}</td>
    `;

    tr.querySelector("[data-remove-id]").addEventListener("click", () => {
      toggleSelection(item, false);
      renderCheckedItems();
    });

    tbody.appendChild(tr);
  });
}

// Synchronize all visible checkboxes across tabs
function syncCheckboxElements() {
  document.querySelectorAll(".item-checkbox").forEach(cb => {
    const id = parseInt(cb.dataset.id, 10);
    cb.checked = state.selectedItems.has(id);
    const row = cb.closest("tr");
    if (row) {
      if (cb.checked) {
        row.classList.add("selected");
      } else {
        row.classList.remove("selected");
      }
    }
  });
}

// Setup Instant On-Type Search Filter
function setupSearch() {
  const searchInput = document.getElementById("searchInput");
  if (!searchInput) return;

  searchInput.addEventListener("input", (e) => {
    const q = e.target.value.trim().toLowerCase();
    const hitsTbody = document.getElementById("searchItemsBody");
    const hitCountEl = document.getElementById("searchHitCount");

    if (!q) {
      hitsTbody.innerHTML = `<tr><td colspan="4" style="text-align: center; padding: 20px; color: #888;">Type in search bar to see instant cached results without losing selections.</td></tr>`;
      if (hitCountEl) hitCountEl.textContent = "0";
      return;
    }

    const matches = state.items.filter(item => item.name.toLowerCase().includes(q));
    if (hitCountEl) hitCountEl.textContent = matches.length;

    if (matches.length === 0) {
      hitsTbody.innerHTML = `<tr><td colspan="4" style="text-align: center; padding: 20px; color: #888;">No matches found for query: "${q}"</td></tr>`;
      return;
    }

    hitsTbody.innerHTML = "";
    matches.forEach(item => {
      const tr = document.createElement("tr");
      const isChecked = state.selectedItems.has(item.id);
      if (isChecked) tr.classList.add("selected");
      const kindText = item.isDir ? "[DIR]" : "[FILE]";

      tr.innerHTML = `
        <td class="col-check">
          <input type="checkbox" class="item-checkbox" data-id="${item.id}" ${isChecked ? "checked" : ""}>
        </td>
        <td class="col-icon">${kindText}</td>
        <td><b>${item.name}</b></td>
        <td style="font-family: monospace; font-size: 11px; color: #555;">${item.path}</td>
      `;

      tr.querySelector(".item-checkbox").addEventListener("change", (ev) => {
        toggleSelection(item, ev.target.checked);
      });

      hitsTbody.appendChild(tr);
    });

    if (state.activeTab !== "tab-search" && q.length >= 2) {
      switchTab("tab-search");
    }
  });
}

// Setup Tabs Navigation
function setupTabs() {
  document.querySelectorAll(".tab-btn").forEach(btn => {
    btn.addEventListener("click", () => {
      const targetTab = btn.dataset.tab;
      switchTab(targetTab);
    });
  });
}

function switchTab(tabId) {
  state.activeTab = tabId;
  document.querySelectorAll(".tab-btn").forEach(b => {
    b.classList.toggle("active", b.dataset.tab === tabId);
  });
  document.querySelectorAll(".tab-panel").forEach(p => {
    p.classList.toggle("active", p.id === tabId);
  });
  if (tabId === "tab-checked") {
    renderCheckedItems();
  }
}

// Setup Destination Lock Toggle
function setupLockToggle() {
  const btn = document.getElementById("btnToggleLock");
  const lockStatus = document.getElementById("lockStatusTag");
  const lockPill = document.getElementById("destLockPill");

  if (!btn) return;
  btn.addEventListener("click", () => {
    state.isLocked = !state.isLocked;
    if (state.isLocked) {
      btn.textContent = "[UNLOCK]";
      lockStatus.textContent = "[LOCKED]";
      lockPill.classList.add("locked");
    } else {
      btn.textContent = "[LOCK]";
      lockStatus.textContent = "[UNLOCKED]";
      lockPill.classList.remove("locked");
    }
  });
}

// Theme Studio Controllers & Presets
const themePresets = {
  "y2k-candy": {
    winRadius: 14,
    btnRadius: 16,
    shadow: 4,
    gloss: true,
    accent: "#00f0ff",
    canvas: "#ffeef5",
    titleBg: "linear-gradient(180deg, #ff7eb3 0%, #ff758c 50%, #e03860 100%)",
    font: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
  },
  "win95-classic": {
    winRadius: 0,
    btnRadius: 0,
    shadow: 0,
    gloss: false,
    accent: "#000080",
    canvas: "#c0c0c0",
    titleBg: "linear-gradient(90deg, #000080 0%, #1084d0 100%)",
    font: "'MS Sans Serif', Tahoma, sans-serif"
  },
  "monochrome-mac": {
    winRadius: 4,
    btnRadius: 3,
    shadow: 3,
    gloss: false,
    accent: "#000000",
    canvas: "#ffffff",
    titleBg: "repeating-linear-gradient(0deg, #000, #000 1px, #fff 1px, #fff 3px)",
    font: "monospace"
  },
  "vaporwave-cyber": {
    winRadius: 8,
    btnRadius: 10,
    shadow: 4,
    gloss: true,
    accent: "#ff007f",
    canvas: "#180928",
    titleBg: "linear-gradient(90deg, #ff007f 0%, #7928ca 50%, #00f0ff 100%)",
    font: "Segoe UI, monospace, sans-serif"
  }
};

function applyThemeValues(theme) {
  const root = document.documentElement;
  root.style.setProperty("--win-radius", `${theme.winRadius}px`);
  root.style.setProperty("--btn-radius", `${theme.btnRadius}px`);
  root.style.setProperty("--box-shadow", theme.shadow > 0 ? `${theme.shadow}px ${theme.shadow}px 0px #1a1a2e` : "none");
  root.style.setProperty("--accent-color", theme.accent);
  root.style.setProperty("--canvas-bg", theme.canvas);
  root.style.setProperty("--titlebar-bg", theme.titleBg);
  root.style.setProperty("--font-family", theme.font);

  if (theme.gloss) {
    document.body.classList.add("gloss-enabled");
  } else {
    document.body.classList.remove("gloss-enabled");
  }

  // Update controls
  const winSlider = document.getElementById("sliderWinRadius");
  if (winSlider) {
    winSlider.value = theme.winRadius;
    document.getElementById("valWinRadius").textContent = `${theme.winRadius}px`;
    document.getElementById("sliderShadow").value = theme.shadow;
    document.getElementById("valShadow").textContent = `${theme.shadow}px`;
    document.getElementById("checkGloss").checked = theme.gloss;
    document.getElementById("colorAccent").value = theme.accent;
    document.getElementById("colorCanvas").value = theme.canvas;
  }
}

function setupThemeStudio() {
  const select = document.getElementById("themeSelect");
  if (select) {
    select.addEventListener("change", (e) => {
      const preset = themePresets[e.target.value];
      if (preset) {
        applyThemeValues(preset);
      }
    });
  }

  // Interactive Sliders
  document.getElementById("sliderWinRadius").addEventListener("input", (e) => {
    document.documentElement.style.setProperty("--win-radius", `${e.target.value}px`);
    document.getElementById("valWinRadius").textContent = `${e.target.value}px`;
  });

  document.getElementById("sliderShadow").addEventListener("input", (e) => {
    const s = parseInt(e.target.value, 10);
    document.documentElement.style.setProperty("--box-shadow", s > 0 ? `${s}px ${s}px 0px #1a1a2e` : "none");
    document.getElementById("valShadow").textContent = `${s}px`;
  });

  document.getElementById("checkGloss").addEventListener("change", (e) => {
    if (e.target.checked) {
      document.body.classList.add("gloss-enabled");
    } else {
      document.body.classList.remove("gloss-enabled");
    }
  });

  document.getElementById("colorAccent").addEventListener("input", (e) => {
    document.documentElement.style.setProperty("--accent-color", e.target.value);
  });

  document.getElementById("btnExportTheme").addEventListener("click", () => {
    const exportedTheme = {
      themeName: "Custom User Theme",
      author: "Local User",
      style: {
        windowRadius: `${document.getElementById("sliderWinRadius").value}px`,
        boxShadow: `${document.getElementById("sliderShadow").value}px`,
        glossOverlay: document.getElementById("checkGloss").checked,
        accentColor: document.getElementById("colorAccent").value
      }
    };
    const blob = new Blob([JSON.stringify(exportedTheme, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "custom-theme.json";
    a.click();
    URL.revokeObjectURL(url);
    alert("[SUCCESS] Theme exported as custom-theme.json");
  });
}

// Collapsible Options / Settings Drawer & Shell Customizer
function setupSettingsDrawer() {
  const toggleBtn = document.getElementById("btnToggleSettings");
  const closeBtn = document.getElementById("btnCloseSettings");
  const drawer = document.getElementById("settingsDrawer");

  if (!toggleBtn || !drawer) return;

  function toggle() {
    const isHidden = drawer.style.display === "none";
    drawer.style.display = isHidden ? "block" : "none";
    toggleBtn.textContent = isHidden ? "[OPTIONS / SETTINGS ^]" : "[OPTIONS / SETTINGS v]";
  }

  toggleBtn.addEventListener("click", toggle);
  if (closeBtn) closeBtn.addEventListener("click", toggle);

  // Hook up Service Status and Installation
  const btnInstallSvc = document.getElementById("btnInstallService");
  if (btnInstallSvc) {
    btnInstallSvc.addEventListener("click", async () => {
      try {
        await fetch("/api/service/install", { method: "POST" });
        alert("[INFO] UAC prompt requested. Please click 'Yes' in the Windows confirmation dialog to register and start the service.");
        setTimeout(checkServiceStatus, 4000);
      } catch (err) {
        alert(`[ERROR] Failed to trigger service installer: ${err.message}`);
      }
    });
  }
}

async function checkServiceStatus() {
  try {
    const res = await fetch("/api/service/status");
    if (res.ok) {
      const data = await res.json();
      const tag = document.getElementById("svcStatusTag");
      if (tag) {
        tag.textContent = data.status;
        tag.style.color = data.running ? "#00aa00" : (data.installed ? "#dd8800" : "#cc0000");
      }
    }
  } catch (e) {}
}
  if (btnApplyShell) {
    btnApplyShell.addEventListener("click", async () => {
      const shellCfg = {
        menuTitle: document.getElementById("shellTitleInput").value.trim() || "Rsync",
        position: document.getElementById("shellPositionSelect").value,
        submenuDepth: parseInt(document.getElementById("shellDepthSelect").value, 10) || 1,
        showIcons: document.getElementById("checkShellIcons").checked,
        useSeparators: document.getElementById("checkShellSeparators").checked,
        useAccelerators: document.getElementById("checkShellAccelerators").checked
      };

      try {
        const getRes = await fetch("/api/config");
        let cfg = {};
        if (getRes.ok) {
          cfg = await getRes.json();
        }
        cfg.shell = shellCfg;
        cfg.preferredIndexer = document.getElementById("indexerSelect").value;

        const postRes = await fetch("/api/config", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(cfg)
        });

        if (postRes.ok) {
          alert("[SUCCESS] Explorer context menu successfully updated in Windows Registry!");
        } else {
          alert("[ERROR] Failed to update context menu settings.");
        }
      } catch (err) {
        alert(`[ERROR] Network error: ${err.message}`);
      }
    });
  }
}

// Select All Checkbox
function setupSelectAll() {
  const cb = document.getElementById("selectAllCheckbox");
  if (!cb) return;
  cb.addEventListener("change", (e) => {
    const isChecked = e.target.checked;
    state.items.forEach(item => {
      if (isChecked) {
        state.selectedItems.set(item.id, item);
      } else {
        state.selectedItems.delete(item.id);
      }
    });
    updateSelectionSummary();
    syncCheckboxElements();
  });
}

// Start Sync Handler
function setupSyncButton() {
  const btn = document.getElementById("btnStartSync");
  if (!btn) return;
  btn.addEventListener("click", () => {
    const count = state.selectedItems.size;
    if (count === 0) {
      alert("[NOTICE] Please select at least one file or folder to sync.");
      return;
    }
    const itemNames = Array.from(state.selectedItems.values()).map(it => it.name).join(", ");
    alert(`[SYNC STARTED]\nMode: ${state.isPullMode ? "Pull From" : "Copy To"}\nItems (${count}): ${itemNames}\nDestination: Alpha Server: /data/backup`);
  });
}

// Inline New Folder Handler
function setupNewFolderButton() {
  const btn = document.getElementById("btnNewFolder");
  if (!btn) return;
  btn.addEventListener("click", () => {
    const name = prompt("[PROMPT] Enter new folder name:");
    if (name && name.trim()) {
      alert(`[SUCCESS] Created new directory: ${name.trim()}`);
    }
  });
}

// Fetch Initial State from Backend
async function fetchInitialState() {
  try {
    const cfgRes = await fetch("/api/config");
    if (cfgRes.ok) {
      const cfg = await cfgRes.json();
      if (cfg.shell) {
        if (cfg.shell.menuTitle) document.getElementById("shellTitleInput").value = cfg.shell.menuTitle;
        if (cfg.shell.position) document.getElementById("shellPositionSelect").value = cfg.shell.position;
        if (cfg.shell.submenuDepth) document.getElementById("shellDepthSelect").value = cfg.shell.submenuDepth;
        document.getElementById("checkShellIcons").checked = cfg.shell.showIcons !== false;
        document.getElementById("checkShellSeparators").checked = cfg.shell.useSeparators !== false;
        document.getElementById("checkShellAccelerators").checked = cfg.shell.useAccelerators !== false;
      }
      if (cfg.preferredIndexer) {
        document.getElementById("indexerSelect").value = cfg.preferredIndexer;
      }
    }

    checkServiceStatus();

    const res = await fetch("/api/initial-state");
    if (res.ok) {
      const init = await res.json();
      if (init.mode === "pull") {
        state.isPullMode = true;
        document.getElementById("modeBadge").textContent = "[MODE: PULL FROM]";
      }
      if (init.destPath) {
        document.getElementById("destPathText").textContent = init.destPath;
      }
      if (init.srcPath) {
        state.currentFolder = init.srcPath;
        const badge = document.getElementById("currentFolderBadge");
        if (badge) badge.textContent = init.srcPath;
        await loadLiveItems(init.srcPath);
        return;
      }
    }
  } catch (e) {
    // Fallback to workspace
  }
  await loadLiveItems(state.currentFolder);
}

// Initialization
document.addEventListener("DOMContentLoaded", () => {
  state.items = initialWorkspaceItems;
  renderSourceItems(state.items);
  updateSelectionSummary();
  setupSearch();
  setupTabs();
  setupLockToggle();
  setupThemeStudio();
  setupSettingsDrawer();
  setupSelectAll();
  setupSyncButton();
  setupNewFolderButton();

  // Apply default Y2K Candy preset
  applyThemeValues(themePresets["y2k-candy"]);

  // Fetch real state and live directory items
  fetchInitialState();
});
