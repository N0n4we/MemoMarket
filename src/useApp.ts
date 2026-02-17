import { ref, computed, onMounted, watch } from "vue";
import { invoke } from "@tauri-apps/api/core";
import {
  type Channel,
  fetchServerInfo, registerOnServer,
  publishRulePack as apiPublishRulePack,
  listRulePacks as apiListRulePacks,
} from "./api";

export interface MemoRule {
  title: string;
  updateRule: string;
}

export interface Memo {
  title: string;
  content: string;
}

export interface RulePack {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  systemPrompt: string;
  rules: MemoRule[];
  memos: Memo[];
  tags: string[];
  createdAt: string;
  updatedAt: string;
}

export type View = "browse" | "create" | "detail" | "settings" | "publish" | "channels";
export type PanelState = "closed" | "expanding" | "expanded" | "collapsing";

function generateId() {
  return `pack_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
}

function nowISO() {
  return new Date().toISOString().slice(0, 19);
}

function toBackend(pack: RulePack) {
  return {
    id: pack.id,
    name: pack.name,
    description: pack.description,
    author: pack.author,
    version: pack.version,
    system_prompt: pack.systemPrompt,
    rules: pack.rules.map(r => ({ title: r.title, update_rule: r.updateRule })),
    memos: pack.memos.map(m => ({ title: m.title, content: m.content })),
    tags: pack.tags,
    created_at: pack.createdAt,
    updated_at: pack.updatedAt,
  };
}

function fromBackend(raw: any): RulePack {
  return {
    id: raw.id,
    name: raw.name,
    description: raw.description,
    author: raw.author || raw.author_name || "",
    version: raw.version,
    systemPrompt: raw.system_prompt || "",
    rules: (raw.rules || []).map((r: any) => ({ title: r.title, updateRule: r.update_rule || r.updateRule })),
    memos: (raw.memos || []).map((m: any) => ({ title: m.title, content: m.content })),
    tags: raw.tags || [],
    createdAt: raw.created_at || "",
    updatedAt: raw.updated_at || "",
  };
}

export function useApp() {
  const packs = ref<RulePack[]>([]);
  const installedIds = ref<string[]>([]);
  const currentView = ref<View>("browse");
  const selectedPack = ref<RulePack | null>(null);
  const searchQuery = ref("");
  const filterTag = ref("");

  // Settings
  const settingsState = ref<PanelState>("closed");
  const settingsContentVisible = ref(false);
  const settingsBtnRef = ref<HTMLButtonElement | null>(null);
  const settingsTitleRef = ref<HTMLHeadingElement | null>(null);
  const settingsBtnRect = ref({ top: 0, left: 0, width: 0, height: 0 });
  const settingsTitleRect = ref({ top: 0, left: 0, width: 0, height: 0 });

  // Create/Edit form
  const editPack = ref<RulePack>({
    id: "", name: "", description: "", author: "", version: "1.0.0",
    systemPrompt: "", rules: [], memos: [], tags: [], createdAt: "", updatedAt: "",
  });

  // Channels — each channel = one backend server URL + token
  const channels = ref<Channel[]>([]);
  const selectedChannelId = ref("");
  const publishing = ref(false);
  const publishError = ref("");
  const publishSuccess = ref("");

  // New channel form
  const newChannelUrl = ref("");
  const newChannelToken = ref("");
  const addingChannel = ref(false);
  const addChannelError = ref("");

  // Local / Remote toggle
  const viewMode = ref<"local" | "remote">("local");
  const remotePacks = ref<RulePack[]>([]);
  const loadingRemote = ref(false);

  onMounted(() => {
    loadConfig();
    loadPacks();
    loadInstalled();
  });

  async function loadConfig() {
    try {
      const config = await invoke<{
        api_key: string; model_id: string; base_url: string;
        reasoning_enabled: boolean; channels_json: string;
      }>("load_config");
      if (config) {
        if (config.channels_json) {
          try {
            channels.value = JSON.parse(config.channels_json);
            if (channels.value.length > 0 && !selectedChannelId.value) {
              selectedChannelId.value = channels.value[0].id;
            }
          } catch { channels.value = []; }
        }
      }
    } catch (e) {
      console.error("Failed to load config:", e);
    }
  }

  async function saveConfig() {
    try {
      await invoke("save_config", {
        apiKey: "",
        modelId: "",
        baseUrl: "",
        reasoningEnabled: false,
        channelsJson: JSON.stringify(channels.value),
      });
    } catch (e) {
      console.error("Failed to save config:", e);
    }
  }

  // Save channels when they change
  watch(channels, () => { saveConfig(); }, { deep: true });

  async function loadPacks() {
    try {
      const raw = await invoke<any[]>("load_packs");
      packs.value = (raw || []).map(fromBackend);
    } catch (e) {
      console.error("Failed to load packs:", e);
    }
  }

  async function loadInstalled() {
    try {
      const ids = await invoke<string[]>("load_installed");
      installedIds.value = ids || [];
    } catch (e) {
      console.error("Failed to load installed:", e);
    }
  }

  async function savePack(pack: RulePack) {
    try {
      await invoke("save_pack", { pack: toBackend(pack) });
      await loadPacks();
    } catch (e) {
      console.error("Failed to save pack:", e);
    }
  }

  async function deletePack(id: string) {
    try {
      await invoke("delete_pack", { id });
      installedIds.value = installedIds.value.filter(i => i !== id);
      await invoke("save_installed", { ids: installedIds.value });
      if (selectedPack.value?.id === id) {
        selectedPack.value = null;
        currentView.value = "browse";
      }
      await loadPacks();
    } catch (e) {
      console.error("Failed to delete pack:", e);
    }
  }

  function toggleInstall(id: string) {
    if (installedIds.value.includes(id)) {
      installedIds.value = installedIds.value.filter(i => i !== id);
    } else {
      installedIds.value.push(id);
    }
    invoke("save_installed", { ids: installedIds.value }).catch(e =>
      console.error("Failed to save installed:", e)
    );
  }

  const filteredPacks = computed(() => {
    let result = viewMode.value === "remote" ? remotePacks.value : packs.value;
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase();
      result = result.filter(p =>
        p.name.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q) ||
        p.author.toLowerCase().includes(q) ||
        p.tags.some(t => t.toLowerCase().includes(q))
      );
    }
    if (filterTag.value) {
      result = result.filter(p => p.tags.includes(filterTag.value));
    }
    return result;
  });

  const allTags = computed(() => {
    const source = viewMode.value === "remote" ? remotePacks.value : packs.value;
    const tags = new Set<string>();
    source.forEach(p => p.tags.forEach(t => tags.add(t)));
    return Array.from(tags).sort();
  });

  function startCreate() {
    editPack.value = {
      id: generateId(), name: "", description: "", author: "", version: "1.0.0",
      systemPrompt: "", rules: [], memos: [], tags: [], createdAt: nowISO(), updatedAt: nowISO(),
    };
    currentView.value = "create";
  }

  function startEdit(pack: RulePack) {
    editPack.value = JSON.parse(JSON.stringify(pack));
    currentView.value = "create";
  }

  function addRule() {
    editPack.value.rules.push({ title: "", updateRule: "" });
  }

  function removeRule(idx: number) {
    editPack.value.rules.splice(idx, 1);
  }

  function addMemo() {
    editPack.value.memos.push({ title: "", content: "" });
  }

  function removeMemo(idx: number) {
    editPack.value.memos.splice(idx, 1);
  }

  function addTag(tag: string) {
    const t = tag.trim().toLowerCase();
    if (t && !editPack.value.tags.includes(t)) {
      editPack.value.tags.push(t);
    }
  }

  function removeTag(idx: number) {
    editPack.value.tags.splice(idx, 1);
  }

  async function saveCurrentPack() {
    editPack.value.updatedAt = nowISO();
    await savePack(editPack.value);
    currentView.value = "browse";
  }

  function viewPack(pack: RulePack) {
    selectedPack.value = pack;
    currentView.value = "detail";
  }

  function goBack() {
    currentView.value = "browse";
    selectedPack.value = null;
  }

  // Export pack as MemoChat-compatible JSON
  function exportForMemoChat(pack: RulePack) {
    const data = {
      systemPrompt: pack.systemPrompt,
      rules: pack.rules.map(r => ({ title: r.title, updateRule: r.updateRule })),
    };
    const json = JSON.stringify(data, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${pack.name.replace(/\s+/g, "-").toLowerCase() || "rules"}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  // Export pack to a directory chosen by user
  async function exportPack(pack: RulePack) {
    try {
      const json = JSON.stringify(toBackend(pack), null, 2);
      const filename = `${pack.name.replace(/\s+/g, "-").toLowerCase() || "pack"}.memomarket.json`;
      await invoke("export_pack", { content: json, filename });
    } catch (e) {
      console.error("Failed to export pack:", e);
    }
  }

  // Import from MemoChat (reads current RulePack + MemoPack from MemoChat)
  async function importFromMemoChat() {
    try {
      const data = await invoke<{ rules: any; memos: any }>("import_from_memochat");
      const now = nowISO();
      const pack: RulePack = {
        id: generateId(),
        name: "Imported from MemoChat",
        description: "Current rules and memos from MemoChat",
        author: "",
        version: "1.0.0",
        systemPrompt: data.rules?.systemPrompt || "",
        rules: (data.rules?.rules || []).map((r: any) => ({ title: r.title, updateRule: r.updateRule })),
        memos: (data.memos || []).map((m: any) => ({ title: m.title, content: m.content })),
        tags: ["imported", "memochat"],
        createdAt: now,
        updatedAt: now,
      };
      await savePack(pack);
      await loadPacks();
    } catch (e) {
      console.error("Failed to import from MemoChat:", e);
    }
  }

  // Import from file
  function importPack() {
    const fileInput = document.createElement("input");
    fileInput.type = "file";
    fileInput.accept = ".json";
    fileInput.onchange = async () => {
      const file = fileInput.files?.[0];
      if (!file) return;
      try {
        const text = await file.text();
        const data = JSON.parse(text);

        // Detect format: MemoChat rules or MemoMarket pack
        if (data.rules && data.systemPrompt !== undefined && !data.id) {
          // MemoChat format
          const now = nowISO();
          const pack: RulePack = {
            id: generateId(),
            name: file.name.replace(/\.json$/, "").replace(/-/g, " "),
            description: "Imported from MemoChat",
            author: "",
            version: "1.0.0",
            systemPrompt: data.systemPrompt || "",
            rules: (data.rules || []).map((r: any) => ({ title: r.title, updateRule: r.updateRule })),
            memos: [],
            tags: ["imported"],
            createdAt: now,
            updatedAt: now,
          };
          await savePack(pack);
        } else if (data.id) {
          // MemoMarket format
          const pack = fromBackend(data);
          pack.id = generateId(); // new ID to avoid conflicts
          await savePack(pack);
        }
      } catch (e) {
        console.error("Failed to import:", e);
      }
    };
    fileInput.click();
  }

  // Settings panel animation (same pattern as MemoChat)
  function openSettings() {
    if (settingsBtnRef.value) {
      const rect = settingsBtnRef.value.getBoundingClientRect();
      settingsBtnRect.value = { top: rect.top, left: rect.left, width: rect.width, height: rect.height };
    }
    const contentMaxWidth = 400;
    const contentPaddingTop = 40;
    const contentPaddingLeft = 24;
    const contentLeft = Math.max((window.innerWidth - contentMaxWidth) / 2, 0);
    settingsTitleRect.value = { top: contentPaddingTop, left: contentLeft + contentPaddingLeft, width: 0, height: 0 };
    settingsState.value = "expanding";
    setTimeout(() => { settingsContentVisible.value = true; }, 150);
    setTimeout(() => { settingsState.value = "expanded"; }, 400);
  }

  function closeSettings() {
    if (settingsBtnRef.value) {
      const rect = settingsBtnRef.value.getBoundingClientRect();
      settingsBtnRect.value = { top: rect.top, left: rect.left, width: rect.width, height: rect.height };
    }
    if (settingsTitleRef.value) {
      const rect = settingsTitleRef.value.getBoundingClientRect();
      settingsTitleRect.value = { top: rect.top, left: rect.left, width: rect.width, height: rect.height };
    }
    settingsContentVisible.value = false;
    settingsState.value = "collapsing";
    setTimeout(() => { settingsState.value = "closed"; }, 400);
  }

  // ---- Remote packs: fetch from all channels ----

  async function fetchRemotePacks() {
    if (channels.value.length === 0) {
      remotePacks.value = [];
      return;
    }
    loadingRemote.value = true;
    try {
      const results = await Promise.allSettled(
        channels.value.map(async (ch) => {
          const res = await apiListRulePacks(ch.url, { limit: 100 });
          return (res.items || []).map((raw: any) => ({
            ...fromBackend(raw),
            _channelName: ch.name,
            _channelUrl: ch.url,
          }));
        })
      );
      const all: RulePack[] = [];
      for (const r of results) {
        if (r.status === "fulfilled") all.push(...r.value);
      }
      remotePacks.value = all;
    } catch (e) {
      console.error("Failed to fetch remote packs:", e);
    } finally {
      loadingRemote.value = false;
    }
  }

  // Auto-fetch remote packs when switching to remote mode
  watch(viewMode, (mode) => {
    if (mode === "remote") fetchRemotePacks();
  });

  // ---- Channel management (each channel = a backend server) ----

  const selectedChannel = computed(() =>
    channels.value.find(c => c.id === selectedChannelId.value) || null
  );

  async function addChannel() {
    const url = newChannelUrl.value.trim().replace(/\/$/, "");
    if (!url) return;
    addingChannel.value = true;
    addChannelError.value = "";
    try {
      // Fetch server info to get name
      const info = await fetchServerInfo(url);
      const ch: Channel = {
        id: `ch_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
        url,
        token: newChannelToken.value.trim(),
        name: info.name || url,
        description: info.description || "",
      };
      channels.value.push(ch);
      if (!selectedChannelId.value) {
        selectedChannelId.value = ch.id;
      }
      newChannelUrl.value = "";
      newChannelToken.value = "";
    } catch (e: any) {
      addChannelError.value = e.message || "Failed to connect to server";
    } finally {
      addingChannel.value = false;
    }
  }

  function removeChannel(id: string) {
    channels.value = channels.value.filter(c => c.id !== id);
    if (selectedChannelId.value === id) {
      selectedChannelId.value = channels.value.length > 0 ? channels.value[0].id : "";
    }
  }

  function updateChannelToken(id: string, token: string) {
    const ch = channels.value.find(c => c.id === id);
    if (ch) ch.token = token;
  }

  async function registerOnChannel(channelId: string, username: string, displayName: string) {
    const ch = channels.value.find(c => c.id === channelId);
    if (!ch) return;
    publishError.value = "";
    try {
      const user = await registerOnServer(ch.url, username, displayName);
      ch.token = user.token || "";
    } catch (e: any) {
      publishError.value = e.message || "Registration failed";
    }
  }

  async function publishPack(pack: RulePack) {
    const ch = selectedChannel.value;
    if (!ch) {
      publishError.value = "Select a channel first";
      return;
    }
    if (!ch.token) {
      publishError.value = "No auth token for this channel. Register or add a token first.";
      return;
    }
    publishing.value = true;
    publishError.value = "";
    publishSuccess.value = "";
    try {
      await apiPublishRulePack(ch.url, ch.token, {
        name: pack.name,
        description: pack.description,
        version: pack.version,
        system_prompt: pack.systemPrompt,
        rules: pack.rules.map(r => ({ title: r.title, update_rule: r.updateRule })),
        tags: pack.tags,
      });
      publishSuccess.value = `"${pack.name}" published to ${ch.name}!`;
    } catch (e: any) {
      publishError.value = e.message || "Publish failed";
    } finally {
      publishing.value = false;
    }
  }

  function openPublish(pack: RulePack) {
    selectedPack.value = pack;
    publishError.value = "";
    publishSuccess.value = "";
    currentView.value = "publish";
  }

  function openChannels() {
    addChannelError.value = "";
    currentView.value = "channels";
  }

  return {
    packs, installedIds, currentView, selectedPack, searchQuery, filterTag,
    filteredPacks, allTags,
    settingsState, settingsContentVisible,
    settingsBtnRef, settingsTitleRef, settingsBtnRect, settingsTitleRect,
    editPack,
    loadPacks, savePack, deletePack, toggleInstall,
    startCreate, startEdit, addRule, removeRule, addMemo, removeMemo, addTag, removeTag,
    saveCurrentPack, viewPack, goBack,
    exportForMemoChat, exportPack, importPack, importFromMemoChat,
    openSettings, closeSettings,
    // Local / Remote toggle
    viewMode, remotePacks, loadingRemote, fetchRemotePacks,
    // Channels (each = a backend server)
    channels, selectedChannelId, selectedChannel,
    publishing, publishError, publishSuccess,
    newChannelUrl, newChannelToken, addingChannel, addChannelError,
    addChannel, removeChannel, updateChannelToken, registerOnChannel,
    publishPack, openPublish, openChannels,
  };
}
