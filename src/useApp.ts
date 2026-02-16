import { ref, computed, onMounted, watch } from "vue";
import { invoke } from "@tauri-apps/api/core";
import { chatCompletion } from "./llm";

export interface MemoRule {
  title: string;
  updateRule: string;
}

export interface RulePack {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  systemPrompt: string;
  rules: MemoRule[];
  tags: string[];
  createdAt: string;
  updatedAt: string;
}

export type View = "browse" | "create" | "detail" | "settings";
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
    author: raw.author,
    version: raw.version,
    systemPrompt: raw.system_prompt || "",
    rules: (raw.rules || []).map((r: any) => ({ title: r.title, updateRule: r.update_rule })),
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
  const apiKey = ref("");
  const modelId = ref("");
  const baseUrl = ref("");
  const reasoningEnabled = ref(false);
  const settingsState = ref<PanelState>("closed");
  const settingsContentVisible = ref(false);
  const settingsBtnRef = ref<HTMLButtonElement | null>(null);
  const settingsTitleRef = ref<HTMLHeadingElement | null>(null);
  const settingsBtnRect = ref({ top: 0, left: 0, width: 0, height: 0 });
  const settingsTitleRect = ref({ top: 0, left: 0, width: 0, height: 0 });

  // Create/Edit form
  const editPack = ref<RulePack>({
    id: "", name: "", description: "", author: "", version: "1.0.0",
    systemPrompt: "", rules: [], tags: [], createdAt: "", updatedAt: "",
  });
  const generating = ref(false);

  onMounted(() => {
    loadConfig();
    loadPacks();
    loadInstalled();
  });

  async function loadConfig() {
    try {
      const config = await invoke<{ api_key: string; model_id: string; base_url: string; reasoning_enabled: boolean }>("load_config");
      if (config) {
        apiKey.value = config.api_key;
        if (config.model_id) modelId.value = config.model_id;
        if (config.base_url) baseUrl.value = config.base_url;
        reasoningEnabled.value = config.reasoning_enabled;
      }
    } catch (e) {
      console.error("Failed to load config:", e);
    }
  }

  async function saveConfig() {
    try {
      await invoke("save_config", {
        apiKey: apiKey.value,
        modelId: modelId.value,
        baseUrl: baseUrl.value,
        reasoningEnabled: reasoningEnabled.value,
      });
    } catch (e) {
      console.error("Failed to save config:", e);
    }
  }

  watch([apiKey, modelId, baseUrl, reasoningEnabled], () => { saveConfig(); });

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
    let result = packs.value;
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
    const tags = new Set<string>();
    packs.value.forEach(p => p.tags.forEach(t => tags.add(t)));
    return Array.from(tags).sort();
  });

  function startCreate() {
    editPack.value = {
      id: generateId(), name: "", description: "", author: "", version: "1.0.0",
      systemPrompt: "", rules: [], tags: [], createdAt: nowISO(), updatedAt: nowISO(),
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

  // AI-powered rule generation
  async function generateRules(topic: string) {
    if (!apiKey.value || !modelId.value || generating.value) return;
    generating.value = true;

    const prompt = `You are a MemoChat rule pack designer. Given a topic, generate a set of memo rules.

Topic: ${topic}

Generate a JSON object with:
- "name": short pack name
- "description": one-line description
- "systemPrompt": a system prompt for the AI assistant
- "rules": array of {"title": "rule title", "updateRule": "instruction for how to update this memo"}
- "tags": array of relevant tags

Output ONLY valid JSON, no markdown fences.`;

    try {
      const config = { baseUrl: baseUrl.value, apiKey: apiKey.value, modelId: modelId.value, reasoningEnabled: reasoningEnabled.value };
      const result = await chatCompletion([{ role: "user", content: prompt }], config);
      const cleaned = result.content.replace(/```json\n?/g, "").replace(/```\n?/g, "").trim();
      const parsed = JSON.parse(cleaned);

      editPack.value.name = parsed.name || editPack.value.name;
      editPack.value.description = parsed.description || "";
      editPack.value.systemPrompt = parsed.systemPrompt || "";
      editPack.value.rules = (parsed.rules || []).map((r: any) => ({
        title: r.title || "", updateRule: r.updateRule || r.update_rule || "",
      }));
      editPack.value.tags = parsed.tags || [];
    } catch (e) {
      console.error("AI generation failed:", e);
    } finally {
      generating.value = false;
    }
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

  // Export as full MemoMarket pack
  function exportPack(pack: RulePack) {
    const json = JSON.stringify(toBackend(pack), null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${pack.name.replace(/\s+/g, "-").toLowerCase() || "pack"}.memomarket.json`;
    a.click();
    URL.revokeObjectURL(url);
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

  return {
    packs, installedIds, currentView, selectedPack, searchQuery, filterTag,
    filteredPacks, allTags,
    apiKey, modelId, baseUrl, reasoningEnabled,
    settingsState, settingsContentVisible,
    settingsBtnRef, settingsTitleRef, settingsBtnRect, settingsTitleRect,
    editPack, generating,
    loadPacks, savePack, deletePack, toggleInstall,
    startCreate, startEdit, addRule, removeRule, addTag, removeTag,
    saveCurrentPack, viewPack, goBack,
    generateRules,
    exportForMemoChat, exportPack, importPack,
    openSettings, closeSettings,
  };
}
